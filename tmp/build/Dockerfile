FROM alpine:3.6

RUN adduser -D integration-controller
USER integration-controller

ADD tmp/_output/bin/integration-controller /usr/local/bin/integration-controller
