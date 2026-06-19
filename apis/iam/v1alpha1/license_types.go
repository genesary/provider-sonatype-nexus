package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LicenseEndpointCredentials holds optional HTTP credentials
// for the license endpoint.
type LicenseEndpointCredentials struct {
	// UsernameSecretRef references a secret key with the HTTP username.
	// +kubebuilder:validation:Optional
	UsernameSecretRef *xpv2.SecretKeySelector `json:"usernameSecretRef,omitempty"`

	// PasswordSecretRef references a secret key with the HTTP password
	// or bearer token.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// LicenseParameters defines the desired state of License.
type LicenseParameters struct {
	// LicenseSecretRef references a Kubernetes secret containing the license
	// data (Behavior 1). Mutually exclusive with EndpointURL.
	// +kubebuilder:validation:Optional
	LicenseSecretRef *xpv2.SecretKeySelector `json:"licenseSecretRef,omitempty"`

	// EndpointURL is the URL from which the license file is fetched
	// (Behavior 2). Mutually exclusive with LicenseSecretRef.
	// +kubebuilder:validation:Optional
	EndpointURL *string `json:"endpointUrl,omitempty"`

	// EndpointCredentials are optional HTTP credentials for the license
	// endpoint. Used only when EndpointURL is set.
	// +kubebuilder:validation:Optional
	EndpointCredentials *LicenseEndpointCredentials `json:"endpointCredentials,omitempty"`
}

// LicenseObservation defines the observed state of License.
type LicenseObservation struct {
	// Fingerprint is the license fingerprint reported by Nexus.
	Fingerprint string `json:"fingerprint,omitempty"`

	// ContactEmail is the license contact email.
	ContactEmail string `json:"contactEmail,omitempty"`

	// ContactName is the license contact name.
	ContactName string `json:"contactName,omitempty"`

	// ContactCompany is the license contact company.
	ContactCompany string `json:"contactCompany,omitempty"`

	// EffectiveDate is the license effective date.
	EffectiveDate string `json:"effectiveDate,omitempty"`

	// ExpirationDate is the license expiration date.
	ExpirationDate string `json:"expirationDate,omitempty"`

	// LicenseType is the type of license installed.
	LicenseType string `json:"licenseType,omitempty"`

	// LicensedUsers is the number of licensed users.
	LicensedUsers string `json:"licensedUsers,omitempty"`

	// InstalledHash is the SHA-256 of the last license bytes applied by
	// this controller. Used to detect drift when the desired license
	// changes in the source secret or endpoint.
	InstalledHash string `json:"installedHash,omitempty"`
}

// LicenseSpec defines the desired state of License.
type LicenseSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider LicenseParameters `json:"forProvider"`
}

// LicenseStatus defines the observed state of License.
type LicenseStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider LicenseObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXPIRATION",type="string",JSONPath=".status.atProvider.expirationDate"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// License manages the Sonatype Nexus instance license.
// It is a singleton resource: one License CR per Nexus instance.
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec"`
	Status LicenseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LicenseList contains a list of License.
type LicenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []License `json:"items"`
}

// init registers the License types with the scheme.
func init() {
	SchemeBuilder.Register(&License{}, &LicenseList{})
}
