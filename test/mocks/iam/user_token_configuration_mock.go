//nolint:dupl // mock files have structurally similar but distinct types
package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserTokenConfigurationClient = &MockUserTokenConfigurationClient{}

// MockUserTokenConfigurationClient is a mock UserTokenConfigurationClient.
type MockUserTokenConfigurationClient struct {
	GetUserTokenConfigurationFn    func(ctx context.Context) (*security.UserTokenConfiguration, error)
	UpdateUserTokenConfigurationFn func(ctx context.Context, config security.UserTokenConfiguration) error

	GetUserTokenConfigurationCalls    int
	UpdateUserTokenConfigurationCalls []security.UserTokenConfiguration
}

// NewMockUserTokenConfigurationClient creates a new mock client.
func NewMockUserTokenConfigurationClient() *MockUserTokenConfigurationClient {
	return &MockUserTokenConfigurationClient{}
}

// GetUserTokenConfiguration mock implementation.
func (m *MockUserTokenConfigurationClient) GetUserTokenConfiguration(ctx context.Context) (*security.UserTokenConfiguration, error) {
	m.GetUserTokenConfigurationCalls++

	if m.GetUserTokenConfigurationFn != nil {
		return m.GetUserTokenConfigurationFn(ctx)
	}

	return nil, errMockNotConfigured
}

// UpdateUserTokenConfiguration mock implementation.
func (m *MockUserTokenConfigurationClient) UpdateUserTokenConfiguration(ctx context.Context, config security.UserTokenConfiguration) error {
	m.UpdateUserTokenConfigurationCalls = append(m.UpdateUserTokenConfigurationCalls, config)

	if m.UpdateUserTokenConfigurationFn != nil {
		return m.UpdateUserTokenConfigurationFn(ctx, config)
	}

	return errMockNotConfigured
}
