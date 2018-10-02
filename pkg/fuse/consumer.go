package fuse

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/api/errors"

	enmasse "github.com/integr8ly/integration-controller/pkg/apis/enmasse/v1"
	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	pkgerrs "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1alpha12 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Consumer struct {
	watchNS    string
	fuseCruder FuseCrudler
}

func NewConsumer(watchNS string, fuseCruder FuseCrudler) *Consumer {
	return &Consumer{watchNS: watchNS, fuseCruder: fuseCruder}
}

func (c *Consumer) GVKs() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{{
		Kind:    enmasse.AddressSpaceKind,
		Group:   enmasse.GroupName,
		Version: enmasse.Version,
	}, {
		Kind:    "ConfigMap",
		Group:   "",
		Version: "v1",
	}}
}

func (c *Consumer) Exists() bool {
	logrus.Debug("fuse consume: checking if a fuse exists")
	syndesisList := v1alpha12.NewSyndesisList()
	if err := c.fuseCruder.List(c.watchNS, syndesisList); err != nil {
		logrus.Error("fuse consumer: failed to check if fuse exists. Will try again ", err)
		return false
	}

	return len(syndesisList.Items) > 0
}

func (c *Consumer) Validate(object runtime.Object) error {
	// validate that we have the annotations
	switch o := object.(type) {
	case *enmasse.AddressSpace:
		if o.Annotations == nil || o.Annotations["enmasse.io/realm-name"] == "" || o.Annotations["enmasse.io/created-by"] == "" {
			return pkgerrs.New("fuse consumer: enmasse address space invalid. missing annotations. Needed: (enmasse.io/realm-name,enmasse.io/created-by)")
		}
	}
	return nil
}

func unMarshalAddressSpace(cfgm *v1.ConfigMap) (*enmasse.AddressSpace, error) {
	addressSpaceData := cfgm.Data["config.json"]

	var addressSpace enmasse.AddressSpace
	err := json.Unmarshal([]byte(addressSpaceData), &addressSpace)
	return &addressSpace, err
}

func (c *Consumer) convertToAddressSpace(o runtime.Object) (*enmasse.AddressSpace, error) {
	switch ro := o.(type) {
	case *v1.ConfigMap:
		return unMarshalAddressSpace(ro)
	case *enmasse.AddressSpace:
		return ro, nil
	default:
		return nil, fmt.Errorf("unexpected type %v  cannot convert to address space", ro)

	}
}

func (c *Consumer) CreateAvailableIntegration(o runtime.Object, namespace string, enabled bool) error {
	logrus.Info("create available integration for fuses")

	as, err := c.convertToAddressSpace(o)
	if err != nil {
		return err
	}
	syndesisList := v1alpha12.NewSyndesisList()
	if err := c.fuseCruder.List(c.watchNS, syndesisList); err != nil {
		logrus.Error("fuse consumer: failed to check if fuse exists ", err)
		return nil
	}
	var errs error
	for _, s := range syndesisList.Items {
		if as.Annotations == nil || s.Annotations == nil {
			continue
		}
		// only create if the same use owns both
		if strings.TrimSpace(as.Annotations["enmasse.io/created-by"]) != strings.TrimSpace(s.Annotations["syndesis.io/created-by"]) {
			logrus.Debug("found an enmasse address space but it does not match the user for fuse. Ignoring")
			continue
		}
		for _, endPointStatus := range as.Status.EndPointStatuses {
			if endPointStatus.Name == "messaging" {
				ingrtn := v1alpha1.NewIntegration()
				ingrtn.ObjectMeta.Name = c.integrationName(as)
				ingrtn.ObjectMeta.Namespace = namespace
				ingrtn.Spec.Client = s.Name
				ingrtn.Status.IntegrationMetaData = map[string]string{}
				ingrtn.Status.IntegrationMetaData[msgHostKey] = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", endPointStatus.Port)
				ingrtn.Status.IntegrationMetaData[realmKey] = as.Annotations["enmasse.io/realm-name"]
				ingrtn.Spec.IntegrationType = "amqp"
				ingrtn.Spec.ServiceProvider = string(v1alpha1.FuseIntegrationTarget)
				ingrtn.Spec.Enabled = enabled

				for _, servicePort := range endPointStatus.ServicePorts {
					if servicePort.Name == "amqp" {
						ingrtn.Status.IntegrationMetaData[msgHostKey] = endPointStatus.ServiceHost + ":" + fmt.Sprintf("%d", servicePort.Port)
					}
				}

				if err := c.fuseCruder.Create(ingrtn); err != nil && !errors.IsAlreadyExists(err) {
					if errs == nil {
						errs = err
						continue
					}
					errs = pkgerrs.Wrap(errs, err.Error())
				}
			}
		}
	}

	return errs
}

func (c *Consumer) RemoveAvailableIntegration(o runtime.Object, namespace string) error {
	logrus.Info("delete available integration called for fuse")
	// get an integration with that name
	as, err := c.convertToAddressSpace(o)
	if err != nil {
		return err
	}
	name := c.integrationName(as)
	ingrtn := v1alpha1.NewIntegration()
	ingrtn.ObjectMeta.Name = name
	ingrtn.ObjectMeta.Namespace = namespace
	if err := c.fuseCruder.Get(ingrtn); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return c.fuseCruder.Delete(ingrtn)
}

func (c *Consumer) integrationName(o *enmasse.AddressSpace) string {
	return "enmasse-" + o.Name + "-to-fuse"
}

//go:generate moq -out fuse_crudler_test.go . FuseCrudler
type FuseCrudler interface {
	List(namespace string, o sdk.Object, option ...sdk.ListOption) error
	Get(into sdk.Object, opts ...sdk.GetOption) error
	Create(object sdk.Object) (err error)
	Delete(object sdk.Object) error
}
