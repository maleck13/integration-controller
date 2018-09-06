package integration

import "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

type FuseService interface {
	AddAMQPConnection(user, pass, messageHost string) error
}

type EnMasseService interface {
	CreateUser(userName string) (*v1alpha1.User, error)
}
