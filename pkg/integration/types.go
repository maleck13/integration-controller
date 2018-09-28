package integration

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"

	integrationAPI "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

type IntegratorDisintegrator interface {
	Integrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	DisIntegrate(context.Context, *integrationAPI.Integration) (*integrationAPI.Integration, error)
	Integrates() string
	Validate(integration *integrationAPI.Integration) error
}

type Consumer interface {
	Exists() bool
	Validate(object runtime.Object) error
	CreateAvailableIntegration(object runtime.Object, targetNS string, enabled bool) error
	RemoveAvailableIntegration(object runtime.Object, targetNS string) error
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
