package integration

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"github.com/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/sirupsen/logrus"
)

type Fuse struct{}

func (f *Fuse) Integrate(ctx context.Context, integration *v1alpha1.Integration) error {
	logrus.Info("handling fuse integration ", integration.Spec.IntegrationType)
	switch integration.Spec.IntegrationType {
	case "amqp":
		update, err := f.addAMQPIntegration(ctx, integration)
		if err != nil {
			return errors.Wrap(err, "failed to add amqp integration")
		}
		return sdk.Update(update)
	}
	return nil
}

func (f *Fuse) addAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	// copy and modify state of the integration
	return nil, nil
}
