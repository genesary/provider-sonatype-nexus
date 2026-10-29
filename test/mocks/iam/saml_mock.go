package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.SAMLClient = &MockSAMLClient{}

// MockSAMLClient is a mock of iamclient.SAMLClient.
type MockSAMLClient struct {
	ReadFn   func() (*v395.SamlConfigurationXO, error)
	ApplyFn  func(saml v395.SamlConfigurationXO) error
	DeleteFn func() error

	ReadCalls   int
	ApplyCalls  []v395.SamlConfigurationXO
	DeleteCalls int
}

// NewMockSAMLClient creates a new MockSAMLClient.
func NewMockSAMLClient() *MockSAMLClient {
	return &MockSAMLClient{}
}

// Read mock implementation.
func (m *MockSAMLClient) Read(_ context.Context) (*v395.SamlConfigurationXO, error) {
	m.ReadCalls++

	if m.ReadFn != nil {
		return m.ReadFn()
	}

	return nil, errMockNotConfigured
}

// Apply mock implementation.
func (m *MockSAMLClient) Apply(_ context.Context, saml v395.SamlConfigurationXO) error {
	m.ApplyCalls = append(m.ApplyCalls, saml)

	if m.ApplyFn != nil {
		return m.ApplyFn(saml)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockSAMLClient) Delete(_ context.Context) error {
	m.DeleteCalls++

	if m.DeleteFn != nil {
		return m.DeleteFn()
	}

	return errMockNotConfigured
}
