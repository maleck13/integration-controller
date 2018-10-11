package fuse

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/integr8ly/integration-controller/pkg/integration"

	"github.com/sirupsen/logrus"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/pkg/errors"
)

const (
	connectionIDKey     = "connectionID"
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
	CreateUser(userName, realm string) (*v1alpha1.User, error)
	DeleteUser(userName, realm string) error
}

type Integrator struct {
	enmasseService EnMasseService
	connectionCrud FuseConnectionCruder
}

func NewIntegrator(service EnMasseService, connectionCrud FuseConnectionCruder) *Integrator {
	return &Integrator{enmasseService: service, connectionCrud: connectionCrud}
}

// Integrate calls out to the fuse api to create a new connection
func (f *Integrator) Integrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse integration ", integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	//need to avoid creating users so need determinstic user name
	//TODO once a user can be added via CRD update
	u, err := f.enmasseService.CreateUser(integration.Name, realm)
	if err != nil && !errors2.IsAlreadyExists(err) {
		return ic, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	amqpHost := fmt.Sprintf("amqp://%s?amqp.saslMechanisms=PLAIN", integration.Status.IntegrationMetaData[msgHostKey])
	id, err := f.connectionCrud.CreateConnection("amqp", integration.Name, u.UserName, u.Password, amqpHost)
	if err != nil {
		return ic, errors.Wrap(err, "failed to create amqp connection in fuse")
	}
	ic.Status.Phase = v1alpha1.PhaseComplete
	if id != "" {
		ic.Status.IntegrationMetaData[connectionIDKey] = id
	}
	return ic, nil
}

// DisIntegrate removes the enmasse user and fuse connection
func (f *Integrator) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse removing integration ", integration.Spec.IntegrationType)
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	if err := f.enmasseService.DeleteUser(integration.Name, realm); err != nil {
		//dont fail here we still want to remove the connection
		logrus.Error("failed to remove user when removing enmasse fuse integration")
	}
	if err := f.connectionCrud.DeleteConnection("amqp", ic.Status.IntegrationMetaData[connectionIDKey]); err != nil {
		return ic, err
	}
	return ic, nil
}

func (f *Integrator) Integrates() []integration.Type {
	return []integration.Type{
		{
			Provider: v1alpha1.FuseIntegrationTarget,
			Type:     "amqp",
		},
	}
}

func (f *Integrator) Validate(i *v1alpha1.Integration) error {
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
