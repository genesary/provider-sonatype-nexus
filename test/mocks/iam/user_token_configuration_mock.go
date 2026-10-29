package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserTokenConfigurationClient = &MockUserTokenConfigurationClient{}

// MockUserTokenConfigurationClient is a mock UserTokenConfigurationClient.
type MockUserTokenConfigurationClient struct {
	GetFn       func() (*v395.UserTokensApiModel, error)
	ConfigureFn func(config v395.UserTokensApiModel) error

	GetCalls       int
	ConfigureCalls []v395.UserTokensApiModel
}

// NewMockUserTokenConfigurationClient creates a new mock client.
func NewMockUserTokenConfigurationClient() *MockUserTokenConfigurationClient {
	return &MockUserTokenConfigurationClient{}
}

// Get mock implementation.
func (m *MockUserTokenConfigurationClient) Get(_ context.Context) (*v395.UserTokensApiModel, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Configure mock implementation.
func (m *MockUserTokenConfigurationClient) Configure(_ context.Context, config v395.UserTokensApiModel) error {
	m.ConfigureCalls = append(m.ConfigureCalls, config)

	if m.ConfigureFn != nil {
		return m.ConfigureFn(config)
	}

	return errMockNotConfigured
}
