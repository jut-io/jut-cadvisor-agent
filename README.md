jut-cadvisor-agent
==================

jut-cadvisor-agent is a companion program to [cAdvisor] (https://github.com/google/cadvisor), which monitors [Docker] (https://www.docker.com/) containers. It runs alongside cAdvisor, polling it for metrics/logs, and sends those metrics/events to a Jut Data Node.

For more information on Jut Data Nodes and how to use the metrics and events collected from Docket containers, please visit our [Web Site] (http://www.jut.io) or read our [Documentation] (http://docs.jut.io).

Usage
-----
jut-cadvisor-agent is typically run as follows:

      jut-cadvisor-agent --apikey=XXXXXX --datanode=data-engine-192-168-37-12.jutdata.io

The full set of command line arguments is:

`apikey=""` - Jut Data Engine API Key. Must be provided.<br>
`cadvisor_url=http://127.0.0.1:8080` - cAdvisor Root URL.<br>
`datanode=""` - Jut Data Node Hostname. Must be provided.<br>
`allow_insecure_ssl=false` - Allow insecure certificates when connecting to Jut Data Node.<br>
`alsologtostderr=false` - log to standard error as well as files.<br>
`full_metrics=false` - Collect and report full set of metrics from containers.<br>
`log_backtrace_at=:0:` - when logging hits line file:N, emit a stack trace.<br>
`log_dir=""` - If non-empty, write log files in this directory.<br>
`logtostderr=false` - log to standard error instead of files.<br>
`stderrthreshold=0` - logs at or above this threshold go to stderr.<br>
`v=0` - log level for V logs.<br>
`vmodule` - comma-separated list of pattern=N settings for file-filtered logging.<br>

jut-cadvisor-agent can (soon, not yet) also be run from Docker Hub:

      docker run jut-io/jut-cadvisor-agent:latest

Metrics Collected
-----------------
jut-cadvisor-agent can be configured to collect and transmit a default set of important metrics for each container, or a much larger set of full metrics.

The default metrics include:

* CPU: total/user/system, load_average
* Memory: usage/working set
* Network: bytes/packets sent overall (not per-interface)

The Full Set of metrics adds the following:

* CPU: per-cpu usage
* Details on disk I/O utilization
* Memory: page fault counts
* Network: per-interface information
* Filesystem: per-filesystem I/O information
* task information (sleeping/running/etc)

Logging
-------
jut-cadvisor-agent uses [glog] (https://github.com/golang/glog) for logging. When run inside a container, jut-cadvisor-agent is configured to log to standard error.











