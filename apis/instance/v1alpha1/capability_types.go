package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CapabilityParameters are the configurable fields of a Capability.
type CapabilityParameters struct {
	// TypeId is the capability type identifier (e.g. "DockerBearerTokenRealm").
	// +kubebuilder:validation:Required
	TypeId string `json:"typeId"`

	// Enabled specifies whether the capability is active.
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled"`

	// Notes is optional free-form text about the capability.
	// +kubebuilder:validation:Optional
	Notes string `json:"notes,omitempty"`

	// Properties are type-specific key/value configuration pairs.
	// +kubebuilder:validation:Optional
	Properties map[string]string `json:"properties,omitempty"`
}

// CapabilityObservation represents the observed state of a Capability.
type CapabilityObservation struct {
	// ID is the server-assigned capability identifier.
	ID string `json:"id,omitempty"`
}

// CapabilitySpec defines the desired state of a Capability.
type CapabilitySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider CapabilityParameters `json:"forProvider"`
}

// CapabilityStatus represents the observed state of a Capability.
type CapabilityStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider CapabilityObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Capability is the Schema for the capabilities API.
type Capability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CapabilitySpec   `json:"spec"`
	Status CapabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CapabilityList contains a list of Capability.
type CapabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Capability `json:"items"`
}

// Capability type metadata.
var (
	CapabilityKind             = reflect.TypeFor[Capability]().Name()
	CapabilityGroupKind        = schema.GroupKind{Group: APIGroup, Kind: CapabilityKind}.String()
	CapabilityKindAPIVersion   = CapabilityKind + "." + SchemeGroupVersion.String()
	CapabilityGroupVersionKind = SchemeGroupVersion.WithKind(CapabilityKind)
)

// init registers Capability types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&Capability{}, &CapabilityList{})
}
