//go:build e2e

/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package content_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestRoutingRuleCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const ruleName = "e2e-test-routing-rule"

	rule := &contentv1alpha1.RoutingRule{
		ObjectMeta: metav1.ObjectMeta{Name: ruleName, Namespace: "default"},
		Spec: contentv1alpha1.RoutingRuleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: contentv1alpha1.RoutingRuleParameters{
				Name:        ruleName,
				Description: ptrTo("Routing rule created by e2e tests"),
				Mode:        "ALLOW",
				Matchers:    []string{`^/com/example/.*`, `^/org/example/.*`},
			},
		},
	}

	f.CreateAndWaitForReady(t, rule, 2*time.Minute)
	e2e.AssertReady(t, rule)
	e2e.AssertSynced(t, rule)

	got, err := f.FetchRoutingRule(ruleName)
	if err != nil {
		t.Fatalf("fetching routing rule from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("routing rule %q not found in Nexus", ruleName)
	}
	if got.Name != ruleName {
		t.Errorf("routing rule name = %q, want %q", got.Name, ruleName)
	}
	if string(got.Mode) != "ALLOW" {
		t.Errorf("routing rule mode = %q, want %q", got.Mode, "ALLOW")
	}
	if len(got.Matchers) != 2 {
		t.Errorf("routing rule matchers len = %d, want 2", len(got.Matchers))
	}
}
