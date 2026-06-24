package content

import (
	nexusschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.RoutingRuleClient = &MockRoutingRuleClient{}

// MockRoutingRuleClient is a test double for contentclient.RoutingRuleClient.
type MockRoutingRuleClient struct {
	GetFn    func(name string) (*nexusschema.RoutingRule, error)
	CreateFn func(rule *nexusschema.RoutingRule) error
	UpdateFn func(rule *nexusschema.RoutingRule) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []*nexusschema.RoutingRule
	UpdateCalls []*nexusschema.RoutingRule
	DeleteCalls []string
}

// NewMockRoutingRuleClient creates a new MockRoutingRuleClient.
func NewMockRoutingRuleClient() *MockRoutingRuleClient {
	return &MockRoutingRuleClient{}
}

// Get mock implementation.
func (m *MockRoutingRuleClient) Get(name string) (*nexusschema.RoutingRule, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockRoutingRuleClient) Create(rule *nexusschema.RoutingRule) error {
	m.CreateCalls = append(m.CreateCalls, rule)

	if m.CreateFn != nil {
		return m.CreateFn(rule)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockRoutingRuleClient) Update(rule *nexusschema.RoutingRule) error {
	m.UpdateCalls = append(m.UpdateCalls, rule)

	if m.UpdateFn != nil {
		return m.UpdateFn(rule)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockRoutingRuleClient) Delete(name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
