//nolint:dupl // mock structs share structural shape by design
package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.EmailClient = &MockEmailClient{}

// MockEmailClient is a test double for instanceclient.EmailClient.
type MockEmailClient struct {
	GetFn    func() (*v395.ApiEmailConfiguration, error)
	UpdateFn func(config v395.ApiEmailConfiguration) error
	ResetFn  func() error

	GetCalls    int
	UpdateCalls []v395.ApiEmailConfiguration
	ResetCalls  int
}

// NewMockEmailClient returns a MockEmailClient with no functions set.
func NewMockEmailClient() *MockEmailClient {
	return &MockEmailClient{}
}

// Get mock implementation.
func (m *MockEmailClient) Get(_ context.Context) (*v395.ApiEmailConfiguration, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockEmailClient) Update(_ context.Context, config v395.ApiEmailConfiguration) error {
	m.UpdateCalls = append(m.UpdateCalls, config)

	if m.UpdateFn != nil {
		return m.UpdateFn(config)
	}

	return errMockNotConfigured
}

// Reset mock implementation.
func (m *MockEmailClient) Reset(_ context.Context) error {
	m.ResetCalls++

	if m.ResetFn != nil {
		return m.ResetFn()
	}

	return errMockNotConfigured
}
