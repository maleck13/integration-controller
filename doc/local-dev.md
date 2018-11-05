# Local Development


The controller expects to be running in a Kubernetes environment. There are multiple ways to get a local Kubernetes environment 
which this document wont go into. This document will focus on developing against a cluster that has been setup with the needed services
using the integreatly [installation scripts](https://github.com/integr8ly/installation)

[mini kube](https://kubernetes.io/docs/setup/minikube/)

[oc cluster up](https://github.com/openshift/origin#installation)


Once you have a local cluster. You should login to it with the system admin user. 

It is worth noting that your local environment will need the services etc installed that the integration controller will plug together.

For an integreatly environment (IE one setup using the integreatly [installation scripts](https://github.com/integr8ly/installation)) you can run the
following command

```
make setup install 
```

This will setup a namespace called ```integration-services``` this is configurable by passing in the ```NAMESPACE``` var to the makefile.
It will also setup the needed roles,rolebindings and crds etc. It will be limited to one namespace in this setup.

Once this is complete you can then run the controller by using:

``` 
make run SA_TOKEN=$(oc sa get-token integration-controller)

```

Depending on the type of integration your are working on, you may need to [port forward pods](https://docs.openshift.com/enterprise/3.0/dev_guide/port_forwarding.html) to your local machine


To work on just the k8s service to fuse online integration you can run
``` 
make install NAMESPACE=my-managed-services USER_NAMESPACE=myproject
make k8sservice-integration NAMESPACE=my-managed-services USER_NAMESPACE=myproject
export LOCAL_DEV=true
make run SA_TOKEN=$(oc sa get-token integration-controller -n project-services) NAMESPACE=project-services USER_NAMESPACE=project1
``` 