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

func TestCleanupPolicyCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	// The cleanup-policies REST endpoint is available only in Nexus Repository Pro.
	// Skip gracefully in OSS/Community Edition installations.
	if !f.CleanupPoliciesAvailable() {
		t.Skip("cleanup-policies REST API not available (requires Nexus Repository Pro)")
	}

	const policyName = "e2e-test-cleanup-policy"

	policy := &contentv1alpha1.CleanupPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: policyName, Namespace: "default"},
		Spec: contentv1alpha1.CleanupPolicySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: contentv1alpha1.CleanupPolicyParameters{
				Name:                    policyName,
				Format:                  "maven2",
				Notes:                   ptrTo("E2E test cleanup policy"),
				CriteriaLastBlobUpdated: ptrTo(30),
				CriteriaLastDownloaded:  ptrTo(60),
				CriteriaReleaseType:     ptrTo("RELEASES"),
			},
		},
	}

	f.CreateAndWaitForReady(t, policy, 2*time.Minute)
	e2e.AssertReady(t, policy)
	e2e.AssertSynced(t, policy)

	got, err := f.FetchCleanupPolicy(policyName)
	if err != nil {
		t.Fatalf("fetching cleanup policy from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("cleanup policy %q not found in Nexus", policyName)
	}
	if got.Name != policyName {
		t.Errorf("policy name = %q, want %q", got.Name, policyName)
	}
}

func ptrTo[T any](v T) *T { return &v }
