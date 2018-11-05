package fuse

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	kfake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/integr8ly/integration-controller/pkg/fuse/client"
	v13 "k8s.io/client-go/kubernetes/typed/core/v1"

	v12 "k8s.io/api/core/v1"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
)

func TestIntegrator_Integrate(t *testing.T) {
	cases := []struct {
		Name           string
		Integration    *v1alpha1.Integration
		EnMasseService func() EnMasseService
		HttpRequester  func() httpRequester
		SecretClient   func() v13.SecretInterface
		ExpectError    bool
		Validate       func(*testing.T, *EnMasseServiceMock, *httpRequesterMock, *v1alpha1.Integration)
	}{
		{
			Name:        "integration should complete successfully calling all needed apis",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{}),
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					CreateUserFunc: func(userName string, realm string) (*v1alpha1.User, *v12.Secret, error) {
						return &v1alpha1.User{ID: "id", UserName: userName, Password: "pass"}, &v12.Secret{ObjectMeta: v1.ObjectMeta{Name: "secret"}}, nil
					},
				}
			},
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{
					DoFunc: func(r *http.Request) (*http.Response, error) {
						if r.Method == http.MethodPost && r.URL.String() == "http://syndesis-server.test.svc/api/v1/connections" {
							return &http.Response{
								StatusCode: 200,
								Status:     http.StatusText(200),
								Body:       ioutil.NopCloser(strings.NewReader(`{"id":"id"}`)),
							}, nil
						}
						return nil, errors.New("unknown request ")
					},
				}
			},
			SecretClient: func() v13.SecretInterface {
				fake := &kfake.Clientset{}
				fake.AddReactor("get", "secrets", func(ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v12.Secret{StringData: map[string]string{"user": "user", "pass": "pass"}}, nil
				})
				return fake.CoreV1().Secrets("test")
			},
			ExpectError: false,
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.CreateUserCalls()) != 1 {
					t.Fatal("expected create user to be called once but was called ", len(mock.CreateUserCalls()))
				}
				if len(mock2.DoCalls()) != 1 {
					t.Fatal("expected 1 called to do request but got ", len(mock2.DoCalls()))
				}
				if integration.Status.IntegrationMetaData[connectionIDKey] != "id" {
					t.Fatal("expected a connection id but got ", integration.Status.IntegrationMetaData[connectionIDKey])
				}

			},
		},
		{
			Name:        "integration should fail when it fails to create a new user",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{}),
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					CreateUserFunc: func(userName string, realm string) (*v1alpha1.User, *v12.Secret, error) {
						return nil, nil, errors.New("failed to create the user")
					},
				}
			},
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{}
			},
			SecretClient: func() v13.SecretInterface {
				fake := &kfake.Clientset{}
				return fake.CoreV1().Secrets("test")
			},
			ExpectError: true,
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.CreateUserCalls()) != 1 {
					t.Fatal("expected create user to be called once but was called ", len(mock.CreateUserCalls()))
				}
				if len(mock2.DoCalls()) != 0 {
					t.Fatal("expected 1 called to do request but got ", len(mock2.DoCalls()))
				}
				if integration != nil && integration.Status.IntegrationMetaData[connectionIDKey] != "" {
					t.Fatal("expected no connection id but got ", integration.Status.IntegrationMetaData[connectionIDKey])
				}

			},
		},
		{
			Name:        "integration should fail when it fails to create a new connection",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{}),
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					CreateUserFunc: func(userName string, realm string) (*v1alpha1.User, *v12.Secret, error) {
						return &v1alpha1.User{ID: "id", UserName: userName, Password: "pass"}, &v12.Secret{ObjectMeta: v1.ObjectMeta{Name: "secret"}}, nil
					},
				}
			},
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{
					DoFunc: func(r *http.Request) (*http.Response, error) {
						if r.Method == http.MethodPost && r.URL.String() == "http://syndesis-server.test.svc/api/v1/connections" {
							return &http.Response{
								StatusCode: 400,
								Status:     http.StatusText(400),
								Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
							}, nil
						}
						return nil, errors.New("unknown request ")
					},
				}
			},
			SecretClient: func() v13.SecretInterface {
				fake := &kfake.Clientset{}
				fake.AddReactor("get", "secrets", func(ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v12.Secret{StringData: map[string]string{"user": "user", "pass": "pass"}}, nil
				})
				return fake.CoreV1().Secrets("test")
			},
			ExpectError: true,
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.CreateUserCalls()) != 1 {
					t.Fatal("expected create user to be called once but was called ", len(mock.CreateUserCalls()))
				}
				if len(mock2.DoCalls()) != 1 {
					t.Fatal("expected 1 called to do request but got ", len(mock2.DoCalls()))
				}
				if integration != nil && integration.Status.IntegrationMetaData[connectionIDKey] != "" {
					t.Fatal("expected no connection id but got ", integration.Status.IntegrationMetaData[connectionIDKey])
				}

			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			enmasseMock := tc.EnMasseService()
			httpMock := tc.HttpRequester()
			cc := client.New(httpMock, tc.SecretClient(), "test", "satoken", "sauser")
			integreator := NewAddressSpaceIntegrator(enmasseMock, cc)
			ctx := context.TODO()
			integration, err := integreator.Integrate(ctx, tc.Integration)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, enmasseMock.(*EnMasseServiceMock), httpMock.(*httpRequesterMock), integration)
			}
		})
	}
}

