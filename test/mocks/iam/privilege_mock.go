package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.PrivilegeClient = &MockPrivilegeClient{}

// MockPrivilegeClient is a mock of iamclient.PrivilegeClient.
type MockPrivilegeClient struct {
	GetPrivilegeFn    func(ctx context.Context, name string) (*security.Privilege, error)
	CreatePrivilegeFn func(ctx context.Context, privCR *iamv1alpha1.Privilege) error
	UpdatePrivilegeFn func(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error
	DeletePrivilegeFn func(ctx context.Context, name string) error

	GetPrivilegeCalls    []string
	CreatePrivilegeCalls []*iamv1alpha1.Privilege
	UpdatePrivilegeCalls []string
	DeletePrivilegeCalls []string
}

// NewMockPrivilegeClient creates a new MockPrivilegeClient.
func NewMockPrivilegeClient() *MockPrivilegeClient {
	return &MockPrivilegeClient{}
}

// GetPrivilege mock implementation.
func (m *MockPrivilegeClient) GetPrivilege(ctx context.Context, name string) (*security.Privilege, error) {
	m.GetPrivilegeCalls = append(m.GetPrivilegeCalls, name)

	if m.GetPrivilegeFn != nil {
		return m.GetPrivilegeFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreatePrivilege mock implementation.
func (m *MockPrivilegeClient) CreatePrivilege(ctx context.Context, privCR *iamv1alpha1.Privilege) error {
	m.CreatePrivilegeCalls = append(m.CreatePrivilegeCalls, privCR)

	if m.CreatePrivilegeFn != nil {
		return m.CreatePrivilegeFn(ctx, privCR)
	}

	return errMockNotConfigured
}

// UpdatePrivilege mock implementation.
func (m *MockPrivilegeClient) UpdatePrivilege(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error {
	m.UpdatePrivilegeCalls = append(m.UpdatePrivilegeCalls, name)

	if m.UpdatePrivilegeFn != nil {
		return m.UpdatePrivilegeFn(ctx, name, privCR)
	}

	return errMockNotConfigured
}

// DeletePrivilege mock implementation.
func (m *MockPrivilegeClient) DeletePrivilege(ctx context.Context, name string) error {
	m.DeletePrivilegeCalls = append(m.DeletePrivilegeCalls, name)

	if m.DeletePrivilegeFn != nil {
		return m.DeletePrivilegeFn(ctx, name)
	}

	return errMockNotConfigured
}
