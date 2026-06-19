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

func TestApplicationPrivilegeCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const privName = "e2e-test-app-privilege"

	priv := &iamv1alpha1.Privilege{
		ObjectMeta: metav1.ObjectMeta{Name: privName, Namespace: "default"},
		Spec: iamv1alpha1.PrivilegeSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PrivilegeParameters{
				Name:        privName,
				Description: ptr.To("Application privilege created by e2e tests"),
				Type:        "application",
				Domain:      ptr.To("analytics"),
				Actions:     []string{"READ"},
			},
		},
	}

	f.CreateAndWaitForReady(t, priv, 2*time.Minute)
	e2e.AssertReady(t, priv)
	e2e.AssertSynced(t, priv)

	got, err := f.FetchPrivilege(privName)
	if err != nil {
		t.Fatalf("fetching privilege from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("privilege %q not found in Nexus", privName)
	}
	if got.Name != privName {
		t.Errorf("privilege name = %q, want %q", got.Name, privName)
	}
	if got.Type != "application" {
		t.Errorf("privilege type = %q, want %q", got.Type, "application")
	}
}

func TestRepositoryViewPrivilegeCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const privName = "e2e-test-repo-privilege"

	priv := &iamv1alpha1.Privilege{
		ObjectMeta: metav1.ObjectMeta{Name: privName, Namespace: "default"},
		Spec: iamv1alpha1.PrivilegeSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.PrivilegeParameters{
				Name:        privName,
				Description: ptr.To("Repository-view privilege created by e2e tests"),
				Type:        "repository-view",
				Format:      ptr.To("maven2"),
				Repository:  ptr.To("*"),
				Actions:     []string{"BROWSE", "READ"},
			},
		},
	}

	f.CreateAndWaitForReady(t, priv, 2*time.Minute)
	e2e.AssertReady(t, priv)
	e2e.AssertSynced(t, priv)

	got, err := f.FetchPrivilege(privName)
	if err != nil {
		t.Fatalf("fetching privilege from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("privilege %q not found in Nexus", privName)
	}
	if got.Name != privName {
		t.Errorf("privilege name = %q, want %q", got.Name, privName)
	}
}
