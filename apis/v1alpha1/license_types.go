package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LicenseParameters defines the desired state of License.
type LicenseParameters struct {
	// LicenseSecretRef is a reference to a Secret containing the Nexus license file.
	// The referenced key should contain the raw license binary content.
	// +kubebuilder:validation:Required
	LicenseSecretRef xpv1.SecretKeySelector `json:"licenseSecretRef"`
}

// LicenseObservation represents the observed state of License in Nexus.
type LicenseObservation struct {
	// ContactCompany is the company associated with the license.
	// +optional
	ContactCompany *string `json:"contactCompany,omitempty"`

	// ContactEmail is the email associated with the license.
	// +optional
	ContactEmail *string `json:"contactEmail,omitempty"`

	// ContactName is the contact name associated with the license.
	// +optional
	ContactName *string `json:"contactName,omitempty"`

	// EffectiveDate is when the license became effective.
	// +optional
	EffectiveDate *string `json:"effectiveDate,omitempty"`

	// ExpirationDate is when the license expires.
	// +optional
	ExpirationDate *string `json:"expirationDate,omitempty"`

	// Features lists the licensed features.
	// +optional
	Features *string `json:"features,omitempty"`

	// Fingerprint is the license fingerprint.
	// +optional
	Fingerprint *string `json:"fingerprint,omitempty"`

	// LicenseType is the type of the license (e.g., "PRO", "STARTER").
	// +optional
	LicenseType *string `json:"licenseType,omitempty"`

	// LicensedUsers is the number of licensed users.
	// +optional
	LicensedUsers *string `json:"licensedUsers,omitempty"`
}

// LicenseSpec defines the desired state of License.
type LicenseSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       LicenseParameters `json:"forProvider"`
}

// LicenseStatus defines the observed state of License.
type LicenseStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LicenseObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="LICENSE-TYPE",type="string",JSONPath=".status.atProvider.licenseType"
// +kubebuilder:printcolumn:name="EXPIRATION",type="string",JSONPath=".status.atProvider.expirationDate"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// License is the Schema for the licenses API.
// This is a singleton resource that manages the Nexus Repository Manager license.
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
	Items           []License `json:"items"`
}

func init() {
	SchemeBuilder.Register(&License{}, &LicenseList{})
}
