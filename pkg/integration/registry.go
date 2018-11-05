package integration

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

type integrators struct {
	integratorMap map[Type]IntegratorDisintegrator
	*sync.Mutex
}

var registered = &integrators{integratorMap: map[Type]IntegratorDisintegrator{}, Mutex: &sync.Mutex{}}

type Registry struct {
}

func (r Registry) RegisterIntegrator(i IntegratorDisintegrator) error {

	registered.Lock()
	defer registered.Unlock()
	keys := i.Integrates()
	for _, key := range keys {
		if _, ok := registered.integratorMap[key]; ok {
			return errors.New("duplicate entry for " + key.String() + "integrator")
		}
		registered.integratorMap[key] = i
	}
	return nil
}

func (r Registry) IntegratorFor(i *v1alpha1.Integration) IntegratorDisintegrator {
	key := Type{Type: i.Spec.IntegrationType, Provider: i.Spec.ServiceProvider}
	logrus.Debug("integrator for ", key)
	it := registered.integratorMap[key]
	return it
}
