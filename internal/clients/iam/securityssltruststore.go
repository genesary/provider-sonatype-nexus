package iam

import (
	"context"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SSLTruststoreClient manages Nexus SSL truststore certificates.
type SSLTruststoreClient interface {
	AddCertificate(ctx context.Context, cert *security.SSLCertificate) error
	RemoveCertificate(ctx context.Context, id string) error
	ListCertificates(ctx context.Context) ([]security.SSLCertificate, error)
}

// NewSSLTruststoreClient returns a new SSLTruststoreClient.
func NewSSLTruststoreClient(creds nexus.Credentials) (SSLTruststoreClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.SSL(), nil
}

// IsCertUpToDate reports whether the CR spec PEM matches observed.
func IsCertUpToDate(cr *iamv1alpha1.SecuritySSLTruststore) bool {
	return strings.TrimSpace(cr.Spec.ForProvider.Pem) == strings.TrimSpace(cr.Status.AtProvider.Pem)
}

// CertToObservation converts an SSLCertificate to an observation struct.
func CertToObservation(cert *security.SSLCertificate) iamv1alpha1.SecuritySSLTruststoreObservation {
	obs := iamv1alpha1.SecuritySSLTruststoreObservation{
		Pem: cert.Pem,
	}

	setCertBasicFields(&obs, cert)
	setCertIssuerFields(&obs, cert)
	setCertSubjectFields(&obs, cert)
	setCertDateFields(&obs, cert)

	return obs
}

// setCertBasicFields sets the ID, fingerprint, and serial number fields.
func setCertBasicFields(obs *iamv1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
	if cert.Id != "" {
		obs.ID = &cert.Id
	}

	if cert.Fingerprint != "" {
		obs.Fingerprint = &cert.Fingerprint
	}

	if cert.SerialNumber != "" {
		obs.SerialNumber = &cert.SerialNumber
	}
}

// setCertIssuerFields sets the issuer-related observation fields.
func setCertIssuerFields(obs *iamv1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
	if cert.IssuerCommonName != "" {
		obs.IssuerCommonName = &cert.IssuerCommonName
	}

	if cert.IssuerOrganization != "" {
		obs.IssuerOrganization = &cert.IssuerOrganization
	}

	if cert.IssuerOrganizationUnit != "" {
		obs.IssuerOrganizationUnit = &cert.IssuerOrganizationUnit
	}
}

// setCertSubjectFields sets the subject-related observation fields.
func setCertSubjectFields(obs *iamv1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
	if cert.SubjectCommonName != "" {
		obs.SubjectCommonName = &cert.SubjectCommonName
	}

	if cert.SubjectOrganization != "" {
		obs.SubjectOrganization = &cert.SubjectOrganization
	}

	if cert.SubjectOrganizationUnit != "" {
		obs.SubjectOrganizationUnit = &cert.SubjectOrganizationUnit
	}
}

// setCertDateFields sets the issued-on and expires-on observation fields.
func setCertDateFields(obs *iamv1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
	if cert.IssuedOn != 0 {
		obs.IssuedOn = &cert.IssuedOn
	}

	if cert.ExpiresOn != 0 {
		obs.ExpiresOn = &cert.ExpiresOn
	}
}
