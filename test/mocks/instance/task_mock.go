// Package instance provides mock implementations for the instance client group.
package instance

import (
	"context"
	"errors"

	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

// errMockNotConfigured is returned when a mock function has not been set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ instanceclient.TaskClient = &MockTaskClient{}

// MockTaskClient is a mock instanceclient.TaskClient for unit tests.
type MockTaskClient struct {
	GetTaskByNameFn func(ctx context.Context, name string) (*nxtask.Task, error)
	CreateTaskFn    func(ctx context.Context, t *nxtask.TaskCreateStruct) (*nxtask.Task, error)
	UpdateTaskFn    func(ctx context.Context, id string, t *nxtask.TaskCreateStruct) error
	DeleteTaskFn    func(ctx context.Context, id string) error

	GetTaskByNameCalls []string
	CreateTaskCalls    []*nxtask.TaskCreateStruct
	UpdateTaskCalls    []string
	DeleteTaskCalls    []string
}

// NewMockTaskClient creates a new MockTaskClient.
func NewMockTaskClient() *MockTaskClient {
	return &MockTaskClient{}
}

// GetTaskByName mock implementation.
func (m *MockTaskClient) GetTaskByName(ctx context.Context, name string) (*nxtask.Task, error) {
	m.GetTaskByNameCalls = append(m.GetTaskByNameCalls, name)

	if m.GetTaskByNameFn != nil {
		return m.GetTaskByNameFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateTask mock implementation.
func (m *MockTaskClient) CreateTask(ctx context.Context, t *nxtask.TaskCreateStruct) (*nxtask.Task, error) {
	m.CreateTaskCalls = append(m.CreateTaskCalls, t)

	if m.CreateTaskFn != nil {
		return m.CreateTaskFn(ctx, t)
	}

	return nil, errMockNotConfigured
}

// UpdateTask mock implementation.
func (m *MockTaskClient) UpdateTask(ctx context.Context, id string, t *nxtask.TaskCreateStruct) error {
	m.UpdateTaskCalls = append(m.UpdateTaskCalls, id)

	if m.UpdateTaskFn != nil {
		return m.UpdateTaskFn(ctx, id, t)
	}

	return errMockNotConfigured
}

// DeleteTask mock implementation.
func (m *MockTaskClient) DeleteTask(ctx context.Context, id string) error {
	m.DeleteTaskCalls = append(m.DeleteTaskCalls, id)

	if m.DeleteTaskFn != nil {
		return m.DeleteTaskFn(ctx, id)
	}

	return errMockNotConfigured
}
