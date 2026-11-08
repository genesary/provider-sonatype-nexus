package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.ScriptClient = &MockScriptClient{}

// MockScriptClient is a test double for instanceclient.ScriptClient.
type MockScriptClient struct {
	CreateFn func(script v395.ScriptXO) error
	DeleteFn func(name string) error
	GetFn    func(name string) (*v395.ScriptXO, error)
	UpdateFn func(name string, script v395.ScriptXO) error

	CreateCalls []v395.ScriptXO
	DeleteCalls []string
	GetCalls    []string
	UpdateCalls []v395.ScriptXO
}

// NewMockScriptClient returns a MockScriptClient with no functions set.
func NewMockScriptClient() *MockScriptClient {
	return &MockScriptClient{}
}

// Create mock implementation.
func (m *MockScriptClient) Create(_ context.Context, script v395.ScriptXO) error {
	m.CreateCalls = append(m.CreateCalls, script)

	if m.CreateFn != nil {
		return m.CreateFn(script)
	}

	return errMockNotConfigured
}

// Delete mock implementation.
func (m *MockScriptClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)

	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return errMockNotConfigured
}

// Get mock implementation.
func (m *MockScriptClient) Get(_ context.Context, name string) (*v395.ScriptXO, error) {
	m.GetCalls = append(m.GetCalls, name)

	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockScriptClient) Update(_ context.Context, name string, script v395.ScriptXO) error {
	m.UpdateCalls = append(m.UpdateCalls, script)

	if m.UpdateFn != nil {
		return m.UpdateFn(name, script)
	}

	return errMockNotConfigured
}
