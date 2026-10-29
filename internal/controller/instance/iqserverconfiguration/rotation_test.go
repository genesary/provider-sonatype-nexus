package iqserverconfiguration

import (
	"context"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// TestCredentialRotation verifies that rotating the IQ Server credentials
// Secret is reported as drift and re-applied to Nexus, even though Nexus
// never returns the password.
func TestCredentialRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	scheme := newTestScheme(t)

	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	cr := newTestIQServer(true)
	cr.Namespace = "default"
	cr.Spec.ForProvider.UsernameSecretRef = &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: "iq-credentials"},
		Key:                  "username",
	}
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: "iq-credentials"},
		Key:                  "password",
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "iq-credentials", Namespace: "default"},
		Data:       map[string][]byte{"username": []byte("iq-user"), "password": []byte("old-password")},
	}
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, secret).WithStatusSubresource(cr).Build()

	var applied []string

	mc := instancemocks.NewMockIQServerClient()
	mc.GetFn = func() (*v395.IqConnectionXo, error) {
		return newObservedConfig(), nil
	}
	mc.UpdateFn = func(config v395.IqConnectionXo) error {
		applied = append(applied, ptr.Deref(config.Password, ""))

		return nil
	}

	e := &external{client: mc, kube: kube}

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

	assertUpToDate("after rotation", false)

	_, err = e.Update(ctx, cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	assertUpToDate("after update", true)

	if len(applied) != 2 || applied[0] != "old-password" || applied[1] != "rotated-password" {
		t.Fatalf("applied passwords = %v, want [old-password rotated-password]", applied)
	}
}
