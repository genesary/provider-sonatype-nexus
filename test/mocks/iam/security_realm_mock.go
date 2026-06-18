// Package iam provides mock implementations for the IAM client group.
package iam

import (
	"context"
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

// errMockNotConfigured is returned when a mock function has not been set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ iamclient.SecurityRealmClient = &MockSecurityRealmClient{}

// MockSecurityRealmClient is a mock of iamclient.SecurityRealmClient.
type MockSecurityRealmClient struct {
	ListActiveRealmsFn    func(ctx context.Context) ([]string, error)
	ListAvailableRealmsFn func(ctx context.Context) ([]security.Realm, error)
	ActivateRealmsFn      func(ctx context.Context, ids []string) error

	ListActiveRealmsCalls    int
	ListAvailableRealmsCalls int
	ActivateRealmsCalls      [][]string
}

// NewMockSecurityRealmClient creates a new MockSecurityRealmClient.
func NewMockSecurityRealmClient() *MockSecurityRealmClient {
	return &MockSecurityRealmClient{}
}

// ListActiveRealms mock implementation.
func (m *MockSecurityRealmClient) ListActiveRealms(ctx context.Context) ([]string, error) {
	m.ListActiveRealmsCalls++

	if m.ListActiveRealmsFn != nil {
		return m.ListActiveRealmsFn(ctx)
	}

	return nil, errMockNotConfigured
}

// ListAvailableRealms mock implementation.
func (m *MockSecurityRealmClient) ListAvailableRealms(ctx context.Context) ([]security.Realm, error) {
	m.ListAvailableRealmsCalls++

	if m.ListAvailableRealmsFn != nil {
		return m.ListAvailableRealmsFn(ctx)
	}

	return nil, errMockNotConfigured
}

// ActivateRealms mock implementation.
func (m *MockSecurityRealmClient) ActivateRealms(ctx context.Context, ids []string) error {
	m.ActivateRealmsCalls = append(m.ActivateRealmsCalls, ids)

	if m.ActivateRealmsFn != nil {
		return m.ActivateRealmsFn(ctx, ids)
	}

	return errMockNotConfigured
}
