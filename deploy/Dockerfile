FROM progrium/busybox
MAINTAINER mstemm@jut.io

# Grab jut-cadvisor-agent from the staging directory.
ADD jut-cadvisor-agent /usr/bin/jut-cadvisor-agent

ENTRYPOINT ["/usr/bin/jut-cadvisor-agent"]