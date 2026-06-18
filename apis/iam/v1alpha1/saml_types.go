package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SAMLParameters defines the desired state of SAML configuration.
type SAMLParameters struct {
	// IdpMetadata is the SAML Identity Provider metadata XML.
	// +kubebuilder:validation:Required
	IdpMetadata string `json:"idpMetadata"`

	// EntityId is the SAML entity ID for this service provider.
	// +kubebuilder:validation:Required
	EntityId string `json:"entityId"`

	// UsernameAttribute is the SAML attribute to use for the username.
	// +kubebuilder:validation:Required
	UsernameAttribute string `json:"usernameAttribute"`

	// FirstNameAttribute is the SAML attribute to use for the first name.
	// +optional
	FirstNameAttribute *string `json:"firstNameAttribute,omitempty"`

	// LastNameAttribute is the SAML attribute to use for the last name.
	// +optional
	LastNameAttribute *string `json:"lastNameAttribute,omitempty"`

	// EmailAttribute is the SAML attribute to use for the email address.
	// +optional
	EmailAttribute *string `json:"emailAttribute,omitempty"`

	// GroupsAttribute is the SAML attribute to use for group membership.
	// +optional
	GroupsAttribute *string `json:"groupsAttribute,omitempty"`

	// ValidateResponseSignature determines if the SAML response signature
	// should be validated.
	// +optional
	ValidateResponseSignature *bool `json:"validateResponseSignature,omitempty"`

	// ValidateAssertionSignature determines if the SAML assertion signature
	// should be validated.
	// +optional
	ValidateAssertionSignature *bool `json:"validateAssertionSignature,omitempty"`
}

// SAMLObservation represents the observed state of SAML configuration.
type SAMLObservation struct {
}

// SAMLSpec defines the desired state of SAML.
type SAMLSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider SAMLParameters `json:"forProvider"`
}

// SAMLStatus defines the observed state of SAML.
type SAMLStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider SAMLObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// SAML is the Schema for the samls API.
// This is a singleton resource that configures SAML SSO.
type SAML struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SAMLSpec   `json:"spec"`
	Status SAMLStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SAMLList contains a list of SAML.
type SAMLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SAML `json:"items"`
}

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&SAML{}, &SAMLList{})
}
