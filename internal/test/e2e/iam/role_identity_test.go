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
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestRoleIdentifiedBySpec checks that a Role whose metadata.name differs
// from its spec.forProvider.id manages the role named by its spec, and never
// touches a Nexus role that happens to bear its metadata.name.
func TestRoleIdentifiedBySpec(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	const (
		decoyID   = "e2e-identity-decoy"
		decoyName = "E2E Identity Decoy"
		roleID    = "e2e-identity-role"
	)

	_, _, err := f.Nexus.SecurityManagementRolesAPI.CreateSecurityRoles(ctx).
		RoleXORequest(*v395.NewRoleXORequest(decoyID, decoyName)).Execute()
	if err != nil {
		t.Fatalf("creating decoy role %q in Nexus: %v", decoyID, err)
	}

	t.Cleanup(func() {
		_, cleanupErr := f.Nexus.SecurityManagementRolesAPI.DeleteSecurityRoles(ctx, decoyID).Execute()
		if cleanupErr != nil {
			t.Logf("cleanup: deleting decoy role %q from Nexus: %v", decoyID, cleanupErr)
		}
	})

	role := &iamv1alpha1.Role{
		// Named after the decoy, which crossplane-runtime's default
		// initializer would have made the external name.
		ObjectMeta: metav1.ObjectMeta{Name: decoyID, Namespace: "default"},
		Spec: iamv1alpha1.RoleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.RoleParameters{
				ID:   roleID,
				Name: "E2E Identity Role",
			},
		},
	}

	f.CreateAndWaitForReady(t, role, 2*time.Minute)
	e2e.AssertExternalName(t, role, roleID)

	got, err := f.FetchRole(roleID)
	if err != nil {
		t.Fatalf("fetching role %q from Nexus: %v", roleID, err)
	}

	if got == nil {
		t.Fatalf("role %q not created in Nexus", roleID)
	}

	assertDecoyUnchanged := func() {
		t.Helper()

		decoy, fetchErr := f.FetchRole(decoyID)
		if fetchErr != nil {
			t.Fatalf("fetching decoy role %q from Nexus: %v", decoyID, fetchErr)
		}

		if decoy == nil || decoy.GetName() != decoyName {
			t.Fatalf("decoy role = %+v, want it unchanged with name %q", decoy, decoyName)
		}
	}

	assertDecoyUnchanged()

	f.Delete(t, role)

	err = f.WaitForDeletion(ctx, role, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for Role deletion: %v", err)
	}

	assertDecoyUnchanged()

	got, err = f.FetchRole(roleID)
	if err != nil {
		t.Fatalf("fetching role %q from Nexus: %v", roleID, err)
	}

	if got != nil {
		t.Errorf("role %q still in Nexus after its Role was deleted", roleID)
	}
}
