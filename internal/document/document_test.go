package document

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// testNamespace is the namespace of the managed resources under test.
	testNamespace = "team-a"
	// testKey is the connection Secret key documents are cached under.
	testKey = "document"
	// testCache is the name of the connection Secret.
	testCache = "document-cache"
	// testContent is the content of the document.
	testContent = "document content"
	// unreachableURL is an endpoint that refuses connections.
	unreachableURL = "http://127.0.0.1:1/document"
)

// newManaged returns a managed resource that caches documents in testCache.
func newManaged() *instancev1alpha1.License {
	return &instancev1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNamespace},
		Spec: instancev1alpha1.LicenseSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv2.LocalSecretReference{Name: testCache},
			},
		},
	}
}

// newSecret returns a Secret of testNamespace.
func newSecret(name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}, Data: data}
}

// newKube returns a fake client holding objs.
func newKube(objs ...client.Object) client.Client {
	return crfake.NewClientBuilder().WithObjects(objs...).Build()
}

// selector returns a selector of the given Secret key.
func selector(name, key string) *xpv2.LocalSecretKeySelector {
	return &xpv2.LocalSecretKeySelector{LocalSecretReference: xpv2.LocalSecretReference{Name: name}, Key: key}
}

// fromURL returns the Source that fetches from url.
func fromURL(url string) Source {
	return Source{From: &nexusv1alpha1.DocumentSource{EndpointURL: &url}, Key: testKey}
}

// cacheOf returns the connection Secret data that caches testContent read from
// source.
func cacheOf(source Source) map[string][]byte {
	return source.document([]byte(testContent)).ConnectionDetails
}

// newEndpoint returns an endpoint serving body with the given status.
func newEndpoint(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// TestRead_Secret checks that a document is read from the Secret its source
// references, in the namespace of the managed resource, and that the Secret
// takes precedence over an endpoint.
func TestRead_Secret(t *testing.T) {
	t.Parallel()

	kube := newKube(newSecret("source", map[string][]byte{"key": []byte(testContent)}))

	source := fromURL(unreachableURL)
	source.From.SecretRef = selector("source", "key")

	got, err := source.Read(context.Background(), kube, newManaged())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(got.Data) != testContent {
		t.Errorf("Read() = %q, want %q", got.Data, testContent)
	}

	other := newManaged()
	other.Namespace = "team-b"

	_, err = source.Read(context.Background(), kube, other)
	if err == nil {
		t.Error("Read() from another namespace error = nil, want an error")
	}
}

// TestRead_NoSource checks that a source naming nothing is an error.
func TestRead_NoSource(t *testing.T) {
	t.Parallel()

	_, err := Source{Key: testKey}.Read(context.Background(), newKube(), newManaged())
	if err == nil || !strings.Contains(err.Error(), ErrNoSource) {
		t.Fatalf("Read() error = %v, want %q", err, ErrNoSource)
	}
}

// TestRead_Endpoint checks that a document is fetched from its endpoint.
func TestRead_Endpoint(t *testing.T) {
	t.Parallel()

	server := newEndpoint(t, http.StatusOK, testContent)

	got, err := fromURL(server.URL).Read(context.Background(), newKube(), newManaged())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(got.Data) != testContent || string(got.ConnectionDetails[testKey]) != testContent {
		t.Errorf("Read() = %q caching %q, want %q", got.Data, got.ConnectionDetails[testKey], testContent)
	}
}

// TestRead_Cache checks when an unavailable endpoint falls back on the
// document cached in the connection Secret.
func TestRead_Cache(t *testing.T) {
	t.Parallel()

	source := fromURL(unreachableURL)
	fromSecret := Source{From: &nexusv1alpha1.DocumentSource{SecretRef: selector("source", "key")}, Key: testKey}

	tests := map[string]struct {
		cache   map[string][]byte
		wantErr bool
	}{
		"SameEndpoint":      {cache: cacheOf(source)},
		"UnrecordedByOlder": {cache: map[string][]byte{testKey: []byte(testContent)}},
		"OtherEndpoint":     {cache: cacheOf(fromURL(unreachableURL + "/other")), wantErr: true},
		"ReadFromSecret":    {cache: cacheOf(fromSecret), wantErr: true},
		"NotCached":         {cache: map[string][]byte{}, wantErr: true},
		"NoSecret":          {wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := newKube()
			if tc.cache != nil {
				kube = newKube(newSecret(testCache, tc.cache))
			}

			got, err := source.Read(context.Background(), kube, newManaged())
			if (err != nil) != tc.wantErr {
				t.Fatalf("Read() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				if !strings.Contains(err.Error(), "endpoint request failed") {
					t.Errorf("Read() error = %v, want the endpoint error", err)
				}

				return
			}

			if string(got.Data) != testContent {
				t.Errorf("Read() = %q, want %q", got.Data, testContent)
			}

			// The fallback records an endpoint only where the cache already did.
			_, recorded := got.ConnectionDetails[endpointKey]
			if _, wasRecorded := tc.cache[endpointKey]; recorded != wasRecorded {
				t.Errorf("Read() records the endpoint = %v, want %v", recorded, wasRecorded)
			}
		})
	}
}

