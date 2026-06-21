package content

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
)

var _ contentclient.ScriptClient = &MockScriptClient{}

// MockScriptClient is a test double for contentclient.ScriptClient.
type MockScriptClient struct {
	GetScriptFn    func(ctx context.Context, name string) (*schema.Script, error)
	CreateScriptFn func(ctx context.Context, script *schema.Script) error
	UpdateScriptFn func(ctx context.Context, script *schema.Script) error
	DeleteScriptFn func(ctx context.Context, name string) error

	GetScriptCalls    []string
	CreateScriptCalls []*schema.Script
	UpdateScriptCalls []*schema.Script
	DeleteScriptCalls []string
}

// NewMockScriptClient creates a MockScriptClient with no pre-configured fns.
func NewMockScriptClient() *MockScriptClient {
	return &MockScriptClient{}
}

// GetScript implements contentclient.ScriptClient.
func (m *MockScriptClient) GetScript(ctx context.Context, name string) (*schema.Script, error) {
	m.GetScriptCalls = append(m.GetScriptCalls, name)

	if m.GetScriptFn != nil {
		return m.GetScriptFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateScript implements contentclient.ScriptClient.
func (m *MockScriptClient) CreateScript(ctx context.Context, script *schema.Script) error {
	m.CreateScriptCalls = append(m.CreateScriptCalls, script)

	if m.CreateScriptFn != nil {
		return m.CreateScriptFn(ctx, script)
	}

	return errMockNotConfigured
}

// UpdateScript implements contentclient.ScriptClient.
func (m *MockScriptClient) UpdateScript(ctx context.Context, script *schema.Script) error {
	m.UpdateScriptCalls = append(m.UpdateScriptCalls, script)

	if m.UpdateScriptFn != nil {
		return m.UpdateScriptFn(ctx, script)
	}

	return errMockNotConfigured
}

// DeleteScript implements contentclient.ScriptClient.
func (m *MockScriptClient) DeleteScript(ctx context.Context, name string) error {
	m.DeleteScriptCalls = append(m.DeleteScriptCalls, name)

	if m.DeleteScriptFn != nil {
		return m.DeleteScriptFn(ctx, name)
	}

	return errMockNotConfigured
}
