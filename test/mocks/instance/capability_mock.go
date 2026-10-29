// Package capability provides mock implementations for capability client
// testing.
package capability

import (
	"context"
	"errors"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

// errMockNotConfigured is returned when a mock function is not set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ instanceclient.CapabilityClient = &MockCapabilityClient{}

// MockCapabilityClient is a test double for instanceclient.CapabilityClient.
type MockCapabilityClient struct {
	GetFn       func(id string) (*instanceclient.Capability, error)
	CreateFn    func(create instanceclient.Capability) (*instanceclient.Capability, error)
	UpdateFn    func(id string, update instanceclient.Capability) error
	DeleteFn    func(id string) error
	ListFn      func() ([]instanceclient.Capability, error)
	ListTypesFn func() ([]v395.CapabilityTypeDTO, error)

	GetCalls    []string
	CreateCalls []instanceclient.Capability
	UpdateCalls []instanceclient.Capability
	DeleteCalls []string
}

// NewMockCapabilityClient returns a new MockCapabilityClient with no
// functions configured.
func NewMockCapabilityClient() *MockCapabilityClient {
	return &MockCapabilityClient{}
}

// Get mock implementation.
func (m *MockCapabilityClient) Get(_ context.Context, id string) (*instanceclient.Capability, error) {
	m.GetCalls = append(m.GetCalls, id)
	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockCapabilityClient) Create(_ context.Context, create instanceclient.Capability) (*instanceclient.Capability, error) {
	m.CreateCalls = append(m.CreateCalls, create)
	if m.CreateFn != nil {
		return m.CreateFn(create)
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockCapabilityClient) Update(_ context.Context, id string, update instanceclient.Capability) error {
	m.UpdateCalls = append(m.UpdateCalls, update)
	if m.UpdateFn != nil {
		return m.UpdateFn(id, update)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockCapabilityClient) Delete(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}

// List mock implementation.
func (m *MockCapabilityClient) List(_ context.Context) ([]instanceclient.Capability, error) {
	if m.ListFn != nil {
		return m.ListFn()
	}

	return nil, errMockNotConfigured
}

// ListTypes mock implementation.
func (m *MockCapabilityClient) ListTypes(_ context.Context) ([]v395.CapabilityTypeDTO, error) {
	if m.ListTypesFn != nil {
		return m.ListTypesFn()
	}

	return nil, errMockNotConfigured
}
