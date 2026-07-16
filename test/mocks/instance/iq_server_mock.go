package capability

import (
	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.IQServerClient = &MockIQServerClient{}

// MockIQServerClient is a test double for instanceclient.IQServerClient.
type MockIQServerClient struct {
	GetFn     func() (*nexussdk.IQServerConfiguration, error)
	UpdateFn  func(config nexussdk.IQServerConfiguration) error
	DisableFn func() error

	GetCalls     int
	UpdateCalls  []nexussdk.IQServerConfiguration
	DisableCalls int
}

// NewMockIQServerClient returns a MockIQServerClient with no functions set.
func NewMockIQServerClient() *MockIQServerClient {
	return &MockIQServerClient{}
}

// Get mock implementation.
func (m *MockIQServerClient) Get() (*nexussdk.IQServerConfiguration, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockIQServerClient) Update(config nexussdk.IQServerConfiguration) error {
	m.UpdateCalls = append(m.UpdateCalls, config)

	if m.UpdateFn != nil {
		return m.UpdateFn(config)
	}

	return errMockNotConfigured
}

// Disable mock implementation.
func (m *MockIQServerClient) Disable() error {
	m.DisableCalls++

	if m.DisableFn != nil {
		return m.DisableFn()
	}

	return errMockNotConfigured
}
