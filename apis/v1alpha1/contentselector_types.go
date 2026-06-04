package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContentSelectorParameters defines the desired state of a ContentSelector.
type ContentSelectorParameters struct {
	// Name of the content selector.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the content selector.
	// +optional
	Description *string `json:"description,omitempty"`

	// Expression is the CSEL (Content Selector Expression Language) expression
	// used to select content.
	// +kubebuilder:validation:Required
	Expression string `json:"expression"`
}

// ContentSelectorObservation represents the observed state of a ContentSelector.
type ContentSelectorObservation struct {
}

// ContentSelectorSpec defines the desired state of ContentSelector.
type ContentSelectorSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider ContentSelectorParameters `json:"forProvider"`
}

// ContentSelectorStatus defines the observed state of ContentSelector.
type ContentSelectorStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider ContentSelectorObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// ContentSelector is the Schema for the contentselectors API.
type ContentSelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContentSelectorSpec   `json:"spec"`
	Status ContentSelectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContentSelectorList contains a list of ContentSelector.
type ContentSelectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ContentSelector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContentSelector{}, &ContentSelectorList{})
}
