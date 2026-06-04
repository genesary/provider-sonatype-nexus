package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleParameters defines the desired state of a Role.
type RoleParameters struct {
	// ID is the unique identifier for the role.
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Name of the role.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the role.
	// +optional
	Description *string `json:"description,omitempty"`

	// Privileges assigned to this role.
	// +optional
	Privileges []string `json:"privileges,omitempty"`

	// Roles are other roles contained by this role.
	// +optional
	Roles []string `json:"roles,omitempty"`
}

// RoleObservation represents the observed state of a Role.
type RoleObservation struct {
	// Source of the role.
	Source *string `json:"source,omitempty"`

	// ReadOnly indicates if the role is read-only.
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// RoleSpec defines the desired state of Role.
type RoleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider RoleParameters `json:"forProvider"`
}

// RoleStatus defines the observed state of Role.
type RoleStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider RoleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Role is the Schema for the roles API.
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec"`
	Status RoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RoleList contains a list of Role.
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
