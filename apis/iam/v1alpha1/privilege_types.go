package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrivilegeParameters defines the desired state of a Privilege.
type PrivilegeParameters struct {
	// Name of the privilege.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the privilege.
	// +optional
	Description *string `json:"description,omitempty"`

	// Type of the privilege.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=application;repository-view;repository-admin;repository-content-selector;script;wildcard
	Type string `json:"type"`

	// Actions allowed by this privilege.
	// For application: READ, EDIT, ADD, DELETE, ALL
	// For repository-view: BROWSE, READ, EDIT, ADD, DELETE, ALL
	// For repository-admin: BROWSE, READ, EDIT, ADD, DELETE, ALL
	// For repository-content-selector: BROWSE, READ, EDIT, ADD, DELETE, ALL
	// For script: READ, BROWSE, RUN, EDIT, ADD, DELETE, ALL
	// +optional
	Actions []string `json:"actions,omitempty"`

	// Domain for application type privilege.
	// +optional
	Domain *string `json:"domain,omitempty"`

	// Format for repository type privileges (e.g., maven2, npm, docker).
	// +optional
	Format *string `json:"format,omitempty"`

	// Repository name for repository type privileges.
	// Use * for all repositories.
	// +optional
	Repository *string `json:"repository,omitempty"`

	// ContentSelector name for repository-content-selector type privilege.
	// +optional
	ContentSelector *string `json:"contentSelector,omitempty"`

	// ScriptName for script type privilege.
	// +optional
	ScriptName *string `json:"scriptName,omitempty"`

	// Pattern for wildcard type privilege.
	// +optional
	Pattern *string `json:"pattern,omitempty"`
}

// PrivilegeObservation represents the observed state of a Privilege.
type PrivilegeObservation struct {
	// ReadOnly indicates if the privilege is read-only (built-in).
	ReadOnly *bool `json:"readOnly,omitempty"`
	// Description is the observed privilege description.
	Description string `json:"description,omitempty"`
	// Actions are the observed allowed actions.
	Actions []string `json:"actions,omitempty"`
}

// PrivilegeSpec defines the desired state of Privilege.
type PrivilegeSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider PrivilegeParameters `json:"forProvider"`
}

// PrivilegeStatus defines the observed state of Privilege.
type PrivilegeStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider PrivilegeObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.forProvider.type"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Privilege is the Schema for the privileges API.
type Privilege struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrivilegeSpec   `json:"spec"`
	Status PrivilegeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PrivilegeList contains a list of Privilege.
type PrivilegeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Privilege `json:"items"`
}

// init registers the Privilege types with the scheme.
func init() {
	SchemeBuilder.Register(&Privilege{}, &PrivilegeList{})
}
