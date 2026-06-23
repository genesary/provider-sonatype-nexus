// Package fake provides mock implementations for testing.
package fake

import (
	"context"

	"github.com/pkg/errors"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
)

// errNotImplemented is returned when a mock function has not been configured.
var errNotImplemented = errors.New("mock function not implemented")

// MockLicenseClient is a mock iamclient.LicenseClient for unit tests.
type MockLicenseClient struct {
	GetLicenseFn     func(ctx context.Context) (*iamclient.LicenseInfo, error)
	InstallLicenseFn func(ctx context.Context, licenseData []byte) error
	DeleteLicenseFn  func(ctx context.Context) error
}

// Ensure MockLicenseClient implements LicenseClient.
var _ iamclient.LicenseClient = &MockLicenseClient{}

// NewMockLicenseClient returns a new MockLicenseClient.
func NewMockLicenseClient() *MockLicenseClient {
	return &MockLicenseClient{}
}

// GetLicense implements LicenseClient.
func (m *MockLicenseClient) GetLicense(ctx context.Context) (*iamclient.LicenseInfo, error) {
	if m.GetLicenseFn != nil {
		return m.GetLicenseFn(ctx)
	}

	return nil, errNotImplemented
}

// InstallLicense implements LicenseClient.
func (m *MockLicenseClient) InstallLicense(ctx context.Context, licenseData []byte) error {
	if m.InstallLicenseFn != nil {
		return m.InstallLicenseFn(ctx, licenseData)
	}

	return errNotImplemented
}

// DeleteLicense implements LicenseClient.
func (m *MockLicenseClient) DeleteLicense(ctx context.Context) error {
	if m.DeleteLicenseFn != nil {
		return m.DeleteLicenseFn(ctx)
	}

	return errNotImplemented
}
