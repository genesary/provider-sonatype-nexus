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

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestRepositoryFormats creates a repository of every format and repository
// type the generated Nexus client added, and checks Nexus serves it with that
// format and type. Groups are created last, with the format's first
// repository as their member, because Nexus rejects an empty group.
//
// Terraform hosted repositories are left out: Nexus validates their PGP
// signing key on creation, and the unit and CRD validation tests cover them.
func TestRepositoryFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format    string
		types     []string
		remoteURL string
	}{
		{format: "alpine", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://dl-cdn.alpinelinux.org/alpine/"},
		{format: "ansiblegalaxy", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://galaxy.ansible.com/"},
		{format: "composer", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://repo.packagist.org/"},
		{format: "oci", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://registry-1.docker.io/"},
		{format: "pub", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://pub.dev/"},
		{format: "swift", types: []string{"hosted", "proxy", "group"}, remoteURL: "https://github.com/"},
		{format: "terraform", types: []string{"proxy", "group"}, remoteURL: "https://registry.terraform.io/"},
		{format: "conan", types: []string{"hosted", "group"}},
		{format: "conda", types: []string{"hosted", "group"}},
		{format: "go", types: []string{"hosted"}},
		{format: "helm", types: []string{"hosted", "group"}},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()

			f := e2e.New(t)

			var member string

			for _, repoType := range tt.types {
				repoName := "e2e-test-format-" + tt.format + "-" + repoType

				params := contentv1alpha1.RepositoryParameters{
					Name:    repoName,
					Format:  tt.format,
					Type:    repoType,
					Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				}

				switch repoType {
				case "proxy":
					params.Proxy = &contentv1alpha1.ProxyConfig{RemoteURL: tt.remoteURL}
				case "group":
					params.Group = &contentv1alpha1.GroupConfig{MemberNames: []string{member}}
				}

				if tt.format == "alpine" {
					// Nexus does not validate the key until it signs an index.
					params.AlpineSigning = &contentv1alpha1.AlpineSigningConfig{Keypair: "e2e-test-keypair"}
				}

				repo := &contentv1alpha1.Repository{
					ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
					Spec: contentv1alpha1.RepositorySpec{
						ManagedResourceSpec: xpv2.ManagedResourceSpec{
							ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
						},
						ForProvider: params,
					},
				}

				f.CreateAndWaitForReady(t, repo, reconcileTimeout)
				e2e.AssertReady(t, repo)
				e2e.AssertSynced(t, repo)

				got, err := f.FetchRepository(repoName)
				if err != nil || got == nil {
					t.Fatalf("fetching %s %s repository %q from Nexus: %v", tt.format, repoType, repoName, err)
				}

				if got.GetFormat() != tt.format || got.GetType() != repoType {
					t.Errorf("Nexus serves %q as %s %s, want %s %s", repoName, got.GetFormat(), got.GetType(), tt.format, repoType)
				}

				if member == "" {
					member = repoName
				}
			}
		})
	}
}
