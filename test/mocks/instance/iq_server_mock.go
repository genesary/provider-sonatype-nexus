package capability

import (
	"errors"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

var _ instanceclient.IQServerClient = &MockIQServerClient{}

type MockIQServerClient struct {
	GetFn    func() (*nexussdk.IQServerConfiguration, error)
	UpdateFn func(config nexussdk.IQServerConfiguration) error

	GetCalls    int
	UpdateCalls []nexussdk.IQServerConfiguration
}

func NewMockIQServerClient() *MockIQServerClient {
	return &MockIQServerClient{}
}

func (m *MockIQServerClient) Get() (*nexussdk.IQServerConfiguration, error) {
	m.GetCalls++

	if m.GetFn != nil {
		return m.GetFn()
	}

	return nil, errors.New("mock function not configured")
}

func (m *MockIQServerClient) Update(config nexussdk.IQServerConfiguration) error {
	m.UpdateCalls = append(m.UpdateCalls, config)

	if m.UpdateFn != nil {
		return m.UpdateFn(config)
	}

	return errors.New("mock function not configured")
}
