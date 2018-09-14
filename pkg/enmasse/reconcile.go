package enmasse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	autoEnable bool
}

func NewReconciler() *Reconciler {
	r := &Reconciler{}
	if os.Getenv("INTEGRATION_AUTO_ENABLE") == "true" {
		r.autoEnable = true
	}
	return r
}

var integrationType = "amqp"

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

func createIntegrationResource(as *v1.AddressSpace, fuse *syndesis.Syndesis, namespace string, enabled bool) *integration.Integration {
	logrus.Info("auto enabled ", enabled)
	ingrtn := integration.NewIntegration()
	for _, endPointStatus := range as.Status.EndPointStatuses {
		if endPointStatus.Name == "messaging" {
			ingrtn := integration.NewIntegration()
			ingrtn.ObjectMeta.Name = as.Name + "-messaging-" + fuse.Name + "-fuse"
			ingrtn.ObjectMeta.Namespace = namespace
			ingrtn.Labels = map[string]string{"integration": "disabled"}
			ingrtn.Status.IntegrationMetaData.MessagingHost = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", endPointStatus.Port)
			ingrtn.Status.IntegrationMetaData.Realm = as.Annotations["enmasse.io/realm-name"]
			ingrtn.Spec.IntegrationType = integrationType
			ingrtn.Spec.Service = "fuse"
			ingrtn.Spec.Enabled = enabled

			for _, servicePort := range endPointStatus.ServicePorts {
				if servicePort.Name == integrationType {
					ingrtn.Status.IntegrationMetaData.MessagingHost = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", servicePort.Port)
				}

			}

			return ingrtn
		}
	}
	return ingrtn
}

func (r *Reconciler) Handle(ctx context.Context, event sdk.Event) error {
	configMap, ok := event.Object.(*v12.ConfigMap)
	//logrus.Info("handling address space config map ", configMap.Name, event)
	if !ok {
		return errors.New("expected a config map object but got " + event.Object.GetObjectKind().GroupVersionKind().String())
	}
	logrus.Info("handling object ", event.Object.GetObjectKind().GroupVersionKind().String())

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}

	addressSpace, err := unMarshalAddressSpace(configMap)
	if err != nil {
		logrus.Fatalf("Failed to unmarshall addressspace data: %v", err)
		return err
	}
	syndesisList := syndesis.NewSyndesisList()
	if err := sdk.List(namespace, syndesisList); err != nil {
		return errors.Wrap(err, "Failed to list syndesis custom resources")
	}
	if len(syndesisList.Items) == 0 {
		logrus.Info("no fuse discovered. Doing nothing")
		return nil
	}
	addressCreator := addressSpace.Annotations["enmasse.io/created-by"]
	for _, f := range syndesisList.Items {
		fuseCreator := f.Annotations["syndesis.io/created-by"]
		if f.Status.Phase == syndesis.SyndesisPhaseInstalled && (strings.TrimSpace(fuseCreator) == strings.TrimSpace(addressCreator)) {
			ingrtn := createIntegrationResource(addressSpace, &f, namespace, r.autoEnable)
			if err := sdk.Create(ingrtn); err != nil && !errors2.IsAlreadyExists(err) {
				return err
			}
		}
	}
	return nil
}

type Fuse interface {
	ListFuses()
}
