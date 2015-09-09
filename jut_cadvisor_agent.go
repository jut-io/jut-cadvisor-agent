// Copyright (c) 2015 Jut, Inc. <www.jut.io>
//
// Program to poll from a cadvisor instance, transform those metrics
// to a form suitable for ingest by a Jut data node, and send them to
// the jut data node.

// TODO:
// - DONE Find a place to check it in
// - DONE Make sure I have the go code organized properly
// - DONE Start sending to raw connector
// - DONE Add command line arg parsing
// - DONE Add support for "minimal" metrics
// - DONE Fill in README.md
// - DONE make repository public
// - DONE Create docker hub account, get it downloadable there
// - DONE (handled that via godeps, to at least fix the version) change build to not just pull master of all github modules
// - DONE Add an argument for polling interval
// - Create some sample graphs:
//     - DONE stacked cpu usage for all containers
//     - DONE pie chart of cpu usage
//     - DONE stacked memory usage for all containers
//     - DONE pie chart of memory usage
//     - stacked network activity for all containers (blocked on a new cadvisor release)
//     - pie chart of network activity
//     - capacity management
// - Get automated builds working for docker hub account
// - Write docker integration confluence page.
// - Add support for fetching logs
// - Test for filesystem, network iface, stats that don't show up by default
// - Performance test
// - Report metrics when the script itself is having problems
// - decide whether or not to grab non-metrics stuff like configuraton, etc.

package main

import (
        "os"
        "flag"
        "time"
        "bytes"
        "net/http"
        "net/url"
        "encoding/json"
        "crypto/tls"

        "github.com/golang/glog"
        "github.com/google/cadvisor/client"
        info "github.com/google/cadvisor/info/v1"
)

type Config struct {
        Apikey string
        CadvisorUrl string
        Datanode string
        AllowInsecureSsl bool
        FullMetrics bool
        PollInterval uint
}

var config Config

type DataPointList []interface{}

type DataPointHeader struct {
        Time time.Time `json:"time"`
        ContainerName string  `json:"container_name"`
        ContainerAlias string  `json:"container_alias"`
        SourceType string `json:"source_type"`
}

type DataPoint struct {
        DataPointHeader
        Name string `json:"name"`
        Value uint64 `json:"value"`
}

type PerCpuDataPoint struct {
        DataPoint
        Cpuid uint `json:"cpu_id"`
}

type PerDiskDataPoint struct {
        DataPoint
        Major uint64 `json:"major"`
        Minor uint64 `json:"minor"`
}

type PerIfaceDataPoint struct {
        DataPoint
        Iface string `json:"iface"`
}

type PerFilesystemDataPoint struct {
        DataPoint
        Device string `json:"device"`
}

func addPerDiskDataPoints(hdr *DataPointHeader, perDiskInfos []info.PerDiskStats, statPrefix string) DataPointList {

        var dataPoints DataPointList
        var metrics = []string{"Async", "Read", "Sync", "Total", "Write"}

        for _, perDiskInfo := range perDiskInfos {
                for _, metric := range metrics {
                        dataPoints = append(dataPoints,
                                &PerDiskDataPoint{DataPoint{*hdr,
                                        statPrefix + "." + metric,
                                        perDiskInfo.Stats[metric]},
                                        perDiskInfo.Major,
                                        perDiskInfo.Minor})
                }
        }

        return dataPoints
}

func addIfaceDataPoints(hdr *DataPointHeader, ifaceStat info.InterfaceStats, statPrefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints,
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".rx_bytes", ifaceStat.RxBytes}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".rx_packets", ifaceStat.RxPackets}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".rx_errors", ifaceStat.RxErrors}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".rx_dropped", ifaceStat.RxDropped}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".tx_bytes", ifaceStat.TxBytes}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".tx_packets", ifaceStat.TxPackets}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".tx_errors", ifaceStat.TxErrors}, ifaceStat.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statPrefix + ".tx_dropped", ifaceStat.TxDropped}, ifaceStat.Name},
        )

        return dataPoints
}

