//go:build e2e

package instance_test

import (
	"context"
	"slices"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestSSRFProtectionLifecycle applies SSRF protection exceptions, then checks
// that deleting the resource restores the Nexus default. It is NOT parallel:
// SSRFProtection maps to a singleton in Nexus. The exceptions it adds only
// loosen the default protection, so concurrent tests in other packages are
// unaffected.
func TestSSRFProtectionLifecycle(t *testing.T) {
	f := e2e.New(t)

	const (
		allowedIP     = "10.0.0.50"
		allowedDomain = "e2e-registry.local"
	)

	cr := &instancev1alpha1.SSRFProtection{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-ssrf-protection", Namespace: "default"},
		Spec: instancev1alpha1.SSRFProtectionSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.SSRFProtectionParameters{
				Enabled:        ptr.To(true),
				AllowedIPs:     []string{allowedIP},
				AllowedDomains: []string{allowedDomain},
			},
		},
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)

	got, err := f.FetchSSRFProtection()
	if err != nil {
		t.Fatalf("fetching SSRF protection from Nexus: %v", err)
	}

	if !got.Enabled || !slices.Contains(got.AllowedIPs, allowedIP) || !slices.Contains(got.AllowedDomains, allowedDomain) {
		t.Fatalf("Nexus SSRF protection = %+v, want enabled with %s and %s allowed", got, allowedIP, allowedDomain)
	}

	f.Delete(t, cr)

	err = f.WaitForDeletion(context.Background(), cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for SSRFProtection deletion: %v", err)
	}

	got, err = f.FetchSSRFProtection()
	if err != nil {
		t.Fatalf("fetching SSRF protection from Nexus after deletion: %v", err)
	}

	if !got.Enabled || len(got.AllowedIPs) != 0 || len(got.AllowedDomains) != 0 {
		t.Errorf("Nexus SSRF protection after deletion = %+v, want enabled without exceptions", got)
	}
}
