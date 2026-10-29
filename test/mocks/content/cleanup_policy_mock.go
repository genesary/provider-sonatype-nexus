// Package content provides mock implementations for the content client group.
package content

import (
	"context"
	"errors"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

// errMockNotConfigured is returned when a mock function has not been set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ contentclient.CleanupPolicyClient = &MockCleanupPolicyClient{}

// MockCleanupPolicyClient is a mock of contentclient.CleanupPolicyClient.
type MockCleanupPolicyClient struct {
	GetFn    func(name string) (*contentclient.CleanupPolicy, error)
	CreateFn func(policy *contentclient.CleanupPolicy) error
	UpdateFn func(policy *contentclient.CleanupPolicy) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []*contentclient.CleanupPolicy
	UpdateCalls []*contentclient.CleanupPolicy
	DeleteCalls []string
}

// NewMockCleanupPolicyClient creates a new MockCleanupPolicyClient.
func NewMockCleanupPolicyClient() *MockCleanupPolicyClient {
	return &MockCleanupPolicyClient{}
}

// Get mock implementation.
func (m *MockCleanupPolicyClient) Get(_ context.Context, name string) (*contentclient.CleanupPolicy, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockCleanupPolicyClient) Create(_ context.Context, policy *contentclient.CleanupPolicy) error {
	m.CreateCalls = append(m.CreateCalls, policy)

	if m.CreateFn != nil {
		return m.CreateFn(policy)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockCleanupPolicyClient) Update(_ context.Context, policy *contentclient.CleanupPolicy) error {
	m.UpdateCalls = append(m.UpdateCalls, policy)

	if m.UpdateFn != nil {
		return m.UpdateFn(policy)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockCleanupPolicyClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