func addMemoryDataPoints(hdr *DataPointHeader, memData info.MemoryStatsMemoryData, statPrefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints, &DataPoint{*hdr, statPrefix + "." + "pgfault", memData.Pgfault})
        dataPoints = append(dataPoints, &DataPoint{*hdr, statPrefix + "." + "pgmajfault", memData.Pgmajfault})

        return dataPoints
}

func addFilesystemDataPoints(hdr *DataPointHeader, fsStat info.FsStats, statPrefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints,
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".capacity", fsStat.Limit}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".usage", fsStat.Usage}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".available", fsStat.Available}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".reads_completed", fsStat.ReadsCompleted}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".reads_merged", fsStat.ReadsMerged}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".sectors_read", fsStat.SectorsRead}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".read_time", fsStat.ReadTime}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".writes_completed", fsStat.WritesCompleted}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".writes_merged", fsStat.WritesMerged}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".sectors_written", fsStat.SectorsWritten}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".write_time", fsStat.WriteTime}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".io_in_progress", fsStat.IoInProgress}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".io_time", fsStat.IoTime}, fsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statPrefix + ".weighted_io_time", fsStat.WeightedIoTime}, fsStat.Device},
        )

        return dataPoints
}

func allDataPoints(info info.ContainerInfo) DataPointList {

        stat := info.Stats[0]

        var dataPoints DataPointList

        hdr := &DataPointHeader{stat.Timestamp, info.Name, info.Aliases[0], "metric"}

        dataPoints = append(dataPoints,
                &DataPoint{*hdr, "cpu.usage.total", stat.Cpu.Usage.Total},
                &DataPoint{*hdr, "cpu.usage.user", stat.Cpu.Usage.User},
                &DataPoint{*hdr, "cpu.usage.system", stat.Cpu.Usage.System},
                &DataPoint{*hdr, "cpu.load_average", uint64(stat.Cpu.LoadAverage)},
        )

        if (config.FullMetrics) {
                for idx, perCpuInfo := range stat.Cpu.Usage.PerCpu {
                        dataPoints = append(dataPoints, &PerCpuDataPoint{DataPoint{*hdr, "cpu.usage.per-cpu", perCpuInfo}, uint(idx)})
                }

                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoServiceBytes, "diskio.io_service_bytes")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoServiced, "diskio.io_serviced")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoQueued, "diskio.io_queued")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.Sectors, "diskio.sectors")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoServiceTime, "diskio.io_service_time")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoWaitTime, "diskio.io_wait_time")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoMerged, "diskio.io_merged")...)
                dataPoints = append(dataPoints, addPerDiskDataPoints(hdr, stat.DiskIo.IoTime, "diskio.io_time")...)
        }

        dataPoints = append(dataPoints,
                &DataPoint{*hdr, "memory.usage", stat.Memory.Usage},
                &DataPoint{*hdr, "memory.working_set", stat.Memory.WorkingSet},
        )

        if (config.FullMetrics) {
                dataPoints = append(dataPoints, addMemoryDataPoints(hdr, stat.Memory.ContainerData, "memory.container_data")...)
                dataPoints = append(dataPoints, addMemoryDataPoints(hdr, stat.Memory.HierarchicalData, "memory.hierarchical_data")...)
        }

        dataPoints = append(dataPoints, addIfaceDataPoints(hdr, stat.Network.InterfaceStats, "stat.network")...)

        if (config.FullMetrics) {
                for _, ifaceStat := range stat.Network.Interfaces {
                        dataPoints = append(dataPoints, addIfaceDataPoints(hdr, ifaceStat, "stat.network")...)
                }
        }

        if (config.FullMetrics) {
                for _, fsStat := range stat.Filesystem {
                        dataPoints = append(dataPoints, addFilesystemDataPoints(hdr, fsStat, "stat.fs")...)
                }

                dataPoints = append(dataPoints,
                        &DataPoint{*hdr, "task.nr_sleeping", stat.TaskStats.NrSleeping},
                        &DataPoint{*hdr, "task.nr_running", stat.TaskStats.NrRunning},
                        &DataPoint{*hdr, "task.nr_stopped", stat.TaskStats.NrStopped},
                        &DataPoint{*hdr, "task.nr_uninterruptible", stat.TaskStats.NrUninterruptible},
                        &DataPoint{*hdr, "task.nr_io_wait", stat.TaskStats.NrIoWait},
                )
        }
        return dataPoints
}


