package openshift

import (
	"context"
	"fmt"
	"os"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

	"github.com/pkg/errors"

	"github.com/openshift/api/route/v1"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RouteReconciler struct {
	autoEnable bool
	watchNS    string
}

func NewRouteReconciler() *RouteReconciler {
	r := &RouteReconciler{}
	// TODO move up to main
	if os.Getenv("INTEGRATION_AUTO_ENABLE") == "true" {
		r.autoEnable = true
	}
	r.watchNS = os.Getenv("WATCH_NAMESPACE")
	return r
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
	integreation.Namespace = h.watchNS
	integreation.Name = route.Name + "-fuse"
	if route.Spec.TLS != nil {
		integreation.Spec.IntegrationType = "https"
	}
	integreation.Spec.Enabled = h.autoEnable
	integreation.Spec.Service = "fuse"
	if integreation.Status.IntegrationMetaData.Route == nil {
		integreation.Status.IntegrationMetaData.Route = map[string]string{}
	}
	integreation.Status.IntegrationMetaData.Route[route.Name] = fmt.Sprintf("%s://%s", integreation.Spec.IntegrationType, route.Spec.Host)
	if err := sdk.Create(integreation); err != nil && !errors2.IsAlreadyExists(err) {
		return err
	}
	return nil
}
