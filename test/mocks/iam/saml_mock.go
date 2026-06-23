package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.SAMLClient = &MockSAMLClient{}

// MockSAMLClient is a mock of iamclient.SAMLClient.
type MockSAMLClient struct {
	ReadFn   func() (*security.SAML, error)
	ApplyFn  func(saml security.SAML) error
	DeleteFn func() error

	ReadCalls   int
	ApplyCalls  []security.SAML
	DeleteCalls int
}

// NewMockSAMLClient creates a new MockSAMLClient.
func NewMockSAMLClient() *MockSAMLClient {
	return &MockSAMLClient{}
}

// Read mock implementation.
func (m *MockSAMLClient) Read() (*security.SAML, error) {
	m.ReadCalls++

	if m.ReadFn != nil {
		return m.ReadFn()
	}

	return nil, errMockNotConfigured
}

// Apply mock implementation.
func (m *MockSAMLClient) Apply(saml security.SAML) error {
	m.ApplyCalls = append(m.ApplyCalls, saml)

	if m.ApplyFn != nil {
		return m.ApplyFn(saml)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockSAMLClient) Delete() error {
	m.DeleteCalls++

	if m.DeleteFn != nil {
		return m.DeleteFn()
	}

	return errMockNotConfigured
}
