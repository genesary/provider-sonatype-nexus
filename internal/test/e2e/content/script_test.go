//go:build e2e

package content_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestScriptCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	if !f.ScriptingAvailable() {
		t.Skip("Nexus Scripting API not available: set nexus.scripts.allowCreation=true or upgrade to a supported version")
	}

	const scriptName = "e2e-test-script"

	script := &contentv1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{Name: scriptName, Namespace: "default"},
		Spec: contentv1alpha1.ScriptSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: contentv1alpha1.ScriptParameters{
				Name:    scriptName,
				Type:    "groovy",
				Content: "log.info('e2e test script')",
			},
		},
	}

	f.CreateAndWaitForReady(t, script, 2*time.Minute)
	e2e.AssertReady(t, script)
	e2e.AssertSynced(t, script)

	got, err := f.FetchScript(scriptName)
	if err != nil {
		t.Fatalf("fetching script from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("script %q not found in Nexus", scriptName)
	}

	if got.Name != scriptName {
		t.Errorf("script name = %q, want %q", got.Name, scriptName)
	}

	if got.Type != "groovy" {
		t.Errorf("script type = %q, want %q", got.Type, "groovy")
	}
}
