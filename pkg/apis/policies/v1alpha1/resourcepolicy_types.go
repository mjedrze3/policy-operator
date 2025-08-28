package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResourcePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourcePolicySpec `json:"spec"`
}

type TargetObject struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
}

type ResourceLimits struct {
	RAM  string `json:"ram,omitempty"`
	CPU  string `json:"cpu,omitempty"`
}

type ResourcePolicySpec struct {
	TargetObjects []TargetObject `json:"targetObjects"`
	Limits        ResourceLimits `json:"limits"`
	Policy        string         `json:"policy"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResourcePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ResourcePolicy `json:"items"`
}
