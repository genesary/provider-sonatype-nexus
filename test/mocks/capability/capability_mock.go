// Package capability provides mock implementations for capability client
// testing.
package capability

import (
	"context"
	"errors"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"

	capabilityclient "github.com/genesary/provider-sonatype-nexus/internal/clients/capability"
)

// errMockNotConfigured is returned when a mock function is not set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ capabilityclient.CapabilityClient = &MockCapabilityClient{}

// MockCapabilityClient is a test double for capabilityclient.CapabilityClient.
type MockCapabilityClient struct {
	GetCapabilityFn    func(ctx context.Context, id string) (*nexussdk.Capability, error)
	CreateCapabilityFn func(ctx context.Context, create nexussdk.CapabilityCreate) (*nexussdk.Capability, error)
	UpdateCapabilityFn func(ctx context.Context, id string, update nexussdk.CapabilityUpdate) error
	DeleteCapabilityFn func(ctx context.Context, id string) error

	GetCapabilityCalls    []string
	CreateCapabilityCalls []nexussdk.CapabilityCreate
	UpdateCapabilityCalls []nexussdk.CapabilityUpdate
	DeleteCapabilityCalls []string
}

// NewMockCapabilityClient returns a new MockCapabilityClient with no
// functions configured.
func NewMockCapabilityClient() *MockCapabilityClient {
	return &MockCapabilityClient{}
}

// GetCapability mock implementation.
func (m *MockCapabilityClient) GetCapability(ctx context.Context, id string) (*nexussdk.Capability, error) {
	m.GetCapabilityCalls = append(m.GetCapabilityCalls, id)
	if m.GetCapabilityFn != nil {
		return m.GetCapabilityFn(ctx, id)
	}

	return nil, errMockNotConfigured
}

// CreateCapability mock implementation.
func (m *MockCapabilityClient) CreateCapability(ctx context.Context, create nexussdk.CapabilityCreate) (*nexussdk.Capability, error) {
	m.CreateCapabilityCalls = append(m.CreateCapabilityCalls, create)
	if m.CreateCapabilityFn != nil {
		return m.CreateCapabilityFn(ctx, create)
	}

	return nil, errMockNotConfigured
}

// UpdateCapability mock implementation.
func (m *MockCapabilityClient) UpdateCapability(ctx context.Context, id string, update nexussdk.CapabilityUpdate) error {
	m.UpdateCapabilityCalls = append(m.UpdateCapabilityCalls, update)
	if m.UpdateCapabilityFn != nil {
		return m.UpdateCapabilityFn(ctx, id, update)
	}

	return errMockNotConfigured
}

// DeleteCapability mock implementation.
func (m *MockCapabilityClient) DeleteCapability(ctx context.Context, id string) error {
	m.DeleteCapabilityCalls = append(m.DeleteCapabilityCalls, id)
	if m.DeleteCapabilityFn != nil {
		return m.DeleteCapabilityFn(ctx, id)
	}

	return errMockNotConfigured
}