func collectMetrics(cURL *url.URL, dnURL *url.URL) {

        glog.Info("Collecting Metrics")

        cAdvisorClient, err := client.NewClient(cURL.String())
        if err != nil {
                glog.Errorf("tried to make cAdvisor client and got error: %v", err);
                return
        }

        request := &info.ContainerInfoRequest{
                NumStats: 1,
        }

        cInfos, err := cAdvisorClient.AllDockerContainers(request)

        if err != nil {
                glog.Errorf("unable to get info on all docker containers: %v", err)
                return
        }

        var dataPoints DataPointList

        for _, info := range cInfos {
                dataPoints = append(dataPoints, allDataPoints(info)...)
        }
        jsonDataPoints, err := json.Marshal(dataPoints)

        if err != nil {
                glog.Errorf("Unable to construct JSON metrics: %v", err)
                return
        }

        glog.V(3).Infof("About to send metrics: %v", string(jsonDataPoints))

        tr := &http.Transport{
                TLSClientConfig: &tls.Config{InsecureSkipVerify: config.AllowInsecureSsl},
        }

        dataNodeClient := &http.Client{Transport: tr}

        resp, err := dataNodeClient.Post(dnURL.String(),
                "application/json",
                bytes.NewBuffer(jsonDataPoints))

        if err != nil {
                glog.Errorf("Unable to send metrics to Jut Data Node: %v", err)
                return
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
                glog.Errorf("Unable to send metrics to Jut Data Node: %v", resp.Status)
                return
        }
}

func checkNonEmpty(arg string, argName string) {
        if arg == "" {
                os.Stderr.WriteString("Argument " + argName + " must be provided. Usage:\n")
                flag.PrintDefaults();
                os.Exit(1)
        }
}


func main() {

        flag.StringVar(&config.Apikey, "apikey", "", "Jut Data Engine API Key")
        flag.StringVar(&config.CadvisorUrl, "cadvisor_url", "http://127.0.0.1:8080", "cAdvisor Root URL")
        flag.StringVar(&config.Datanode, "datanode", "", "Jut Data Node Hostname")
        flag.BoolVar(&config.AllowInsecureSsl, "allow_insecure_ssl", false, "Allow insecure certificates when connecting to Jut Data Node")
        flag.BoolVar(&config.FullMetrics, "full_metrics", false, "Collect and transmit full set of metrics from containers")
        flag.UintVar(&config.PollInterval, "poll_interval", 30, "Polling Interval (seconds)")

        flag.Parse()

        checkNonEmpty(config.Apikey, "apikey")
        checkNonEmpty(config.CadvisorUrl, "cadvisor_url")
        checkNonEmpty(config.Datanode, "datanode")

        cURL, err := url.Parse(config.CadvisorUrl)

        if err != nil {
                glog.Fatal(err)
        }

        urlstr := "https://" + config.Datanode + ":3110/api/v1/import/docker?apikey=" + config.Apikey + "&data_source=docker";
        glog.V(2).Info("Full data node url: " + urlstr)
        dnURL, err := url.Parse(urlstr)

        if err != nil {
                glog.Fatal(err)
        }

        for true {
                collectMetrics(cURL, dnURL)
                time.Sleep(time.Duration(config.PollInterval) * time.Second)
        }
}
