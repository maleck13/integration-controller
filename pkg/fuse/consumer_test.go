package fuse

import (
	"testing"

	"github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func TestConsumer_Exists(t *testing.T) {
	cases := []struct {
		Name        string
		WatchNS     string
		ShouldExist bool
		Cruder      func() FuseCrudler
		Validate    func(t *testing.T, mock *FuseCrudlerMock)
	}{{
		Name:        "test exists returns true when a fuse exists",
		WatchNS:     "test",
		ShouldExist: true,
		Cruder: func() FuseCrudler {
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
			Cruder: func() FuseCrudler {
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
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mockCrud := tc.Cruder()
			consumer := NewConsumer(tc.WatchNS, mockCrud)
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
