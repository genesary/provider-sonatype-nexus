package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserTokenConfigurationParameters defines the desired state.
type UserTokenConfigurationParameters struct {
	// Enabled determines if user tokens are enabled.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// ProtectContent determines if content should be protected with user tokens.
	// +kubebuilder:validation:Optional
	ProtectContent *bool `json:"protectContent,omitempty"`

	// ExpirationEnabled determines if token expiration is enabled.
	// +kubebuilder:validation:Optional
	ExpirationEnabled *bool `json:"expirationEnabled,omitempty"`

	// ExpirationDays is the number of days before a token expires.
	// Only applicable if ExpirationEnabled is true.
	// +kubebuilder:validation:Optional
	ExpirationDays *int32 `json:"expirationDays,omitempty"`
}

// UserTokenConfigurationObservation is the observed state.
type UserTokenConfigurationObservation struct {
	// Enabled is the observed enabled state.
	Enabled bool `json:"enabled,omitempty"`
	// ProtectContent is the observed protect content setting.
	ProtectContent bool `json:"protectContent,omitempty"`
	// ExpirationEnabled is the observed expiration enabled state.
	ExpirationEnabled bool `json:"expirationEnabled,omitempty"`
	// ExpirationDays is the observed expiration days.
	ExpirationDays int `json:"expirationDays,omitempty"`
}

// UserTokenConfigurationSpec defines the desired state.
type UserTokenConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider UserTokenConfigurationParameters `json:"forProvider"`
}

// UserTokenConfigurationStatus defines the observed state.
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

// UserTokenConfiguration configures user token settings.
// This is a singleton resource managing Nexus user token configuration.
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

// init registers the UserTokenConfiguration types with the scheme.
func init() {
	SchemeBuilder.Register(&UserTokenConfiguration{}, &UserTokenConfigurationList{})
}
