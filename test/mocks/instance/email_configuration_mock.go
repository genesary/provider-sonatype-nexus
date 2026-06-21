// Package instance provides mock implementations for instance-level clients.
package instance

import (
	"context"
	"errors"

	mailschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

// errMockNotConfigured is returned when a mock function field is not set.
var errMockNotConfigured = errors.New("mock function not configured")

var _ instanceclient.EmailConfigurationClient = &MockEmailConfigurationClient{}

// MockEmailConfigurationClient is a test double for
// instanceclient.EmailConfigurationClient.
type MockEmailConfigurationClient struct {
	GetEmailConfigurationFn    func(ctx context.Context) (*mailschema.MailConfig, error)
	UpdateEmailConfigurationFn func(ctx context.Context, config mailschema.MailConfig) error

	GetEmailConfigurationCalls    int
	UpdateEmailConfigurationCalls []mailschema.MailConfig
}

// NewMockEmailConfigurationClient returns a MockEmailConfigurationClient
// with nil function fields.
func NewMockEmailConfigurationClient() *MockEmailConfigurationClient {
	return &MockEmailConfigurationClient{}
}

// GetEmailConfiguration implements EmailConfigurationClient.
func (m *MockEmailConfigurationClient) GetEmailConfiguration(ctx context.Context) (*mailschema.MailConfig, error) {
	m.GetEmailConfigurationCalls++

	if m.GetEmailConfigurationFn != nil {
		return m.GetEmailConfigurationFn(ctx)
	}

	return nil, errMockNotConfigured
}

// UpdateEmailConfiguration implements EmailConfigurationClient.
func (m *MockEmailConfigurationClient) UpdateEmailConfiguration(ctx context.Context, config mailschema.MailConfig) error {
	m.UpdateEmailConfigurationCalls = append(m.UpdateEmailConfigurationCalls, config)

	if m.UpdateEmailConfigurationFn != nil {
		return m.UpdateEmailConfigurationFn(ctx, config)
	}

	return errMockNotConfigured
}
