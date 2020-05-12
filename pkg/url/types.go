package url

import (
	"github.com/openshift/odo/pkg/envinfo"
	iextensionsv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// URL is
type URL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              URLSpec   `json:"spec,omitempty"`
	Status            URLStatus `json:"status,omitempty"`
}

// IngressURL contains Ingress Object and the URL status
type IngressURL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              iextensionsv1.IngressSpec `json:"spec,omitempty"`
	Status            URLStatus                 `json:"status,omitempty"`
}

// URLSpec is
type URLSpec struct {
	Host      string `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Port      int    `json:"port,omitempty"`
	Secure    bool   `json:"secure"`
	urlKind   envinfo.URLKind
	tLSSecret string
}

// AppList is a list of applications
type URLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []URL `json:"items"`
}

type IngressURLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressURL `json:"items"`
}

// URLStatus is Status of url
type URLStatus struct {
	// "Pushed" or "Not Pushed" or "Locally Delted"
	State StateType `json:"state"`
}

type StateType string

const (
	// StateTypePushed means that URL is present both locally and on cluster
	StateTypePushed = "Pushed"
	// StateTypeNotPushed means that URL is only in local config, but not on the cluster
	StateTypeNotPushed = "Not Pushed"
	// StateTypeLocallyDeleted means that URL was deleted from the local config, but it is still present on the cluster
	StateTypeLocallyDeleted = "Locally Deleted"
)
