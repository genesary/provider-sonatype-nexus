package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.TaskClient = &MockTaskClient{}

// MockTaskClient is a test double for instanceclient.TaskClient.
type MockTaskClient struct {
	CreateFn    func(task v395.TaskTemplateXO) (*v395.TaskXO, error)
	DeleteFn    func(id string) error
	GetByNameFn func(name string) (*v395.TaskXO, error)
	UpdateFn    func(id string, task v395.UpdateTasksRequest) error

	CreateCalls    []v395.TaskTemplateXO
	DeleteCalls    []string
	GetByNameCalls []string
	UpdateIDs      []string
	UpdateCalls    []v395.UpdateTasksRequest
}

// NewMockTaskClient returns a MockTaskClient with no functions set.
func NewMockTaskClient() *MockTaskClient {
	return &MockTaskClient{}
}

// Create mock implementation.
func (m *MockTaskClient) Create(_ context.Context, task v395.TaskTemplateXO) (*v395.TaskXO, error) {
	m.CreateCalls = append(m.CreateCalls, task)

	if m.CreateFn != nil {
		return m.CreateFn(task)
	}

	return nil, errMockNotConfigured
}

// Delete mock implementation.
func (m *MockTaskClient) Delete(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)

	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}

// GetByName mock implementation.
func (m *MockTaskClient) GetByName(_ context.Context, name string) (*v395.TaskXO, error) {
	m.GetByNameCalls = append(m.GetByNameCalls, name)

	if m.GetByNameFn != nil {
		return m.GetByNameFn(name)
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockTaskClient) Update(_ context.Context, id string, task v395.UpdateTasksRequest) error {
	m.UpdateIDs = append(m.UpdateIDs, id)
	m.UpdateCalls = append(m.UpdateCalls, task)

	if m.UpdateFn != nil {
		return m.UpdateFn(id, task)
	}

	return errMockNotConfigured
}
