package fuse

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) AddAMQPConnection(name, user, pass, messageHost, namespace string) (string, error) {
	//amqp://messaging.enmasse-eval.svc:5672?amqp.saslMechanisms=PLAIN
	msgHost := fmt.Sprintf("amqp://%s?amqp.saslMechanisms=PLAIN", messageHost)
	connection := v1alpha1.Connection{
		Spec: v1alpha1.ConnectionSpec{
			ConnectionType: "amqp",
			Password:       pass,
			Username:       user,
			URL:            msgHost,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "fuse-amqp-" + name,
			Labels:    map[string]string{"integration": "fuse-enmasse"},
			Namespace: namespace,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Connection",
			APIVersion: "syndesis.io/v1alpha1",
		},
	}
	logrus.Info("adding new fuse amqp connection")
	if err := sdk.Create(&connection); err != nil {
		return "", err
	}
	return connection.Name, nil
}

func (s *Service) AddRouteConnection(name, route, routeProtocol, namespace string) error {
	logrus.Debug("adding route connection")
	connection := v1alpha1.Connection{
		Spec: v1alpha1.ConnectionSpec{
			ConnectionType: routeProtocol,
			URL:            route,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "fuse-route-" + name,
			Labels:    map[string]string{"integration": "fuse-routes"},
			Namespace: namespace,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Connection",
			APIVersion: "syndesis.io/v1alpha1",
		},
	}
	logrus.Info("adding new fuse route connection")
	if err := sdk.Create(&connection); err != nil {
		return err
	}
	return nil
}

func (s *Service) DoesConnectionExist(connType, name, namespace string) (bool, error) {
	connection := v1alpha1.Connection{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fuse-" + connType + "-" + name,
			Namespace: namespace,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Connection",
			APIVersion: "syndesis.io/v1alpha1",
		},
	}

	if err := sdk.Get(&connection, sdk.WithGetOptions(&v1.GetOptions{})); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil

}
