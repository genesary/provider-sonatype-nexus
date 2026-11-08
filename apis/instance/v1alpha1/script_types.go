package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ScriptParameters are the configurable fields of a Script.
type ScriptParameters struct {
	// Name of the script in Nexus. It is the script's identity, so it cannot
	// change.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\-]{1}[a-zA-Z0-9_\-\.]*$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Type is the script language. Nexus only runs groovy scripts. Nexus
	// keeps the type a script was created with, so it cannot change.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=groovy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
	Type string `json:"type,omitempty"`

	// Content is the script source.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Content string `json:"content"`
}

// ScriptObservation is the observed state of a Script.
type ScriptObservation struct {
	// Name is the observed script name.
	Name string `json:"name,omitempty"`

	// Type is the observed script language.
	Type string `json:"type,omitempty"`

	// ContentSHA256 is the hex encoded SHA-256 digest of the observed script
	// source. The source itself is not copied into the status.
	ContentSHA256 string `json:"contentSha256,omitempty"`
}

// ScriptSpec defines the desired state of a Script.
type ScriptSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider ScriptParameters `json:"forProvider"`
}

// ScriptStatus holds the observed state of a Script.
type ScriptStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider ScriptObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Script manages a script stored in Nexus through its scripting API. The
// provider only stores the script; it never runs it. A stored script is
// arbitrary code that anyone allowed to run scripts can execute with the full
// rights of the Nexus server, so restrict who may create Scripts. Nexus
// disables script creation by default: it must be enabled with
// nexus.scripts.allowCreation=true in nexus.properties.
type Script struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScriptSpec   `json:"spec"`
	Status ScriptStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScriptList contains a list of Script.
type ScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Script `json:"items"`
}

// Script type metadata.
var (
	ScriptKind             = reflect.TypeFor[Script]().Name()
	ScriptGroupKind        = schema.GroupKind{Group: APIGroup, Kind: ScriptKind}.String()
	ScriptKindAPIVersion   = ScriptKind + "." + SchemeGroupVersion.String()
	ScriptGroupVersionKind = SchemeGroupVersion.WithKind(ScriptKind)
)

// init registers Script types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&Script{}, &ScriptList{})
}
