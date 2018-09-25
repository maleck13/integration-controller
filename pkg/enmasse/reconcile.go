package enmasse

import (
	"context"
	"encoding/json"

	"github.com/integr8ly/integration-controller/pkg/integration"

	"github.com/integr8ly/integration-controller/pkg/apis/enmasse/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Reconciler struct {
	consumerRegistery integration.ConsumerRegistery
	watchNS           string
}

func NewReconciler(cr integration.ConsumerRegistery, watchNS string) *Reconciler {
	return &Reconciler{consumerRegistery: cr, watchNS: watchNS}
}

func (r *Reconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: "v1",
		Group:   "",
		Kind:    "ConfigMap",
	}

}

func unMarshalAddressSpace(cfgm *v12.ConfigMap) (*v1.AddressSpace, error) {
	addressSpaceData := cfgm.Data["config.json"]

	var addressSpace v1.AddressSpace
	err := json.Unmarshal([]byte(addressSpaceData), &addressSpace)
	return &addressSpace, err
}

func (r *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())
	configMap, ok := event.Object.(*v12.ConfigMap)
	if !ok {
		return errors.New("expected a config map object but got " + event.Object.GetObjectKind().GroupVersionKind().String())
	}

	addressSpace, err := unMarshalAddressSpace(configMap)
	if err != nil {
		logrus.Fatalf("Failed to unmarshall addressspace data: %v", err)
	}

	consumers := r.consumerRegistery.ConsumersForKind(addressSpace.GetObjectKind().GroupVersionKind())

	if event.Deleted == true {
		logrus.Debug("handling deleted address space", addressSpace.Name)
		var multiErr error
		for _, consumer := range consumers {
			if consumer.Exists() {
				if err := consumer.RemoveAvailableIntegration(addressSpace, r.watchNS); err != nil {
					if multiErr == nil {
						multiErr = errors.New(err.Error())
						continue
					}
					multiErr = errors.Wrap(multiErr, err.Error())
				}
			}
		}
		if multiErr != nil {
			return multiErr
		}
		return nil
	}
	logrus.Debug("handling address-space ", addressSpace.Name)
	for _, consumer := range consumers {
		if consumer.Exists() {
			if err := consumer.Validate(addressSpace); err != nil {
				return errors.Wrap(err, "address-space validation failed")
			}
			logrus.Debug("fuse exists creating integration")
			if err := consumer.CreateAvailableIntegration(addressSpace, r.watchNS, false); err != nil {
				return errors.Wrap(err, "failed to create an integration based on enmasse address space "+addressSpace.Name)
			}
		}
	}

	return nil
}
