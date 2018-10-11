package fuse

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/integr8ly/integration-controller/pkg/integration"
)

type HTTPIntegrator struct {
	serviceClient      Crudler
	fuseConnectionCrud FuseConnectionCruder
}

func NewHTTPIntegrator(serviceClient Crudler, fuseCruder FuseConnectionCruder) *HTTPIntegrator {
	return &HTTPIntegrator{
		serviceClient:      serviceClient,
		fuseConnectionCrud: fuseCruder,
	}
}

func (si *HTTPIntegrator) Integrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Info("handing integration ", integration.Spec.ServiceProvider, integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	// add service discovery annotations to the service
	// get the service based on the discovery_resource

	// if the port is empty then it is a route
	//todo consider adding the service port anyway?
	if integration.Status.IntegrationMetaData["port"] != "" {
		if err := si.updateServiceWithDiscovery(integration.Status.IntegrationMetaData, integration.Status.DiscoveryResource); err != nil {
			return ic, err
		}
	}
	// add connection to fuse
	//for http we need to have the host the port and the path, if the port is 0 then leave it out
	url := fmt.Sprintf("%s://%s%s%s", integration.Status.IntegrationMetaData["scheme"], integration.Status.IntegrationMetaData["host"], integration.Status.IntegrationMetaData["api-path"], integration.Status.IntegrationMetaData["port"])
	id, err := si.fuseConnectionCrud.CreateConnection(integration.Spec.IntegrationType, integration.Name, "", "", url)
	if err != nil {
		return ic, errors.Wrap(err, "failed to create connection in fuse")
	}
	ic.Status.IntegrationMetaData[connectionIDKey] = id
	ic.Status.Phase = v1alpha1.PhaseComplete
	return ic, nil
}

func (si *HTTPIntegrator) updateServiceWithDiscovery(integrationMeta map[string]string, resource v1alpha1.DiscoveryResource) error {
	into := &v1.Service{
		ObjectMeta: v12.ObjectMeta{
			Name: resource.Name,

			Namespace: resource.Namespace,
		},
		TypeMeta: v12.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}
	if err := si.serviceClient.Get(into, sdk.WithGetOptions(&v12.GetOptions{})); err != nil {
		return errors.Wrap(err, "failed to find the service backing the integration")
	}

	if into.Annotations == nil {
		into.Annotations = map[string]string{}
	}
	into.Annotations[serviceDiscoveryPort] = integrationMeta["port"]
	into.Annotations[serviceDiscoveryScheme] = integrationMeta["scheme"]
	if v, ok := integrationMeta["description-doc-path"]; ok {
		into.Annotations[serviceDiscoveryDoc] = v
	}
	if v, ok := integrationMeta["api-path"]; ok {
		into.Annotations[serviceDiscoveryPath] = v
	}

	if err := si.serviceClient.Update(into); err != nil {
		return errors.Wrap(err, "failed to add service discovery annotations")
	}
	return nil
}

func (si *HTTPIntegrator) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	if err := si.fuseConnectionCrud.DeleteConnection(ic.Status.IntegrationMetaData["scheme"], ic.Status.IntegrationMetaData[connectionIDKey]); err != nil {
		return ic, errors.Wrap(err, "failed to remove connection")
	}
	return integration, nil
}

func (si *HTTPIntegrator) Validate(integration *v1alpha1.Integration) error {
	if integration.Status.IntegrationMetaData == nil {
		return errors.New("no integration meta data")
	}
	if integration.Status.IntegrationMetaData["scheme"] == "" {
		return errors.New("no service discovery scheme found")
	}
	if integration.Status.IntegrationMetaData["host"] == "" {
		return errors.New("no service discovery host found")
	}
	return nil
}

func (si *HTTPIntegrator) Integrates() []integration.Type {
	return []integration.Type{
		{
			Type:     "http",
			Provider: v1alpha1.FuseIntegrationTarget,
		},
		{
			Type:     "https",
			Provider: v1alpha1.FuseIntegrationTarget,
		},
	}
}
