package enmasse

import (
	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Service struct {
	k8sclient   kubernetes.Interface
	routeClient routev1.RouteInterface
}

func NewService(k8sclient kubernetes.Interface, routeClient routev1.RouteInterface) *Service {
	return &Service{k8sclient: k8sclient, routeClient: routeClient}
}

func (s *Service) CreateUser(userName string) (*v1alpha1.User, error) {
	routes, err := s.routeClient.List(metav1.ListOptions{LabelSelector: "app=enmasse"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list routes for enmasse")
	}
	if len(routes.Items) == 0 {
		return nil, errors.Wrap(err, "no route for enmasse keycloak found")
	}
	route := routes.Items[0]
	logrus.Info("found route for keycloak ", route.Name)
	// find route to keycloak
	// find secret for keycloak
	// create user via keycloak api
	u := &v1alpha1.User{UserName: "", Password: ""}
	return u, nil
}
