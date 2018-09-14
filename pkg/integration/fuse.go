package integration

import (
	"context"
	"time"

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
		logrus.Debug("adding fuse amqp integration")
		update, err := f.addAMQPIntegration(ctx, integration)
		if err != nil {
			return errors.Wrap(err, "failed to add amqp integration")
		}
		if update == nil {
			return nil
		}
		return sdk.Update(update)
	case "http", "https":
		logrus.Debug("adding fuse route integration")
		update, err := f.addRouteIntegration(ctx, integration)
		if err != nil {
			return errors.Wrap(err, "failed to add route integration")
		}
		if update == nil {
			return nil
		}
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
	case "http,https":
		update, err = f.removeRouteIntegration(ctx, integration)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	v1alpha1.RemoveFinalizer(update, v1alpha1.Finalizer)
	return nil
}

func (f *Fuse) removeAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	// remove user
	// remove user secret
	// remove connection
	// if deleted remove finalizer
	ic := integration.DeepCopy()

	return ic, nil
}

func (f *Fuse) removeRouteIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	// remove connection
	// if deleted remove finalizer
	ic := integration.DeepCopy()

	return ic, nil
}

func (f *Fuse) addRouteIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Info("adding route integration ", integration.Status.IntegrationMetaData.Route)
	ic := integration.DeepCopy()
	logrus.Info("adding route integration ", integration.Status.IntegrationMetaData.Route)
	var errs string
	for n, r := range ic.Status.IntegrationMetaData.Route {
		if err := f.fuseService.AddRouteConnection(n, r, integration.Spec.IntegrationType, ic.Namespace); err != nil && !errors3.IsAlreadyExists(err) {
			errs += " " + err.Error()
		}
	}
	if errs != "" {
		return nil, errors.New(errs)
	}
	ic.Status.Phase = v1alpha1.PhaseComplete
	if ic.Labels == nil {
		ic.Labels = map[string]string{}
	}
	ic.Labels["integration"] = "enabled"
	return ic, nil
}

func (f *Fuse) addAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData.Realm
	if realm == "" {
		return nil, errors.New("missing realm in metadata")
	}
	messagingHost := ic.Status.IntegrationMetaData.MessagingHost
	// check if a connection already exists, if it does we will rerun through the create once every 5 mins
	exists, err := f.fuseService.DoesConnectionExist(integration.Spec.IntegrationType, integration.Status.IntegrationMetaData.Realm, integration.Namespace)
	if err != nil {
		logrus.Error("failed to check if amqp connection already exists. Assuming it doesn't", err)
	}
	if exists && (time.Now().Unix()-ic.Status.LastCheck) < int64(time.Second*30) {
		logrus.Info("connection exists ")
		return nil, nil
	}

	//need to avoid creating users so need determinstic user name
	u, err := f.enmasseService.CreateUser(integration.Name, realm)
	if err != nil && !errors2.IsAlreadyExistsErr(err) {
		return nil, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	if _, err := f.fuseService.AddAMQPConnection(integration.Status.IntegrationMetaData.Realm, u.UserName, u.Password, messagingHost, integration.Namespace); err != nil && !errors3.IsAlreadyExists(err) {
		return nil, errors.Wrap(err, "failed to create amqp connection in fuse")
	}
	secretFound := false
	for _, is := range ic.Status.IntegrationMetaData.Secrets {
		if is == u.UserName {
			secretFound = true
			break
		}
	}
	if !secretFound {
		ic.Status.IntegrationMetaData.Secrets = append(ic.Status.IntegrationMetaData.Secrets, u.UserName)
	}
	ic.Status.LastCheck = time.Now().Unix()
	ic.Status.Phase = v1alpha1.PhaseComplete
	ic.Labels["integration"] = "enabled"
	return ic, nil
}
