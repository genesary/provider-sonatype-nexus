package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.RoleClient = &MockRoleClient{}

// MockRoleClient is a mock of iamclient.RoleClient.
type MockRoleClient struct {
	GetRoleFn    func(ctx context.Context, id string) (*security.Role, error)
	CreateRoleFn func(ctx context.Context, role security.Role) error
	UpdateRoleFn func(ctx context.Context, id string, role security.Role) error
	DeleteRoleFn func(ctx context.Context, id string) error

	GetRoleCalls    []string
	CreateRoleCalls []security.Role
	UpdateRoleCalls []string
	DeleteRoleCalls []string
}

// NewMockRoleClient creates a new MockRoleClient.
func NewMockRoleClient() *MockRoleClient {
	return &MockRoleClient{}
}

// GetRole mock implementation.
func (m *MockRoleClient) GetRole(ctx context.Context, id string) (*security.Role, error) {
	m.GetRoleCalls = append(m.GetRoleCalls, id)

	if m.GetRoleFn != nil {
		return m.GetRoleFn(ctx, id)
	}

	return nil, errMockNotConfigured
}

// CreateRole mock implementation.
func (m *MockRoleClient) CreateRole(ctx context.Context, role security.Role) error {
	m.CreateRoleCalls = append(m.CreateRoleCalls, role)

	if m.CreateRoleFn != nil {
		return m.CreateRoleFn(ctx, role)
	}

	return errMockNotConfigured
}

// UpdateRole mock implementation.
func (m *MockRoleClient) UpdateRole(ctx context.Context, id string, role security.Role) error {
	m.UpdateRoleCalls = append(m.UpdateRoleCalls, id)

	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, id, role)
	}

	return errMockNotConfigured
}

// DeleteRole mock implementation.
func (m *MockRoleClient) DeleteRole(ctx context.Context, id string) error {
	m.DeleteRoleCalls = append(m.DeleteRoleCalls, id)

	if m.DeleteRoleFn != nil {
		return m.DeleteRoleFn(ctx, id)
	}

	return errMockNotConfigured
}
