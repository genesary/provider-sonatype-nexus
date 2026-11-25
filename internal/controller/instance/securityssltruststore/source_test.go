package securityssltruststore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

const (
	// testNamespace is the namespace of the certificate under test.
	testNamespace = "default"
	// testSecret is the name of the Secret holding the certificate.
	testSecret = "ca"
	// oldID is the ID of the certificate Nexus holds.
	oldID = "old-id"
)

// privateKeyPem is a PEM block that must never be sent to Nexus.
const privateKeyPem = "-----BEGIN PRIVATE KEY-----\nc2VjcmV0\n-----END PRIVATE KEY-----\n"

// newTruststoreFrom returns a CR reading its certificate from source, which
// was added to Nexus as oldID.
func newTruststoreFrom(source nexusv1alpha1.DocumentSource) *instancev1alpha1.SecuritySSLTruststore {
	cr := newTruststoreCR("")
	cr.Namespace = testNamespace
	cr.Spec.ForProvider.Pem = nil
	cr.Spec.ForProvider.PemFrom = &source
	meta.SetExternalName(cr, oldID)

	return cr
}

// fromSecret is the source reading the given key of testSecret.
func fromSecret(key string) nexusv1alpha1.DocumentSource {
	return nexusv1alpha1.DocumentSource{SecretRef: &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: testSecret},
		Key:                  key,
	}}
}

// newExternal returns an external reading testSecret, whose Nexus holds
// testPem as oldID and knows cr.
func newExternal(t *testing.T, cr *instancev1alpha1.SecuritySSLTruststore) (*external, *iammocks.MockSSLTruststoreClient) {
	t.Helper()

	mock := iammocks.NewMockSSLTruststoreClient()
	mock.RemoveCertificateFn = func(string) error { return nil }
	mock.AddCertificateFn = func(string) (*v395.ApiCertificate, error) {
		return &v395.ApiCertificate{Id: new("new-id")}, nil
	}
	mock.ListCertificatesFn = func() ([]v395.ApiCertificate, error) {
		return []v395.ApiCertificate{{Id: new(oldID), Pem: new(testPem)}}, nil
	}

	scheme := newTestScheme(t)

	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(core) failed: %v", err)
	}

	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace},
		Data: map[string][]byte{
			"same":    []byte(testPemCRLF),
			"bundle":  []byte(testPem + "\n" + otherPem),
			"rotated": []byte(privateKeyPem + otherPem),
			"key":     []byte(privateKeyPem),
		},
	}).Build()

	return &external{client: mock, kube: kube}, mock
}

// TestObserve_PemFrom checks that Observe compares the certificate Nexus holds
// with the one of the source, so that a rotated Secret or document is drift.
func TestObserve_PemFrom(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testPemCRLF))
	}))
	t.Cleanup(server.Close)

	tests := map[string]struct {
		source       nexusv1alpha1.DocumentSource
		wantUpToDate bool
		wantErr      bool
	}{
		"SecretUpToDate":   {source: fromSecret("same"), wantUpToDate: true},
		"FirstOfBundle":    {source: fromSecret("bundle"), wantUpToDate: true},
		"SecretRotated":    {source: fromSecret("rotated")},
		"SecretMissingKey": {source: fromSecret("nope"), wantErr: true},
		"NotCertificate":   {source: fromSecret("key"), wantErr: true},
		"URL":              {source: nexusv1alpha1.DocumentSource{EndpointURL: &server.URL}, wantUpToDate: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cr := newTruststoreFrom(tc.source)
			ext, _ := newExternal(t, cr)

			obs, err := ext.Observe(context.Background(), cr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if !obs.ResourceExists || obs.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("Observe() = %+v, want existing with ResourceUpToDate %v", obs, tc.wantUpToDate)
			}

			// A certificate Nexus has not added yet is not cached: it could
			// replace the trusted one as the fallback with one Nexus refuses.
			if cached := len(obs.ConnectionDetails[pemKey]) != 0; cached != tc.wantUpToDate {
				t.Errorf("Observe() caches the certificate = %v, want %v", cached, tc.wantUpToDate)
			}
		})
	}
}

// TestObserve_DeletedWithoutSource checks that a missing source does not block
// the deletion of a certificate.
func TestObserve_DeletedWithoutSource(t *testing.T) {
	t.Parallel()

	cr := newTruststoreFrom(fromSecret("nope"))
	ext, _ := newExternal(t, cr)

	cr.DeletionTimestamp = new(metav1.Now())

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true so that Delete is called")
	}
}

// TestCreate_PemFrom checks that Create only sends the first certificate of
// the source, and caches it.
func TestCreate_PemFrom(t *testing.T) {
	t.Parallel()

	cr := newTruststoreFrom(fromSecret("rotated"))
	ext, mock := newExternal(t, cr)

	got, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(mock.AddCertificateCalls) != 1 || mock.AddCertificateCalls[0] != otherPem+"\n" {
		t.Errorf("Create() sent %q, want only the certificate", mock.AddCertificateCalls)
	}

	if string(got.ConnectionDetails[pemKey]) != otherPem+"\n" {
		t.Errorf("Create() ConnectionDetails = %q, want the certificate cached", got.ConnectionDetails[pemKey])
	}
}

// TestUpdate_PemFrom checks that Update replaces the certificate with the one
// of the source, and leaves it alone when the source cannot be read.
func TestUpdate_PemFrom(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key     string
		wantErr bool
	}{
		"Rotated":        {key: "rotated"},
		"MissingKey":     {key: "nope", wantErr: true},
		"NotCertificate": {key: "key", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cr := newTruststoreFrom(fromSecret(tc.key))
			ext, mock := newExternal(t, cr)

			_, err := ext.Update(context.Background(), cr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Update() error = %v, wantErr %v", err, tc.wantErr)
			}

			wantCalls := 1
			if tc.wantErr {
				wantCalls = 0
			}

			if len(mock.AddCertificateCalls) != wantCalls || len(mock.RemoveCertificateCalls) != wantCalls {
				t.Errorf("Update() added %d and removed %d certificates, want %d of each",
					len(mock.AddCertificateCalls), len(mock.RemoveCertificateCalls), wantCalls)
			}
		})
	}
}

// TestUpdate_KeepsObservedStatus checks that recording the external name does
// not overwrite the status Observe generated.
func TestUpdate_KeepsObservedStatus(t *testing.T) {
	t.Parallel()

	cr := newTruststoreFrom(fromSecret("rotated"))
	ext, _ := newExternal(t, cr)

	cr.Status.AtProvider.Pem = testPem

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if cr.Status.AtProvider.Pem != testPem {
		t.Errorf("Update() left status %+v, want the observed one", cr.Status.AtProvider)
	}
}
