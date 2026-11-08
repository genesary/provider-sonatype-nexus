//go:build e2e

package instance_test

import (
	"context"
	"slices"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestSSRFProtectionLifecycle checks that SSRFProtection replaces the whole
// Nexus configuration, corrects drift made outside the provider, and restores
// the Nexus default once deleted. It is NOT parallel: SSRFProtection maps to a
// singleton in Nexus. Every configuration it sets, including the drift it
// injects, only loosens the default protection, so concurrent tests in other
// packages are unaffected.
func TestSSRFProtectionLifecycle(t *testing.T) {
	f := e2e.New(t)

	const (
		allowedIP     = "10.0.0.50"
		allowedDomain = "e2e-registry.local"
		staleIP       = "10.0.0.99"
		staleDomain   = "e2e-stale.local"
	)

	// Restore the default even if the test fails before deleting the
	// resource. Registered before CreateAndWaitForReady, it runs after the
	// resource's own cleanup.
	t.Cleanup(func() {
		err := f.SetSSRFProtection(v395.SsrfProtectionConfigurationXO{Enabled: true})
		if err != nil {
			t.Logf("restoring the default SSRF protection: %v", err)
		}
	})

	// Exceptions that exist before the resource does must be removed: the
	// spec replaces the configuration rather than merging into it.
	err := f.SetSSRFProtection(v395.SsrfProtectionConfigurationXO{
		Enabled:        true,
		AllowedIPs:     []string{staleIP},
		AllowedDomains: []string{staleDomain},
	})
	if err != nil {
		t.Fatalf("seeding a stale SSRF protection exception: %v", err)
	}

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

	matchesSpec := func(got *v395.SsrfProtectionConfigurationXO) bool {
		return got.Enabled && sameSet(got.AllowedIPs, []string{allowedIP}) && sameSet(got.AllowedDomains, []string{allowedDomain})
	}

	// Synced=True means the reconcile that applied the spec succeeded, so
	// Nexus must already hold exactly the spec.
	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)

	got, err := f.FetchSSRFProtection()
	if err != nil {
		t.Fatalf("fetching SSRF protection from Nexus: %v", err)
	}

	if !matchesSpec(got) {
		t.Fatalf("Nexus SSRF protection = %+v, want enabled with exactly %s and %s allowed", got, allowedIP, allowedDomain)
	}

	// Drift made directly in Nexus must be reverted to the spec.
	err = f.SetSSRFProtection(v395.SsrfProtectionConfigurationXO{
		Enabled:        false,
		AllowedIPs:     []string{allowedIP, staleIP},
		AllowedDomains: []string{staleDomain},
	})
	if err != nil {
		t.Fatalf("injecting SSRF protection drift: %v", err)
	}

	err = f.Nudge(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.WaitForNexus(context.Background(), 2*time.Minute, func() (bool, error) {
		observed, fetchErr := f.FetchSSRFProtection()
		if fetchErr != nil {
			return false, fetchErr
		}

		return matchesSpec(observed), nil
	})
	if err != nil {
		observed, _ := f.FetchSSRFProtection()
		t.Fatalf("SSRF protection drift was not corrected: %v\n  nexus reports: %+v", err, observed)
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

// sameSet reports whether a and b hold the same strings, ignoring order.
func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)

	return slices.Equal(slices.Compact(a), slices.Compact(b))
}
