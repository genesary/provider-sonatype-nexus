package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.SSLTruststoreClient = &MockSSLTruststoreClient{}

// MockSSLTruststoreClient is a mock of iamclient.SSLTruststoreClient.
type MockSSLTruststoreClient struct {
	AddCertificateFn    func(ctx context.Context, cert *security.SSLCertificate) error
	RemoveCertificateFn func(ctx context.Context, id string) error
	ListCertificatesFn  func(ctx context.Context) ([]security.SSLCertificate, error)

	AddCertificateCalls    []*security.SSLCertificate
	RemoveCertificateCalls []string
	ListCertificatesCalls  int
}

// NewMockSSLTruststoreClient creates a new MockSSLTruststoreClient.
func NewMockSSLTruststoreClient() *MockSSLTruststoreClient {
	return &MockSSLTruststoreClient{}
}

// AddCertificate mock implementation.
func (m *MockSSLTruststoreClient) AddCertificate(ctx context.Context, cert *security.SSLCertificate) error {
	m.AddCertificateCalls = append(m.AddCertificateCalls, cert)

	if m.AddCertificateFn != nil {
		return m.AddCertificateFn(ctx, cert)
	}

	return errMockNotConfigured
}

// RemoveCertificate mock implementation.
func (m *MockSSLTruststoreClient) RemoveCertificate(ctx context.Context, id string) error {
	m.RemoveCertificateCalls = append(m.RemoveCertificateCalls, id)

	if m.RemoveCertificateFn != nil {
		return m.RemoveCertificateFn(ctx, id)
	}

	return errMockNotConfigured
}

// ListCertificates mock implementation.
func (m *MockSSLTruststoreClient) ListCertificates(ctx context.Context) ([]security.SSLCertificate, error) {
	m.ListCertificatesCalls++

	if m.ListCertificatesFn != nil {
		return m.ListCertificatesFn(ctx)
	}

	return nil, errMockNotConfigured
}
