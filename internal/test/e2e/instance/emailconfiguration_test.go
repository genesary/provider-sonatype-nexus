//go:build e2e

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestEmailConfigurationLifecycle checks that EmailConfiguration applies the
// email server settings with a password from a Secret, corrects drift made
// outside the provider, and resets the settings once deleted. It is NOT
// parallel: EmailConfiguration maps to a singleton in Nexus.
//
// The SMTP host is an address from the TEST-NET-3 documentation range: Nexus
// resolves the host when the settings are applied, refusing names it cannot
// resolve and, under the default SSRF protection, private addresses, while a
// literal public address needs no DNS. Nexus never connects to it.
func TestEmailConfigurationLifecycle(t *testing.T) {
	f := e2e.New(t)

	const (
		host        = "203.0.113.5"
		driftHost   = "192.0.2.10"
		fromAddress = "nexus-e2e@example.com"
		secretName  = "e2e-email-configuration-password"
	)

	// Reset the settings even if the test fails before deleting the
	// resource. Registered before CreateAndWaitForReady, it runs after the
	// resource's own cleanup.
	t.Cleanup(func() {
		err := f.ResetEmailConfiguration()
		if err != nil {
			t.Logf("resetting the email server configuration: %v", err)
		}
	})

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"password": "E2eSmtpPassword123!"},
	}

	err := f.Kube.Create(context.Background(), secret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating password secret: %v", err)
	}

	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), secret)
	})

	cr := &instancev1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-email-configuration", Namespace: "default"},
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Enabled:     ptr.To(true),
				Host:        host,
				Port:        587,
				Username:    ptr.To("nexus-e2e"),
				FromAddress: fromAddress,
				PasswordSecretRef: &xpv2.LocalSecretKeySelector{
					LocalSecretReference: xpv2.LocalSecretReference{Name: secretName},
					Key:                  "password",
				},
				SubjectPrefix:   ptr.To("[Nexus e2e]"),
				StartTLSEnabled: ptr.To(true),
			},
		},
	}

	matchesSpec := func(got *v395.ApiEmailConfiguration) bool {
		return got.GetEnabled() && got.Host == host && got.Port == 587 && got.GetUsername() == "nexus-e2e" &&
			got.FromAddress == fromAddress && got.GetSubjectPrefix() == "[Nexus e2e]" && got.GetStartTlsEnabled()
	}

	// Synced=True means the reconcile that applied the spec succeeded, so
	// Nexus must already hold exactly the spec.
	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)

	got, err := f.FetchEmailConfiguration()
	if err != nil {
		t.Fatalf("fetching the email server configuration from Nexus: %v", err)
	}

	if !matchesSpec(got) {
		t.Fatalf("Nexus email server configuration = %+v, want the spec", got)
	}

	// Drift made directly in Nexus must be reverted to the spec.
	err = f.SetEmailConfiguration(v395.ApiEmailConfiguration{
		Enabled:     ptr.To(false),
		Host:        driftHost,
		Port:        25,
		FromAddress: fromAddress,
	})
	if err != nil {
		t.Fatalf("injecting email server configuration drift: %v", err)
	}

	err = f.Nudge(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.WaitForNexus(context.Background(), 2*time.Minute, func() (bool, error) {
		observed, fetchErr := f.FetchEmailConfiguration()
		if fetchErr != nil {
			return false, fetchErr
		}

		return matchesSpec(observed), nil
	})
	if err != nil {
		observed, _ := f.FetchEmailConfiguration()
		t.Fatalf("email server configuration drift was not corrected: %v\n  nexus reports: %+v", err, observed)
	}

	f.Delete(t, cr)

	err = f.WaitForDeletion(context.Background(), cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for EmailConfiguration deletion: %v", err)
	}

	got, err = f.FetchEmailConfiguration()
	if err != nil {
		t.Fatalf("fetching the email server configuration from Nexus after deletion: %v", err)
	}

	if got.GetEnabled() || got.Host != "" {
		t.Errorf("Nexus email server configuration after deletion = %+v, want it reset", got)
	}
}
