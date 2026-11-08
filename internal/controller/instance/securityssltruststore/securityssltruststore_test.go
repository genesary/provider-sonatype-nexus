package securityssltruststore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

// testPem is a self-signed test certificate.
const testPem = `-----BEGIN CERTIFICATE-----
MIIDETCCAfmgAwIBAgIULjaoxQ6pRnkQYrHMqHCOoa4I6wYwDQYJKoZIhvcNAQEL
BQAwGDEWMBQGA1UEAwwNY2xhdWRlLXRlc3QtYTAeFw0yNjExMDgwMzAyMjFaFw0y
NjEyMDgwMzAyMjFaMBgxFjAUBgNVBAMMDWNsYXVkZS10ZXN0LWEwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDSGh1igB4rw6maggGQQdXq4f8lqeD7RjoZ
QKqsa+JCZ/RIc4+atN3g1+JHA6QQXYUGr+d0DUVWrmeQFj7sSesqAC/9qwezbR2e
MmzoKqV07c7gw3Ac+EnV0QhDjp6Mvvfjykrg557/XaPKUIbaX9m/DbaOMg2vhP4r
DBmwWOiZm7Wp8Nf3lpZz9jbTbHEmGpA5SNY/Gr1TCMbeGEs8SmU7gBAer7mE/V6e
NK0SqWiPXatGWtNUfIWpfbvCgfPIsZwjEHCmV+iQx80jVZMII3NBB/L4NTlMU3AU
JxKslM/rL1cNmdB/F/sv8tNBErqkyhSs0ediGYQuWSJNHhktWuUhAgMBAAGjUzBR
MB0GA1UdDgQWBBTUCQ4lU4cnI6VYGMEfofgiDiXatDAfBgNVHSMEGDAWgBTUCQ4l
U4cnI6VYGMEfofgiDiXatDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUA
A4IBAQBIGoUHTikABdyG1tBhzq/8r2pw31avfBynq2ry+k4jQyojXGC3lhNufew8
9/+nF3yJeAHk6vGheotEMed7w7fuozqnQu5qOspKeXpD2m7FAHd7MFuc8XgYVUdK
daF9FDTxB1NQRfVNsfkpkWfsJlD5NcKYjKz9xyyL81KxY8g30KMNbfzpLx3Gor5G
vbAQC9aTRnkdDVWTBDDUHtyCmpA94hXPIihWi/P4cypNsgFiDUK8YPemmie0UVK2
VuZXvnBuK6wxqcy1Y1MK3N6OneCXqKlOZn2RWR4PllApmqbWGSxZmzRb4D6TR43j
qknoPzJD0I9euF9C+l9ggWuRMDF0
-----END CERTIFICATE-----`

// otherPem is a second self-signed test certificate.
const otherPem = `-----BEGIN CERTIFICATE-----
MIIDETCCAfmgAwIBAgIUHLdNumgmiC9AfWdZNwElqAB48M0wDQYJKoZIhvcNAQEL
BQAwGDEWMBQGA1UEAwwNY2xhdWRlLXRlc3QtYjAeFw0yNjExMDgwMzAyMjFaFw0y
NjEyMDgwMzAyMjFaMBgxFjAUBgNVBAMMDWNsYXVkZS10ZXN0LWIwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQCkwokhfFoygpbtlxP+ZVEBXcdWBokq9Nga
qV4DKWue/98pChnw51YYpkwGkytd6CfNxfdsZpffivDPknXonffgs0jFRdnIn2TX
GHJtPLD+94g/+cUBKhHzPV4fvLiZEgS8IssVpwjga961U3P/6x471Dl5pIMI66dW
pigX9YPkx0MKeiK6cdSsH7xsIH0GNWfipYcOzoRESwKlYDK5IuPAp3GWEc72nvlR
2XC/8Bp8vlVonbYM0W/lMqAxnqZFNA/aEk+ktIomTz8zkybHPe9FCnJ9yGwf0+MW
Y/4ECGqCvl9aVCn1ZbZFV5NqgVQeXj6a8IQJNBgR+/QaPkg5LxL5AgMBAAGjUzBR
MB0GA1UdDgQWBBSGe1WGI6u9bZ8vzqEBPj/hrruBvDAfBgNVHSMEGDAWgBSGe1WG
I6u9bZ8vzqEBPj/hrruBvDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUA
A4IBAQAZ6WQuqdbpmJ7WfP4W5oRp20ufG1TS438RwI/Xp0b8T3w1MjynTo7A6S3m
QA9N3yl9isTI4aJTtiS4IgxGIZREupbQtg9TULZ6AGhe71pt8JKtvJpqCdEu2p0s
vg1dOi0lC80CXEIibDqsJsuI2S2LfVMXwCADnfNl30cxf8a5vdpCiEOgifT6GXMV
Z8jsO23XJQ68JJrPAYpifUxSH3cdnaUukw52rLEL03i9PUgpQ5/UJkt1/1ahorFz
YrfSksyOoYkXE7lIz0Eh7sEe7T58JrFLvBO+UB7zDzSt42AUu5YSMUWhOXGMmPwl
DWbap+TJ02EnkWfdq+Mg1ujkJ990
-----END CERTIFICATE-----`

