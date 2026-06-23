// Package content provides mock implementations for the content client group.
package content

import (
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

// errMockNotConfigured is returned when a mock function has not been set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ contentclient.CleanupPolicyClient = &MockCleanupPolicyClient{}

// MockCleanupPolicyClient is a mock of contentclient.CleanupPolicyClient.
type MockCleanupPolicyClient struct {
	GetFn    func(name string) (*cleanuppolicies.CleanupPolicy, error)
	CreateFn func(policy *cleanuppolicies.CleanupPolicy) error
	UpdateFn func(policy *cleanuppolicies.CleanupPolicy) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []*cleanuppolicies.CleanupPolicy
	UpdateCalls []*cleanuppolicies.CleanupPolicy
	DeleteCalls []string
}

// NewMockCleanupPolicyClient creates a new MockCleanupPolicyClient.
func NewMockCleanupPolicyClient() *MockCleanupPolicyClient {
	return &MockCleanupPolicyClient{}
}

// Get mock implementation.
func (m *MockCleanupPolicyClient) Get(name string) (*cleanuppolicies.CleanupPolicy, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockCleanupPolicyClient) Create(policy *cleanuppolicies.CleanupPolicy) error {
	m.CreateCalls = append(m.CreateCalls, policy)

	if m.CreateFn != nil {
		return m.CreateFn(policy)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockCleanupPolicyClient) Update(policy *cleanuppolicies.CleanupPolicy) error {
	m.UpdateCalls = append(m.UpdateCalls, policy)

	if m.UpdateFn != nil {
		return m.UpdateFn(policy)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockCleanupPolicyClient) Delete(name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
