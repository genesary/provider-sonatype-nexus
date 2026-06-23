package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ iamclient.SSLTruststoreClient = &MockSSLTruststoreClient{}

// MockSSLTruststoreClient is a mock of iamclient.SSLTruststoreClient.
type MockSSLTruststoreClient struct {
	AddCertificateFn    func(cert *security.SSLCertificate) error
	RemoveCertificateFn func(id string) error
	ListCertificatesFn  func() ([]security.SSLCertificate, error)

	AddCertificateCalls    []*security.SSLCertificate
	RemoveCertificateCalls []string
	ListCertificatesCalls  int
}

// NewMockSSLTruststoreClient creates a new MockSSLTruststoreClient.
func NewMockSSLTruststoreClient() *MockSSLTruststoreClient {
	return &MockSSLTruststoreClient{}
}

// AddCertificate mock implementation.
func (m *MockSSLTruststoreClient) AddCertificate(cert *security.SSLCertificate) error {
	m.AddCertificateCalls = append(m.AddCertificateCalls, cert)

	if m.AddCertificateFn != nil {
		return m.AddCertificateFn(cert)
	}

	return errMockNotConfigured
}

// RemoveCertificate mock implementation.
func (m *MockSSLTruststoreClient) RemoveCertificate(id string) error {
	m.RemoveCertificateCalls = append(m.RemoveCertificateCalls, id)

	if m.RemoveCertificateFn != nil {
		return m.RemoveCertificateFn(id)
	}

	return errMockNotConfigured
}

// ListCertificates mock implementation.
func (m *MockSSLTruststoreClient) ListCertificates() ([]security.SSLCertificate, error) {
	m.ListCertificatesCalls++

	if m.ListCertificatesFn != nil {
		return m.ListCertificatesFn()
	}

	return nil, errMockNotConfigured
}
