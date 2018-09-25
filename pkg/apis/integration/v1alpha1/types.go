package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Integration `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              IntegrationSpec   `json:"spec"`
	Status            IntegrationStatus `json:"status,omitempty"`
}

func NewIntegration() *Integration {
	return &Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Integration",
			APIVersion: GroupName + "/" + Version,
		},
	}
}

type IntegrationSpec struct {
	ServiceProvider string `json:"serviceProvider"`
	IntegrationType string `json:"integrationType"`
	Client          string `json:"client"`
	Enabled         bool   `json:"enabled"`
}
type IntegrationStatus struct {
	Phase               Phase             `json:"phase"`
	IntegrationMetaData map[string]string `json:"metaData"`
	StatusMessage       string            `json:"statusMessage"`
}

type Phase string

type User struct {
	UserName string
	Password string
	ID       string
}

const (
	PhaseNone        Phase = ""
	PhaseAccepted    Phase = "accepted"
	PhaseIntegrating Phase = "integrating"
	PhaseComplete    Phase = "complete"
	Finalizer              = "integreatly.org"
)

// GetFinalizers gets the list of finalizers on obj
func GetFinalizers(obj runtime.Object) ([]string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	return accessor.GetFinalizers(), nil
}

// AddFinalizer adds value to the list of finalizers on obj
func AddFinalizer(obj runtime.Object, value string) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	finalizers.Insert(value)
	accessor.SetFinalizers(finalizers.List())
	return nil
}

// RemoveFinalizer removes the given value from the list of finalizers in obj, then returns a new list
// of finalizers after value has been removed.
func RemoveFinalizer(obj runtime.Object, value string) ([]string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	finalizers.Delete(value)
	newFinalizers := finalizers.List()
	accessor.SetFinalizers(newFinalizers)
	return newFinalizers, nil
}

const (
	FuseIntegrationTarget string = "fuse"
)
