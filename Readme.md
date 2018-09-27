# Integration Controller

Knows how to enable various integrations between certain sets of products.

Things it can currently integrate

- EnMasse address-spaces to fuse online as connections

# Design concept
Integrations are made up of consumers and integrators these are the core interfaces in the controller:

[consumer](https://github.com/integr8ly/integration-controller/blob/master/pkg/integration/types.go#L29)
[integrator_disintegrator](https://github.com/integr8ly/integration-controller/blob/master/pkg/integration/types.go#L22)

The idea behind the integration controller is to be a dumb pipe. IE when it sees a resource it knows about, it looks for consumers that can integrate with what that resource represents. The controller checks for the existence of a consumer and then creates an ```integration``` resource. **Ideally** once the integration resouce is enabled, the integration controller will hand off to an ```integrator``` which may create a new custom resource and hand off the complexities of the api integration to another operator or it may also call the services API directly to set up the integration (this is the case between enmasse and fuse for example). Once the right CRDs are put in place in the upstream dependencies it would then be delegated to another operator and the API specific code removed.

Example of a consumer integrator/disintegrator implementation [fuse online](https://github.com/integr8ly/integration-controller/tree/master/pkg/fuse) in the code base.

In this example there are alot of direct api calls. However the hope is to remove those and hand them off to each services operator



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
