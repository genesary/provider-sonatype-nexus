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

// TestEmailConfigurationCRUD tests the EmailConfiguration resource lifecycle.
// This test is NOT parallel since EmailConfiguration maps to a singleton in
// Nexus — concurrent tests would collide.
func TestEmailConfigurationCRUD(t *testing.T) {
	f := e2e.New(t)

	emailConfig := &instancev1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-email-config"},
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Enabled:     false,
				Host:        "smtp.example.com",
				Port:        25,
				FromAddress: "nexus@example.com",
			},
		},
	}

	f.CreateAndWaitForReady(t, emailConfig, 2*time.Minute)
	e2e.AssertReady(t, emailConfig)
	e2e.AssertSynced(t, emailConfig)

	got, err := f.FetchEmailConfiguration()
	if err != nil {
		t.Fatalf("fetching email configuration from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("email configuration not found in Nexus")
	}

	if got.Enabled != nil && *got.Enabled {
		t.Errorf("email config enabled = true, want false")
	}

	if got.Host != "smtp.example.com" {
		t.Errorf("email config host = %q, want smtp.example.com", got.Host)
	}
}
