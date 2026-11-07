package instance

import (
	"context"
	"strings"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SSLTruststoreClient manages Nexus SSL truststore certificates.
type SSLTruststoreClient interface {
	AddCertificate(ctx context.Context, pem string) error
	RemoveCertificate(ctx context.Context, id string) error
	ListCertificates(ctx context.Context) ([]v395.ApiCertificate, error)
}

// sslTruststoreClient implements SSLTruststoreClient over the Nexus API.
type sslTruststoreClient struct {
	api *v395.SecurityCertificatesAPIService
}

// AddCertificate adds the PEM encoded certificate to the Nexus truststore.
func (c *sslTruststoreClient) AddCertificate(ctx context.Context, pem string) error {
	_, resp, err := c.api.CreateSecuritySslTruststore(ctx).Body(pem).Execute()

	return nexus.APIError(resp, err)
}

// RemoveCertificate removes the certificate with the given ID from the Nexus
// truststore.
func (c *sslTruststoreClient) RemoveCertificate(ctx context.Context, id string) error {
	return nexus.APIError(c.api.DeleteSecuritySslTruststore(ctx, id).Execute())
}

// ListCertificates returns all certificates in the Nexus truststore.
func (c *sslTruststoreClient) ListCertificates(ctx context.Context) ([]v395.ApiCertificate, error) {
	certs, resp, err := c.api.ListSecuritySslTruststore(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return certs, nil
}

// NewSSLTruststoreClient returns a new SSLTruststoreClient.
func NewSSLTruststoreClient(creds nexus.Credentials) (SSLTruststoreClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &sslTruststoreClient{api: nc.SecurityCertificatesAPI}, nil
}

// IsCertUpToDate reports whether the CR spec PEM matches observed.
func IsCertUpToDate(cr *instancev1alpha1.SecuritySSLTruststore) bool {
	return strings.TrimSpace(cr.Spec.ForProvider.Pem) == strings.TrimSpace(cr.Status.AtProvider.Pem)
}

// CertToObservation converts a truststore certificate to an observation
// struct. Fields Nexus reports empty are left unset.
func CertToObservation(cert *v395.ApiCertificate) instancev1alpha1.SecuritySSLTruststoreObservation {
	return instancev1alpha1.SecuritySSLTruststoreObservation{
		Pem:                     cert.GetPem(),
		ID:                      nonEmpty(cert.Id),
		Fingerprint:             nonEmpty(cert.Fingerprint),
		SerialNumber:            nonEmpty(cert.SerialNumber),
		IssuerCommonName:        nonEmpty(cert.IssuerCommonName),
		IssuerOrganization:      nonEmpty(cert.IssuerOrganization),
		IssuerOrganizationUnit:  nonEmpty(cert.IssuerOrganizationalUnit),
		SubjectCommonName:       nonEmpty(cert.SubjectCommonName),
		SubjectOrganization:     nonEmpty(cert.SubjectOrganization),
		SubjectOrganizationUnit: nonEmpty(cert.SubjectOrganizationalUnit),
		IssuedOn:                nonZero(cert.IssuedOn),
		ExpiresOn:               nonZero(cert.ExpiresOn),
	}
}

// nonEmpty returns value, or nil when it points at an empty string.
func nonEmpty(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	return value
}

// nonZero returns value, or nil when it points at zero.
func nonZero(value *int64) *int64 {
	if value == nil || *value == 0 {
		return nil
	}

	return value
}
