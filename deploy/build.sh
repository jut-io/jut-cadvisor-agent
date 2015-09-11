#!/bin/bash

# Did you run "go get github.com/tools/godep" yet?

set -e
set -x

godep go build -a github.com/jut-io/jut-cadvisor-agent

sudo docker build -t jutd/jut-cadvisor-agent:latest .
