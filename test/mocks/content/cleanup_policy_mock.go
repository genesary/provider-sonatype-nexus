// Package content provides mock implementations for the content client group.
package content

import (
	"context"
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

// errMockNotConfigured is returned when a mock function has not been set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ contentclient.CleanupPolicyClient = &MockCleanupPolicyClient{}

// MockCleanupPolicyClient is a mock of contentclient.CleanupPolicyClient.
type MockCleanupPolicyClient struct {
	GetCleanupPolicyFn    func(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error)
	CreateCleanupPolicyFn func(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	UpdateCleanupPolicyFn func(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	DeleteCleanupPolicyFn func(ctx context.Context, name string) error

	GetCleanupPolicyCalls    []string
	CreateCleanupPolicyCalls []*cleanuppolicies.CleanupPolicy
	UpdateCleanupPolicyCalls []*cleanuppolicies.CleanupPolicy
	DeleteCleanupPolicyCalls []string
}

// NewMockCleanupPolicyClient creates a new MockCleanupPolicyClient.
func NewMockCleanupPolicyClient() *MockCleanupPolicyClient {
	return &MockCleanupPolicyClient{}
}

// GetCleanupPolicy mock implementation.
func (m *MockCleanupPolicyClient) GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error) {
	m.GetCleanupPolicyCalls = append(m.GetCleanupPolicyCalls, name)

	if m.GetCleanupPolicyFn != nil {
		return m.GetCleanupPolicyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCleanupPolicy mock implementation.
func (m *MockCleanupPolicyClient) CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	m.CreateCleanupPolicyCalls = append(m.CreateCleanupPolicyCalls, policy)

	if m.CreateCleanupPolicyFn != nil {
		return m.CreateCleanupPolicyFn(ctx, policy)
	}

	return errMockNotConfigured
}

// UpdateCleanupPolicy mock implementation.
func (m *MockCleanupPolicyClient) UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	m.UpdateCleanupPolicyCalls = append(m.UpdateCleanupPolicyCalls, policy)

	if m.UpdateCleanupPolicyFn != nil {
		return m.UpdateCleanupPolicyFn(ctx, policy)
	}

	return errMockNotConfigured
}

// DeleteCleanupPolicy mock implementation.
func (m *MockCleanupPolicyClient) DeleteCleanupPolicy(ctx context.Context, name string) error {
	m.DeleteCleanupPolicyCalls = append(m.DeleteCleanupPolicyCalls, name)

	if m.DeleteCleanupPolicyFn != nil {
		return m.DeleteCleanupPolicyFn(ctx, name)
	}

	return errMockNotConfigured
}
