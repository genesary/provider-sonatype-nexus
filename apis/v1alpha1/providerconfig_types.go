package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderConfigSpec defines the desired state of ProviderConfig.
type ProviderConfigSpec struct {
	// Credentials required to authenticate to Nexus.
	Credentials ProviderCredentials `json:"credentials"`
}

// ProviderCredentials defines the credentials for Nexus authentication.
type ProviderCredentials struct {
	// SecretRef is a reference to a secret containing credentials.
	// +optional
	xpv2.CommonCredentialSelectors `json:",inline"`

	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=None;Secret
	Source xpv2.CredentialsSource `json:"source"`
}

// ProviderConfigStatus defines the observed state of ProviderConfig.
type ProviderConfigStatus struct {
	xpv2.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,nexus}

// ProviderConfig is the Schema for the providerconfigs API.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".spec.providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".spec.resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".spec.resourceRef.name"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,nexus}

// ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ProviderConfigUsage `json:"items"`
}

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
}
