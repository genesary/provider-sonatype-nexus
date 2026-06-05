package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.AnonymousAccessClient = &MockAnonymousAccessClient{}

// MockAnonymousAccessClient is a mock of iamclient.AnonymousAccessClient.
type MockAnonymousAccessClient struct {
	GetAnonymousAccessFn    func(ctx context.Context) (*security.AnonymousAccessSettings, error)
	UpdateAnonymousAccessFn func(ctx context.Context, settings security.AnonymousAccessSettings) error

	GetAnonymousAccessCalls    int
	UpdateAnonymousAccessCalls []security.AnonymousAccessSettings
}

// NewMockAnonymousAccessClient creates a new MockAnonymousAccessClient.
func NewMockAnonymousAccessClient() *MockAnonymousAccessClient {
	return &MockAnonymousAccessClient{}
}

// GetAnonymousAccess mock implementation.
func (m *MockAnonymousAccessClient) GetAnonymousAccess(ctx context.Context) (*security.AnonymousAccessSettings, error) {
	m.GetAnonymousAccessCalls++

	if m.GetAnonymousAccessFn != nil {
		return m.GetAnonymousAccessFn(ctx)
	}

	return nil, errMockNotConfigured
}

// UpdateAnonymousAccess mock implementation.
func (m *MockAnonymousAccessClient) UpdateAnonymousAccess(ctx context.Context, settings security.AnonymousAccessSettings) error {
	m.UpdateAnonymousAccessCalls = append(m.UpdateAnonymousAccessCalls, settings)

	if m.UpdateAnonymousAccessFn != nil {
		return m.UpdateAnonymousAccessFn(ctx, settings)
	}

	return errMockNotConfigured
}
