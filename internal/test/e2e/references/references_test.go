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

// Package references_test checks that managed resources referencing each
// other are reconciled once the resources they reference are.
package references_test

import (
	"context"
	"slices"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestReferenceChain applies, all at once and in reverse order, a Role that
// references a Privilege, that references a Repository, that references a
// BlobStore, and checks that each resolves to the Nexus name of what it
// references. The Kubernetes names differ from the Nexus names on purpose.
func TestReferenceChain(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	const (
		blobStoreName  = "e2e-ref-store"
		repositoryName = "e2e-ref-maven"
		privilegeName  = "e2e-ref-maven-view"
		roleID         = "e2e-ref-role"
	)

	pcRef := xpv2.ManagedResourceSpec{
		ProviderConfigReference: &xpv2.ProviderConfigReference{Kind: "ProviderConfig", Name: f.ProviderConfigName},
	}

	blobStore := &instancev1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "store", Namespace: "default"},
		Spec: instancev1alpha1.BlobStoreSpec{
			ManagedResourceSpec: pcRef,
			ForProvider: instancev1alpha1.BlobStoreParameters{
				Name: blobStoreName,
				Type: "File",
				Path: new("/nexus-data/blobs/" + blobStoreName),
			},
		},
	}

	repository := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "maven", Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: pcRef,
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repositoryName,
				Format: "maven2",
				Type:   "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreNameRef: &xpv2.Reference{Name: blobStore.GetName()},
				},
				Maven: &contentv1alpha1.MavenConfig{
					VersionPolicy: new("RELEASE"),
					LayoutPolicy:  new("STRICT"),
				},
			},
		},
	}

	privilege := &iamv1alpha1.Privilege{
		ObjectMeta: metav1.ObjectMeta{Name: "maven-view", Namespace: "default"},
		Spec: iamv1alpha1.PrivilegeSpec{
			ManagedResourceSpec: pcRef,
			ForProvider: iamv1alpha1.PrivilegeParameters{
				Name:          privilegeName,
				Type:          "repository-view",
				Format:        new("maven2"),
				RepositoryRef: &xpv2.Reference{Name: repository.GetName()},
				Actions:       []string{"BROWSE", "READ"},
			},
		},
	}

	role := &iamv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "maven-readers", Namespace: "default"},
		Spec: iamv1alpha1.RoleSpec{
			ManagedResourceSpec: pcRef,
			ForProvider: iamv1alpha1.RoleParameters{
				ID:            roleID,
				Name:          "E2E Reference Role",
				PrivilegeRefs: []xpv2.Reference{{Name: privilege.GetName()}},
			},
		},
	}

	// Cleanups run last in, first out, so register the BlobStore first: it
	// is deleted last, once nothing uses it.
	resources := []e2e.Conditioned{blobStore, repository, privilege, role}
	for _, resource := range resources {
		t.Cleanup(func() {
			f.Delete(t, resource)

			waitErr := f.WaitForDeletion(ctx, resource, e2e.DefaultDeleteTimeout)
			if waitErr != nil {
				t.Logf("cleanup: waiting for deletion of %s: %v", resource.GetName(), waitErr)
			}
		})
	}

	for _, resource := range slices.Backward(resources) {
		f.Apply(t, resource)
	}

	for _, resource := range resources {
		err := f.WaitForReady(ctx, resource, 3*time.Minute)
		if err != nil {
			t.Fatalf("waiting for %s to be Ready: %v\n  conditions: %s", resource.GetName(), err, e2e.SummariseConditions(resource))
		}
	}

	if got := repository.Spec.ForProvider.Storage.BlobStoreName; got != blobStoreName {
		t.Errorf("Repository storage.blobStoreName = %q, want %q", got, blobStoreName)
	}

	gotRepository, err := f.FetchMavenHostedRepo(repositoryName)
	if err != nil || gotRepository == nil {
		t.Fatalf("fetching repository %q from Nexus: %v (found %t)", repositoryName, err, gotRepository != nil)
	}

	if got := gotRepository.Storage.GetBlobStoreName(); got != blobStoreName {
		t.Errorf("repository blob store in Nexus = %q, want %q", got, blobStoreName)
	}

	gotPrivilege, err := f.FetchPrivilege(privilegeName)
	if err != nil || gotPrivilege == nil {
		t.Fatalf("fetching privilege %q from Nexus: %v (found %t)", privilegeName, err, gotPrivilege != nil)
	}

	if got := gotPrivilege.GetRepository(); got != repositoryName {
		t.Errorf("privilege repository in Nexus = %q, want %q", got, repositoryName)
	}

	gotRole, err := f.FetchRole(roleID)
	if err != nil || gotRole == nil {
		t.Fatalf("fetching role %q from Nexus: %v (found %t)", roleID, err, gotRole != nil)
	}

	if got := gotRole.GetPrivileges(); !slices.Equal(got, []string{privilegeName}) {
		t.Errorf("role privileges in Nexus = %v, want [%s]", got, privilegeName)
	}
}
