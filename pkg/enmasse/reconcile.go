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

func (r *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	// check for a fuse cr in the same namespace.
	// if present create an integration resource
	return nil
}
