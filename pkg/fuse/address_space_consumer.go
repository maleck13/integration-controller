package fuse

import (
	"encoding/json"
	"fmt"
	"strings"

	syndesis "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

type AddressSpaceConsumer struct {
	*FuseExistsChecker
	watchNS           string
	integrationCruder Crudler
}

func NewAddressSpaceConsumer(watchNS string, cruder Crudler) *AddressSpaceConsumer {
	return &AddressSpaceConsumer{
		watchNS:           watchNS,
		integrationCruder: cruder,
		FuseExistsChecker: NewFuseExistsChecker(watchNS, cruder),
	}
}

func (c *AddressSpaceConsumer) GVKs() []schema.GroupVersionKind {
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

func (c *AddressSpaceConsumer) Validate(object runtime.Object) error {
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

func (c *AddressSpaceConsumer) convertToAddressSpace(o runtime.Object) (*enmasse.AddressSpace, error) {
	switch ro := o.(type) {
	case *v1.ConfigMap:
		return unMarshalAddressSpace(ro)
	case *enmasse.AddressSpace:
		return ro, nil
	default:
		return nil, fmt.Errorf("unexpected type %v  cannot convert to address space", ro)

	}
}

func (c *AddressSpaceConsumer) CreateAvailableIntegration(o runtime.Object, enabled bool) error {
	logrus.Info("create available integration for fuses")

	as, err := c.convertToAddressSpace(o)
	if err != nil {
		return err
	}
	syndesisList := v1alpha12.NewSyndesisList()
	if err := c.integrationCruder.List(c.watchNS, syndesisList); err != nil {
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
				ingrtn := v1alpha1.NewIntegration(c.integrationName(as))
				ingrtn.ObjectMeta.Namespace = c.watchNS
				ingrtn.Spec.Client = s.Name
				objectMeta := o.(v12.ObjectMetaAccessor).GetObjectMeta()
				ingrtn.Status.DiscoveryResource = v1alpha1.DiscoveryResource{Namespace: objectMeta.GetNamespace(), Name: objectMeta.GetName(), GroupVersionKind: o.GetObjectKind().GroupVersionKind()}
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

				if err := c.integrationCruder.Create(ingrtn); err != nil && !errors.IsAlreadyExists(err) {
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

func (c *AddressSpaceConsumer) RemoveAvailableIntegration(o runtime.Object) error {
	logrus.Info("delete available integration called for fuse")
	// get an integration with that name
	as, err := c.convertToAddressSpace(o)
	if err != nil {
		return err
	}
	name := c.integrationName(as)
	ingrtn := v1alpha1.NewIntegration(name)
	ingrtn.ObjectMeta.Name = name
	ingrtn.ObjectMeta.Namespace = c.watchNS
	if err := c.integrationCruder.Get(ingrtn); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return c.integrationCruder.Delete(ingrtn)
}

func (c *AddressSpaceConsumer) integrationName(o *enmasse.AddressSpace) string {
	return "enmasse-" + o.Name + "-to-fuse"
}

//go:generate moq -out crudler_mock_test.go . Crudler
type Crudler interface {
	List(namespace string, o sdk.Object, option ...sdk.ListOption) error
	Get(into sdk.Object, opts ...sdk.GetOption) error
	Create(object sdk.Object) (err error)
	Delete(object sdk.Object) error
	Update(object sdk.Object) error
}

//go:generate moq -out fuse_client_test.go . FuseClient
type FuseClient interface {
	// TODO perhaps should take a cutomisation object
	CreateCustomisation(c *syndesis.Connection) (*syndesis.Connection, error)
	DeleteConnector(c *syndesis.Connection) error
	ConnectionExists(c *syndesis.Connection) (bool, error)
	CreateConnection(c *syndesis.Connection) (*syndesis.Connection, error)
	UpdateConnection(c *syndesis.Connection) (*syndesis.Connection, error)
	DeleteConnection(c *syndesis.Connection) error
}
