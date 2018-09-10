// love the name of this it is actually a unit test:)
package integration_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/integr8ly/integration-controller/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

	"github.com/integr8ly/integration-controller/pkg/integration"
)

func TestAccept(t *testing.T) {
	cases := []struct {
		Name        string
		Integration *v1alpha1.Integration
		ExpectErr   bool
		Validate    func(err error, integration *v1alpha1.Integration, t *testing.T)
	}{
		{
			Name: "should not be accepted when not enabled",
			Integration: &v1alpha1.Integration{
				Spec: v1alpha1.IntegrationSpec{
					Service:         "fuse",
					IntegrationType: "amqp",
				},
			},
			ExpectErr: true,
			Validate: func(err error, integration *v1alpha1.Integration, t *testing.T) {
				if !errors.IsNotEnabledErr(err) {
					t.Fatal("Expeted a not enabled error but it was a ", reflect.TypeOf(err))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			r := &integration.Reconciler{}
			updated, err := r.Accept(context.TODO(), tc.Integration)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got not")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("did not expect an error but got ", err)
			}
			if tc.Validate != nil {
				tc.Validate(err, updated, t)
			}
		})
	}
}
