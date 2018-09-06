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
	fuse *Fuse
}

func NewReconciler(fuse *Fuse) *Reconciler {
	return &Reconciler{fuse: fuse}
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
	logrus.Info("handling integration ", integration.Name, integration.Spec, event)
	if !ok {
		return errors.New("expected a integration object but got " + event.Object.GetObjectKind().GroupVersionKind().String())
	}
	if integration.Status.Phase == v1alpha1.PhaseNone {
		ic, err := h.Accept(ctx, integration)
		if err != nil && controllerErr.IsNotEnabledErr(err) {
			logrus.Debug("integration is not enabled ", integration.Name, " doing nothing")
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "failed to accept new integration")
		}

		return sdk.Update(ic)
	}
	switch integration.Spec.Service {
	case "fuse":
		if event.Deleted {
			return h.fuse.DisIntegrate(ctx, integration)
		}
		return h.fuse.Integrate(ctx, integration)
	default:
		return errors.New("unknown integration type " + integration.Spec.Service)
	}
	return nil
}

func (h *Reconciler) Accept(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	if !ic.Spec.Enabled {
		return nil, &controllerErr.NotEnabledErr{}
	}
	if err := v1alpha1.AddFinalizer(integration, v1alpha1.Finalizer); err != nil {
		return nil, errors.Wrap(err, "failed to accept integration could not add finalizer")
	}
	ic.Status.Phase = v1alpha1.PhaseAccepted
	return ic, nil
}
