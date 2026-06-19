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

package iam_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestAnonymousAccessCRUD tests the AnonymousAccess resource lifecycle.
// This test is NOT parallel since AnonymousAccess maps to a singleton
// configuration in Nexus — concurrent tests would collide.
func TestAnonymousAccessCRUD(t *testing.T) {
	f := e2e.New(t)

	anon := &iamv1alpha1.AnonymousAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-anonymous", Namespace: "default"},
		Spec: iamv1alpha1.AnonymousAccessSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.AnonymousAccessParameters{
				Enabled:   true,
				UserID:    "anonymous",
				RealmName: "NexusAuthorizingRealm",
			},
		},
	}

	f.CreateAndWaitForReady(t, anon, 2*time.Minute)
	e2e.AssertReady(t, anon)
	e2e.AssertSynced(t, anon)

	got, err := f.FetchAnonymousAccess()
	if err != nil {
		t.Fatalf("fetching anonymous access from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("anonymous access settings not found in Nexus")
	}
	if !got.Enabled {
		t.Errorf("anonymous access enabled = false, want true")
	}
}
