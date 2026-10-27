package repository

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// TestCredentialRotation verifies that changes to the credentials Nexus
// redacts on read - the proxy password Secret, the proxy authentication and
// the signing keys - are reported as drift even though the repository Nexus
// returns still matches.
func TestCredentialRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	scheme := runtime.NewScheme()

	err := repositoryv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(content) failed: %v", err)
	}

	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	cr := newTestRepo("apt-proxy", "apt", repoTypeProxy)
	cr.Namespace = "default"
	cr.Spec.ForProvider.AptSigning = &repositoryv1alpha1.AptSigningConfig{Keypair: "old-key"}
	cr.Spec.ForProvider.HTTPClient = &repositoryv1alpha1.HTTPClientConfig{
		Authentication: &repositoryv1alpha1.HTTPClientAuthentication{
			Type:     new("username"),
			Username: new("upstream"),
			PasswordSecretRef: &xpv2.LocalSecretKeySelector{
				LocalSecretReference: xpv2.LocalSecretReference{Name: "upstream"},
				Key:                  "password",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "upstream", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte("old-password")},
	}
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, secret).WithStatusSubresource(cr).Build()

	e := newTestExternal(&mockHandler{exists: true, upToDate: true})
	e.kube = kube

	assertUpToDate := func(step string, want bool) {
		t.Helper()

		obs, observeErr := e.Observe(ctx, cr)
		if observeErr != nil {
			t.Fatalf("%s: Observe() unexpected error: %v", step, observeErr)
		}

		if obs.ResourceUpToDate != want {
			t.Fatalf("%s: ResourceUpToDate = %v, want %v", step, obs.ResourceUpToDate, want)
		}
	}

	update := func(step string) {
		t.Helper()

		_, updateErr := e.Update(ctx, cr)
		if updateErr != nil {
			t.Fatalf("%s: Update() unexpected error: %v", step, updateErr)
		}

		assertUpToDate(step, true)
	}

	_, err = e.Create(ctx, cr)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	assertUpToDate("after create", true)

	secret.Data["password"] = []byte("rotated-password")

	err = kube.Update(ctx, secret)
	if err != nil {
		t.Fatalf("rotating secret: %v", err)
	}

	assertUpToDate("after password rotation", false)
	update("after password update")

	cr.Spec.ForProvider.HTTPClient.Authentication.Username = new("other-upstream")
	assertUpToDate("after username change", false)
	update("after username update")

	cr.Spec.ForProvider.AptSigning.Keypair = "new-key"
	assertUpToDate("after signing key change", false)
	update("after signing key update")
}
