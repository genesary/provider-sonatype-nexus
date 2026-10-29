package content

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.RoutingRuleClient = &MockRoutingRuleClient{}

// MockRoutingRuleClient is a test double for contentclient.RoutingRuleClient.
type MockRoutingRuleClient struct {
	GetFn    func(name string) (*v395.RoutingRuleXO, error)
	CreateFn func(rule *v395.RoutingRuleXO) error
	UpdateFn func(rule *v395.RoutingRuleXO) error
	DeleteFn func(name string) error

	GetCalls    []string
	CreateCalls []*v395.RoutingRuleXO
	UpdateCalls []*v395.RoutingRuleXO
	DeleteCalls []string
}

// NewMockRoutingRuleClient creates a new MockRoutingRuleClient.
func NewMockRoutingRuleClient() *MockRoutingRuleClient {
	return &MockRoutingRuleClient{}
}

// Get mock implementation.
func (m *MockRoutingRuleClient) Get(_ context.Context, name string) (*v395.RoutingRuleXO, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Create mock implementation.
func (m *MockRoutingRuleClient) Create(_ context.Context, rule *v395.RoutingRuleXO) error {
	m.CreateCalls = append(m.CreateCalls, rule)

	if m.CreateFn != nil {
		return m.CreateFn(rule)
	}

	return errMockNotConfigured
}

// Update mock implementation.
func (m *MockRoutingRuleClient) Update(_ context.Context, rule *v395.RoutingRuleXO) error {
	m.UpdateCalls = append(m.UpdateCalls, rule)

	if m.UpdateFn != nil {
		return m.UpdateFn(rule)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockRoutingRuleClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}
