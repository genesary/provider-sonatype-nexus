package securityssltruststore

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
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
func newTruststoreCR(pem string) *iamv1alpha1.SecuritySSLTruststore {
	return &iamv1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-cert",
			Annotations: map[string]string{},
		},
		Spec: iamv1alpha1.SecuritySSLTruststoreSpec{
			ForProvider: iamv1alpha1.SecuritySSLTruststoreParameters{
				Pem: pem,
			},
		},
	}
}

// newTestScheme registers iam and nexus v1alpha1 types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := iamv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(iam) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *iamv1alpha1.SecuritySSLTruststore
		mockSetup    func(*iammocks.MockSSLTruststoreClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:         "NoExternalName",
			cr:           newTruststoreCR(testPem),
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
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
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR("new-pem-content")
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
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
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-deleted")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{}, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "APIError",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
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

			mc := iammocks.NewMockSSLTruststoreClient()
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
		cr        *iamv1alpha1.SecuritySSLTruststore
		mockSetup func(*iammocks.MockSSLTruststoreClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ *security.SSLCertificate) error {
					return nil
				}
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
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
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ *security.SSLCertificate) error {
					return errors.New("add error")
				}
			},
			wantErr: true,
		},
		{
			name: "CreateCantFindAfterAdd",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ *security.SSLCertificate) error {
					return nil
				}
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{}, nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockSSLTruststoreClient()
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
		cr        *iamv1alpha1.SecuritySSLTruststore
		mockSetup func(*iammocks.MockSSLTruststoreClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "old-cert-id")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
				mc.AddCertificateFn = func(_ *security.SSLCertificate) error {
					return nil
				}
				mc.ListCertificatesFn = func() ([]security.SSLCertificate, error) {
					return []security.SSLCertificate{
						{Id: "new-cert-id", Pem: testPem},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateAddError",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "old-cert-id")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
				mc.AddCertificateFn = func(_ *security.SSLCertificate) error {
					return errors.New("add error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockSSLTruststoreClient()
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
		cr        *iamv1alpha1.SecuritySSLTruststore
		mockSetup func(*iammocks.MockSSLTruststoreClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name:    "DeleteNoExternalName",
			cr:      newTruststoreCR(testPem),
			wantErr: false,
		},
		{
			name: "DeleteAPIError",
			cr: func() *iamv1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockSSLTruststoreClient()
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

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockSSLTruststoreClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_WrongType tests Connect with wrong resource type.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotTruststore {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotTruststore)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTruststoreCR(testPem)
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests ProviderConfig get failure.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTruststoreCR(testPem)
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}
