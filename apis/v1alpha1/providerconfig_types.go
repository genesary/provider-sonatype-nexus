package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderConfigSpec defines the desired state of ProviderConfig.
type ProviderConfigSpec struct {
	// URL is the URL of the Nexus instance.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// InsecureSkipVerify indicates whether to skip TLS certificate verification.
	// +kubebuilder:validation:Optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`

	// Username credentials to authenticate with Nexus.
	Username ProviderCredentials `json:"username"`

	// Password credentials to authenticate with Nexus.
	Password ProviderCredentials `json:"password"`
}

// ProviderCredentials defines the credentials for Nexus authentication. The
// referenced Secret is always in the ProviderConfig's namespace.
type ProviderCredentials struct {
	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=None;Secret
	Source xpv2.CredentialsSource `json:"source"`

	// SecretRef selects a key of a Secret in the ProviderConfig's namespace.
	// +kubebuilder:validation:Optional
	SecretRef *xpv2.LocalSecretKeySelector `json:"secretRef,omitempty"`
}

// ClusterProviderConfigSpec defines the desired state of ClusterProviderConfig.
type ClusterProviderConfigSpec struct {
	// URL is the URL of the Nexus instance.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// InsecureSkipVerify indicates whether to skip TLS certificate verification.
	// +kubebuilder:validation:Optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`

	// Username credentials to authenticate with Nexus.
	Username ClusterProviderCredentials `json:"username"`

	// Password credentials to authenticate with Nexus.
	Password ClusterProviderCredentials `json:"password"`
}

// ClusterProviderCredentials defines the credentials for Nexus authentication
// of a cluster-scoped config, whose secretRef names its own namespace.
type ClusterProviderCredentials struct {
	// +kubebuilder:validation:Optional
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
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="USERNAME-SECRET",type="string",JSONPath=".spec.username.secretRef.name",priority=1
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,nexus}

// ProviderConfig is the Schema for the namespaced providerconfigs API.
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
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,nexus}

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

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="USERNAME-SECRET",type="string",JSONPath=".spec.username.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,nexus}

// ClusterProviderConfig is the Schema for the cluster-scoped
// providerconfigs API.
type ClusterProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterProviderConfigSpec `json:"spec"`
	Status ProviderConfigStatus      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterProviderConfigList contains a list of ClusterProviderConfig.
type ClusterProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,nexus}

// ClusterProviderConfigUsage indicates that a resource is using
// a ClusterProviderConfig.
type ClusterProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ClusterProviderConfigUsageList contains a list of ClusterProviderConfigUsage.
type ClusterProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterProviderConfigUsage `json:"items"`
}

// init registers these types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
	SchemeBuilder.Register(&ClusterProviderConfig{}, &ClusterProviderConfigList{})
	SchemeBuilder.Register(&ClusterProviderConfigUsage{}, &ClusterProviderConfigUsageList{})
}
