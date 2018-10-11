package integration

import (
	"context"

	controllerErr "github.com/integr8ly/integration-controller/pkg/errors"

	"github.com/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Reconciler struct {
	integrationReg IntegratorRegistery
}

func NewReconciler(integrationReg IntegratorRegistery) *Reconciler {
	return &Reconciler{integrationReg: integrationReg}
}

func (h *Reconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: v1alpha1.Version,
		Group:   v1alpha1.GroupName,
		Kind:    v1alpha1.IntegrationKind,
	}
}

func (h *Reconciler) Handle(ctx context.Context, event sdk.Event) error {

	integration, ok := event.Object.(*v1alpha1.Integration)
	if !ok {
		return errors.New("expected a integration object but got " + event.Object.GetObjectKind().GroupVersionKind().String())
	}
	logrus.Debug("handling integration ", integration.Name, integration.Spec)
	integrator := h.integrationReg.IntegratorFor(integration)
	if integrator == nil {
		return errors.New("no integrators registered for " + integration.Spec.ServiceProvider)
	}
	if integration.Status.Phase == v1alpha1.PhaseNone {
		ic, err := h.Accept(ctx, integration, integrator.Validate)
		if err != nil && controllerErr.IsNotEnabledErr(err) {
			logrus.Debug("integration is not enabled ", integration.Name, " doing nothing")
			return nil
		}
		if err != nil {
			errMsg := errors.Wrap(err, "failed to accept new integration").Error()
			ic.Status.StatusMessage = errMsg
			logrus.Error(errMsg)
			if err := sdk.Update(ic); err != nil {
				return errors.Wrap(err, errMsg+" : failed to update the integration")
			}
			return errors.Wrap(err, "failed to accept new integration")
		}
		return sdk.Update(ic)
	}
	if event.Deleted || (integration.Status.Phase == v1alpha1.PhaseComplete && integration.Spec.Enabled == false) {
		itg, err := integrator.DisIntegrate(ctx, integration)
		if err != nil {
			itg.Status.StatusMessage = err.Error()
			logrus.Error(err)
			if updateErr := sdk.Update(itg); updateErr != nil {
				return errors.Wrap(updateErr, err.Error()+" : failed to update the integration")
			}
			return err
		}
		if _, err := v1alpha1.RemoveFinalizer(itg, v1alpha1.Finalizer); err != nil {
			return err
		}

		return sdk.Update(itg)
	}
	itg, err := integrator.Integrate(ctx, integration)
	if err != nil {
		itg.Status.StatusMessage = err.Error()
		logrus.Error(err)
		if updateErr := sdk.Update(itg); updateErr != nil {
			return errors.Wrap(updateErr, err.Error()+" : failed to update the integration")
		}
		return err
	}
	return sdk.Update(itg)
}

type validator func(integration *v1alpha1.Integration) error

func (h *Reconciler) Accept(ctx context.Context, integration *v1alpha1.Integration, validate validator) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	if !ic.Spec.Enabled {
		return nil, &controllerErr.NotEnabledErr{}
	}
	if err := validate(ic); err != nil {
		// not going to error here but instead allow the user to see the issue on the resource and keep trying to reconcile
		ic.Status.StatusMessage = err.Error()
		logrus.Error("integration failed validation ", err)
		return ic, nil
	}
	if err := v1alpha1.AddFinalizer(integration, v1alpha1.Finalizer); err != nil {
		return nil, errors.Wrap(err, "failed to accept integration could not add finalizer")
	}
	ic.Status.Phase = v1alpha1.PhaseAccepted
	return ic, nil
}
