package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserParameters defines the desired state of a User.
type UserParameters struct {
	// UserID is the unique identifier for the user.
	// +kubebuilder:validation:Required
	UserID string `json:"userId"`

	// FirstName of the user.
	// +kubebuilder:validation:Required
	FirstName string `json:"firstName"`

	// LastName of the user.
	// +kubebuilder:validation:Required
	LastName string `json:"lastName"`

	// EmailAddress of the user.
	// +kubebuilder:validation:Required
	EmailAddress string `json:"emailAddress"`

	// Status of the user account.
	// +kubebuilder:validation:Enum=active;locked;disabled;changepassword
	// +kubebuilder:default=active
	// +optional
	Status *string `json:"status,omitempty"`

	// Roles assigned to the user.
	// +optional
	Roles []string `json:"roles,omitempty"`

	// PasswordSecretRef is a reference to a secret with the user password.
	// +optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// Source of the user (default is "default" for local users).
	// +kubebuilder:default="default"
	// +optional
	Source *string `json:"source,omitempty"`
}

// UserObservation is the observed state of a User.
type UserObservation struct {
	// ReadOnly indicates if the user is read-only.
	ReadOnly *bool `json:"readOnly,omitempty"`

	// ExternalRoles are roles from external sources.
	ExternalRoles []string `json:"externalRoles,omitempty"`
}

// UserSpec defines the desired state of User.
type UserSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider UserParameters `json:"forProvider"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// User is the Schema for the users API.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

// init registers the User types with the scheme.
func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