func TestIntegrator_DisIntegrate(t *testing.T) {
	cases := []struct {
		Name           string
		Integration    *v1alpha1.Integration
		EnMasseService func() EnMasseService
		HttpRequester  func() httpRequester
		ExpectError    bool
		Validate       func(*testing.T, *EnMasseServiceMock, *httpRequesterMock, *v1alpha1.Integration)
	}{
		{
			Name:        "test disintegrate completes succesfully calling all needed apis",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{connectionIDKey: "testid"}),
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{
					DoFunc: func(r *http.Request) (*http.Response, error) {
						if r.Method == http.MethodDelete && r.URL.String() == "http://syndesis-server.test.svc/api/v1/connections/testid" {
							return &http.Response{
								Status:     http.StatusText(204),
								StatusCode: http.StatusNoContent,
							}, nil
						}
						return nil, errors.New("unexpectd request")
					},
				}
			},
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					DeleteUserFunc: func(userName string, realm string) error {
						return nil
					},
				}
			},
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.DeleteUserCalls()) != 1 {
					t.Fatal("expected one call to delete user but got ", len(mock.DeleteUserCalls()))
				}
				if len(mock2.DoCalls()) != 1 {
					t.Fatal("expected one call to request do but got ", len(mock2.DoCalls()))
				}
			},
			ExpectError: false,
		},
		{
			Name:        "test disintegrate still removes the connection even if removing the user from enmasse fails",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{connectionIDKey: "testid"}),
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{
					DoFunc: func(r *http.Request) (*http.Response, error) {
						if r.Method == http.MethodDelete && r.URL.String() == "http://syndesis-server.test.svc/api/v1/connections/testid" {
							return &http.Response{
								Status:     http.StatusText(204),
								StatusCode: http.StatusNoContent,
							}, nil
						}
						return nil, errors.New("unexpectd request")
					},
				}
			},
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					DeleteUserFunc: func(userName string, realm string) error {
						return errors.New("failed to remove user")
					},
				}
			},
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.DeleteUserCalls()) != 1 {
					t.Fatal("expected one call to delete user but got ", len(mock.DeleteUserCalls()))
				}
				if len(mock2.DoCalls()) != 1 {
					t.Fatal("expected one call to request do but got ", len(mock2.DoCalls()))
				}
			},
			ExpectError: false,
		},
		{
			Name:        "test disintegrate fails if removing the connection fails",
			Integration: validIntegration("test", "amqp", "fuse", "enmasse", map[string]string{connectionIDKey: "testid"}),
			HttpRequester: func() httpRequester {
				return &httpRequesterMock{
					DoFunc: func(r *http.Request) (*http.Response, error) {
						if r.Method == http.MethodDelete && r.URL.String() == "http://syndesis-server.test.svc/api/v1/connections/testid" {
							return &http.Response{
								Status:     http.StatusText(500),
								StatusCode: http.StatusInternalServerError,
							}, nil
						}
						return nil, errors.New("unexpectd request")
					},
				}
			},
			EnMasseService: func() EnMasseService {
				return &EnMasseServiceMock{
					DeleteUserFunc: func(userName string, realm string) error {
						return nil
					},
				}
			},
			Validate: func(t *testing.T, mock *EnMasseServiceMock, mock2 *httpRequesterMock, integration *v1alpha1.Integration) {
				if len(mock.DeleteUserCalls()) != 1 {
					t.Fatal("expected one call to delete user but got ", len(mock.DeleteUserCalls()))
				}
				if len(mock2.DoCalls()) != 1 {
					t.Fatal("expected one call to request do but got ", len(mock2.DoCalls()))
				}
			},
			ExpectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			enmasseSvc := tc.EnMasseService()
			httpRequester := tc.HttpRequester()
			ctx := context.TODO()
			cc := client.New(httpRequester, nil, "test", "satoken", "sauser")
			disintegrator := NewAddressSpaceIntegrator(enmasseSvc, cc)
			it, err := disintegrator.DisIntegrate(ctx, tc.Integration)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, enmasseSvc.(*EnMasseServiceMock), httpRequester.(*httpRequesterMock), it)
			}

		})
	}

}

func validIntegration(name, integrationType, client, provider string, meta map[string]string) *v1alpha1.Integration {
	return &v1alpha1.Integration{
		ObjectMeta: v1.ObjectMeta{
			Name:        name,
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.IntegrationSpec{
			IntegrationType: integrationType,
			Enabled:         true,
			Client:          client,
			ServiceProvider: provider,
		},
		Status: v1alpha1.IntegrationStatus{

			IntegrationMetaData: meta,
		},
	}
}
