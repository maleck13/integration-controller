package integration

import (
	"context"

	errors3 "k8s.io/apimachinery/pkg/api/errors"

	errors2 "github.com/integr8ly/integration-controller/pkg/errors"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"github.com/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/sirupsen/logrus"
)

type Fuse struct {
	enmasseService EnMasseService
	fuseService    FuseService
}

func NewFuse(service EnMasseService, fuseService FuseService) *Fuse {
	return &Fuse{enmasseService: service, fuseService: fuseService}
}

func (f *Fuse) Integrate(ctx context.Context, integration *v1alpha1.Integration) error {
	logrus.Info("handling fuse integration ", integration.Spec.IntegrationType)
	switch integration.Spec.IntegrationType {
	case "amqp":
		logrus.Debug("adding amqp integration")
		update, err := f.addAMQPIntegration(ctx, integration)
		if err != nil {
			return errors.Wrap(err, "failed to add amqp integration")
		}
		logrus.Debug("updating integration")
		return sdk.Update(update)
	}
	return nil
}

func (f *Fuse) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) error {
	logrus.Debug("removing amqp integration")
	var update *v1alpha1.Integration
	var err error
	switch integration.Spec.IntegrationType {
	case "amqp":
		update, err = f.removeAMQPIntegration(ctx, integration)
		if err != nil {
			return err
		}
	}

	v1alpha1.RemoveFinalizer(update, v1alpha1.Finalizer)
	return nil
}

func (f *Fuse) removeAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	// remove user
	// remove connection
	// if deleted remove finalizer
	ic := integration.DeepCopy()

	return ic, nil
}

func (f *Fuse) addAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	realm := ic.Spec.Realm
	if realm == "" {
		return nil, errors.New("missing realm in metadata")
	}
	logrus.Debug("integration metaData ", ic.Spec.Realm)
	//need to avoid creating users so need determinstic user name
	u, err := f.enmasseService.CreateUser(integration.Name, realm)
	if err != nil && !errors2.IsAlreadyExistsErr(err) {
		return nil, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	if _, err := f.fuseService.AddAMQPConnection(integration.Name, u.UserName, u.Password, integration.Spec.MessagingHost, integration.Namespace); err != nil && !errors3.IsAlreadyExists(err) {
		return nil, errors.Wrap(err, "failed to create amqp connection in fuse")
	}
	ic.Status.Phase = v1alpha1.PhaseComplete
	ic.Labels["integration"] = "enabled"
	return ic, nil
}
