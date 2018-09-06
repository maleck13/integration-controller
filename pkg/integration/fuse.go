package integration

import (
	"context"

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
		logrus.Debug(update)
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
	// remove finalizer
	ic := integration.DeepCopy()

	return ic, nil
}

func (f *Fuse) addAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	//need to avoid creating users so need determinstic user name
	u, err := f.enmasseService.CreateUser(integration.Name)
	if err != nil {
		return nil, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	f.fuseService.AddAMQPConnection(u.UserName, u.Password, "")
	ic.Status.Phase = v1alpha1.PhaseComplete
	return ic, nil
}
