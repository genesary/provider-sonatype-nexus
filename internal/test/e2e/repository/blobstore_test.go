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

package repository_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestBlobStoreCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const bsName = "e2e-test-file"

	bs := &instancev1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-file-blobstore", Namespace: "default"},
		Spec: instancev1alpha1.BlobStoreSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.BlobStoreParameters{
				Name: bsName,
				Type: "File",
				Path: ptrTo("/nexus-data/blobs/e2e-test-file"),
				SoftQuota: &instancev1alpha1.SoftQuota{
					Type:  ptrTo("spaceRemainingQuota"),
					Limit: ptrTo(int64(104857600)),
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, bs, 2*time.Minute)
	e2e.AssertReady(t, bs)
	e2e.AssertSynced(t, bs)

	got, err := f.FetchBlobStoreFile(bsName)
	if err != nil {
		t.Fatalf("fetching blob store from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("blob store %q not found in Nexus", bsName)
	}
	if got.Name != bsName {
		t.Errorf("blob store name = %q, want %q", got.Name, bsName)
	}
}

func ptrTo[T any](v T) *T { return &v }
