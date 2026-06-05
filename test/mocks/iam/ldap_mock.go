//nolint:dupl // mock structs share structural shape by design
package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.LDAPClient = &MockLDAPClient{}

// MockLDAPClient is a mock of iamclient.LDAPClient.
type MockLDAPClient struct {
	GetLDAPFn    func(ctx context.Context, name string) (*security.LDAP, error)
	CreateLDAPFn func(ctx context.Context, ldap security.LDAP) error
	UpdateLDAPFn func(ctx context.Context, name string, ldap security.LDAP) error
	DeleteLDAPFn func(ctx context.Context, name string) error

	GetLDAPCalls    []string
	CreateLDAPCalls []security.LDAP
	UpdateLDAPCalls []string
	DeleteLDAPCalls []string
}

// NewMockLDAPClient creates a new MockLDAPClient.
func NewMockLDAPClient() *MockLDAPClient {
	return &MockLDAPClient{}
}

// GetLDAP mock implementation.
func (m *MockLDAPClient) GetLDAP(ctx context.Context, name string) (*security.LDAP, error) {
	m.GetLDAPCalls = append(m.GetLDAPCalls, name)

	if m.GetLDAPFn != nil {
		return m.GetLDAPFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateLDAP mock implementation.
func (m *MockLDAPClient) CreateLDAP(ctx context.Context, ldap security.LDAP) error {
	m.CreateLDAPCalls = append(m.CreateLDAPCalls, ldap)

	if m.CreateLDAPFn != nil {
		return m.CreateLDAPFn(ctx, ldap)
	}

	return errMockNotConfigured
}

// UpdateLDAP mock implementation.
func (m *MockLDAPClient) UpdateLDAP(ctx context.Context, name string, ldap security.LDAP) error {
	m.UpdateLDAPCalls = append(m.UpdateLDAPCalls, name)

	if m.UpdateLDAPFn != nil {
		return m.UpdateLDAPFn(ctx, name, ldap)
	}

	return errMockNotConfigured
}

// DeleteLDAP mock implementation.
func (m *MockLDAPClient) DeleteLDAP(ctx context.Context, name string) error {
	m.DeleteLDAPCalls = append(m.DeleteLDAPCalls, name)

	if m.DeleteLDAPFn != nil {
		return m.DeleteLDAPFn(ctx, name)
	}

	return errMockNotConfigured
}
