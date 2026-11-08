package httpclient

import (
	"context"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// TestCredentialRotation verifies that rotating the proxy password Secret is
// reported as drift and re-applied to Nexus, even though Nexus never returns
// the password.
func TestCredentialRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cr := newTestHTTPClient()
	cr.Spec.ForProvider.HTTPProxy.Authentication = &instancev1alpha1.HTTPClientProxyAuthentication{
		Username: "proxy-user",
		PasswordSecretRef: xpv2.LocalSecretKeySelector{
			LocalSecretReference: xpv2.LocalSecretReference{Name: "proxy-credentials"},
			Key:                  "password",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "proxy-credentials", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte("old-password")},
	}
	kube := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(cr, secret).WithStatusSubresource(cr).Build()

	var applied []string

	mc := instancemocks.NewMockHTTPSettingsClient()
	mc.GetFn = func() (*v395.HttpSettingsXo, error) {
		settings := newObservedSettings()
		settings.HttpProxy.Get().AuthInfo = v395.AuthSettingsXo{Enabled: true, Username: "proxy-user"}

		return settings, nil
	}
	mc.UpdateFn = func(settings v395.HttpSettingsXo) error {
		applied = append(applied, settings.HttpProxy.Get().AuthInfo.Password)

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

	// Nothing has been applied yet, so the password must be.
	assertUpToDate("before create", false)

	_, err := e.Create(ctx, cr)
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
