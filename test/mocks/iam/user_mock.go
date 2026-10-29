package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserClient = &MockUserClient{}

// MockUserClient is a mock of iamclient.UserClient.
type MockUserClient struct {
	GetFn            func(id string) (*v395.ApiUser, error)
	CreateFn         func(user v395.ApiCreateUser) error
	UpdateFn         func(id string, user v395.ApiUser) error
	DeleteFn         func(id string) error
	ChangePasswordFn func(id, password string) error

	GetCalls            []string
	CreateCalls         []v395.ApiCreateUser
	UpdateCalls         []string
	DeleteCalls         []string
	ChangePasswordCalls []string
}

// NewMockUserClient creates a new MockUserClient.
func NewMockUserClient() *MockUserClient {
	return &MockUserClient{}
}

// Get mock implementation.
func (m *MockUserClient) Get(_ context.Context, id string) (*v395.ApiUser, error) {
	m.GetCalls = append(m.GetCalls, id)

	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockUserClient) Create(_ context.Context, user v395.ApiCreateUser) error {
	m.CreateCalls = append(m.CreateCalls, user)

	if m.CreateFn != nil {
		return m.CreateFn(user)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockUserClient) Update(_ context.Context, id string, user v395.ApiUser) error {
	m.UpdateCalls = append(m.UpdateCalls, id)

	if m.UpdateFn != nil {
		return m.UpdateFn(id, user)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockUserClient) Delete(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)

	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}

// ChangePassword mock implementation.
func (m *MockUserClient) ChangePassword(_ context.Context, id, password string) error {
	m.ChangePasswordCalls = append(m.ChangePasswordCalls, id)

	if m.ChangePasswordFn != nil {
		return m.ChangePasswordFn(id, password)
	}

	return errMockNotConfigured
}
