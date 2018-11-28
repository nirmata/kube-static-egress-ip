FROM alpine:3.8

MAINTAINER Jim Bugwadia <jim@nirmata.com>

ADD egressip-controller /egressip-controller

ENTRYPOINT ["/egressip-controller"]
