package iam

import (
	"context"
	"errors"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

// errMockNotConfigured is returned when no mock function has been configured.
var errMockNotConfigured = errors.New("mock function not configured")

var _ iamclient.AnonymousAccessClient = &MockAnonymousAccessClient{}

// MockAnonymousAccessClient is a mock of iamclient.AnonymousAccessClient.
type MockAnonymousAccessClient struct {
	ReadFn   func() (*v395.AnonymousAccessSettingsXO, error)
	UpdateFn func(settings v395.AnonymousAccessSettingsXO) error

	ReadCalls   int
	UpdateCalls []v395.AnonymousAccessSettingsXO
}

// NewMockAnonymousAccessClient creates a new MockAnonymousAccessClient.
func NewMockAnonymousAccessClient() *MockAnonymousAccessClient {
	return &MockAnonymousAccessClient{}
}

// Read mock implementation.
func (m *MockAnonymousAccessClient) Read(_ context.Context) (*v395.AnonymousAccessSettingsXO, error) {
	m.ReadCalls++

	if m.ReadFn != nil {
		return m.ReadFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockAnonymousAccessClient) Update(_ context.Context, settings v395.AnonymousAccessSettingsXO) error {
	m.UpdateCalls = append(m.UpdateCalls, settings)

	if m.UpdateFn != nil {
		return m.UpdateFn(settings)
	}

	return errMockNotConfigured
}
