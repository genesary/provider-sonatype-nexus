//nolint:dupl // mock structs share structural shape by design
package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.RoleClient = &MockRoleClient{}

// MockRoleClient is a mock of iamclient.RoleClient.
type MockRoleClient struct {
	GetFn    func(id string) (*v395.RoleXOResponse, error)
	CreateFn func(role v395.RoleXORequest) error
	UpdateFn func(id string, role v395.RoleXORequest) error
	DeleteFn func(id string) error

	GetCalls    []string
	CreateCalls []v395.RoleXORequest
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockRoleClient creates a new MockRoleClient.
func NewMockRoleClient() *MockRoleClient {
	return &MockRoleClient{}
}

// Get mock implementation.
func (m *MockRoleClient) Get(_ context.Context, id string) (*v395.RoleXOResponse, error) {
	m.GetCalls = append(m.GetCalls, id)

	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockRoleClient) Create(_ context.Context, role v395.RoleXORequest) error {
	m.CreateCalls = append(m.CreateCalls, role)

	if m.CreateFn != nil {
		return m.CreateFn(role)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockRoleClient) Update(_ context.Context, id string, role v395.RoleXORequest) error {
	m.UpdateCalls = append(m.UpdateCalls, id)

	if m.UpdateFn != nil {
		return m.UpdateFn(id, role)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockRoleClient) Delete(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)

	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}
