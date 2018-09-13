package fuse

import (
	"fmt"

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
			Password: pass,
			Username: user,
			URL:      msgHost,
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
	logrus.Info("adding new fuse connection")
	if err := sdk.Create(&connection); err != nil {
		return "", err
	}
	return connection.Name, nil
}

func (s *Service) AddHTTPConnection() error {
	return nil
}
