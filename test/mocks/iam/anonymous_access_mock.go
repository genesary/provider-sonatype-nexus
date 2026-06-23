package iam

import (
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

// errMockNotConfigured is returned when no mock function has been configured.
var errMockNotConfigured = errors.New("mock function not configured")

var _ iamclient.AnonymousAccessClient = &MockAnonymousAccessClient{}

// MockAnonymousAccessClient is a mock of iamclient.AnonymousAccessClient.
type MockAnonymousAccessClient struct {
	ReadFn   func() (*security.AnonymousAccessSettings, error)
	UpdateFn func(settings security.AnonymousAccessSettings) error

	ReadCalls   int
	UpdateCalls []security.AnonymousAccessSettings
}

// NewMockAnonymousAccessClient creates a new MockAnonymousAccessClient.
func NewMockAnonymousAccessClient() *MockAnonymousAccessClient {
	return &MockAnonymousAccessClient{}
}

// Read mock implementation.
func (m *MockAnonymousAccessClient) Read() (*security.AnonymousAccessSettings, error) {
	m.ReadCalls++

	if m.ReadFn != nil {
		return m.ReadFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockAnonymousAccessClient) Update(settings security.AnonymousAccessSettings) error {
	m.UpdateCalls = append(m.UpdateCalls, settings)

	if m.UpdateFn != nil {
		return m.UpdateFn(settings)
	}

	return errMockNotConfigured
}
