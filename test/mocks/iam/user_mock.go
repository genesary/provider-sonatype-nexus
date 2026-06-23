package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserClient = &MockUserClient{}

// MockUserClient is a mock of iamclient.UserClient.
type MockUserClient struct {
	GetFn            func(id string) (*security.User, error)
	CreateFn         func(user security.User) error
	UpdateFn         func(id string, user security.User) error
	DeleteFn         func(id string) error
	ChangePasswordFn func(id, password string) error

	GetCalls            []string
	CreateCalls         []security.User
	UpdateCalls         []string
	DeleteCalls         []string
	ChangePasswordCalls []string
}

// NewMockUserClient creates a new MockUserClient.
func NewMockUserClient() *MockUserClient {
	return &MockUserClient{}
}

// Get mock implementation.
func (m *MockUserClient) Get(id string) (*security.User, error) {
	m.GetCalls = append(m.GetCalls, id)

	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockUserClient) Create(user security.User) error {
	m.CreateCalls = append(m.CreateCalls, user)

	if m.CreateFn != nil {
		return m.CreateFn(user)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockUserClient) Update(id string, user security.User) error {
	m.UpdateCalls = append(m.UpdateCalls, id)

	if m.UpdateFn != nil {
		return m.UpdateFn(id, user)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockUserClient) Delete(id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)

	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}

// ChangePassword mock implementation.
func (m *MockUserClient) ChangePassword(id, password string) error {
	m.ChangePasswordCalls = append(m.ChangePasswordCalls, id)

	if m.ChangePasswordFn != nil {
		return m.ChangePasswordFn(id, password)
	}

	return errMockNotConfigured
}
