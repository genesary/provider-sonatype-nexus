package content

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.ContentSelectorClient = &MockContentSelectorClient{}

// MockContentSelectorClient is a mock of contentclient.ContentSelectorClient.
type MockContentSelectorClient struct {
	GetContentSelectorFn    func(ctx context.Context, name string) (*security.ContentSelector, error)
	CreateContentSelectorFn func(ctx context.Context, cs security.ContentSelector) error
	UpdateContentSelectorFn func(ctx context.Context, name string, cs security.ContentSelector) error
	DeleteContentSelectorFn func(ctx context.Context, name string) error

	GetContentSelectorCalls    []string
	CreateContentSelectorCalls []security.ContentSelector
	UpdateContentSelectorCalls []security.ContentSelector
	DeleteContentSelectorCalls []string
}

// NewMockContentSelectorClient creates a new MockContentSelectorClient.
func NewMockContentSelectorClient() *MockContentSelectorClient {
	return &MockContentSelectorClient{}
}

// GetContentSelector mock implementation.
func (m *MockContentSelectorClient) GetContentSelector(ctx context.Context, name string) (*security.ContentSelector, error) {
	m.GetContentSelectorCalls = append(m.GetContentSelectorCalls, name)

	if m.GetContentSelectorFn != nil {
		return m.GetContentSelectorFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateContentSelector mock implementation.
func (m *MockContentSelectorClient) CreateContentSelector(ctx context.Context, cs security.ContentSelector) error {
	m.CreateContentSelectorCalls = append(m.CreateContentSelectorCalls, cs)

	if m.CreateContentSelectorFn != nil {
		return m.CreateContentSelectorFn(ctx, cs)
	}

	return errMockNotConfigured
}

// UpdateContentSelector mock implementation.
func (m *MockContentSelectorClient) UpdateContentSelector(ctx context.Context, name string, cs security.ContentSelector) error {
	m.UpdateContentSelectorCalls = append(m.UpdateContentSelectorCalls, cs)

	if m.UpdateContentSelectorFn != nil {
		return m.UpdateContentSelectorFn(ctx, name, cs)
	}

	return errMockNotConfigured
}

// DeleteContentSelector mock implementation.
func (m *MockContentSelectorClient) DeleteContentSelector(ctx context.Context, name string) error {
	m.DeleteContentSelectorCalls = append(m.DeleteContentSelectorCalls, name)

	if m.DeleteContentSelectorFn != nil {
		return m.DeleteContentSelectorFn(ctx, name)
	}

	return errMockNotConfigured
}
