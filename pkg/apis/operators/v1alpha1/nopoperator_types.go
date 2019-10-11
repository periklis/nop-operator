package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type OperatorChannel struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Version  string `json:"version"`
	Replicas int    `json:"replicas,omitempty"`
}

// NopOperatorSpec defines the desired state of NopOperator
// +k8s:openapi-gen=true
type NopOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Operators []OperatorChannel `json:"operators"`
}

// NopOperatorStatus defines the observed state of NopOperator
// +k8s:openapi-gen=true
type NopOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NopOperator is the Schema for the nopoperators API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nopoperators,scope=Namespaced
type NopOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NopOperatorSpec   `json:"spec,omitempty"`
	Status NopOperatorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NopOperatorList contains a list of NopOperator
type NopOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NopOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NopOperator{}, &NopOperatorList{})
}
