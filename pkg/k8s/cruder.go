package k8s

import "github.com/operator-framework/operator-sdk/pkg/sdk"

type K8sCrudler struct {
}

func (k *K8sCrudler) List(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
	return sdk.List(namespace, into, opts...)
}

func (k *K8sCrudler) Get(into sdk.Object, opts ...sdk.GetOption) error {
	return sdk.Get(into, opts...)
}

func (k *K8sCrudler) Create(object sdk.Object) (err error) {
	return sdk.Create(object)
}

func (k *K8sCrudler) Delete(object sdk.Object) error {
	return sdk.Delete(object)
}

func (k *K8sCrudler) Update(object sdk.Object) error {
	return sdk.Update(object)
}
