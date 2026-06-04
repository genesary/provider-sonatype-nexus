package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserTokenConfigurationParameters defines the desired state of
// UserTokenConfiguration.
type UserTokenConfigurationParameters struct {
	// Enabled determines if user tokens are enabled.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// ProtectContent determines if content should be protected with user tokens.
	// +optional
	ProtectContent *bool `json:"protectContent,omitempty"`

	// ExpirationEnabled determines if token expiration is enabled.
	// +optional
	ExpirationEnabled *bool `json:"expirationEnabled,omitempty"`

	// ExpirationDays is the number of days before a token expires.
	// Only applicable if ExpirationEnabled is true.
	// +optional
	ExpirationDays *int32 `json:"expirationDays,omitempty"`
}

// UserTokenConfigurationObservation represents the observed state of
// UserTokenConfiguration.
type UserTokenConfigurationObservation struct {
}

// UserTokenConfigurationSpec defines the desired state of
// UserTokenConfiguration.
type UserTokenConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider UserTokenConfigurationParameters `json:"forProvider"`
}

// UserTokenConfigurationStatus defines the observed state of
// UserTokenConfiguration.
type UserTokenConfigurationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider UserTokenConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".spec.forProvider.enabled"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// UserTokenConfiguration is the Schema for the usertokenconfigurations API.
// This is a singleton resource that configures user token settings.
type UserTokenConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserTokenConfigurationSpec   `json:"spec"`
	Status UserTokenConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserTokenConfigurationList contains a list of UserTokenConfiguration.
type UserTokenConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserTokenConfiguration `json:"items"`
}

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&UserTokenConfiguration{}, &UserTokenConfigurationList{})
}
