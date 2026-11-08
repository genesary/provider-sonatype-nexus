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

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestRoleObserveOnly checks that a Role with the Observe management policy
// reports an existing Nexus role without changing or deleting it.
func TestRoleObserveOnly(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	const (
		roleID   = "e2e-observe-only-role"
		roleName = "E2E Observe Only Role"
	)

	// Remove a role left behind by an aborted run, which would make the
	// creation below fail. A 404 is the expected outcome and is ignored.
	_, _ = f.Nexus.SecurityManagementRolesAPI.DeleteSecurityRoles(ctx, roleID).Execute()

	_, _, err := f.Nexus.SecurityManagementRolesAPI.CreateSecurityRoles(ctx).
		RoleXORequest(*v395.NewRoleXORequest(roleID, roleName)).Execute()
	if err != nil {
		t.Fatalf("creating role %q in Nexus: %v", roleID, err)
	}

	t.Cleanup(func() {
		_, cleanupErr := f.Nexus.SecurityManagementRolesAPI.DeleteSecurityRoles(ctx, roleID).Execute()
		if cleanupErr != nil {
			t.Logf("cleanup: deleting role %q from Nexus: %v", roleID, cleanupErr)
		}
	})

	role := &iamv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: roleID, Namespace: "default"},
		Spec: iamv1alpha1.RoleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
				ManagementPolicies: xpv2.ManagementPolicies{xpv2.ManagementActionObserve},
			},
			ForProvider: iamv1alpha1.RoleParameters{
				ID: roleID,
				// Differs from Nexus: an observe-only Role must not update it.
				Name: "Name Crossplane must not apply",
			},
		},
	}
	meta.SetExternalName(role, roleID)

	f.CreateAndWaitForReady(t, role, 2*time.Minute)

	if role.Status.AtProvider.ID != roleID {
		t.Errorf("status.atProvider.id = %q, want %q", role.Status.AtProvider.ID, roleID)
	}

	got, err := f.FetchRole(roleID)
	if err != nil {
		t.Fatalf("fetching role from Nexus: %v", err)
	}

	if got == nil || got.GetName() != roleName {
		t.Fatalf("role in Nexus = %+v, want it unchanged with name %q", got, roleName)
	}

	f.Delete(t, role)

	err = f.WaitForDeletion(ctx, role, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for Role deletion: %v", err)
	}

	got, err = f.FetchRole(roleID)
	if err != nil {
		t.Fatalf("fetching role from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("deleting an observe-only Role deleted role %q from Nexus", roleID)
	}
}
