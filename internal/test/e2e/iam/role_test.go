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
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestRoleCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const (
		roleID   = "e2e-test-role"
		roleName = "E2E Test Role"
	)

	role := &iamv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-role", Namespace: "default"},
		Spec: iamv1alpha1.RoleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.RoleParameters{
				ID:          roleID,
				Name:        roleName,
				Description: ptr.To("Role created by e2e tests"),
				Privileges: []string{
					"nx-repository-view-*-*-browse",
					"nx-repository-view-*-*-read",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, role, 2*time.Minute)
	e2e.AssertReady(t, role)
	e2e.AssertSynced(t, role)

	got, err := f.FetchRole(roleID)
	if err != nil {
		t.Fatalf("fetching role from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("role %q not found in Nexus", roleID)
	}
	if got.ID != roleID {
		t.Errorf("role ID = %q, want %q", got.ID, roleID)
	}
	if got.Name != roleName {
		t.Errorf("role name = %q, want %q", got.Name, roleName)
	}
}
