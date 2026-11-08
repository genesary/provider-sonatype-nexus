//nolint:dupl // mock structs share structural shape by design
package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.HTTPSettingsClient = &MockHTTPSettingsClient{}

// MockHTTPSettingsClient is a test double for
// instanceclient.HTTPSettingsClient.
type MockHTTPSettingsClient struct {
	GetFn    func() (*v395.HttpSettingsXo, error)
	UpdateFn func(settings v395.HttpSettingsXo) error
	ResetFn  func() error

	GetCalls    int
	UpdateCalls []v395.HttpSettingsXo
	ResetCalls  int
}

// NewMockHTTPSettingsClient returns a MockHTTPSettingsClient with no
// functions set.
func NewMockHTTPSettingsClient() *MockHTTPSettingsClient {
	return &MockHTTPSettingsClient{}
}

// Get mock implementation.
func (m *MockHTTPSettingsClient) Get(_ context.Context) (*v395.HttpSettingsXo, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockHTTPSettingsClient) Update(_ context.Context, settings v395.HttpSettingsXo) error {
	m.UpdateCalls = append(m.UpdateCalls, settings)

	if m.UpdateFn != nil {
		return m.UpdateFn(settings)
	}

	return errMockNotConfigured
}

// Reset mock implementation.
func (m *MockHTTPSettingsClient) Reset(_ context.Context) error {
	m.ResetCalls++

	if m.ResetFn != nil {
		return m.ResetFn()
	}

	return errMockNotConfigured
}
