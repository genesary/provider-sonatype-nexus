package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecuritySSLTruststoreParameters defines the desired state of a truststore certificate.
type SecuritySSLTruststoreParameters struct {
	// Pem is the certificate in PEM format to add to the Nexus truststore.
	// +kubebuilder:validation:Required
	Pem string `json:"pem"`
}

// SecuritySSLTruststoreObservation represents the observed state of a truststore certificate.
type SecuritySSLTruststoreObservation struct {
	// ID is the certificate identifier in Nexus.
	// +optional
	ID *string `json:"id,omitempty"`

	// Fingerprint is the certificate fingerprint.
	// +optional
	Fingerprint *string `json:"fingerprint,omitempty"`

	// SerialNumber is the certificate serial number.
	// +optional
	SerialNumber *string `json:"serialNumber,omitempty"`

	// IssuerCommonName is the issuer common name.
	// +optional
	IssuerCommonName *string `json:"issuerCommonName,omitempty"`

	// IssuerOrganization is the issuer organization.
	// +optional
	IssuerOrganization *string `json:"issuerOrganization,omitempty"`

	// IssuerOrganizationUnit is the issuer organizational unit.
	// +optional
	IssuerOrganizationUnit *string `json:"issuerOrganizationUnit,omitempty"`

	// SubjectCommonName is the subject common name.
	// +optional
	SubjectCommonName *string `json:"subjectCommonName,omitempty"`

	// SubjectOrganization is the subject organization.
	// +optional
	SubjectOrganization *string `json:"subjectOrganization,omitempty"`

	// SubjectOrganizationUnit is the subject organizational unit.
	// +optional
	SubjectOrganizationUnit *string `json:"subjectOrganizationUnit,omitempty"`

	// IssuedOn is the timestamp when the certificate was issued (epoch millis).
	// +optional
	IssuedOn *int64 `json:"issuedOn,omitempty"`

	// ExpiresOn is the timestamp when the certificate expires (epoch millis).
	// +optional
	ExpiresOn *int64 `json:"expiresOn,omitempty"`
}

// SecuritySSLTruststoreSpec defines the desired state of SecuritySSLTruststore.
type SecuritySSLTruststoreSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	ForProvider SecuritySSLTruststoreParameters `json:"forProvider"`
}

// SecuritySSLTruststoreStatus defines the observed state of SecuritySSLTruststore.
type SecuritySSLTruststoreStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider SecuritySSLTruststoreObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="SUBJECT",type="string",JSONPath=".status.atProvider.subjectCommonName"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// SecuritySSLTruststore is the Schema for the security SSL truststore API.
// Each instance represents a certificate in the Nexus truststore.
type SecuritySSLTruststore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecuritySSLTruststoreSpec   `json:"spec"`
	Status SecuritySSLTruststoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecuritySSLTruststoreList contains a list of SecuritySSLTruststore.
type SecuritySSLTruststoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecuritySSLTruststore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecuritySSLTruststore{}, &SecuritySSLTruststoreList{})
}
