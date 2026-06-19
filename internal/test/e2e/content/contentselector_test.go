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

func TestContentSelectorCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const selectorName = "e2e-test-selector"

	selector := &contentv1alpha1.ContentSelector{
		ObjectMeta: metav1.ObjectMeta{Name: selectorName, Namespace: "default"},
		Spec: contentv1alpha1.ContentSelectorSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: contentv1alpha1.ContentSelectorParameters{
				Name:        selectorName,
				Description: ptrTo("Selector created by e2e tests"),
				Expression:  `format == "maven2" and path =^ "/test/"`,
			},
		},
	}

	f.CreateAndWaitForReady(t, selector, 2*time.Minute)
	e2e.AssertReady(t, selector)
	e2e.AssertSynced(t, selector)

	got, err := f.FetchContentSelector(selectorName)
	if err != nil {
		t.Fatalf("fetching content selector from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("content selector %q not found in Nexus", selectorName)
	}
	if got.Name != selectorName {
		t.Errorf("selector name = %q, want %q", got.Name, selectorName)
	}
}
