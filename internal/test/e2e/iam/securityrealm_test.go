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

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestSecurityRealmCRUD tests the SecurityRealm resource lifecycle.
// This test is NOT parallel since SecurityRealm maps to a singleton
// active-realm list in Nexus — concurrent tests would collide.
func TestSecurityRealmCRUD(t *testing.T) {
	f := e2e.New(t)

	const wantRealm = "DockerToken"

	realm := &iamv1alpha1.SecurityRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-realms", Namespace: "default"},
		Spec: iamv1alpha1.SecurityRealmSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.SecurityRealmParameters{
				ActiveRealms: []string{
					"NexusAuthenticatingRealm",
					wantRealm,
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, realm, 2*time.Minute)
	e2e.AssertReady(t, realm)
	e2e.AssertSynced(t, realm)

	active, err := f.ListActiveRealms()
	if err != nil {
		t.Fatalf("listing active realms from Nexus: %v", err)
	}

	found := false
	for _, r := range active {
		if r == wantRealm {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("realm %q not found in active realms: %v", wantRealm, active)
	}
}
