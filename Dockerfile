FROM alpine:3.8

MAINTAINER Jim Bugwadia <jim@nirmata.com>

RUN apk add --no-cache \
      iptables \
      ipset \
      iproute2

ADD egressip-controller /usr/local/bin//egressip-controller

ENTRYPOINT ["/usr/local/bin//egressip-controller"]
