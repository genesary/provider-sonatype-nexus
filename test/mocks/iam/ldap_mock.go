//nolint:dupl // mock structs share structural shape by design
package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.LDAPClient = &MockLDAPClient{}

// MockLDAPClient is a mock of iamclient.LDAPClient.
type MockLDAPClient struct {
	GetFn    func(name string) (*security.LDAP, error)
	CreateFn func(ldap security.LDAP) error
	UpdateFn func(name string, ldap security.LDAP) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []security.LDAP
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockLDAPClient creates a new MockLDAPClient.
func NewMockLDAPClient() *MockLDAPClient {
	return &MockLDAPClient{}
}

// Get mock implementation.
func (m *MockLDAPClient) Get(name string) (*security.LDAP, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockLDAPClient) Create(ldap security.LDAP) error {
	m.CreateCalls = append(m.CreateCalls, ldap)

	if m.CreateFn != nil {
		return m.CreateFn(ldap)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockLDAPClient) Update(name string, ldap security.LDAP) error {
	m.UpdateCalls = append(m.UpdateCalls, name)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, ldap)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockLDAPClient) Delete(name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
