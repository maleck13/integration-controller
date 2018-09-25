package k8s

import "github.com/operator-framework/operator-sdk/pkg/sdk"

type K8sCrudler struct {
}

func (k *K8sCrudler) List(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
	return sdk.List(namespace, into, opts...)
}
