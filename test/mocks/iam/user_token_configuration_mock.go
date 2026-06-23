package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserTokenConfigurationClient = &MockUserTokenConfigurationClient{}

// MockUserTokenConfigurationClient is a mock UserTokenConfigurationClient.
type MockUserTokenConfigurationClient struct {
	GetFn       func() (*security.UserTokenConfiguration, error)
	ConfigureFn func(config security.UserTokenConfiguration) error

	GetCalls       int
	ConfigureCalls []security.UserTokenConfiguration
}

// NewMockUserTokenConfigurationClient creates a new mock client.
func NewMockUserTokenConfigurationClient() *MockUserTokenConfigurationClient {
	return &MockUserTokenConfigurationClient{}
}

// Get mock implementation.
func (m *MockUserTokenConfigurationClient) Get() (*security.UserTokenConfiguration, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Configure mock implementation.
func (m *MockUserTokenConfigurationClient) Configure(config security.UserTokenConfiguration) error {
	m.ConfigureCalls = append(m.ConfigureCalls, config)

	if m.ConfigureFn != nil {
		return m.ConfigureFn(config)
	}

	return errMockNotConfigured
}
