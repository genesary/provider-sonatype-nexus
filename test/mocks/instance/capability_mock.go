// Package capability provides mock implementations for capability client
// testing.
package capability

import (
	"errors"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

// errMockNotConfigured is returned when a mock function is not set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ instanceclient.CapabilityClient = &MockCapabilityClient{}

// MockCapabilityClient is a test double for instanceclient.CapabilityClient.
type MockCapabilityClient struct {
	GetFn       func(id string) (*nexussdk.Capability, error)
	CreateFn    func(create nexussdk.CapabilityCreate) (*nexussdk.Capability, error)
	UpdateFn    func(id string, update nexussdk.CapabilityUpdate) error
	DeleteFn    func(id string) error
	ListFn      func() ([]nexussdk.Capability, error)
	ListTypesFn func() ([]nexussdk.TypeDescriptor, error)

	GetCalls    []string
	CreateCalls []nexussdk.CapabilityCreate
	UpdateCalls []nexussdk.CapabilityUpdate
	DeleteCalls []string
}

// NewMockCapabilityClient returns a new MockCapabilityClient with no
// functions configured.
func NewMockCapabilityClient() *MockCapabilityClient {
	return &MockCapabilityClient{}
}

// Get mock implementation.
func (m *MockCapabilityClient) Get(id string) (*nexussdk.Capability, error) {
	m.GetCalls = append(m.GetCalls, id)
	if m.GetFn != nil {
		return m.GetFn(id)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockCapabilityClient) Create(create nexussdk.CapabilityCreate) (*nexussdk.Capability, error) {
	m.CreateCalls = append(m.CreateCalls, create)
	if m.CreateFn != nil {
		return m.CreateFn(create)
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockCapabilityClient) Update(id string, update nexussdk.CapabilityUpdate) error {
	m.UpdateCalls = append(m.UpdateCalls, update)
	if m.UpdateFn != nil {
		return m.UpdateFn(id, update)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockCapabilityClient) Delete(id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return errMockNotConfigured
}

// List mock implementation.
func (m *MockCapabilityClient) List() ([]nexussdk.Capability, error) {
	if m.ListFn != nil {
		return m.ListFn()
	}

	return nil, errMockNotConfigured
}

// ListTypes mock implementation.
func (m *MockCapabilityClient) ListTypes() ([]nexussdk.TypeDescriptor, error) {
	if m.ListTypesFn != nil {
		return m.ListTypesFn()
	}

	return nil, errMockNotConfigured
}
