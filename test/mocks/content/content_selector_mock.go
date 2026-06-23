package content

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.ContentSelectorClient = &MockContentSelectorClient{}

// MockContentSelectorClient is a mock of contentclient.ContentSelectorClient.
type MockContentSelectorClient struct {
	GetFn    func(name string) (*security.ContentSelector, error)
	CreateFn func(cs security.ContentSelector) error
	UpdateFn func(name string, cs security.ContentSelector) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []security.ContentSelector
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockContentSelectorClient creates a new MockContentSelectorClient.
func NewMockContentSelectorClient() *MockContentSelectorClient {
	return &MockContentSelectorClient{}
}

// Get mock implementation.
func (m *MockContentSelectorClient) Get(name string) (*security.ContentSelector, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockContentSelectorClient) Create(cs security.ContentSelector) error {
	m.CreateCalls = append(m.CreateCalls, cs)

	if m.CreateFn != nil {
		return m.CreateFn(cs)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockContentSelectorClient) Update(name string, cs security.ContentSelector) error {
	m.UpdateCalls = append(m.UpdateCalls, name)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, cs)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockContentSelectorClient) Delete(name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
