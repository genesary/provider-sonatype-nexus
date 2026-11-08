package capability

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.SSRFProtectionClient = &MockSSRFProtectionClient{}

// MockSSRFProtectionClient is a test double for
// instanceclient.SSRFProtectionClient.
type MockSSRFProtectionClient struct {
	GetFn    func() (*v395.SsrfProtectionConfigurationXO, error)
	UpdateFn func(config v395.SsrfProtectionConfigurationXO) error

	GetCalls    int
	UpdateCalls []v395.SsrfProtectionConfigurationXO
}

// NewMockSSRFProtectionClient returns a MockSSRFProtectionClient with no
// functions set.
func NewMockSSRFProtectionClient() *MockSSRFProtectionClient {
	return &MockSSRFProtectionClient{}
}

// Get mock implementation.
func (m *MockSSRFProtectionClient) Get(_ context.Context) (*v395.SsrfProtectionConfigurationXO, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errMockNotConfigured
}

// Update mock implementation.
func (m *MockSSRFProtectionClient) Update(_ context.Context, config v395.SsrfProtectionConfigurationXO) error {
	m.UpdateCalls = append(m.UpdateCalls, config)

	if m.UpdateFn != nil {
		return m.UpdateFn(config)
	}

	return errMockNotConfigured
}
