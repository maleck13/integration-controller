# Integration Design

## Auto discovery of services and routes that can become connections in Syndesis

## Services / Resources Involved

Syndesis

OpenShift Routes / K8s services

## Use Case

As a developer working with OpenShift on applications I intend to be used by Syndesis, I would like a k8s native way to indicate that a service or route I have
created should be available as a http(s) connection in Syndesis. I should also be able to indicate that I have a swagger definition
and its location, in which case I would expect and API connection in Syndesis. I should not need to deploy this application to the same
namespace as my Syndesis instance.

# Proposed Approach

## Resource Annotations and Labels
There is an existing proposal for service discovery. We should follow its guidelines for defining annotations

[service discovery proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/network/service-discovery.md) 
    
## A note on watching user namespaces
- The integration controller lives within a "managed-services-x" namespace. There could by many of these namespaces. 
- The concept behind these managed service namespaces, is that they are provisioned by an admin on behalf of a team or user (either by manually setting it up
or by the creation of another Custom Resource that will trigger an operator to setup one of these namespaces with all the pieces needed)
- Each of these managed services namespaces contains the custom resources that define each of the services or the integration controller
knows in which namespaces to find these resources.
- In order to watch user namespaces, the integration controller will be configured with a list
of namespaces to watch. Once configured RoleBindings will be assumed to have been setup to allow to get and list certain resources. In this case it would
be services and routes 

## Creating the connection

- When the integration controller sees one of these resources (service or route), it will look for an existing Syndesis resource in its managed services namespace 
- If found it will create an integration resource in the users' namespace. If it is a route, it will not create one for the service that route
is referencing
- In the case of a route, the integration controller should be able to figure out if a http or https connection is needed based on the route
definition.
- This integration resource will have placeholders for  certain parameters such as the location of a swagger definition
- If one of the above specified annotations exists on the service or route resource, the integration will be automatically enabled and the parameters automatically filled.
- a connection resource (initially a configmap see https://github.com/syndesisio/syndesis/issues/3692) will then be created by
the integration controller for Sydnesis to act on.
- *Note* a connection to a service between namespaces will only work if the networking policies allow this communication. (this may be something that 
the integration controller could check before creating the integration based on a service)

**Possible Connection Resource Example**

```json 
{
   "connectionType": "api",
   "url": "http://someservice/swagger.json",
   "credentials":{},
   "certs":{}
}
```
## Updating the connection

During each reconciliation loop, the integration controller would ensure the connection resource was up to date.
Syndesis server would ensure the connection it had created reflected that state.  


## Removing the connection

If the route or service is deleted from one of the watched namespaces, the integration object will also be deleted via owner ref.
Once the integration operator sees this it will ensure the configmap is also removed. Can't be done by owner refs at the moment as
they don't cross namespaces. 
Syndesis will need to see that this has been removed and remove the associated connection.


## Work Flows

1)
- Developer using integreatly cluster deploys a new application to namespace being watched by the integration controller
- Developer uses oc tool or UI to see what available integrations there are.
- Developer enabled Fuse Online integration and fills in any needed params
- Connection shows up in the Fuse Console

2)
- Developer is using an integreatly cluster and knows about the annotations. 
- He adds the needed annotations and deploys the application
- Connection shows up in the Fuse Online console.