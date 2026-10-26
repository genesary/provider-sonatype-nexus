package license

import (
	"context"
	"net/http"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// newVictimSecretKube returns a fake client holding a Secret only in a
// namespace other than the managed resource's own.
func newVictimSecretKube() kubeclient.Client {
	return fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "nexus-admin", Namespace: "victim-ns"},
		Data:       map[string][]byte{"username": []byte("admin"), "password": []byte("victim-secret")},
	}).Build()
}

// newVictimSelector returns a selector pointing at key in the victim Secret.
func newVictimSelector(key string) *xpv2.SecretKeySelector {
	return &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{Name: "nexus-admin", Namespace: "victim-ns"},
		Key:             key,
	}
}

// assertNamespaceMismatch fails unless err rejects a cross-namespace selector
// and no secret value was resolved.
func assertNamespaceMismatch(t *testing.T, value string, err error) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), nexus.ErrSecretNamespaceMismatch) {
		t.Fatalf("error = %v, want %q", err, nexus.ErrSecretNamespaceMismatch)
	}

	if value != "" {
		t.Fatalf("cross-namespace secret value %q was resolved", value)
	}
}

// TestFetchDesiredLicense_CrossNamespaceSecretRejected verifies that a License
// cannot read its license file from another namespace.
func TestFetchDesiredLicense_CrossNamespaceSecretRejected(t *testing.T) {
	t.Parallel()

	cr := newTestLicense("nexus-license", "nexus-admin", "victim-ns", "password")
	cr.Namespace = "tenant-a"

	e := &external{kube: newVictimSecretKube()}

	data, err := e.fetchDesiredLicense(context.Background(), cr)
	assertNamespaceMismatch(t, string(data), err)
}

// TestApplyEndpointCredentials_CrossNamespaceSecretRejected verifies that the
// license endpoint credentials cannot come from another namespace, so they
// are never attached to a request to the endpoint URL.
func TestApplyEndpointCredentials_CrossNamespaceSecretRejected(t *testing.T) {
	t.Parallel()

	tests := map[string]*iamv1alpha1.LicenseEndpointCredentials{
		"Basic": {
			UsernameSecretRef: newVictimSelector("username"),
			PasswordSecretRef: newVictimSelector("password"),
		},
		"Bearer": {
			PasswordSecretRef: newVictimSelector("password"),
		},
	}

	for name, creds := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://license.example.com", http.NoBody)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}

			e := &external{kube: newVictimSecretKube()}

			err = e.applyEndpointCredentials(context.Background(), "tenant-a", req, creds)
			assertNamespaceMismatch(t, req.Header.Get("Authorization"), err)
		})
	}
}
