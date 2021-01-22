package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type XServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              XServerSpec   `json:"spec"`
	Status            XServerStatus `json:"status"`
}

type XServerSpec struct {
	Image    string `json:"image,omitempty"`
	Replicas int32  `json:"replicas"`
}

type XServerState string

type XServerStatus struct {
	State    XServerState `json:"state"`
	Address  string       `json:"address"`
	Port     int32        `json:"Port"`
	Capacity int32        `json:"capacity"`
	Count    int32        `json:"count"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XServerList is a list of XServer resources
type XServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []XServer `json:"items"`
}
