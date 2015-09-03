// Program to poll from a cadvisor instance, transform those metrics
// to a form suitable for ingest by a Jut data node, and send them to
// the jut data node.

// TODO:
//  Choose between these options:
//    - http fetch of json + untyped parsing + flattening of stats section
//    - google client fetch of json + typed parsing + flattening of stats section
//    - google client fetch of json + typed parsing + typed conversion to our metrics
//      Q: for these, how do I fetch just 1 stat? !: do a POST instead of a get, with an application/json body like: {"num_stats":1,"start":"0001-01-01T00:00:00Z","end":"0001-01-01T00:00:00Z"}
//  events fetch
//  do I want to grab non-metrics stuff like configuraton, etc.
//  do I want to grab container information or just docker

package main

import (
	"flag"
        //        "fmt"
        "time"
        "bytes"
        "net/http"
        "encoding/json"
        "crypto/tls"

	log "github.com/Sirupsen/logrus"
        "github.com/google/cadvisor/client"
        info "github.com/google/cadvisor/info/v1"
)

type DataPointList []interface{}

// Things to fetch:
//   - All docker container info, under docker endpoint:
//   - All events, under events endpoint:

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

func (hdr DataPointHeader) CreateDataPoint(name string, value uint64) *DataPoint {
        return &DataPoint{hdr, name, value}
}

func addPerDiskDataPoints(hdr *DataPointHeader, perDiskInfos []info.PerDiskStats, statprefix string) DataPointList {

        var dataPoints DataPointList
        var metrics = []string{"Async", "Read", "Sync", "Total", "Write"}

        for _, perDiskInfo := range perDiskInfos {
                for _, metric := range metrics {
                        dataPoints = append(dataPoints,
                                &PerDiskDataPoint{DataPoint{*hdr,
                                        statprefix + "." + metric,
                                        perDiskInfo.Stats[metric]},
                                        perDiskInfo.Major,
                                        perDiskInfo.Minor})
                }
        }

        return dataPoints
}

func addIfaceDataPoints(hdr *DataPointHeader, IfaceStats info.InterfaceStats, statprefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints,
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".rx_bytes", IfaceStats.RxBytes}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".rx_packets", IfaceStats.RxPackets}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".rx_errors", IfaceStats.RxErrors}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".rx_dropped", IfaceStats.RxDropped}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".tx_bytes", IfaceStats.TxBytes}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".tx_packets", IfaceStats.TxPackets}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".tx_errors", IfaceStats.TxErrors}, IfaceStats.Name},
                &PerIfaceDataPoint{DataPoint{*hdr, statprefix + ".tx_dropped", IfaceStats.TxDropped}, IfaceStats.Name},
        )

        return dataPoints
}

func addMemoryDataPoints(hdr *DataPointHeader, MemData info.MemoryStatsMemoryData, statprefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints, &DataPoint{*hdr, statprefix + "." + "pgfault", MemData.Pgfault})
        dataPoints = append(dataPoints, &DataPoint{*hdr, statprefix + "." + "pgmajfault", MemData.Pgmajfault})

        return dataPoints
}

func addFilesystemDataPoints(hdr *DataPointHeader, FsStat info.FsStats, statprefix string) DataPointList {

        var dataPoints DataPointList

        dataPoints = append(dataPoints,
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".capacity", FsStat.Limit}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".usage", FsStat.Usage}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".available", FsStat.Available}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".reads_completed", FsStat.ReadsCompleted}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".reads_merged", FsStat.ReadsMerged}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".sectors_read", FsStat.SectorsRead}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".read_time", FsStat.ReadTime}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".writes_completed", FsStat.WritesCompleted}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".writes_merged", FsStat.WritesMerged}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".sectors_written", FsStat.SectorsWritten}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".write_time", FsStat.WriteTime}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".io_in_progress", FsStat.IoInProgress}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".io_time", FsStat.IoTime}, FsStat.Device},
                &PerFilesystemDataPoint{DataPoint{*hdr, statprefix + ".weighted_io_time", FsStat.WeightedIoTime}, FsStat.Device},
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

        dataPoints = append(dataPoints,
                &DataPoint{*hdr, "memory.usage", stat.Memory.Usage},
                &DataPoint{*hdr, "memory.working_set", stat.Memory.WorkingSet},
        )

        dataPoints = append(dataPoints, addMemoryDataPoints(hdr, stat.Memory.ContainerData, "memory.container_data")...)
        dataPoints = append(dataPoints, addMemoryDataPoints(hdr, stat.Memory.ContainerData, "memory.heirarchical_data")...)

        dataPoints = append(dataPoints, addIfaceDataPoints(hdr, stat.Network.InterfaceStats, "stat.network")...)
        for _, IfaceStats := range stat.Network.Interfaces {
                dataPoints = append(dataPoints, addIfaceDataPoints(hdr, IfaceStats, "stat.network")...)
        }

        for _, FsStat := range stat.Filesystem {
                dataPoints = append(dataPoints, addFilesystemDataPoints(hdr, FsStat, "stat.fs")...)
        }

        dataPoints = append(dataPoints,
                &DataPoint{*hdr, "task.nr_sleeping", stat.TaskStats.NrSleeping},
                &DataPoint{*hdr, "task.nr_running", stat.TaskStats.NrRunning},
                &DataPoint{*hdr, "task.nr_stopped", stat.TaskStats.NrStopped},
                &DataPoint{*hdr, "task.nr_uninterruptible", stat.TaskStats.NrUninterruptible},
                &DataPoint{*hdr, "task.nr_io_wait", stat.TaskStats.NrIoWait},
        )
        return dataPoints
}


func collect_metrics(apikey string) {

	log.Info("Collecting Metrics");
	
        staticClient, err := client.NewClient("http://localhost:8080")
        if err != nil {
                log.WithFields(log.Fields{
			"err": err,
		}).Error("tried to make client and got error");
                return
        }

        request := &info.ContainerInfoRequest{
                NumStats: 1,
        }

        cInfos, err := staticClient.AllDockerContainers(request)

        if err != nil {
                log.WithFields(log.Fields{
			"err": err,
		}).Error("unable to get info on all docker containers")
                return
        }

        var dataPoints DataPointList

        for _, info := range cInfos {
                dataPoints = append(dataPoints, allDataPoints(info)...)
        }
        str, err := json.Marshal(dataPoints)

        if err != nil {
                log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to construct JSON metrics")
                return
        }

        log.WithFields(log.Fields{
		"metrics": str,
	}).Debug("About to send metrics")

        tr := &http.Transport{
                TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }

        client := &http.Client{Transport: tr}

        resp, err := client.Post("https://127.0.0.1:3110/api/v1/import/docker?apikey=" + apikey + "&data_source=docker",
                "application/json",
                bytes.NewBuffer(str))

        if err != nil {
                log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to send metrics to Jut Data Node")
                return
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
                log.WithFields(log.Fields{
			"err": resp.Status,
		}).Error("Unable to send metrics to Jut Data Node: %v")
                return
        }
}

func main() {

	var apikey = flag.String("apikey", "", "Jut Data Engine API Key")
	flag.Parse();
	
	for true {
		collect_metrics(*apikey)
		time.Sleep(30 * time.Second)
	}
}
