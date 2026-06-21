package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmailConfigurationParameters defines the desired state of the Nexus SMTP
// email configuration.
type EmailConfigurationParameters struct {
	// Enabled determines whether outbound email is active.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// Host is the SMTP server hostname or IP address.
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Port is the SMTP server port.
	// +kubebuilder:validation:Required
	Port int `json:"port"`

	// Username is the SMTP account username.
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the SMTP
	// password. The password is write-only and is never returned by the API.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// FromAddress is the email address used as the sender.
	// +kubebuilder:validation:Required
	FromAddress string `json:"fromAddress"`

	// SubjectPrefix is prepended to all outbound email subjects.
	// +kubebuilder:validation:Optional
	SubjectPrefix *string `json:"subjectPrefix,omitempty"`

	// StartTlsEnabled enables STARTTLS support.
	// +kubebuilder:validation:Optional
	StartTlsEnabled *bool `json:"startTlsEnabled,omitempty"`

	// StartTlsRequired requires STARTTLS for all connections.
	// +kubebuilder:validation:Optional
	StartTlsRequired *bool `json:"startTlsRequired,omitempty"`

	// SslOnConnectEnabled enables SSL/TLS on connect (SMTPS).
	// +kubebuilder:validation:Optional
	SslOnConnectEnabled *bool `json:"sslOnConnectEnabled,omitempty"`

	// SslServerIdentityCheckEnabled enables SSL certificate identity checks.
	// +kubebuilder:validation:Optional
	SslServerIdentityCheckEnabled *bool `json:"sslServerIdentityCheckEnabled,omitempty"`

	// NexusTrustStoreEnabled uses the Nexus truststore for TLS verification.
	// +kubebuilder:validation:Optional
	NexusTrustStoreEnabled *bool `json:"nexusTrustStoreEnabled,omitempty"`
}

// EmailConfigurationObservation is the observed state of the Nexus email
// configuration.
type EmailConfigurationObservation struct {
	// Enabled is the observed enabled state.
	Enabled bool `json:"enabled,omitempty"`
	// Host is the observed SMTP hostname.
	Host string `json:"host,omitempty"`
	// Port is the observed SMTP port.
	Port int `json:"port,omitempty"`
	// Username is the observed SMTP username.
	Username string `json:"username,omitempty"`
	// FromAddress is the observed sender address.
	FromAddress string `json:"fromAddress,omitempty"`
	// SubjectPrefix is the observed subject prefix.
	SubjectPrefix string `json:"subjectPrefix,omitempty"`
	// StartTlsEnabled is the observed STARTTLS enabled state.
	StartTlsEnabled bool `json:"startTlsEnabled,omitempty"`
	// StartTlsRequired is the observed STARTTLS required state.
	StartTlsRequired bool `json:"startTlsRequired,omitempty"`
	// SslOnConnectEnabled is the observed SSL-on-connect state.
	SslOnConnectEnabled bool `json:"sslOnConnectEnabled,omitempty"`
	// SslServerIdentityCheckEnabled is the observed SSL identity check state.
	SslServerIdentityCheckEnabled bool `json:"sslServerIdentityCheckEnabled,omitempty"`
	// NexusTrustStoreEnabled is the observed Nexus truststore state.
	NexusTrustStoreEnabled bool `json:"nexusTrustStoreEnabled,omitempty"`
}

// EmailConfigurationSpec defines the desired state of EmailConfiguration.
type EmailConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider EmailConfigurationParameters `json:"forProvider"`
}

// EmailConfigurationStatus defines the observed state of EmailConfiguration.
type EmailConfigurationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider EmailConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".spec.forProvider.enabled"
// +kubebuilder:printcolumn:name="HOST",type="string",JSONPath=".spec.forProvider.host"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// EmailConfiguration is the Schema for the Nexus SMTP email configuration API.
type EmailConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmailConfigurationSpec   `json:"spec"`
	Status EmailConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EmailConfigurationList contains a list of EmailConfiguration.
type EmailConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []EmailConfiguration `json:"items"`
}

// init registers EmailConfiguration types with the scheme.
func init() {
	SchemeBuilder.Register(&EmailConfiguration{}, &EmailConfigurationList{})
}
