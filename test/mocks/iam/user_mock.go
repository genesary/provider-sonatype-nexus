package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.UserClient = &MockUserClient{}

// MockUserClient is a mock of iamclient.UserClient.
type MockUserClient struct {
	GetUserFn        func(ctx context.Context, id string) (*security.User, error)
	CreateUserFn     func(ctx context.Context, user security.User) error
	UpdateUserFn     func(ctx context.Context, id string, user security.User) error
	DeleteUserFn     func(ctx context.Context, id string) error
	ChangePasswordFn func(ctx context.Context, id, password string) error

	GetUserCalls        []string
	CreateUserCalls     []security.User
	UpdateUserCalls     []string
	DeleteUserCalls     []string
	ChangePasswordCalls []string
}

// NewMockUserClient creates a new MockUserClient.
func NewMockUserClient() *MockUserClient {
	return &MockUserClient{}
}

// GetUser mock implementation.
func (m *MockUserClient) GetUser(ctx context.Context, id string) (*security.User, error) {
	m.GetUserCalls = append(m.GetUserCalls, id)

	if m.GetUserFn != nil {
		return m.GetUserFn(ctx, id)
	}

	return nil, errMockNotConfigured
}

// CreateUser mock implementation.
func (m *MockUserClient) CreateUser(ctx context.Context, user security.User) error {
	m.CreateUserCalls = append(m.CreateUserCalls, user)

	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, user)
	}

	return errMockNotConfigured
}

// UpdateUser mock implementation.
func (m *MockUserClient) UpdateUser(ctx context.Context, id string, user security.User) error {
	m.UpdateUserCalls = append(m.UpdateUserCalls, id)

	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, id, user)
	}

	return errMockNotConfigured
}

// DeleteUser mock implementation.
func (m *MockUserClient) DeleteUser(ctx context.Context, id string) error {
	m.DeleteUserCalls = append(m.DeleteUserCalls, id)

	if m.DeleteUserFn != nil {
		return m.DeleteUserFn(ctx, id)
	}

	return errMockNotConfigured
}

// ChangePassword mock implementation.
func (m *MockUserClient) ChangePassword(ctx context.Context, id, password string) error {
	m.ChangePasswordCalls = append(m.ChangePasswordCalls, id)

	if m.ChangePasswordFn != nil {
		return m.ChangePasswordFn(ctx, id, password)
	}

	return errMockNotConfigured
}
