#!/bin/bash

set -e
set -x

godep go build -a github.com/jut-io/jut-cadvisor-agent

docker build -t jut-io/jut-cadvisor-agent:latest .
