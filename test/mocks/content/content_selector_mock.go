package content

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.ContentSelectorClient = &MockContentSelectorClient{}

// MockContentSelectorClient is a mock of contentclient.ContentSelectorClient.
type MockContentSelectorClient struct {
	GetFn    func(name string) (*v395.ContentSelectorApiResponse, error)
	CreateFn func(cs v395.ContentSelectorApiCreateRequest) error
	UpdateFn func(name string, cs v395.ContentSelectorApiUpdateRequest) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []v395.ContentSelectorApiCreateRequest
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockContentSelectorClient creates a new MockContentSelectorClient.
func NewMockContentSelectorClient() *MockContentSelectorClient {
	return &MockContentSelectorClient{}
}

// Get mock implementation.
func (m *MockContentSelectorClient) Get(_ context.Context, name string) (*v395.ContentSelectorApiResponse, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockContentSelectorClient) Create(_ context.Context, cs v395.ContentSelectorApiCreateRequest) error {
	m.CreateCalls = append(m.CreateCalls, cs)

	if m.CreateFn != nil {
		return m.CreateFn(cs)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockContentSelectorClient) Update(_ context.Context, name string, cs v395.ContentSelectorApiUpdateRequest) error {
	m.UpdateCalls = append(m.UpdateCalls, name)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, cs)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockContentSelectorClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
