//nolint:dupl // mock structs share structural shape by design
package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.RoleClient = &MockRoleClient{}

// MockRoleClient is a mock of iamclient.RoleClient.
type MockRoleClient struct {
	GetFn    func(id string) (*security.Role, error)
	CreateFn func(role security.Role) error
	UpdateFn func(id string, role security.Role) error
	DeleteFn func(id string) error

	GetCalls    []string
	CreateCalls []security.Role
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockRoleClient creates a new MockRoleClient.
func NewMockRoleClient() *MockRoleClient {
	return &MockRoleClient{}
}

// Get mock implementation.
func (m *MockRoleClient) Get(id string) (*security.Role, error) {
	m.GetCalls = append(m.GetCalls, id)

	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockRoleClient) Create(role security.Role) error {
	m.CreateCalls = append(m.CreateCalls, role)

	if m.CreateFn != nil {
		return m.CreateFn(role)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockRoleClient) Update(id string, role security.Role) error {
	m.UpdateCalls = append(m.UpdateCalls, id)

	if m.UpdateFn != nil {
		return m.UpdateFn(id, role)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockRoleClient) Delete(id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)

	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}
