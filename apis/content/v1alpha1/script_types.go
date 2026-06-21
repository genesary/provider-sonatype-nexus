package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScriptParameters defines desired state of a Nexus Groovy script.
type ScriptParameters struct {
	// Name of the script.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the script (e.g. "groovy").
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Content is the script body.
	// +kubebuilder:validation:Required
	Content string `json:"content"`
}

// ScriptObservation observed state of a Nexus Groovy script.
type ScriptObservation struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

// ScriptSpec defines desired state of Script.
type ScriptSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider ScriptParameters `json:"forProvider"`
}

// ScriptStatus defines observed state of Script.
type ScriptStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider ScriptObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=crossplane
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Script is a Nexus Groovy script managed resource.
// The Nexus Scripting API must be enabled via nexus.scripts.allowCreation=true.
// It is deprecated in Nexus 3.21+ and removed in 3.70+.
type Script struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScriptSpec   `json:"spec"`
	Status ScriptStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScriptList contains a list of Script managed resources.
type ScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Script `json:"items"`
}

// init registers the Script types with the scheme.
func init() {
	SchemeBuilder.Register(&Script{}, &ScriptList{})
}
