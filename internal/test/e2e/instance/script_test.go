//go:build e2e

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestScriptLifecycle checks that a Script is stored in Nexus, that a source
// changed outside the provider is restored, that a spec change reaches Nexus
// and that deleting the resource deletes the script. The e2e Nexus runs with
// nexus.scripts.allowCreation=true (cluster/local/nexus.yaml); Nexus disables
// script creation by default.
func TestScriptLifecycle(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const (
		scriptName = "e2e-test-script"
		original   = "return 'e2e original'"
		drifted    = "return 'e2e drifted'"
		updated    = "return 'e2e updated'"
	)

	cr := &instancev1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-script", Namespace: "default"},
		Spec: instancev1alpha1.ScriptSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.ScriptParameters{
				Name:    scriptName,
				Content: original,
			},
		},
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)
	e2e.AssertExternalName(t, cr, scriptName)

	got, err := f.FetchScript(scriptName)
	if err != nil {
		t.Fatalf("fetching script from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("script %q not found in Nexus", scriptName)
	}

	if got.Content != original || got.Type != "groovy" {
		t.Fatalf("Nexus script = %+v, want groovy with the original source", got)
	}

	waitForContent := func(want, what string) {
		t.Helper()

		waitErr := f.WaitForNexus(context.Background(), 2*time.Minute, func() (bool, error) {
			observed, fetchErr := f.FetchScript(scriptName)
			if fetchErr != nil || observed == nil {
				return false, fetchErr
			}

			return observed.Content == want, nil
		})
		if waitErr != nil {
			observed, _ := f.FetchScript(scriptName)
			t.Fatalf("%s: %v\n  nexus reports: %+v", what, waitErr, observed)
		}
	}

	// A source changed directly in Nexus must be restored.
	err = f.SetScript(v395.ScriptXO{Name: scriptName, Type: "groovy", Content: drifted})
	if err != nil {
		t.Fatalf("injecting script drift: %v", err)
	}

	err = f.Nudge(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	waitForContent(original, "script drift was not corrected")

	// A spec change must reach Nexus.
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		getErr := f.Kube.Get(context.Background(), client.ObjectKeyFromObject(cr), cr)
		if getErr != nil {
			return getErr
		}

		cr.Spec.ForProvider.Content = updated

		return f.Kube.Update(context.Background(), cr)
	})
	if err != nil {
		t.Fatalf("updating the script spec: %v", err)
	}

	waitForContent(updated, "script spec change did not reach Nexus")

	f.Delete(t, cr)

	err = f.WaitForDeletion(context.Background(), cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for Script deletion: %v", err)
	}

	got, err = f.FetchScript(scriptName)
	if err != nil {
		t.Fatalf("fetching script from Nexus after deletion: %v", err)
	}

	if got != nil {
		t.Errorf("script %q still exists in Nexus after deletion", scriptName)
	}
}
