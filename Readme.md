# Integration Controller

Knows how to enable various integrations between certain sets of products.

Things it can currently integrate

- EnMasse address-spaces to fuse online as connections

# Status
POC

# Test it locally

*Note*: You will need a running Kubernetes or OpenShift cluster to use the Operator

- clone this repo to `$GOPATH/src/github.com/integr8ly/integration-controller`
- run `make setup install run`

Note that you only need to run `setup` the first time. After that you can simply run `make run`.

You should see something like:

```go
INFO[0000] Go Version: go1.10.2
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git

```



# Tear it down

```make uninstall```
