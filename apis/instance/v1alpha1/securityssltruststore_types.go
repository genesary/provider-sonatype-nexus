package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// SecuritySSLTruststoreParameters defines the desired state of a
// truststore certificate.
// +kubebuilder:validation:XValidation:rule="has(self.pem) != has(self.pemFrom)",message="exactly one of pem and pemFrom is required."
type SecuritySSLTruststoreParameters struct {
	// Pem is the certificate in PEM format to add to the Nexus truststore. Of
	// several certificates, Nexus only adds the first. Mutually exclusive with
	// PemFrom.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Pem *string `json:"pem,omitempty"`

	// PemFrom is where the certificate in PEM format is read from: a secret or
	// a URL. Of several certificates, such as a CA bundle, only the first is
	// added. Anything else there, such as a private key, is not sent to Nexus.
	// Mutually exclusive with Pem.
	// +kubebuilder:validation:Optional
	PemFrom *nexusv1alpha1.DocumentSource `json:"pemFrom,omitempty"`
}

// SecuritySSLTruststoreObservation represents the observed state of a
// truststore certificate.
type SecuritySSLTruststoreObservation struct {
	// ID is the certificate identifier in Nexus.
	// +kubebuilder:validation:Optional
	ID *string `json:"id,omitempty"`

	// Pem is the observed certificate in PEM format.
	// +kubebuilder:validation:Optional
	Pem string `json:"pem,omitempty"`

	// Fingerprint is the certificate fingerprint.
	// +kubebuilder:validation:Optional
	Fingerprint *string `json:"fingerprint,omitempty"`

	// SerialNumber is the certificate serial number.
	// +kubebuilder:validation:Optional
	SerialNumber *string `json:"serialNumber,omitempty"`

	// IssuerCommonName is the issuer common name.
	// +kubebuilder:validation:Optional
	IssuerCommonName *string `json:"issuerCommonName,omitempty"`

	// IssuerOrganization is the issuer organization.
	// +kubebuilder:validation:Optional
	IssuerOrganization *string `json:"issuerOrganization,omitempty"`

	// IssuerOrganizationUnit is the issuer organizational unit.
	// +kubebuilder:validation:Optional
	IssuerOrganizationUnit *string `json:"issuerOrganizationUnit,omitempty"`

	// SubjectCommonName is the subject common name.
	// +kubebuilder:validation:Optional
	SubjectCommonName *string `json:"subjectCommonName,omitempty"`

	// SubjectOrganization is the subject organization.
	// +kubebuilder:validation:Optional
	SubjectOrganization *string `json:"subjectOrganization,omitempty"`

	// SubjectOrganizationUnit is the subject organizational unit.
	// +kubebuilder:validation:Optional
	SubjectOrganizationUnit *string `json:"subjectOrganizationUnit,omitempty"`

	// IssuedOn is the timestamp when the certificate was issued (epoch
	// millis).
	// +kubebuilder:validation:Optional
	IssuedOn *int64 `json:"issuedOn,omitempty"`

	// ExpiresOn is the timestamp when the certificate expires (epoch
	// millis).
	// +kubebuilder:validation:Optional
	ExpiresOn *int64 `json:"expiresOn,omitempty"`
}

// SecuritySSLTruststoreSpec defines the desired state of
// SecuritySSLTruststore.
type SecuritySSLTruststoreSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider SecuritySSLTruststoreParameters `json:"forProvider"`
}

// SecuritySSLTruststoreStatus defines the observed state of
// SecuritySSLTruststore.
type SecuritySSLTruststoreStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider SecuritySSLTruststoreObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="SUBJECT",type="string",JSONPath=".status.atProvider.subjectCommonName"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

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

// SecuritySSLTruststore type metadata.
var (
	SecuritySSLTruststoreKind             = reflect.TypeFor[SecuritySSLTruststore]().Name()
	SecuritySSLTruststoreGroupKind        = schema.GroupKind{Group: APIGroup, Kind: SecuritySSLTruststoreKind}.String()
	SecuritySSLTruststoreKindAPIVersion   = SecuritySSLTruststoreKind + "." + SchemeGroupVersion.String()
	SecuritySSLTruststoreGroupVersionKind = SchemeGroupVersion.WithKind(SecuritySSLTruststoreKind)
)

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&SecuritySSLTruststore{}, &SecuritySSLTruststoreList{})
}
