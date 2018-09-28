package integration

import (
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

type integrators struct {
	integratorMap map[string]IntegratorDisintegrator
	*sync.Mutex
}

var registered = &integrators{integratorMap: map[string]IntegratorDisintegrator{}, Mutex: &sync.Mutex{}}

type Registry struct {
}

func (r Registry) RegisterIntegrator(i IntegratorDisintegrator) error {

	registered.Lock()
	defer registered.Unlock()
	key := i.Integrates()
	if _, ok := registered.integratorMap[key]; ok {
		return errors.New("duplicate entry for " + key + "integrator")
	}
	registered.integratorMap[key] = i
	return nil
}

func (r Registry) IntegratorFor(i *v1alpha1.Integration) IntegratorDisintegrator {
	key := strings.TrimSpace(i.Spec.ServiceProvider)
	it := registered.integratorMap[key]
	logrus.Info("registered ", registered, key, it)
	return it
}
