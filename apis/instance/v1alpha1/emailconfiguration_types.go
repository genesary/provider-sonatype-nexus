package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EmailConfigurationParameters are the configurable Nexus email server
// settings (Administration, System, Email Server). They replace the whole
// Nexus configuration: an unset boolean is applied as its documented default
// and an unset string as empty.
type EmailConfigurationParameters struct {
	// Enabled indicates whether Nexus sends emails.
	// If unset, will default to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`

	// Host is the SMTP server host name or IP address. Nexus resolves it when
	// the configuration is applied and refuses a host it cannot resolve. It
	// also refuses a host that resolves to a private, loopback or link-local
	// address while SSRF protection is enabled, unless the host or address is
	// allowed by the SSRFProtection settings.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`

	// Port is the SMTP server port.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port"`

	// Username is the SMTP user name. If unset, Nexus connects without
	// authentication.
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef references the Kubernetes Secret key holding the
	// SMTP password. The secret must be in the same namespace as the
	// EmailConfiguration.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.LocalSecretKeySelector `json:"passwordSecretRef,omitempty"`

	// FromAddress is the sender address of the emails Nexus sends.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	FromAddress string `json:"fromAddress"`

	// SubjectPrefix is prepended to the subject of every email Nexus sends.
	// +kubebuilder:validation:Optional
	SubjectPrefix *string `json:"subjectPrefix,omitempty"`

	// StartTLSEnabled enables STARTTLS when the SMTP server supports it.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	StartTLSEnabled *bool `json:"startTlsEnabled,omitempty"`

	// StartTLSRequired refuses to send emails when the SMTP server does not
	// support STARTTLS.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	StartTLSRequired *bool `json:"startTlsRequired,omitempty"`

	// SSLOnConnectEnabled connects to the SMTP server over SSL/TLS.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	SSLOnConnectEnabled *bool `json:"sslOnConnectEnabled,omitempty"`

	// SSLServerIdentityCheckEnabled verifies the SMTP server certificate
	// when STARTTLS or SSL/TLS is used. Disabling it lets anyone able to
	// intercept the connection read the SMTP password and the emails. It has
	// no effect on a plain connection.
	// If unset, will default to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	SSLServerIdentityCheckEnabled *bool `json:"sslServerIdentityCheckEnabled,omitempty"`

	// NexusTrustStoreEnabled verifies the SMTP server certificate against
	// the Nexus truststore (see SecuritySSLTruststore) rather than the JVM
	// one.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	NexusTrustStoreEnabled *bool `json:"nexusTrustStoreEnabled,omitempty"`
}

// EmailConfigurationObservation is the observed Nexus email server state.
type EmailConfigurationObservation struct {
	// Enabled reflects whether Nexus sends emails.
	Enabled bool `json:"enabled"`

	// Host is the observed SMTP server host.
	Host string `json:"host,omitempty"`

	// Port is the observed SMTP server port.
	Port int `json:"port,omitempty"`

	// Username is the observed SMTP user name.
	Username string `json:"username,omitempty"`

	// FromAddress is the observed sender address.
	FromAddress string `json:"fromAddress,omitempty"`

	// SubjectPrefix is the observed email subject prefix.
	SubjectPrefix string `json:"subjectPrefix,omitempty"`

	// StartTLSEnabled reflects whether STARTTLS is enabled.
	StartTLSEnabled bool `json:"startTlsEnabled,omitempty"`

	// StartTLSRequired reflects whether STARTTLS is required.
	StartTLSRequired bool `json:"startTlsRequired,omitempty"`

	// SSLOnConnectEnabled reflects whether Nexus connects over SSL/TLS.
	SSLOnConnectEnabled bool `json:"sslOnConnectEnabled,omitempty"`

	// SSLServerIdentityCheckEnabled reflects whether the server certificate
	// is verified.
	SSLServerIdentityCheckEnabled bool `json:"sslServerIdentityCheckEnabled,omitempty"`

	// NexusTrustStoreEnabled reflects whether the Nexus truststore is used.
	NexusTrustStoreEnabled bool `json:"nexusTrustStoreEnabled,omitempty"`

	// CredentialsFingerprint identifies the password last applied to Nexus,
	// which Nexus never reports back. It changes when the referenced Secret
	// changes or is repointed, which triggers an update.
	CredentialsFingerprint string `json:"credentialsFingerprint,omitempty"`
}

// EmailConfigurationSpec defines the desired Nexus email server state.
type EmailConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider EmailConfigurationParameters `json:"forProvider"`
}

// EmailConfigurationStatus holds the observed Nexus email server state.
type EmailConfigurationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider EmailConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".status.atProvider.enabled"
// +kubebuilder:printcolumn:name="HOST",type="string",JSONPath=".status.atProvider.host"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// EmailConfiguration manages the Nexus email server settings, which Nexus
// uses to send notifications such as task alerts and password resets.
//
// It is a singleton: declare at most one EmailConfiguration per Nexus
// instance, across all namespaces and ProviderConfigs pointing at that
// instance. The provider does not enforce this. With two of them, each
// reconcile replaces the settings the other applied. Deleting it resets the
// email server settings, which disables email.
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

// EmailConfiguration type metadata.
var (
	EmailConfigurationKind             = reflect.TypeFor[EmailConfiguration]().Name()
	EmailConfigurationGroupKind        = schema.GroupKind{Group: APIGroup, Kind: EmailConfigurationKind}.String()
	EmailConfigurationKindAPIVersion   = EmailConfigurationKind + "." + SchemeGroupVersion.String()
	EmailConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(EmailConfigurationKind)
)

// init registers EmailConfiguration types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&EmailConfiguration{}, &EmailConfigurationList{})
}
