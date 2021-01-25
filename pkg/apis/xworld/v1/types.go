package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"xworld.cn/pkg"
	//"xworld.cn/pkg/apis/xworld/v1"
	"xworld.cn/pkg/apis/xworld"
)

const (
	XServerStateOpen  XServerState = "Open"
	XServerStateClose XServerState = "Close"
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
	Image string `json:"image,omitempty"`
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

func (xs *XServer) ApplyDefaults() {
	// VersionAnnotation is the annotation that stores
	// the version of sdk which runs in a sidecar
	if xs.ObjectMeta.Annotations == nil {
		xs.ObjectMeta.Annotations = map[string]string{}
	}
	xs.ObjectMeta.Annotations[VersionAnnotation] = xworld.Version
	xs.ObjectMeta.Finalizers = append(xs.ObjectMeta.Finalizers, xworld.GroupName)

	xs.Spec.ApplyDefaults()
	xs.applyStatusDefaults()
}

func (xs *XServer) applyStatusDefaults() {
	if xs.Status.State == "" {
		xs.Status.State = XServerStateOpen
	}
}

func (xss *XServerSpec) ApplyDefaults() {
	xss.Image = ""
}
