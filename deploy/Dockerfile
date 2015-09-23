FROM progrium/busybox
MAINTAINER mstemm@jut.io

# Grab the CA certificates list. This should have been copied into the
# current directory by build.sh.
ADD ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Grab jut-cadvisor-agent from the staging directory.
ADD jut-cadvisor-agent /usr/bin/jut-cadvisor-agent

ENTRYPOINT ["/usr/bin/jut-cadvisor-agent"]