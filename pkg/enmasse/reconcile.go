package enmasse

import (
	"context"
	"encoding/json"
	"fmt"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/enmasse/v1"
	integration "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	syndesis "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Reconciler struct {
}

var integrationType = "amqp"

func (r *Reconciler) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: "v1",
		Group:   "",
		Kind:    "ConfigMap",
	}

}

func syndesisExists(namespace string) (bool, error) {
	syndesisList := syndesis.NewSyndesisList()
	if err := sdk.List(namespace, syndesisList); err != nil {
		return false, errors.New("Failed to list syndesis custom resources")
	}

	return len(syndesisList.Items) > 0, nil
}

func unMarshalAddressSpace(cfgm *v12.ConfigMap) (*v1.AddressSpace, error) {
	addressSpaceData := cfgm.Data["config.json"]

	var addressSpace v1.AddressSpace
	err := json.Unmarshal([]byte(addressSpaceData), &addressSpace)
	return &addressSpace, err
}

func createIntegrationResource(as *v1.AddressSpace, namespace string) *integration.Integration {
	ingrtn := integration.NewIntegration()
	for _, endPointStatus := range as.Status.EndPointStatuses {
		if endPointStatus.Name == "messaging" {
			ingrtn := integration.NewIntegration()
			ingrtn.ObjectMeta.Name = as.Name
			ingrtn.ObjectMeta.Namespace = namespace
			ingrtn.Spec.MessagingHost = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", endPointStatus.Port)
			ingrtn.Spec.Realm = as.Annotations["enmasse.io/realm-name"]
			ingrtn.Spec.IntegrationType = integrationType
			ingrtn.Spec.Service = "fuse"
			ingrtn.Spec.Enabled = false

			for _, servicePort := range endPointStatus.ServicePorts {
				if servicePort.Name == integrationType {
					ingrtn.Spec.MessagingHost = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", servicePort.Port)
				}

			}

			return ingrtn
		}
	}
	return ingrtn
}

func (r *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}
	hasSyndesis, err := syndesisExists(namespace)
	if err != nil {
		return err
	}
	if hasSyndesis == false {
		return nil
	}

	configMap, ok := event.Object.(*v12.ConfigMap)
	//logrus.Info("handling address space config map ", configMap.Name, event)
	if !ok {
		return errors.New("expected a config map object but got " + event.Object.GetObjectKind().GroupVersionKind().String())
	}

	addressSpace, err := unMarshalAddressSpace(configMap)
	if err != nil {
		logrus.Fatalf("Failed to unmarshall addressspace data: %v", err)
	}

	ingrtn := createIntegrationResource(addressSpace, namespace)
	if event.Deleted == true {
		return sdk.Delete(ingrtn)
	} else {
		if err := sdk.Create(ingrtn); err != nil && !errors2.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
