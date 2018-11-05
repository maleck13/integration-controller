package fuse

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"

	v1alpha12 "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"

	"github.com/integr8ly/integration-controller/pkg/integration"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/pkg/errors"
)

const (
	connectionIDKey     = "connectionID"
	connectionName      = "connectionName"
	realmKey            = "realm"
	msgHostKey          = "msgHost"
	integrationTypeAMQP = "amqp"
	integrationTypeHTTP = "api"
)

var validIntegrationTypes = []string{integrationTypeAMQP, integrationTypeHTTP}

//go:generate moq -out requester_test.go . httpRequester
type httpRequester interface {
	Do(r *http.Request) (*http.Response, error)
}

//go:generate moq -out enmasse_service_test.go . EnMasseService
type EnMasseService interface {
	CreateUser(userName, realm string) (*v1alpha1.User, *v1.Secret, error)
	DeleteUser(userName, realm string) error
}

type AddressSpaceIntegrator struct {
	enmasseService EnMasseService
	connectionCrud FuseClient
}

func NewAddressSpaceIntegrator(service EnMasseService, connectionCrud FuseClient) *AddressSpaceIntegrator {
	return &AddressSpaceIntegrator{enmasseService: service, connectionCrud: connectionCrud}
}

// Integrate calls out to the fuse api to create a new connection
func (f *AddressSpaceIntegrator) Integrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse integration ", integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	//need to avoid creating lots users so need determinstic user name
	//TODO once a user can be added via CRD update
	_, s, err := f.enmasseService.CreateUser(integration.Name, realm)
	if err != nil {
		return ic, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	var (
		connection    = v1alpha12.NewConnection(integration.Name, integration.Namespace, integration.Spec.IntegrationType)
		connectionErr error
	)

	connection.Spec.Credentials = s.Name
	amqpHost := fmt.Sprintf("amqp://%s?amqp.saslMechanisms=PLAIN", integration.Status.IntegrationMetaData[msgHostKey])
	connection.Spec.URL = amqpHost
	connection, connectionErr = f.connectionCrud.CreateConnection(connection)
	if connectionErr != nil && errors2.IsAlreadyExists(connectionErr) {
		connection, connectionErr = f.connectionCrud.UpdateConnection(connection)
	}
	if connectionErr != nil {
		return ic, connectionErr
	}
	ic.Status.IntegrationMetaData[connectionIDKey] = connection.Status.SyndesisID
	ic.Status.Phase = v1alpha1.PhaseComplete
	return ic, nil
}

// DisIntegrate removes the enmasse user and fuse connection
func (f *AddressSpaceIntegrator) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse removing integration ", integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	if err := f.enmasseService.DeleteUser(integration.Name, realm); err != nil {
		//dont fail here we still want to remove the connection
		logrus.Error("failed to remove user when removing enmasse fuse integration")
	}
	connection := v1alpha12.NewConnection(integration.Name, integration.Namespace, integration.Spec.IntegrationType)
	connection.Status.SyndesisID = ic.Status.IntegrationMetaData[connectionIDKey]
	if err := f.connectionCrud.DeleteConnection(connection); err != nil {
		return ic, err
	}
	return ic, nil
}

func (f *AddressSpaceIntegrator) Integrates() []integration.Type {
	return []integration.Type{
		{
			Provider: v1alpha1.FuseIntegrationTarget,
			Type:     "amqp",
		},
	}
}

func (f *AddressSpaceIntegrator) Validate(i *v1alpha1.Integration) error {
	valid := false
	var err error
	for _, it := range validIntegrationTypes {
		if i.Spec.IntegrationType == it {
			valid = true
		}
	}
	if valid != true {
		err = errors.New("unknown integration type should be one of " + strings.Join(validIntegrationTypes, ","))
		return err
	}
	if _, ok := i.Status.IntegrationMetaData[msgHostKey]; !ok && i.Spec.IntegrationType == "amqp" {
		return errors.New("expected to find the key " + msgHostKey + " in the metadata but it was missing")
	}
	if _, ok := i.Status.IntegrationMetaData[realmKey]; !ok && i.Spec.IntegrationType == "amqp" {
		return errors.New("expected to find the key " + realmKey + " in the metadata but it was missing")
	}
	return nil
}
