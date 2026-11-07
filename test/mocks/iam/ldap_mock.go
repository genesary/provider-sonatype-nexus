//nolint:dupl // mock structs share structural shape by design
package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.LDAPClient = &MockLDAPClient{}

// MockLDAPClient is a mock of iamclient.LDAPClient.
type MockLDAPClient struct {
	GetFn    func(name string) (*v395.ReadLdapServerXo, error)
	CreateFn func(ldap v395.CreateLdapServerXo) error
	UpdateFn func(name string, ldap v395.UpdateLdapServerXo) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []v395.CreateLdapServerXo
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockLDAPClient creates a new MockLDAPClient.
func NewMockLDAPClient() *MockLDAPClient {
	return &MockLDAPClient{}
}

// Get mock implementation.
func (m *MockLDAPClient) Get(_ context.Context, name string) (*v395.ReadLdapServerXo, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockLDAPClient) Create(_ context.Context, ldap v395.CreateLdapServerXo) error {
	m.CreateCalls = append(m.CreateCalls, ldap)

	if m.CreateFn != nil {
		return m.CreateFn(ldap)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockLDAPClient) Update(_ context.Context, name string, ldap v395.UpdateLdapServerXo) error {
	m.UpdateCalls = append(m.UpdateCalls, name)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, ldap)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockLDAPClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
