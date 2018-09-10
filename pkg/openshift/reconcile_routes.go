package openshift

import (
	"context"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

	"github.com/pkg/errors"

	"github.com/openshift/api/route/v1"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RouteReconciler struct {
}

func NewRouteReconciler() *RouteReconciler {
	return &RouteReconciler{}
}

func (h *RouteReconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: "v1",
		Group:   "route.openshift.io",
		Kind:    "Route",
	}
}

func (h *RouteReconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling route ", event.Object.GetObjectKind().GroupVersionKind().String())
	route, ok := event.Object.(*v1.Route)
	if !ok {
		return errors.New("not a route")
	}
	integreation := v1alpha1.NewIntegration()
	integreation.Spec.IntegrationType = "http"
	if route.Spec.TLS != nil {
		integreation.Spec.IntegrationType = "https"
	}
	integreation.Spec.Service = "fuse"
	// create integration
	return nil
}
