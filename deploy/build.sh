#!/bin/bash

# Did you run "go get github.com/tools/godep" yet?

set -e
set -x

godep go build -a github.com/jut-io/jut-cadvisor-agent

# jut-cadvisor-agent is looking for a file called ca-certificates.crt,
# hence the rename. The Dockerfile will additionally copy it into an
# appropriate location for the go program.
cp /etc/ssl/certs/ca-bundle.crt ./ca-certificates.crt

sudo docker build -t jutd/jut-cadvisor-agent:latest .
