// Package iam provides mock implementations for the IAM client group.
package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ iamclient.SecurityRealmClient = &MockSecurityRealmClient{}

// MockSecurityRealmClient is a mock of iamclient.SecurityRealmClient.
type MockSecurityRealmClient struct {
	ListActiveFn    func() ([]string, error)
	ListAvailableFn func() ([]v395.RealmApiXO, error)
	ActivateFn      func(ids []string) error

	ListActiveCalls    int
	ListAvailableCalls int
	ActivateCalls      [][]string
}

// NewMockSecurityRealmClient creates a new MockSecurityRealmClient.
func NewMockSecurityRealmClient() *MockSecurityRealmClient {
	return &MockSecurityRealmClient{}
}

// ListActive mock implementation.
func (m *MockSecurityRealmClient) ListActive(_ context.Context) ([]string, error) {
	m.ListActiveCalls++

	if m.ListActiveFn != nil {
		return m.ListActiveFn()
	}

	return nil, errMockNotConfigured
}

// ListAvailable mock implementation.
func (m *MockSecurityRealmClient) ListAvailable(_ context.Context) ([]v395.RealmApiXO, error) {
	m.ListAvailableCalls++

	if m.ListAvailableFn != nil {
		return m.ListAvailableFn()
	}

	return nil, errMockNotConfigured
}

// Activate mock implementation.
func (m *MockSecurityRealmClient) Activate(_ context.Context, ids []string) error {
	m.ActivateCalls = append(m.ActivateCalls, ids)

	if m.ActivateFn != nil {
		return m.ActivateFn(ids)
	}

	return errMockNotConfigured
}
