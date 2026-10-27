package nexus

import (
	"context"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

const (
	// tenantNamespace is the namespace of the referencing objects.
	tenantNamespace = "tenant-a"
	// victimNamespace is a namespace the tenant must not read Secrets from.
	victimNamespace = "victim-ns"
)

// testManaged is the minimal managed resource GetCredentials needs.
type testManaged struct {
	namespace string
	ref       *xpv2.ProviderConfigReference
}

// GetNamespace returns the namespace of the test managed resource.
func (m testManaged) GetNamespace() string { return m.namespace }

// GetProviderConfigReference returns the test providerConfigRef.
func (m testManaged) GetProviderConfigReference() *xpv2.ProviderConfigReference { return m.ref }

// newTestScheme registers the provider config and core types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := v1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(v1alpha1): %v", err)
	}

	err = corev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(core): %v", err)
	}

	return s
}

// newCredentialsSecret returns a Secret holding a username and password.
func newCredentialsSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "nexus-creds", Namespace: namespace},
		Data: map[string][]byte{
			"username": []byte("admin-" + namespace),
			"password": []byte("secret-" + namespace),
		},
	}
}

// newSelector selects key in the nexus-creds Secret of namespace.
func newSelector(namespace, key string) *xpv2.SecretKeySelector {
	return &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{Name: "nexus-creds", Namespace: namespace},
		Key:             key,
	}
}

// newLocalSelector selects key in the nexus-creds Secret of the owner's
// namespace.
func newLocalSelector(key string) *xpv2.LocalSecretKeySelector {
	return &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: "nexus-creds"},
		Key:                  key,
	}
}

// newSpec returns a namespaced spec whose credentials are local references.
func newSpec() v1alpha1.ProviderConfigSpec {
	return v1alpha1.ProviderConfigSpec{
		URL: "https://nexus.example.com",
		Username: v1alpha1.ProviderCredentials{
			Source:    xpv2.CredentialsSourceSecret,
			SecretRef: newLocalSelector("username"),
		},
		Password: v1alpha1.ProviderCredentials{
			Source:    xpv2.CredentialsSourceSecret,
			SecretRef: newLocalSelector("password"),
		},
	}
}

// newClusterSpec returns a cluster spec whose credentials live in
// secretNamespace.
func newClusterSpec(secretNamespace string) v1alpha1.ClusterProviderConfigSpec {
	return v1alpha1.ClusterProviderConfigSpec{
		URL: "https://nexus.example.com",
		Username: v1alpha1.ClusterProviderCredentials{
			Source:                    xpv2.CredentialsSourceSecret,
			CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: newSelector(secretNamespace, "username")},
		},
		Password: v1alpha1.ClusterProviderCredentials{
			Source:                    xpv2.CredentialsSourceSecret,
			CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: newSelector(secretNamespace, "password")},
		},
	}
}

// newKube returns a fake client holding credential Secrets in both test
// namespaces plus objs.
func newKube(t *testing.T, objs ...kubeclient.Object) kubeclient.Client {
	t.Helper()

	objs = append(objs, newCredentialsSecret(tenantNamespace), newCredentialsSecret(victimNamespace))

	return fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(objs...).Build()
}

// TestGetSecretValueUsesOwnerNamespace verifies that a local selector is
// always resolved in the namespace of the object that declares it.
func TestGetSecretValueUsesOwnerNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		namespace string
		selector  *xpv2.LocalSecretKeySelector
		want      string
		wantErr   string
	}{
		{name: "OwnerNamespace", namespace: tenantNamespace, selector: newLocalSelector("password"), want: "secret-" + tenantNamespace},
		{name: "OtherNamespace", namespace: victimNamespace, selector: newLocalSelector("password"), want: "secret-" + victimNamespace},
		{name: "NilSelector", namespace: tenantNamespace, wantErr: "secretKeySelector is nil"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kube := newKube(t)

			got, err := GetSecretValue(context.Background(), kube, tc.namespace, tc.selector)
			assertSecretResult(t, "GetSecretValue", got, err, tc.want, tc.wantErr)

			gotBytes, err := GetSecretBytes(context.Background(), kube, tc.namespace, tc.selector)
			assertSecretResult(t, "GetSecretBytes", string(gotBytes), err, tc.want, tc.wantErr)
		})
	}
}

// assertSecretResult checks a secret lookup against want or wantErr.
func assertSecretResult(t *testing.T, fn, got string, err error, want, wantErr string) {
	t.Helper()

	if wantErr != "" {
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("%s() error = %v, want error containing %q", fn, err, wantErr)
		}

		if got != "" {
			t.Fatalf("%s() returned %q alongside an error", fn, got)
		}

		return
	}

	if err != nil {
		t.Fatalf("%s() unexpected error: %v", fn, err)
	}

	if got != want {
		t.Fatalf("%s() = %q, want %q", fn, got, want)
	}
}

// TestGetCredentialsNamespaceBinding verifies that a namespaced ProviderConfig
// reads Secrets from its own namespace, while a ClusterProviderConfig keeps its
// explicit selector namespace.
func TestGetCredentialsNamespaceBinding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		obj          kubeclient.Object
		ref          xpv2.ProviderConfigReference
		wantPassword string
	}{
		{
			name: "ProviderConfig",
			obj: &v1alpha1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: tenantNamespace},
				Spec:       newSpec(),
			},
			ref:          xpv2.ProviderConfigReference{Kind: "ProviderConfig", Name: "pc"},
			wantPassword: "secret-" + tenantNamespace,
		},
		{
			name: "ClusterProviderConfigAnyNamespace",
			obj: &v1alpha1.ClusterProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec:       newClusterSpec(victimNamespace),
			},
			ref:          xpv2.ProviderConfigReference{Kind: clusterProviderConfigKind, Name: "default"},
			wantPassword: "secret-" + victimNamespace,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mg := testManaged{namespace: tenantNamespace, ref: &tc.ref}

			creds, err := GetCredentials(context.Background(), newKube(t, tc.obj), mg)
			assertSecretResult(t, "GetCredentials", creds.Password, err, tc.wantPassword, "")
		})
	}
}
