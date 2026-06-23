// Package iam provides mock implementations for the IAM client group.
package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ iamclient.SecurityRealmClient = &MockSecurityRealmClient{}

// MockSecurityRealmClient is a mock of iamclient.SecurityRealmClient.
type MockSecurityRealmClient struct {
	ListActiveFn    func() ([]string, error)
	ListAvailableFn func() ([]security.Realm, error)
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
func (m *MockSecurityRealmClient) ListActive() ([]string, error) {
	m.ListActiveCalls++

	if m.ListActiveFn != nil {
		return m.ListActiveFn()
	}

	return nil, errMockNotConfigured
}

// ListAvailable mock implementation.
func (m *MockSecurityRealmClient) ListAvailable() ([]security.Realm, error) {
	m.ListAvailableCalls++

	if m.ListAvailableFn != nil {
		return m.ListAvailableFn()
	}

	return nil, errMockNotConfigured
}

// Activate mock implementation.
func (m *MockSecurityRealmClient) Activate(ids []string) error {
	m.ActivateCalls = append(m.ActivateCalls, ids)

	if m.ActivateFn != nil {
		return m.ActivateFn(ids)
	}

	return errMockNotConfigured
}
