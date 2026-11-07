package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ iamclient.SSLTruststoreClient = &MockSSLTruststoreClient{}

// MockSSLTruststoreClient is a mock of iamclient.SSLTruststoreClient.
type MockSSLTruststoreClient struct {
	AddCertificateFn    func(pem string) error
	RemoveCertificateFn func(id string) error
	ListCertificatesFn  func() ([]v395.ApiCertificate, error)

	AddCertificateCalls    []string
	RemoveCertificateCalls []string
	ListCertificatesCalls  int
}

// NewMockSSLTruststoreClient creates a new MockSSLTruststoreClient.
func NewMockSSLTruststoreClient() *MockSSLTruststoreClient {
	return &MockSSLTruststoreClient{}
}

// AddCertificate mock implementation.
func (m *MockSSLTruststoreClient) AddCertificate(_ context.Context, pem string) error {
	m.AddCertificateCalls = append(m.AddCertificateCalls, pem)

	if m.AddCertificateFn != nil {
		return m.AddCertificateFn(pem)
	}

	return errMockNotConfigured
}

// RemoveCertificate mock implementation.
func (m *MockSSLTruststoreClient) RemoveCertificate(_ context.Context, id string) error {
	m.RemoveCertificateCalls = append(m.RemoveCertificateCalls, id)

	if m.RemoveCertificateFn != nil {
		return m.RemoveCertificateFn(id)
	}

	return errMockNotConfigured
}

// ListCertificates mock implementation.
func (m *MockSSLTruststoreClient) ListCertificates(_ context.Context) ([]v395.ApiCertificate, error) {
	m.ListCertificatesCalls++

	if m.ListCertificatesFn != nil {
		return m.ListCertificatesFn()
	}

	return nil, errMockNotConfigured
}
