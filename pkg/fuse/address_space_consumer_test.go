package fuse

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	v1alpha12 "github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/pkg/errors"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/integr8ly/integration-controller/pkg/apis/enmasse/v1"

	"github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	kerr "k8s.io/apimachinery/pkg/api/errors"
)

func TestAddressSpaceConsumer_Exists(t *testing.T) {
	cases := []struct {
		Name        string
		WatchNS     string
		ShouldExist bool
		Cruder      func() Crudler
		Validate    func(t *testing.T, mock *FuseCrudlerMock)
	}{{
		Name:        "test exists returns true when a fuse exists",
		WatchNS:     "test",
		ShouldExist: true,
		Cruder: func() Crudler {
			return &FuseCrudlerMock{
				ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
					slist := o.(*v1alpha1.SyndesisList)
					slist.Items = []v1alpha1.Syndesis{
						{},
					}
					return nil
				},
			}
		},
		Validate: func(t *testing.T, mock *FuseCrudlerMock) {
			if len(mock.ListCalls()) != 1 {
				t.Fatal("failed should have called list at least once")
			}
		},
	},
		{
			Name:        "test exists returns false when a fuse is missing",
			WatchNS:     "test",
			ShouldExist: false,
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
						return nil
					},
				}
			},
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.ListCalls()) != 1 {
					t.Fatal("failed should have called list at least once")
				}
			},
		},
		{
			Name:        "test exists returns false when err reading fuse",
			WatchNS:     "test",
			ShouldExist: false,
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
						return errors.New("some error ")
					},
				}
			},
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.ListCalls()) != 1 {
					t.Fatal("failed should have called list at least once")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mockCrud := tc.Cruder()
			consumer := NewAddressSpaceConsumer(tc.WatchNS, mockCrud)
			exists := consumer.Exists()
			if exists != tc.ShouldExist {
				t.Fatal("expected fuse to exists")
			}
			if tc.Validate != nil {
				tc.Validate(t, mockCrud.(*FuseCrudlerMock))
			}

		})
	}
}

func TestAddressSpaceConsumer_Validate(t *testing.T) {
	cases := []struct {
		Name        string
		WatchNS     string
		Address     *v1.AddressSpace
		ExpectError bool
	}{
		{
			Name:    "test valid address-space passes",
			WatchNS: "test",
			Address: &v1.AddressSpace{ObjectMeta: v12.ObjectMeta{
				Annotations: map[string]string{"enmasse.io/realm-name": "test", "enmasse.io/created-by": "me"}},
			},
			ExpectError: false,
		},
		{
			Name:        "test invalid address-space fails",
			WatchNS:     "test",
			Address:     &v1.AddressSpace{},
			ExpectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			consumer := NewAddressSpaceConsumer(tc.WatchNS, nil)
			err := consumer.Validate(tc.Address)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect and error but got one ", err)
			}

		})
	}
}

