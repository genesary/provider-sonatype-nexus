package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.PrivilegeClient = &MockPrivilegeClient{}

// MockPrivilegeClient is a mock of iamclient.PrivilegeClient.
type MockPrivilegeClient struct {
	GetFn    func(name string) (*security.Privilege, error)
	CreateFn func(privCR *iamv1alpha1.Privilege) error
	UpdateFn func(name string, privCR *iamv1alpha1.Privilege) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []*iamv1alpha1.Privilege
	UpdateCalls []string
	DeleteCalls []string
}

// NewMockPrivilegeClient creates a new MockPrivilegeClient.
func NewMockPrivilegeClient() *MockPrivilegeClient {
	return &MockPrivilegeClient{}
}

// Get mock implementation.
func (m *MockPrivilegeClient) Get(name string) (*security.Privilege, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockPrivilegeClient) Create(privCR *iamv1alpha1.Privilege) error {
	m.CreateCalls = append(m.CreateCalls, privCR)

	if m.CreateFn != nil {
		return m.CreateFn(privCR)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockPrivilegeClient) Update(name string, privCR *iamv1alpha1.Privilege) error {
	m.UpdateCalls = append(m.UpdateCalls, name)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, privCR)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockPrivilegeClient) Delete(name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
