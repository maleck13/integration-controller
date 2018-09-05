package integration

import (
	"context"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Reconciler struct{}

func (h *Reconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: v1alpha1.Version,
		Group:   v1alpha1.GroupName,
		Kind:    v1alpha1.IntegrationKind,
	}
}

func (h *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	return nil
}
