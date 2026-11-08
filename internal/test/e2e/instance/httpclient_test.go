//go:build e2e

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestHTTPClientLifecycle applies HTTP client settings, then checks that
// deleting the resource resets them. It is NOT parallel: HTTPClient maps to a
// singleton in Nexus. It sets no proxy, because a proxy would break the
// outbound requests of concurrent tests in other packages.
func TestHTTPClientLifecycle(t *testing.T) {
	f := e2e.New(t)

	// The HTTP system settings endpoint is available only in Nexus
	// Repository Pro. Skip gracefully in Community Edition installations.
	if !f.HTTPSettingsAvailable() {
		t.Skip("HTTP system settings REST API not available (requires Nexus Repository Pro)")
	}

	const userAgentSuffix = "e2e-provider-sonatype-nexus"

	cr := &instancev1alpha1.HTTPClient{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-http-client", Namespace: "default"},
		Spec: instancev1alpha1.HTTPClientSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.HTTPClientParameters{
				UserAgentSuffix: ptr.To(userAgentSuffix),
				NonProxyHosts:   []string{"e2e.internal.example.com"},
			},
		},
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)

	got, err := f.FetchHTTPSettings()
	if err != nil {
		t.Fatalf("fetching HTTP settings from Nexus: %v", err)
	}

	if ua := got.UserAgent.Get(); ua == nil || *ua != userAgentSuffix {
		t.Fatalf("Nexus user agent suffix = %v, want %q", ua, userAgentSuffix)
	}

	f.Delete(t, cr)

	err = f.WaitForDeletion(context.Background(), cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for HTTPClient deletion: %v", err)
	}

	got, err = f.FetchHTTPSettings()
	if err != nil {
		t.Fatalf("fetching HTTP settings from Nexus after deletion: %v", err)
	}

	if ua := got.UserAgent.Get(); ua != nil && *ua == userAgentSuffix {
		t.Errorf("Nexus user agent suffix after deletion = %q, want it reset", *ua)
	}
}
