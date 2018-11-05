package consumer

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

	"github.com/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/integration"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func NewHandler(consumerRegistry integration.ConsumerRegistery, integrationRec *integration.Reconciler, watchNS string) sdk.Handler {
	return &Handler{
		consumerRegistery:     consumerRegistry,
		integrationReconciler: integrationRec,
		watchNS:               watchNS,
	}
}

type Handler struct {
	consumerRegistery     integration.ConsumerRegistery
	watchNS               string
	integrationReconciler *integration.Reconciler
}

func handleConsumerErr(existingErr, err error) error {
	var retErr error
	if existingErr == nil {
		retErr = err
	} else {
		retErr = errors.WithMessage(retErr, err.Error())
	}
	return retErr
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {

	object := event.Object
	var multiErr error
	switch o := object.(type) {
	case *v1alpha1.Integration:
		return h.integrationReconciler.Handle(ctx, event)
	case v1.ObjectMetaAccessor:
		logrus.Debug("handing type ", object.GetObjectKind().GroupVersionKind(), event.Deleted, o.GetObjectMeta().GetDeletionTimestamp())
		consumers := h.consumerRegistery.ConsumersForKind(object.GetObjectKind().GroupVersionKind())
		if event.Deleted == true {
			for _, consumer := range consumers {
				if consumer.Exists() {
					if err := consumer.RemoveAvailableIntegration(object); err != nil {
						multiErr = handleConsumerErr(multiErr, err)
						continue
					}
				}
			}
			if multiErr != nil {
				return multiErr
			}
			return nil
		}
		//creation
		for _, consumer := range consumers {
			if consumer.Exists() {
				if err := consumer.Validate(object); err != nil {
					multiErr = handleConsumerErr(multiErr, err)
					continue
				}
				if err := consumer.CreateAvailableIntegration(object, false); err != nil {
					multiErr = handleConsumerErr(multiErr, err)
					continue
				}
			}
		}
		if multiErr != nil {
			logrus.Error("failed to setup integration object ", multiErr)
		}
		return nil
	}
	return errors.New("unknown object kind")
}
