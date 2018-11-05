package fuse

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	v1alpha12 "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/integr8ly/integration-controller/pkg/integration"
)

type HTTPIntegrator struct {
	k8sclient          Crudler
	fuseConnectionCrud FuseClient
	targetNS           string
}

func NewHTTPIntegrator(serviceClient Crudler, fuseCruder FuseClient, targetNS string) *HTTPIntegrator {
	return &HTTPIntegrator{
		k8sclient:          serviceClient,
		fuseConnectionCrud: fuseCruder,
		targetNS:           targetNS,
	}
}

// Integrate takes the following actions.
// add service discovery annotations to the service that is the target for the integration
// if there is a connection id present in the integration object, then look to see does the connection exist
// if it exists then do an update to ensure everything is as expected
// if it doesn't exist then create the connection and store the id on the integration object
func (si *HTTPIntegrator) Integrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("** handing integration ", integration.Spec.ServiceProvider, integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	port := integration.Status.IntegrationMetaData["port"]
	if port != "" {
		if err := si.updateServiceWithDiscovery(integration.Status.IntegrationMetaData, integration.Status.DiscoveryResource); err != nil {
			return ic, err
		}
	}
	if port != "" {
		port = ":" + port
	}
	var (
		id         string
		err        error
		serviceURL string
	)
	if integration.Status.IntegrationMetaData["external-route"] == "true" {
		serviceURL = fmt.Sprintf("%s://%s%s", integration.Status.IntegrationMetaData["scheme"], integration.Status.IntegrationMetaData["host"], integration.Status.IntegrationMetaData["api-path"])
	} else {
		serviceURL = fmt.Sprintf("%s://%s%s%s", integration.Status.IntegrationMetaData["scheme"], integration.Status.IntegrationMetaData["host"], integration.Status.IntegrationMetaData["api-path"], port)
	}

	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return ic, errors.Wrap(err, "invalid url for service")
	}
	connID := ic.Status.IntegrationMetaData[connectionIDKey]
	var (
		connection = v1alpha12.NewConnection(ic.Name, ic.Namespace, ic.Spec.IntegrationType)
	)
	connection.Status.SyndesisID = connID
	connection.Spec.URL = baseUrl.String()

	switch connection.Spec.Type {
	case "http", "https":
		id, err = si.createConnectionIntegration(connection)
	case "api":
		swaggerPath, err := url.Parse(integration.Status.IntegrationMetaData["description-doc-path"])
		if err != nil {
			return ic, err
		}
		connection.Spec.URL = baseUrl.ResolveReference(swaggerPath).String()
		id, err = si.createCustomisationIntegration(connection)
		if err != nil {
			return ic, err
		}
	default:
		return ic, errors.New("unknown connection type " + connection.Spec.Type)
	}
	if err != nil {
		return ic, err
	}
	ic.Status.IntegrationMetaData[connectionIDKey] = id
	ic.Status.Phase = v1alpha1.PhaseComplete
	return ic, nil
}

func (si *HTTPIntegrator) createCustomisationIntegration(connection *v1alpha12.Connection) (string, error) {
	logrus.Debug("create customisation called")
	exists, err := si.fuseConnectionCrud.ConnectionExists(connection)
	if err != nil {
		return "", errors.Wrap(err, "could not check if the connection exists")
	}
	if exists {
		logrus.Debug("cusomisation already exists")
		return connection.Status.SyndesisID, nil
	}
	created, err := si.fuseConnectionCrud.CreateCustomisation(connection)
	if err != nil {
		return "", err
	}
	return created.Status.SyndesisID, nil
}

func (si *HTTPIntegrator) createConnectionIntegration(connection *v1alpha12.Connection) (string, error) {
	var connectionErr error
	exists, err := si.fuseConnectionCrud.ConnectionExists(connection)
	if err != nil {
		return "", errors.Wrap(err, "could not check if the connection exists")
	}
	if exists {
		logrus.Debug("connection already exists updating")
		connection, connectionErr = si.fuseConnectionCrud.UpdateConnection(connection)

	} else {
		logrus.Debug("connection does not exist creating ")
		connection, connectionErr = si.fuseConnectionCrud.CreateConnection(connection)
	}
	if connectionErr != nil {
		return "", connectionErr
	}
	return connection.Status.SyndesisID, nil

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
	if err := si.k8sclient.Get(into, sdk.WithGetOptions(&v12.GetOptions{})); err != nil {
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

	if err := si.k8sclient.Update(into); err != nil {
		return errors.Wrap(err, "failed to add service discovery annotations")
	}
	return nil
}

func (si *HTTPIntegrator) removeServiceDiscovery(resource v1alpha1.DiscoveryResource) error {
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
	if err := si.k8sclient.Get(into, sdk.WithGetOptions(&v12.GetOptions{})); err != nil {
		return errors.Wrap(err, "failed to find the service backing the integration")
	}
	if into.Annotations == nil {
		return nil
	}
	delete(into.Annotations, serviceDiscoveryPort)
	delete(into.Annotations, serviceDiscoveryScheme)
	delete(into.Annotations, serviceDiscoveryPath)
	delete(into.Annotations, serviceDiscoveryDoc)
	if err := si.k8sclient.Update(into); err != nil {
		return errors.Wrap(err, "failed to remove service discovery annotations")
	}
	return nil
}

func (si *HTTPIntegrator) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()

	connection := v1alpha12.NewConnection(ic.Name, ic.Namespace, ic.Spec.IntegrationType)
	//if err := si.k8sclient.Get(connection, sdk.WithGetOptions(&v12.GetOptions{})); err != nil {
	//	return ic, err
	//}
	connection.Status.SyndesisID = ic.Status.IntegrationMetaData[connectionIDKey]
	var errs []string
	if err := si.removeServiceDiscovery(ic.Status.DiscoveryResource); err != nil {
		errs = append(errs, err.Error())
	}
	if integration.Spec.IntegrationType == "api" {
		if err := si.fuseConnectionCrud.DeleteConnector(connection); err != nil {
			errs = append(errs, err.Error())
		}
	} else {
		if err := si.fuseConnectionCrud.DeleteConnection(connection); err != nil {
			errs = append(errs, err.Error())
		}
	}
	var err error
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, " : "))
	}
	return ic, err
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
		{
			Type:     "api",
			Provider: v1alpha1.FuseIntegrationTarget,
		},
	}
}
