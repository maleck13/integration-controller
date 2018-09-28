package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddressSpace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AddressSpaceSpec   `json:"spec"`
	Status            AddressSpaceStatus `json:"status"`
}

type AddressSpaceSpec struct {
	Type string `json:"type"`
	Plan string `json:"plan"`
}

type AddressSpaceStatus struct {
	IsReady          bool             `json:"isReady"`
	EndPointStatuses []EndPointStatus `json:"endpointStatuses"`
}

type EndPointStatus struct {
	metav1.ObjectMeta `json:"metadata"`
	Name              string        `json:"name"`
	ServiceHost       string        `json:"serviceHost"`
	ServicePorts      []ServicePort `json:"servicePorts"`
	Host              string        `json:"host"`
	Port              int           `json:"port"`
}

type ServicePort struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AddressSpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AddressSpace `json:"items"`
}
