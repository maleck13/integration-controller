package enmasse

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Reconciler struct {
}

func (r *Reconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: "v1",
		Group:   "",
		Kind:    "ConfigMap",
	}

}

func (h *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	return nil
}
