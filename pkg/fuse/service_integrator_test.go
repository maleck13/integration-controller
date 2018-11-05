package fuse

import (
	"context"
	"errors"
	"testing"

	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/api/core/v1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	v1alpha12 "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

func TestHTTPIntegrator_Integrate(t *testing.T) {
	cases := []struct {
		Name             string
		k8sCruder        func() Crudler
		Integration      func() *v1alpha1.Integration
		ConnectionCruder func() FuseClient
		ExpectErr        bool
		Validate         func(t *testing.T, crudMock *CrudlerMock, fuseMock *FuseClientMock, integration *v1alpha1.Integration)
	}{
		{
			Name:        "test integration succeeds by creating a new connection",
			k8sCruder:   successFullCruder,
			Integration: validHttpIntegration,
			ConnectionCruder: func() FuseClient {
				return &FuseClientMock{
					CreateConnectionFunc: func(c *v1alpha12.Connection) (*v1alpha12.Connection, error) {
						return &v1alpha12.Connection{Status: v1alpha12.ConnectionStatus{SyndesisID: "id"}}, nil
					},
					ConnectionExistsFunc: func(c *v1alpha12.Connection) (bool, error) {
						return false, nil
					},
				}
			},
			Validate: func(t *testing.T, crudMock *CrudlerMock, fuseMock *FuseClientMock, integration *v1alpha1.Integration) {
				if len(crudMock.UpdateCalls()) != 1 {
					t.Fatal("expected update to be called once but was called", len(crudMock.UpdateCalls()))
				}
				if len(crudMock.GetCalls()) != 1 {
					t.Fatal("expected get to be called once but was called", len(crudMock.GetCalls()))
				}
				if len(fuseMock.ConnectionExistsCalls()) != 1 {
					t.Fatal("expected ConnectionExists to be called once but was called", len(fuseMock.ConnectionExistsCalls()))
				}
				if len(fuseMock.CreateConnectionCalls()) != 1 {
					t.Fatal("expected create connection to be called once but was called", len(fuseMock.CreateConnectionCalls()))
				}
				if len(fuseMock.UpdateConnectionCalls()) != 0 {
					t.Fatal("expected update connection not to be called  was called", len(fuseMock.UpdateConnectionCalls()))
				}
				if integration.Status.Phase != v1alpha1.PhaseComplete {
					t.Fatal("expected integration to be complete but was ", integration.Status.Phase)
				}
				if integration.Status.IntegrationMetaData[connectionIDKey] != "id" {
					t.Fatal("expected connection id to be id but was ", integration.Status.IntegrationMetaData[connectionIDKey])
				}
			},
		},
		{
			Name: "test integration succeeds by updating an existing connection",
			ConnectionCruder: func() FuseClient {
				return &FuseClientMock{
					CreateConnectionFunc: func(c *v1alpha12.Connection) (*v1alpha12.Connection, error) {
						return nil, errors2.NewAlreadyExists(schema.GroupResource{Group: "syndesis.io", Resource: "connection"}, "connection")
					},
					ConnectionExistsFunc: func(c *v1alpha12.Connection) (bool, error) {
						return true, nil
					},
					UpdateConnectionFunc: func(c *v1alpha12.Connection) (*v1alpha12.Connection, error) {
						conn := v1alpha12.NewConnection("test", "test", "http")
						conn.Status.SyndesisID = "id"
						return conn, nil
					},
				}
			},
			Integration: validHttpIntegration,
			k8sCruder:   successFullCruder,
			Validate: func(t *testing.T, crudMock *CrudlerMock, fuseMock *FuseClientMock, integration *v1alpha1.Integration) {
				if len(crudMock.UpdateCalls()) != 1 {
					t.Fatal("expected update to be called once but was called", len(crudMock.UpdateCalls()))
				}
				if len(crudMock.GetCalls()) != 1 {
					t.Fatal("expected get to be called once but was called", len(crudMock.GetCalls()))
				}
				if len(fuseMock.ConnectionExistsCalls()) != 1 {
					t.Fatal("expected ConnectionExists to be called once but was called", len(fuseMock.ConnectionExistsCalls()))
				}
				if len(fuseMock.CreateConnectionCalls()) != 0 {
					t.Fatal("expected create connection not to be called but was called", len(fuseMock.CreateConnectionCalls()))
				}
				if len(fuseMock.UpdateConnectionCalls()) != 1 {
					t.Fatal("expected update connection to be called once  was called", len(fuseMock.UpdateConnectionCalls()))
				}
				if integration.Status.Phase != v1alpha1.PhaseComplete {
					t.Fatal("expected integration to be complete but was ", integration.Status.Phase)
				}
				if integration.Status.IntegrationMetaData[connectionIDKey] != "id" {
					t.Fatal("expected connection id to be id but was ", integration.Status.IntegrationMetaData[connectionIDKey])
				}
			},
		},
		{
			Name:        "test integration fails when we fail to create the connection",
			Integration: validHttpIntegration,
			ExpectErr:   true,
			k8sCruder:   successFullCruder,
			ConnectionCruder: func() FuseClient {
				return &FuseClientMock{
					CreateConnectionFunc: func(c *v1alpha12.Connection) (*v1alpha12.Connection, error) {
						return nil, errors.New("something went very wrong")
					},
					ConnectionExistsFunc: func(c *v1alpha12.Connection) (bool, error) {
						return false, nil
					},
				}
			},
			Validate: func(t *testing.T, crudMock *CrudlerMock, fuseMock *FuseClientMock, integration *v1alpha1.Integration) {
				if len(crudMock.UpdateCalls()) != 1 {
					t.Fatal("expected update to be called once but was called", len(crudMock.UpdateCalls()))
				}
				if len(crudMock.GetCalls()) != 1 {
					t.Fatal("expected get to be called once but was called", len(crudMock.GetCalls()))
				}
				if len(fuseMock.ConnectionExistsCalls()) != 1 {
					t.Fatal("expected ConnectionExists to be called once but was called", len(fuseMock.ConnectionExistsCalls()))
				}
				if len(fuseMock.CreateConnectionCalls()) != 1 {
					t.Fatal("expected create connection to be called once but was called", len(fuseMock.CreateConnectionCalls()))
				}
				if len(fuseMock.UpdateConnectionCalls()) != 0 {
					t.Fatal("expected update connection not to be called but was called", len(fuseMock.CreateConnectionCalls()))
				}
				if integration.Status.Phase == v1alpha1.PhaseComplete {
					t.Fatal("the integration should not be set to complete")
				}
				if integration.Status.IntegrationMetaData[connectionIDKey] != "" {
					t.Fatal("expected connection id to be empty but was ", integration.Status.IntegrationMetaData[connectionIDKey])
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			k8sCruder := tc.k8sCruder()
			connectionCruder := tc.ConnectionCruder()
			integreator := NewHTTPIntegrator(k8sCruder, connectionCruder, "test")
			i, err := integreator.Integrate(context.TODO(), tc.Integration())
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, k8sCruder.(*CrudlerMock), connectionCruder.(*FuseClientMock), i)
			}
		})
	}
}

func validHttpIntegration() *v1alpha1.Integration {
	i := v1alpha1.NewIntegration("test")
	i.Spec.IntegrationType = "http"
	i.Spec.Enabled = true
	i.Spec.ServiceProvider = "fuse"
	i.Status.IntegrationMetaData = map[string]string{"port": "8080", "scheme": "http", "host": "my.test.svc", "api-path": ""}
	return i
}

func successFullCruder() Crudler {
	return &CrudlerMock{
		GetFunc: func(into sdk.Object, opts ...sdk.GetOption) error {
			return nil
		},
		UpdateFunc: func(object sdk.Object) error {
			s := object.(*v1.Service)
			if _, ok := s.Annotations[serviceDiscoveryPath]; !ok {
				return errors.New("missing annotation " + serviceDiscoveryPath)
			}
			if _, ok := s.Annotations[serviceDiscoveryScheme]; !ok {
				return errors.New("missing annotation " + serviceDiscoveryScheme)
			}
			if _, ok := s.Annotations[serviceDiscoveryPort]; !ok {
				return errors.New("missing annotation " + serviceDiscoveryPort)
			}
			return nil
		},
	}
}
