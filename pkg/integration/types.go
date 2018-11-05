package integration

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"

	integrationAPI "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

type IntegratorDisintegrator interface {
	// Integrate is where the business logic for creating the actual integration represented by the *Integration object exists
	Integrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	// DisIntegrate removes anything set up in the consuming and producing services created during integration
	DisIntegrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	// Integrates announces to the integration registry which services this integrator can handle
	Integrates() []Type
	// Validate validates the integration object and ensure everything that is needed is present
	Validate(integration *integrationAPI.Integration) error
}

type Consumer interface {
	// Exists will check to see does the consuming service exist
	Exists() bool
	// Validate will check the runtime object being consumed is valid for creating the integration
	Validate(object runtime.Object) error
	// CreateAvailableIntegration sets up and creates a new integration object
	CreateAvailableIntegration(object runtime.Object, enabled bool) error
	// RemoveAvailableIntegration removed the integration object created by CreateAvailableIntegration
	RemoveAvailableIntegration(object runtime.Object) error
	// GVKs announces to the registry which objects this consumer is interested in
	GVKs() []schema.GroupVersionKind
}

type ConsumerRegistery interface {
	ConsumersForKind(schema.GroupVersionKind) []Consumer
	RegisterConsumer(Consumer)
}

type IntegratorRegistery interface {
	IntegratorFor(*integrationAPI.Integration) IntegratorDisintegrator
	RegisterIntegrator(i IntegratorDisintegrator) error
}

type Type struct {
	Provider string
	Type     string
}

func (t Type) String() string {
	return t.Provider + ":" + t.Type
}
