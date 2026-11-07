package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.IQServerClient = &MockIQServerClient{}

// MockIQServerClient is a test double for instanceclient.IQServerClient.
type MockIQServerClient struct {
	GetFn     func() (*v395.IqConnectionXo, error)
	UpdateFn  func(config v395.IqConnectionXo) error
	DisableFn func() error

	GetCalls     int
	UpdateCalls  []v395.IqConnectionXo
	DisableCalls int
}

// NewMockIQServerClient returns a MockIQServerClient with no functions set.
func NewMockIQServerClient() *MockIQServerClient {
	return &MockIQServerClient{}
}

// Get mock implementation.
func (m *MockIQServerClient) Get(_ context.Context) (*v395.IqConnectionXo, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockIQServerClient) Update(_ context.Context, config v395.IqConnectionXo) error {
	m.UpdateCalls = append(m.UpdateCalls, config)

	if m.UpdateFn != nil {
		return m.UpdateFn(config)
	}

	return errMockNotConfigured
}

// Disable mock implementation.
func (m *MockIQServerClient) Disable(_ context.Context) error {
	m.DisableCalls++

	if m.DisableFn != nil {
		return m.DisableFn()
	}

	return errMockNotConfigured
}
