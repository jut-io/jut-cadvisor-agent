jut-cadvisor-agent
==================

jut-cadvisor-agent is a companion program to [cAdvisor] (https://github.com/google/cadvisor), which monitors [Docker] (https://www.docker.com/) containers. It runs alongside cAdvisor, polling it for metrics/events, and sends those metrics/events to a Jut Data Node.

For more information on Jut Data Nodes and how to use the metrics and events collected from Docket containers, please visit our [Web Site] (http://www.jut.io) or read our [Documentation] (http://docs.jut.io).

Usage
-----
jut-cadvisor-agent is typically run as follows:

     jut-cadvisor-agent --apikey=<apikey> --datanode=<jut data node hostname>

*apikey* is a string specific to your deployment. The *jut data node
 hostname* is a hostname of the form
 `data-engine-192-168-37-12.jutdata.io`.

Configuration
=============

jut-cadvisor-agent can be configured from the command line or the
environment. Command line arguments take precedence over environment
variables.

###Environment Variables

The following environment variables control the behavior of jut-cadvisor-agent:

`JUT_APIKEY` - Jut Data Engine API Key. Must be provided in environment or on command line.<br>
`JUT_CADVISOR_URL` - cAdvisor Root URL.<br>
`JUT_DATA_SOURCE` - Data Source to use for points.<br>
`JUT_DATANODE` - Jut Data Node Hostname. Must be provided in environment or on command line.<br>
`JUT_POLL_INTERVAL` - How often to poll containers (seconds).<br>
`JUT_ALLOW_INSECURE_SSL` - Allow insecure certificates when connecting to Jut Data Node.<br>
`JUT_METRICS` - Collect Metrics from cAdvisor and send to Data Node.<br>
`JUT_EVENTS` - Collect Events from cAdvisor and send to Data Node.<br>
`JUT_FULL_METRICS` - Collect and report full set of metrics from containers.<br>
`JUT_SPACE` - Jut Space to use for points.<br>

###Command Line Arguments

The following command line arguments correspond to the above
environment variables. Also included are their default values.

`--apikey=""`<br>
`--cadvisor_url=http://127.0.0.1:8080`<br>
`--data_source="docker"`<br>
`--datanode=""`<br>
`--poll_interval=10`<br>
`--allow_insecure_ssl=false`<br>
`--metrics=true`<br>
`--events=true`<br>
`--full_metrics=false`<br>
`--space="default"`<br>

Additionally, the following command line arguments can be provided to
control logging behavior. They do not have corresponding environment
variables.

`--alsologtostderr=true` - log to standard error as well as files.<br>
`--log_backtrace_at=:0:` - when logging hits line file:N, emit a stack trace.<br>
`--log_dir=""` - If non-empty, write log files in this directory.<br>
`--logtostderr=false` - log to standard error instead of files.<br>
`--stderrthreshold=0` - logs at or above this threshold go to stderr.<br>
`--v=0` - log level for V logs.<br>
`--vmodule` - comma-separated list of pattern=N settings for file-filtered logging.<br>

jut-cadvisor-agent can also be run from Docker Hub. Assuming you have
cAdvisor running in a container named "cadvisor", this `docker run`
command will start jut-cadvisor-agent to poll cAdvisor and send
metrics/events to your jut Data Node:

     docker run --detach=true --name jut-cadvisor-agent \
     --net=container:cadvisor jutd/jut-cadvisor-agent:latest \
     --apikey=<apikey> --datanode=<jut data node hostname>

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
jut-cadvisor-agent uses [glog] (https://github.com/golang/glog) for logging. When run inside a container, jut-cadvisor-agent can be configured to log to standard error.











