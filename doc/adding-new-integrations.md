# Adding a new integration

There are two main interfaces that control managing and creating integrations ```IntegratorDisintegrator``` and ```Consumer```

```go
type IntegratorDisintegrator interface {
	// Integrate is where the business logic for creating the actual integration represented by the *Integration object exists
	Integrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	// DisIntegrate removes anything set up in the consuming and producing services created during integration
	DisIntegrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	// Integrates announces to the integration registry which services this integrator can handle
	Integrates() string
	// Validate validates the integration object and ensure everything that is needed is present
	Validate(integration *integrationAPI.Integration) error
}

```
[Code](https://github.com/integr8ly/integration-controller/blob/master/pkg/integration/types.go#L13)

```go

type Consumer interface {
	// Exists will check to see does the consuming service exist
	Exists() bool
	// Validate will check the runtime object being consumed is valid for creating the integration
	Validate(object runtime.Object) error
	// CreateAvailableIntegration sets up and creates a new integration object
	CreateAvailableIntegration(object runtime.Object, targetNS string, enabled bool) error
	// RemoveAvailableIntegration removed the integration object created by CreateAvailableIntegration
	RemoveAvailableIntegration(object runtime.Object, targetNS string) error
	// GVKs announces to the registry which objects this consumer is interested in
	GVKs() []schema.GroupVersionKind
}

```
[Code](https://github.com/integr8ly/integration-controller/blob/master/pkg/integration/types.go#L20)

## Add new integration for a new consumer

If there is a product or service that can consume certain services and capabilities offered by another product, then we would add
a new package for that product or service.

An example of this is the ```fuse``` package. Inside the fuse package is an implementation of a ```IntegratorDisintegrator``` and a ```Consumer```
Once your implementation is ready it can be registered in ```main.go``` in a standalone block

```go
{
    // add fuse integrations
    c := fuse.NewConsumer(...)
    consumerRegistery.RegisterConsumer(c)
    i := fuse.NewIntegrator(...)
    if err := integrationRegistery.RegisterIntegrator(i); err != nil {
        panic(err)
    }
}

```  

This should be all you need to do to add a new integration. 


## utility services

Sometimes it may be necessary to interact with not only the consuming service but also the producing service. If this is needed an interface and implementation should be created in a
package named after the producing service. 
An Example of this can be seen under the ```EnMasse``` pkg.


## Adding to an existing integration

If for example you wanted to add something new to the fuse integration, you could do this by modifying the existing consumer and integrator or by adding a new one.
The preference to would be to add a new one as shown above. 


