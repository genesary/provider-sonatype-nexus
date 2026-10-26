package iqserverconfiguration

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
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

// TestResolveCredentials_CrossNamespaceSecretRejected verifies that an IQ
// Server configuration cannot read its credentials from another namespace.
func TestResolveCredentials_CrossNamespaceSecretRejected(t *testing.T) {
	t.Parallel()

	tests := map[string]func(cr *instancev1alpha1.IQServerConfiguration){
		"Username": func(cr *instancev1alpha1.IQServerConfiguration) {
			cr.Spec.ForProvider.UsernameSecretRef = newVictimSelector("username")
		},
		"Password": func(cr *instancev1alpha1.IQServerConfiguration) {
			cr.Spec.ForProvider.PasswordSecretRef = newVictimSelector("password")
		},
	}

	for name, setRef := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cr := newTestIQServer(true)
			cr.Namespace = "tenant-a"
			setRef(cr)

			e := &external{kube: newVictimSecretKube()}

			username, password, err := e.resolveCredentials(context.Background(), cr)
			assertNamespaceMismatch(t, username+password, err)
		})
	}
}
