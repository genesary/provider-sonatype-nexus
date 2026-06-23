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

package instance_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestCapabilityCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	cr := &instancev1alpha1.Capability{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-base-url", Namespace: "default"},
		Spec: instancev1alpha1.CapabilitySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.CapabilityParameters{
				TypeId:  "baseurl",
				Enabled: true,
				Notes:   "E2E test capability",
				Properties: map[string]string{
					"url": "http://e2e-test.example.com",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)

	if cr.Status.AtProvider.ID == "" {
		t.Fatal("AtProvider.ID should be set after creation")
	}

	got, err := f.FetchCapability(cr.Status.AtProvider.ID)
	if err != nil {
		t.Fatalf("fetching capability from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("capability %q not found in Nexus", cr.Status.AtProvider.ID)
	}

	if got.Type != cr.Spec.ForProvider.TypeId {
		t.Errorf("capability type = %q, want %q", got.Type, cr.Spec.ForProvider.TypeId)
	}

	if !got.Enabled {
		t.Error("capability should be enabled")
	}
}
