package integration

import "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

type FuseService interface {
	AddAMQPConnection(name, user, pass, messageHost, namespace string) (string, error)
}

type EnMasseService interface {
	CreateUser(userName, realm string) (*v1alpha1.User, error)
}
