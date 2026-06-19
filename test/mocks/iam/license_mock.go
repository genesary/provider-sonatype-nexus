package iam

import (
	"context"

	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
)

var _ iamclient.LicenseClient = &MockLicenseClient{}

// MockLicenseClient is a mock of iamclient.LicenseClient.
type MockLicenseClient struct {
	GetLicenseFn     func(ctx context.Context) (*iamclient.LicenseInfo, error)
	InstallLicenseFn func(ctx context.Context, licenseData []byte) error
	DeleteLicenseFn  func(ctx context.Context) error

	GetLicenseCalls     int
	InstallLicenseCalls [][]byte
	DeleteLicenseCalls  int
}

// NewMockLicenseClient creates a new MockLicenseClient.
func NewMockLicenseClient() *MockLicenseClient {
	return &MockLicenseClient{}
}

// GetLicense mock implementation.
func (m *MockLicenseClient) GetLicense(ctx context.Context) (*iamclient.LicenseInfo, error) {
	m.GetLicenseCalls++

	if m.GetLicenseFn != nil {
		return m.GetLicenseFn(ctx)
	}

	return nil, errMockNotConfigured
}

// InstallLicense mock implementation.
func (m *MockLicenseClient) InstallLicense(ctx context.Context, licenseData []byte) error {
	m.InstallLicenseCalls = append(m.InstallLicenseCalls, licenseData)

	if m.InstallLicenseFn != nil {
		return m.InstallLicenseFn(ctx, licenseData)
	}

	return errMockNotConfigured
}

// DeleteLicense mock implementation.
func (m *MockLicenseClient) DeleteLicense(ctx context.Context) error {
	m.DeleteLicenseCalls++

	if m.DeleteLicenseFn != nil {
		return m.DeleteLicenseFn(ctx)
	}

	return errMockNotConfigured
}
