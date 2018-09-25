# Overview

This document will walk you through trying out the integration controller.


- We will use an integreatly cluster 
- setup a user namespace
- deploy fuse
- deploy integration controller
- create enmasse address
- see it auto populate into fuse


Using a cluster that has had the integreatly installer #link run against it, provision fuse
via the catalog into a namespace take note of the namespace name. For the purpose of this doc
it will be referred to as ```<YOUR_NAMESPACE>``` 

Using an account with cluster admin perform the following steps to install the controller

- Find the fuse namespace that was created it will be something like fuse-ce3947f7-bd8f-11e8-a624-0a580a820006
- set up the rbac cluster roles
    
    ```
        oc create -f deploy/enmasse-cluster-role.yaml
        oc create -f deploy/applications/route-services-viewer-cluster-role.yaml
     ```
    
- create the service account
    ```oc create -f deploy/sa.yaml -n <FUSE_NAMESPACE>```
    
- create the needed role bindings ensure to replace the ```YOUR_FUSE_NAMESPACE``` in commands below with your own fuse namespace
    ```
        oc create -f deploy/rbac.yaml
        cat deploy/enmasse/enmasse-role-binding.yaml | sed -e 's/FUSE_NAMESPACE/YOUR_FUSE_NAMESPACE/g' | oc create -n enmasse -f -
        cat deploy/applications/route-services-viewer-role-binding.yaml | sed -e 's/FUSE_NAMESPACE/YOUR_FUSE_NAMESPACE/g' | oc create -n <YOUR_NAMESPACE> -f - 
    ```
- setup the integration crd
    ``` 
        oc create -f deploy/crd.yaml
    ```    
- deploy the controller to the fuse namespace

    ```
       oc create -f operator 
    
    ```    