// testPemCRLF is testPem re-encoded with CRLF line endings and 76-column
// lines, a form Nexus accepts and returns re-encoded as testPem.
var testPemCRLF = rewrapPEM(testPem, 76, "\r\n")

// rewrapPEM re-encodes a single-block PEM with the given line width and
// line ending.
func rewrapPEM(pemText string, width int, eol string) string {
	lines := strings.Split(strings.TrimSpace(pemText), "\n")
	body := strings.Join(lines[1:len(lines)-1], "")

	var out strings.Builder

	out.WriteString(lines[0] + eol)

	for len(body) > width {
		out.WriteString(body[:width] + eol)
		body = body[width:]
	}

	out.WriteString(body + eol + lines[len(lines)-1] + eol)

	return out.String()
}

// newTruststoreCR creates a new SecuritySSLTruststore CR for testing.
func newTruststoreCR(pem string) *instancev1alpha1.SecuritySSLTruststore {
	return &instancev1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-cert",
			Annotations: map[string]string{},
		},
		Spec: instancev1alpha1.SecuritySSLTruststoreSpec{
			ForProvider: instancev1alpha1.SecuritySSLTruststoreParameters{
				Pem: pem,
			},
		},
	}
}

// newTestScheme registers iam and nexus v1alpha1 types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := instancev1alpha1.AddToScheme(s)
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
		cr           *instancev1alpha1.SecuritySSLTruststore
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
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{
						{
							Id:                new("cert-id-123"),
							Pem:               new(testPem),
							Fingerprint:       new("AB:CD:EF"),
							SubjectCommonName: new("testCA"),
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
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(otherPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{
						{Id: new("cert-id-123"), Pem: new(testPem)},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			// Nexus returns the certificate re-encoded; a spec with other line
			// endings and wrapping must not report drift forever.
			name: "ExistsUpToDateDespiteEncoding",
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPemCRLF)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{
						{Id: new("cert-id-123"), Pem: new(testPem + "\n")},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "CertNotFoundInList",
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-deleted")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{}, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "APIError",
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
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
		cr        *instancev1alpha1.SecuritySSLTruststore
		mockSetup func(*iammocks.MockSSLTruststoreClient)
		wantID    string
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return &v395.ApiCertificate{Id: new("new-cert-id"), Pem: new(testPem)}, nil
				}
			},
			wantID: "new-cert-id",
		},
		{
			// Nexus re-encodes the PEM, so the ID must come from its response
			// rather than from a text match against the spec.
			name: "CreateSuccessCRLF",
			cr:   newTruststoreCR(testPemCRLF),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return &v395.ApiCertificate{Id: new("new-cert-id"), Pem: new(testPem + "\n")}, nil
				}
			},
			wantID: "new-cert-id",
		},
		{
			name: "CreateAPIError",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, errors.New("add error")
				}
			},
			wantErr: true,
		},
		{
			name: "CreateAdoptsExistingDespiteEncoding",
			cr:   newTruststoreCR(testPemCRLF),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, instanceclient.ErrCertificateExists
				}
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{
						{Id: new("other-id"), Pem: new(otherPem)},
						{Id: new("existing-id"), Pem: new(testPem + "\n")},
					}, nil
				}
			},
			wantID: "existing-id",
		},
		{
			name: "CreateConflictButNotListed",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, instanceclient.ErrCertificateExists
				}
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{{Id: new("other-id"), Pem: new(otherPem)}}, nil
				}
			},
			wantErr: true,
		},
		{
			name: "CreateConflictListError",
			cr:   newTruststoreCR(testPem),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, instanceclient.ErrCertificateExists
				}
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return nil, errors.New("list error")
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
				if externalName != tt.wantID {
					t.Errorf("Create() external name = %q, want %q", externalName, tt.wantID)
				}
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	withExternalName := func(pemText, id string) *instancev1alpha1.SecuritySSLTruststore {
		cr := newTruststoreCR(pemText)
		meta.SetExternalName(cr, id)

		return cr
	}

	tests := []struct {
		name        string
		cr          *instancev1alpha1.SecuritySSLTruststore
		mockSetup   func(*iammocks.MockSSLTruststoreClient)
		wantID      string
		wantRemoved []string
		wantErr     bool
	}{
		{
			name: "UpdateSuccess",
			cr:   withExternalName(testPem, "old-cert-id"),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return &v395.ApiCertificate{Id: new("new-cert-id"), Pem: new(testPem)}, nil
				}
			},
			wantID:      "new-cert-id",
			wantRemoved: []string{"old-cert-id"},
		},
		{
			name: "UpdateOldAlreadyGone",
			cr:   withExternalName(testPem, "old-cert-id"),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nexus.ErrNotFound
				}
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return &v395.ApiCertificate{Id: new("new-cert-id"), Pem: new(testPem)}, nil
				}
			},
			wantID:      "new-cert-id",
			wantRemoved: []string{"old-cert-id"},
		},
		{
			name: "UpdateAdoptsNewAlreadyPresent",
			cr:   withExternalName(testPem, "old-cert-id"),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, instanceclient.ErrCertificateExists
				}
				mc.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
					return []v395.ApiCertificate{{Id: new("existing-id"), Pem: new(testPem)}}, nil
				}
			},
			wantID:      "existing-id",
			wantRemoved: []string{"old-cert-id"},
		},
		{
			name: "UpdateRemoveError",
			cr:   withExternalName(testPem, "old-cert-id"),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return errors.New("remove error")
				}
			},
			wantRemoved: []string{"old-cert-id"},
			wantErr:     true,
		},
		{
			name: "UpdateAddError",
			cr:   withExternalName(testPem, "old-cert-id"),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nil
				}
				mc.AddCertificateFn = func(_ string) (*v395.ApiCertificate, error) {
					return nil, errors.New("add error")
				}
			},
			wantRemoved: []string{"old-cert-id"},
			wantErr:     true,
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

			if strings.Join(mc.RemoveCertificateCalls, ",") != strings.Join(tt.wantRemoved, ",") {
				t.Errorf("Update() removed %v, want %v", mc.RemoveCertificateCalls, tt.wantRemoved)
			}

			if !tt.wantErr {
				externalName := meta.GetExternalName(tt.cr)
				if externalName != tt.wantID {
					t.Errorf("Update() external name = %q, want %q", externalName, tt.wantID)
				}
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *instancev1alpha1.SecuritySSLTruststore
		mockSetup func(*iammocks.MockSSLTruststoreClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
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
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
				cr := newTruststoreCR(testPem)
				meta.SetExternalName(cr, "cert-id-123")

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockSSLTruststoreClient) {
				mc.RemoveCertificateFn = func(_ string) error {
					return nexus.ErrNotFound
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
			cr: func() *instancev1alpha1.SecuritySSLTruststore {
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
