package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Connection `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Connection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ConnectionSpec   `json:"spec"`
	Status            ConnectionStatus `json:"status,omitempty"`
}

type ConnectionSpec struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	// secret name to look for in the current namespace, will have "user" "pass" fields
	Credentials string `json:"credentials"`
	// secret name to look for in the current namespace
	Certs string `json:"certs"`
}
type ConnectionStatus struct {
	SyndesisID string `json:"syndesisId"`
	Message    string `json:"message"`
}

func NewConnection(name, namespace, connType string) *Connection {
	return &Connection{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupName + "/" + Version,
			Kind:       ConnectionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Spec: ConnectionSpec{
			Type: connType,
		},
	}
}
