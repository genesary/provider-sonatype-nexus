package securityssltruststore

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/test/mocks"
)

// testPem is a test PEM certificate.
const testPem = `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJALRiMLAh0ERXMA0GCSqGSIb3DQEBBQUAMBExDzANBgNVBAMMBnRl
c3RDQTAYHDI1MDEwMTAwMDAwMFoYDzIwNTAwMTAxMDAwMDAwWjARMQ8wDQYDVQQD
DAZ0ZXN0Q0EwXDANBgkqhkiG9w0BAQEFAANLADBIAkEA0Z3VS5JJcds3xf0GRHWB
L1JohBCR0MNMIyBjR0FNBHiPl0BoO/Iu2k0U4MAlr7KCi/ByMLBmsAy0JwPfJbm
5wIDAQABoyMwITAfBgNVHREEGDAWhwR/AAABhwQKAAABhwSsEQABMA0GCSqGSIb3
DQEBBQUAA0EAxSPMb7r3v4fhfW6oSaqJN8JgRJAJBfBNOsNhLZYMaO5YoKWXYhGA
-----END CERTIFICATE-----`

// newTruststoreCR creates a new SecuritySSLTruststore CR for testing.
func newTruststoreCR(pem string) *v1alpha1.SecuritySSLTruststore {
	cr := &v1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-cert",
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.SecuritySSLTruststoreSpec{
			ForProvider: v1alpha1.SecuritySSLTruststoreParameters{
				Pem: pem,
			},
		},
	}

	return cr
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *v1alpha1.SecuritySSLTruststore
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NoExternalName",
			cr:   newTruststoreCR(testPem),
			// no external name set → resource doesn't exist yet
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{
						{
							Id:                "cert-id-123",
							Pem:               testPem,
							Fingerprint:       "AB:CD:EF",
							SubjectCommonName: "testCA",
						},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButPemChanged",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR("new-pem-content")
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{
						{Id: "cert-id-123", Pem: testPem},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "CertNotFoundInList",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-deleted")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{}, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "APIError",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return nil, errors.New("connection error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			obs, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.SecuritySSLTruststore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.AddCertificateFn = func(ctx context.Context, cert *security.SSLCertificate) error {
					return nil
				}
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{
						{Id: "new-cert-id", Pem: testPem},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateAPIError",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.AddCertificateFn = func(ctx context.Context, cert *security.SSLCertificate) error {
					return errors.New("add error")
				}
			},
			wantErr: true,
		},
		{
			name: "CreateCantFindAfterAdd",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.AddCertificateFn = func(ctx context.Context, cert *security.SSLCertificate) error {
					return nil
				}
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{}, nil // not found
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				externalName := meta.GetExternalName(tt.cr)
				if externalName != "new-cert-id" {
					t.Errorf("Create() external name = %q, want %q", externalName, "new-cert-id")
				}
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.SecuritySSLTruststore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "old-cert-id")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.RemoveCertificateFn = func(ctx context.Context, id string) error {
					return nil
				}
				mc.MockSSL.AddCertificateFn = func(ctx context.Context, cert *security.SSLCertificate) error {
					return nil
				}
				mc.MockSSL.ListCertificatesFn = func(ctx context.Context) ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{
						{Id: "new-cert-id", Pem: testPem},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateAddError",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "old-cert-id")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.RemoveCertificateFn = func(ctx context.Context, id string) error {
					return nil
				}
				mc.MockSSL.AddCertificateFn = func(ctx context.Context, cert *security.SSLCertificate) error {
					return errors.New("add error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.SecuritySSLTruststore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.RemoveCertificateFn = func(ctx context.Context, id string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.RemoveCertificateFn = func(ctx context.Context, id string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNoExternalName",
			cr:   newTruststoreCR(testPem),
			// no external name → nothing to delete
			wantErr: false,
		},
		{
			name: "DeleteAPIError",
			cr: func() *v1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSSL.RemoveCertificateFn = func(ctx context.Context, id string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
