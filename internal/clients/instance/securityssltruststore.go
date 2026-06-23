package instance

import (
	"strings"

	nexuspkgsecurity "github.com/datadrivers/go-nexus-client/nexus3/pkg/security"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SSLTruststoreClient manages Nexus SSL truststore certificates.
type SSLTruststoreClient interface {
	AddCertificate(cert *security.SSLCertificate) error
	RemoveCertificate(id string) error
	ListCertificates() ([]security.SSLCertificate, error)
}

// sslTruststoreClientImpl wraps SecuritySSLService to normalize the
// ListCertificates return type from *[]T to []T.
type sslTruststoreClientImpl struct {
	svc *nexuspkgsecurity.SecuritySSLService
}

// AddCertificate adds the given certificate to the Nexus truststore.
func (c *sslTruststoreClientImpl) AddCertificate(cert *security.SSLCertificate) error {
	return c.svc.AddCertificate(cert)
}

// RemoveCertificate removes the certificate with the given ID from the Nexus
// truststore.
func (c *sslTruststoreClientImpl) RemoveCertificate(id string) error {
	return c.svc.RemoveCertificate(id)
}

// ListCertificates returns all certificates in the Nexus truststore.
func (c *sslTruststoreClientImpl) ListCertificates() ([]security.SSLCertificate, error) {
	result, err := c.svc.ListCertificates()
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return *result, nil
}

// NewSSLTruststoreClient returns a new SSLTruststoreClient.
func NewSSLTruststoreClient(creds nexus.Credentials) (SSLTruststoreClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &sslTruststoreClientImpl{svc: nc.Security.SSL}, nil
}

// IsCertUpToDate reports whether the CR spec PEM matches observed.
func IsCertUpToDate(cr *instancev1alpha1.SecuritySSLTruststore) bool {
	return strings.TrimSpace(cr.Spec.ForProvider.Pem) == strings.TrimSpace(cr.Status.AtProvider.Pem)
}

// CertToObservation converts an SSLCertificate to an observation struct.
func CertToObservation(cert *security.SSLCertificate) instancev1alpha1.SecuritySSLTruststoreObservation {
	obs := instancev1alpha1.SecuritySSLTruststoreObservation{
		Pem: cert.Pem,
	}

	setCertBasicFields(&obs, cert)
	setCertIssuerFields(&obs, cert)
	setCertSubjectFields(&obs, cert)
	setCertDateFields(&obs, cert)

	return obs
}

// setCertBasicFields populates ID, Fingerprint, and SerialNumber from the
// certificate.
func setCertBasicFields(obs *instancev1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
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

// setCertIssuerFields populates issuer fields from the certificate.
func setCertIssuerFields(obs *instancev1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
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

// setCertSubjectFields populates subject fields from the certificate.
func setCertSubjectFields(obs *instancev1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
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

// setCertDateFields populates IssuedOn and ExpiresOn from the certificate.
func setCertDateFields(obs *instancev1alpha1.SecuritySSLTruststoreObservation, cert *security.SSLCertificate) {
	if cert.IssuedOn != 0 {
		obs.IssuedOn = &cert.IssuedOn
	}

	if cert.ExpiresOn != 0 {
		obs.ExpiresOn = &cert.ExpiresOn
	}
}
