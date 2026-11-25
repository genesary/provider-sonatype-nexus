package saml

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

const (
	// testNamespace is the namespace of the SAML under test.
	testNamespace = "default"
	// testSecret is the name of the Secret holding the metadata.
	testSecret = "idp"
	// testMetadata is minimal SAML identity provider metadata.
	testMetadata = `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="a"/>`
	// rotatedMetadata is the metadata the identity provider replaces
	// testMetadata with.
	rotatedMetadata = `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="b"/>`
)

// newSAMLFrom returns a SAML CR reading its metadata from source.
func newSAMLFrom(source nexusv1alpha1.DocumentSource) *iamv1alpha1.SAML {
	cr := newTestSAML()
	cr.Namespace = testNamespace
	cr.Spec.ForProvider.IdpMetadata = nil
	cr.Spec.ForProvider.IdpMetadataFrom = &source

	return cr
}

// fromSecret is the source reading the given key of testSecret.
func fromSecret(key string) nexusv1alpha1.DocumentSource {
	return nexusv1alpha1.DocumentSource{SecretRef: &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: testSecret},
		Key:                  key,
	}}
}

// newExternal returns an external reading Secrets holding data as testSecret,
// and whose Nexus holds applied as metadata.
func newExternal(data map[string][]byte, applied string) (*external, *iammocks.MockSAMLClient) {
	mock := iammocks.NewMockSAMLClient()
	mock.ApplyFn = func(v395.SamlConfigurationXO) error { return nil }
	mock.ReadFn = func() (*v395.SamlConfigurationXO, error) {
		return &v395.SamlConfigurationXO{
			IdpMetadata:       applied,
			EntityId:          new("nexus-entity"),
			UsernameAttribute: "username",
		}, nil
	}

	kube := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace},
		Data:       data,
	}).Build()

	return &external{client: mock, kube: kube}, mock
}

// TestObserve_MetadataFrom checks that Observe compares what Nexus holds with
// the metadata of the source, so that a rotated Secret or document is drift.
func TestObserve_MetadataFrom(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testMetadata))
	}))
	t.Cleanup(server.Close)

	tests := map[string]struct {
		source       nexusv1alpha1.DocumentSource
		applied      string
		wantUpToDate bool
		wantErr      bool
	}{
		"SecretUpToDate":   {source: fromSecret("metadata"), applied: testMetadata, wantUpToDate: true},
		"SecretRotated":    {source: fromSecret("metadata"), applied: rotatedMetadata},
		"SecretMissingKey": {source: fromSecret("nope"), applied: testMetadata, wantErr: true},
		"NotMetadata":      {source: fromSecret("privateKey"), applied: testMetadata, wantErr: true},
		"URLUpToDate":      {source: nexusv1alpha1.DocumentSource{EndpointURL: &server.URL}, applied: testMetadata, wantUpToDate: true},
		"URLRotated":       {source: nexusv1alpha1.DocumentSource{EndpointURL: &server.URL}, applied: rotatedMetadata},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ext, _ := newExternal(map[string][]byte{
				"metadata":   []byte(testMetadata + "\n"),
				"privateKey": []byte("-----BEGIN PRIVATE KEY-----"),
			}, tc.applied)

			obs, err := ext.Observe(context.Background(), newSAMLFrom(tc.source))
			if (err != nil) != tc.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if !obs.ResourceExists || obs.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("Observe() = %+v, want existing with ResourceUpToDate %v", obs, tc.wantUpToDate)
			}

			// Metadata Nexus has not taken yet is not cached: it could replace
			// the applied one as the fallback with metadata Nexus refuses.
			if cached := string(obs.ConnectionDetails[idpMetadataKey]); tc.wantUpToDate && cached != testMetadata || !tc.wantUpToDate && obs.ConnectionDetails != nil {
				t.Errorf("Observe() ConnectionDetails = %v, want the metadata cached only when up to date", obs.ConnectionDetails)
			}
		})
	}
}

// TestObserve_DeletedWithoutSource checks that a missing source does not block
// the deletion of a SAML.
func TestObserve_DeletedWithoutSource(t *testing.T) {
	t.Parallel()

	ext, _ := newExternal(nil, testMetadata)

	cr := newSAMLFrom(fromSecret("nope"))
	cr.DeletionTimestamp = new(metav1.Now())

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true so that Delete is called")
	}
}

// TestApply_MetadataFrom checks that Create and Update apply the metadata of
// the source and cache it, and apply nothing when it cannot be read.
func TestApply_MetadataFrom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	data := map[string][]byte{"metadata": []byte(testMetadata + "\n")}

	calls := map[string]func(*external, *iamv1alpha1.SAML) (managed.ExternalUpdate, error){
		"Create": func(ext *external, cr *iamv1alpha1.SAML) (managed.ExternalUpdate, error) {
			got, err := ext.Create(ctx, cr)

			return managed.ExternalUpdate{ConnectionDetails: got.ConnectionDetails}, err
		},
		"Update": func(ext *external, cr *iamv1alpha1.SAML) (managed.ExternalUpdate, error) {
			return ext.Update(ctx, cr)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ext, mock := newExternal(data, rotatedMetadata)

			_, err := call(ext, newSAMLFrom(fromSecret("nope")))
			if err == nil || len(mock.ApplyCalls) != 0 {
				t.Fatalf("without source: error = %v after applying %v, want an error and nothing applied", err, mock.ApplyCalls)
			}

			got, err := call(ext, newSAMLFrom(fromSecret("metadata")))
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			if len(mock.ApplyCalls) != 1 || mock.ApplyCalls[0].IdpMetadata != testMetadata {
				t.Errorf("applied %+v, want the metadata of the Secret", mock.ApplyCalls)
			}

			if string(got.ConnectionDetails[idpMetadataKey]) != testMetadata {
				t.Errorf("ConnectionDetails = %v, want the metadata cached", got.ConnectionDetails)
			}
		})
	}
}
