# Integration Design

## EnMasse address-space to Syndesis amqp connection

**Services**

- [EnMasse:](http://enmasse.io/) provides messaging as a service on Kubernertes / OpenShift.
- [Syndesis:](https://syndesis.io/) provides an online integration platform and can consume an amqp connection to power integrations


## Overview

In a managed environment we want to be able to automatically create integrations between services. For this integration, we want
to discover address-spaces created by a user and create a corresponding connection ready to be used in syndesis. 

## PoC approach 

### Creating the integration

- The integration controller watches configmaps in the EnMasse namespace labelled with the address-space type
- When the integration controller sees one of these, it looks in its' own namespace for an existing Syndesis resource
- When a Sysndesis resource is found, an integration object is created.
- When the integration object is enabled, the integration controller takes the following steps
    - create a new user in the EnMasse Keycloak under the realm for the address-space
    - invoke the Syndesis rest server API to create the connection


### Updating the integration

- The integration controller, during each reconciliation loop attempts to create the connection, if it already exists, then it will do an update of the connection to ensure it is always up to date

### Deleting disabling the integration

- The integration controller , when it sees an address-spaced deleted, or an integration disabled, will delete the user from keycloak and remove the connection via the Syndesis Server API    

## Post Poc / Future approach
Requires upstream changes to land first.

https://github.com/syndesisio/syndesis/issues/3692

### Creating the integration
- The integration controller watches for address-space resources in its own namespace (created via the EnMasse API Server). 
- When it sees one, it looks for a Syndesis resource in its namespace
- If found it will create an integration object
- When enabled, the integration controller will create a new custom resource to add a new user to the EnMasse Realm
- It will also crate a resource to represent the discovered connection.
- Syndesis in will see this resource and create the connection.

### Updating the connection
- The integration controller will ensure the connection resource is kept up to date
- Sysdesis will ensure the created connection is up to date with the resource representation


### Deleting the connection
- The integration controller , when it sees an address-space deleted, or an integration disabled, will delete the user custom resources and the connection custom resource
- EnMasse will clean up the user from keycloak
- Syndesis will clean up the connection 