func TestAddressSpaceConsumer_CreateAvailableIntegration(t *testing.T) {
	cases := []struct {
		Name        string
		WatchNS     string
		Cruder      func() Crudler
		Address     *v1.AddressSpace
		Enabled     bool
		ExpectError bool
		Validate    func(t *testing.T, mock *FuseCrudlerMock)
	}{
		{
			Name:    "test create available integration from address-space successful",
			WatchNS: "test",
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
						slist := o.(*v1alpha1.SyndesisList)
						slist.Items = []v1alpha1.Syndesis{
							syndesis("me"),
						}
						return nil
					},
					CreateFunc: func(object sdk.Object) error {
						if object.(*v1alpha12.Integration).Spec.Enabled != true {
							return errors.New("expected the integration to be enabled")
						}
						if object.(*v1alpha12.Integration).Spec.IntegrationType != "amqp" {
							return errors.New("expected the integration to be of type amqp")
						}
						return nil
					},
				}
			},
			Address: validAddressSpace(),
			Enabled: true,
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.CreateCalls()) != 1 {
					t.Fatal("expected create to be called once but was called ", len(mock.CreateCalls()))
				}
			},
		},
		{
			Name:    "test create available integration from address-space fails",
			WatchNS: "test",
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
						slist := o.(*v1alpha1.SyndesisList)
						slist.Items = []v1alpha1.Syndesis{
							syndesis("notme"),
						}
						return nil
					},
					CreateFunc: func(object sdk.Object) error {
						if object.(*v1alpha12.Integration).Spec.Enabled != true {
							return errors.New("expected the integration to be enabled")
						}
						if object.(*v1alpha12.Integration).Spec.IntegrationType != "amqp" {
							return errors.New("expected the integration to be of type amqp")
						}
						return nil
					},
				}
			},
			Address: validAddressSpace(),
			Enabled: true,
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.CreateCalls()) != 0 {
					t.Fatal("expected create not to to be called was called ", len(mock.CreateCalls()))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mockCrud := tc.Cruder()
			consumer := NewAddressSpaceConsumer(tc.WatchNS, mockCrud)
			err := consumer.CreateAvailableIntegration(tc.Address, tc.WatchNS, tc.Enabled)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, mockCrud.(*FuseCrudlerMock))
			}

		})
	}

}

func TestAddressSpaceConsumer_RemoveAvailableIntegration(t *testing.T) {
	cases := []struct {
		Name        string
		WatchNS     string
		Cruder      func() Crudler
		Address     *v1.AddressSpace
		ExpectError bool
		Validate    func(t *testing.T, mock *FuseCrudlerMock)
	}{
		{
			Name: "test removing integration successful",
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					GetFunc: func(into sdk.Object, opts ...sdk.GetOption) error {
						return nil
					},

					DeleteFunc: func(object sdk.Object) error {
						return nil
					},
				}
			},
			Address: validAddressSpace(),
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.GetCalls()) != 1 {
					t.Fatal("expected get to be called once but was called ", len(mock.GetCalls()))
				}
				if len(mock.DeleteCalls()) != 1 {
					t.Fatal("expected delete to be called once but was called ", len(mock.DeleteCalls()))
				}
			},
		},
		{
			Name: "test removing integration fails when integration not found",
			Cruder: func() Crudler {
				return &FuseCrudlerMock{
					GetFunc: func(into sdk.Object, opts ...sdk.GetOption) error {
						return kerr.NewNotFound(schema.GroupResource{Group: "integreatly.org", Resource: "integration"}, "")
					},

					DeleteFunc: func(object sdk.Object) error {
						return nil
					},
				}
			},
			Validate: func(t *testing.T, mock *FuseCrudlerMock) {
				if len(mock.GetCalls()) != 1 {
					t.Fatal("expected get to be called once but was called ", len(mock.GetCalls()))
				}
				if len(mock.DeleteCalls()) != 0 {
					t.Fatal("expected delete not to be called but was called ", len(mock.DeleteCalls()))
				}
			},
			Address: validAddressSpace(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			cruder := tc.Cruder()
			consumer := NewAddressSpaceConsumer(tc.WatchNS, cruder)
			err := consumer.RemoveAvailableIntegration(tc.Address, tc.WatchNS)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, cruder.(*FuseCrudlerMock))
			}
		})
	}
}

func validAddressSpace() *v1.AddressSpace {
	return &v1.AddressSpace{ObjectMeta: v12.ObjectMeta{
		Annotations: map[string]string{"enmasse.io/realm-name": "test", "enmasse.io/created-by": "me"}},
		Status: v1.AddressSpaceStatus{
			EndPointStatuses: []v1.EndPointStatus{
				{
					Name: "messaging",
					ServicePorts: []v1.ServicePort{
						{
							Port: 5167,
						},
					},
					ServiceHost: "messaging.enmasse.svc",
				},
			},
		},
	}
}

func syndesis(createdBy string) v1alpha1.Syndesis {
	return v1alpha1.Syndesis{ObjectMeta: v12.ObjectMeta{Annotations: map[string]string{"syndesis.io/created-by": createdBy}}}
}
