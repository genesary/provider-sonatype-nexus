package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.SAMLClient = &MockSAMLClient{}

// MockSAMLClient is a mock of iamclient.SAMLClient.
type MockSAMLClient struct {
	GetSAMLFn    func(ctx context.Context) (*security.SAML, error)
	ApplySAMLFn  func(ctx context.Context, saml security.SAML) error
	DeleteSAMLFn func(ctx context.Context) error

	GetSAMLCalls    int
	ApplySAMLCalls  []security.SAML
	DeleteSAMLCalls int
}

// NewMockSAMLClient creates a new MockSAMLClient.
func NewMockSAMLClient() *MockSAMLClient {
	return &MockSAMLClient{}
}

// GetSAML mock implementation.
func (m *MockSAMLClient) GetSAML(ctx context.Context) (*security.SAML, error) {
	m.GetSAMLCalls++

	if m.GetSAMLFn != nil {
		return m.GetSAMLFn(ctx)
	}

	return nil, errMockNotConfigured
}

// ApplySAML mock implementation.
func (m *MockSAMLClient) ApplySAML(ctx context.Context, saml security.SAML) error {
	m.ApplySAMLCalls = append(m.ApplySAMLCalls, saml)

	if m.ApplySAMLFn != nil {
		return m.ApplySAMLFn(ctx, saml)
	}

	return errMockNotConfigured
}

// DeleteSAML mock implementation.
func (m *MockSAMLClient) DeleteSAML(ctx context.Context) error {
	m.DeleteSAMLCalls++

	if m.DeleteSAMLFn != nil {
		return m.DeleteSAMLFn(ctx)
	}

	return errMockNotConfigured
}
