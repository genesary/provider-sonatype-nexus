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
	"context"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// newSettingsRepo returns a Repository named repoName with params.
func newSettingsRepo(f *e2e.Framework, repoName string, params contentv1alpha1.RepositoryParameters) *contentv1alpha1.Repository {
	params.Name = repoName

	return &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: params,
		},
	}
}

// TestDockerHostedRepositorySettings checks the latest tag policy and the
// proprietary components flag reach Nexus, and that removing them from the
// spec turns them off again, since Nexus resets both when an update leaves
// them out.
func TestDockerHostedRepositorySettings(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-docker-hosted-settings"

	repo := newSettingsRepo(f, repoName, contentv1alpha1.RepositoryParameters{
		Format: "docker",
		Type:   "hosted",
		Storage: &contentv1alpha1.RepositoryStorage{
			BlobStoreName: defaultBlobStore,
			WritePolicy:   new("ALLOW_ONCE"),
			LatestPolicy:  new(true),
		},
		Component: &contentv1alpha1.ComponentConfig{ProprietaryComponents: new(true)},
	})

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchDockerHostedRepo(repoName)
	if err != nil || got == nil {
		t.Fatalf("fetching docker hosted repo from Nexus: %v", err)
	}

	if !got.Storage.GetLatestPolicy() || !got.Component.GetProprietaryComponents() {
		t.Fatalf("after create, Nexus reports storage %+v and component %+v, want the latest policy and proprietary components", got.Storage, got.Component)
	}

	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Storage.LatestPolicy = nil
		spec.Component = nil
	})

	err = f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, fetchErr := f.FetchDockerHostedRepo(repoName)
		if fetchErr != nil || observed == nil {
			return false, fetchErr
		}

		return !observed.Storage.GetLatestPolicy() && !observed.Component.GetProprietaryComponents(), nil
	})
	if err != nil {
		observed, _ := f.FetchDockerHostedRepo(repoName)
		t.Fatalf("docker hosted repo kept its settings after they were removed from the spec: %v\n  nexus reports: %+v", err, observed)
	}
}

// TestRawProxyRepositorySettings checks the shared proxy settings and the raw
// content disposition reach Nexus, and that a change to them is applied.
func TestRawProxyRepositorySettings(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-raw-proxy-settings"

	repo := newSettingsRepo(f, repoName, contentv1alpha1.RepositoryParameters{
		Format:  "raw",
		Type:    "proxy",
		Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
		Proxy: &contentv1alpha1.ProxyConfig{
			RemoteURL:                 "https://example.com/",
			PreserveEncodedCharacters: new(true),
		},
		Firewall: &contentv1alpha1.FirewallConfig{Mode: "DISABLED"},
		Raw:      &contentv1alpha1.RawConfig{ContentDisposition: new("INLINE")},
	})

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchRawProxyRepo(repoName)
	if err != nil || got == nil {
		t.Fatalf("fetching raw proxy repo from Nexus: %v", err)
	}

	if !got.Proxy.GetPreserveEncodedCharacters() || got.Firewall.GetMode() != "DISABLED" || got.Raw.GetContentDisposition() != "INLINE" {
		t.Fatalf("after create, Nexus reports proxy %+v, firewall %+v and raw %+v, want the spec settings", got.Proxy, got.Firewall, got.Raw)
	}

	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Proxy.PreserveEncodedCharacters = nil
		spec.Raw.ContentDisposition = new("ATTACHMENT")
	})

	err = f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, fetchErr := f.FetchRawProxyRepo(repoName)
		if fetchErr != nil || observed == nil {
			return false, fetchErr
		}

		return !observed.Proxy.GetPreserveEncodedCharacters() && observed.Raw.GetContentDisposition() == "ATTACHMENT", nil
	})
	if err != nil {
		observed, _ := f.FetchRawProxyRepo(repoName)
		t.Fatalf("raw proxy repo did not converge on the updated spec: %v\n  nexus reports: %+v", err, observed)
	}
}

// TestPypiProxyIndexPath checks the empty index path, which serves the remote
// index from its root, reaches Nexus and is told apart from the /simple
// default Nexus restores when the spec leaves it out.
func TestPypiProxyIndexPath(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-pypi-proxy-index-path"

	repo := newSettingsRepo(f, repoName, contentv1alpha1.RepositoryParameters{
		Format:  "pypi",
		Type:    "proxy",
		Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
		Proxy:   &contentv1alpha1.ProxyConfig{RemoteURL: "https://pypi.org/"},
		Pypi:    &contentv1alpha1.PypiProxyConfig{IndexPath: new("")},
	})

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchPypiProxyRepo(repoName)
	if err != nil || got == nil {
		t.Fatalf("fetching pypi proxy repo from Nexus: %v", err)
	}

	if got.Pypi.IndexPath == nil || *got.Pypi.IndexPath != "" {
		t.Fatalf("after create, Nexus reports indexPath %q, want the root path", ptr.Deref(got.Pypi.IndexPath, "<unset>"))
	}

	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Pypi = nil
	})

	err = f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, fetchErr := f.FetchPypiProxyRepo(repoName)
		if fetchErr != nil || observed == nil {
			return false, fetchErr
		}

		return observed.Pypi.GetIndexPath() == "/simple", nil
	})
	if err != nil {
		observed, _ := f.FetchPypiProxyRepo(repoName)
		t.Fatalf("pypi proxy repo did not converge on the /simple index path: %v\n  nexus reports: %+v", err, observed)
	}
}