// TestRead_NoConnectionSecret checks that an unavailable endpoint is an error
// for a managed resource without a connection Secret.
func TestRead_NoConnectionSecret(t *testing.T) {
	t.Parallel()

	mg := newManaged()
	mg.Spec.WriteConnectionSecretToReference = nil

	_, err := fromURL(unreachableURL).Read(context.Background(), newKube(), mg)
	if err == nil {
		t.Fatal("Read() error = nil, want an error")
	}
}

// TestRead_ConnectionDetailsHideURL checks that the connection details do not
// hold the endpoint URL, which can carry secrets.
func TestRead_ConnectionDetailsHideURL(t *testing.T) {
	t.Parallel()

	for key, value := range cacheOf(fromURL("https://example.com/?X-Amz-Signature=secret")) {
		if strings.Contains(string(value), "secret") {
			t.Errorf("ConnectionDetails[%q] = %q reports the URL", key, value)
		}
	}
}

// TestFetch_Credentials checks the Authorization header sent for each
// combination of credentials.
func TestFetch_Credentials(t *testing.T) {
	t.Parallel()

	kube := newKube(newSecret("creds", map[string][]byte{"user": []byte("alice"), "pass": []byte("pw")}))

	tests := map[string]struct {
		creds   *nexusv1alpha1.EndpointCredentials
		want    string
		wantErr bool
	}{
		"None":  {},
		"Empty": {creds: &nexusv1alpha1.EndpointCredentials{}},
		"Basic": {
			creds: &nexusv1alpha1.EndpointCredentials{UsernameSecretRef: selector("creds", "user"), PasswordSecretRef: selector("creds", "pass")},
			want:  "Basic YWxpY2U6cHc=",
		},
		"Bearer": {
			creds: &nexusv1alpha1.EndpointCredentials{PasswordSecretRef: selector("creds", "pass")},
			want:  "Bearer pw",
		},
		"MissingUsername": {
			creds:   &nexusv1alpha1.EndpointCredentials{UsernameSecretRef: selector("creds", "nope"), PasswordSecretRef: selector("creds", "pass")},
			wantErr: true,
		},
		"MissingPassword": {
			creds:   &nexusv1alpha1.EndpointCredentials{UsernameSecretRef: selector("creds", "user"), PasswordSecretRef: selector("creds", "nope")},
			wantErr: true,
		},
		"MissingToken": {
			creds:   &nexusv1alpha1.EndpointCredentials{PasswordSecretRef: selector("nope", "pass")},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got string

			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Authorization")
			}))
			defer server.Close()

			_, err := fetch(context.Background(), kube, testNamespace, server.URL, tc.creds)
			if (err != nil) != tc.wantErr {
				t.Fatalf("fetch() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("fetch() sent Authorization %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFetch_NonOKStatus checks that a response other than 200 is an error.
func TestFetch_NonOKStatus(t *testing.T) {
	t.Parallel()

	server := newEndpoint(t, http.StatusForbidden, testContent)

	_, err := fetch(context.Background(), newKube(), testNamespace, server.URL, nil)
	if err == nil {
		t.Fatal("fetch() error = nil, want an error on 403")
	}
}

// TestFetch_OversizedResponse checks that an endpoint streaming more than
// maxBytes is rejected instead of buffered.
func TestFetch_OversizedResponse(t *testing.T) {
	t.Parallel()

	chunk := bytes.Repeat([]byte("A"), 64<<10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for range 4 * maxBytes / len(chunk) {
			if r.Context().Err() != nil {
				return
			}

			_, _ = w.Write(chunk)
		}
	}))
	defer server.Close()

	data, err := fetch(context.Background(), newKube(), testNamespace, server.URL, nil)
	if err == nil || !strings.Contains(err.Error(), nexus.ErrResponseTooLarge) {
		t.Fatalf("fetch() error = %v, want %q", err, nexus.ErrResponseTooLarge)
	}

	if data != nil {
		t.Fatalf("fetch() returned %d bytes alongside an error", len(data))
	}
}

// TestFetch_RedirectFollowed checks that a redirect from the endpoint is
// followed and the target's body is returned.
func TestFetch_RedirectFollowed(t *testing.T) {
	t.Parallel()

	target := newEndpoint(t, http.StatusOK, testContent)

	server := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer server.Close()

	got, err := fetch(context.Background(), newKube(), testNamespace, server.URL, nil)
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}

	if string(got) != testContent {
		t.Fatalf("fetch() = %q, want %q", got, testContent)
	}
}

// TestEndpointClientTimeout checks that the endpoint client bounds every
// request.
func TestEndpointClientTimeout(t *testing.T) {
	t.Parallel()

	if endpointClient.Timeout == 0 {
		t.Fatal("endpointClient has no timeout")
	}
}

// TestFetch_RedactsTheURL checks that a failed fetch does not report the query
// of its URL, which for a pre-signed URL holds its signature.
func TestFetch_RedactsTheURL(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{
		"http://127.0.0.1:1/license.lic?X-Amz-Signature=secret",
		"http://127.0.0.1:1/%zz?X-Amz-Signature=secret",
	} {
		_, err := fetch(context.Background(), newKube(), testNamespace, endpoint, nil)
		if err == nil {
			t.Fatalf("fetch(%q) error = nil, want an error", endpoint)
		}

		if strings.Contains(err.Error(), "secret") {
			t.Errorf("fetch(%q) error = %q reports the URL query", endpoint, err.Error())
		}
	}
}
