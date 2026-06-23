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

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

const defaultBlobStore = "default"

func newProviderConfigRef(name string) *xpv2.ProviderConfigReference {
	return &xpv2.ProviderConfigReference{Kind: "ProviderConfig", Name: name}
}

func TestMavenHostedRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-maven-hosted"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "maven2",
				Type:   "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
				Maven: &contentv1alpha1.MavenConfig{
					VersionPolicy: ptrTo("RELEASE"),
					LayoutPolicy:  ptrTo("STRICT"),
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchMavenHostedRepo(repoName)
	if err != nil {
		t.Fatalf("fetching maven hosted repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("maven hosted repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestMavenProxyRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-maven-proxy"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "maven2",
				Type:   "proxy",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
				Maven: &contentv1alpha1.MavenConfig{
					VersionPolicy: ptrTo("RELEASE"),
					LayoutPolicy:  ptrTo("STRICT"),
				},
				Proxy: &contentv1alpha1.ProxyConfig{
					RemoteURL: "https://repo1.maven.org/maven2/",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchMavenProxyRepo(repoName)
	if err != nil {
		t.Fatalf("fetching maven proxy repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("maven proxy repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestNpmHostedRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-npm-hosted"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "npm",
				Type:   "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchNpmHostedRepo(repoName)
	if err != nil {
		t.Fatalf("fetching npm hosted repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("npm hosted repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestDockerHostedRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-docker-hosted"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "docker",
				Type:   "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
				Docker: &contentv1alpha1.DockerConfig{
					V1Enabled:      ptrTo(false),
					ForceBasicAuth: ptrTo(true),
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchDockerHostedRepo(repoName)
	if err != nil {
		t.Fatalf("fetching docker hosted repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("docker hosted repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestHelmHostedRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-helm-hosted"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "helm",
				Type:   "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchHelmHostedRepo(repoName)
	if err != nil {
		t.Fatalf("fetching helm hosted repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("helm hosted repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestHelmProxyRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-helm-proxy"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "helm",
				Type:   "proxy",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
				Proxy: &contentv1alpha1.ProxyConfig{
					RemoteURL: "https://charts.helm.sh/stable/",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchHelmProxyRepo(repoName)
	if err != nil {
		t.Fatalf("fetching helm proxy repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("helm proxy repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

func TestPypiProxyRepository(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-pypi-proxy"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:   repoName,
				Format: "pypi",
				Type:   "proxy",
				Storage: &contentv1alpha1.RepositoryStorage{
					BlobStoreName: defaultBlobStore,
				},
				Proxy: &contentv1alpha1.ProxyConfig{
					RemoteURL: "https://pypi.org",
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, 2*time.Minute)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchPypiProxyRepo(repoName)
	if err != nil {
		t.Fatalf("fetching pypi proxy repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("pypi proxy repo %q not found in Nexus", repoName)
	}
	if got.Name != repoName {
		t.Errorf("repo name = %q, want %q", got.Name, repoName)
	}
}

// TestMavenGroupRepository creates the necessary member repos first, then
// creates the group repo. It does NOT run in parallel since it creates
// multiple interrelated resources that would conflict with other repo tests
// if they used the same names.
func TestMavenGroupRepository(t *testing.T) {
	f := e2e.New(t)

	const (
		hostedName = "e2e-group-maven-hosted"
		proxyName  = "e2e-group-maven-proxy"
		groupName  = "e2e-test-maven-group"
	)

	hostedRepo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: hostedName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:    hostedName,
				Format:  "maven2",
				Type:    "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				Maven:   &contentv1alpha1.MavenConfig{VersionPolicy: ptrTo("RELEASE")},
			},
		},
	}
	f.CreateAndWaitForReady(t, hostedRepo, 2*time.Minute)

	proxyRepo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: proxyName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:    proxyName,
				Format:  "maven2",
				Type:    "proxy",
				Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				Maven:   &contentv1alpha1.MavenConfig{VersionPolicy: ptrTo("RELEASE")},
				Proxy:   &contentv1alpha1.ProxyConfig{RemoteURL: "https://repo1.maven.org/maven2/"},
			},
		},
	}
	f.CreateAndWaitForReady(t, proxyRepo, 2*time.Minute)

	groupRepo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: groupName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:    groupName,
				Format:  "maven2",
				Type:    "group",
				Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				Group:   &contentv1alpha1.GroupConfig{MemberNames: []string{hostedName, proxyName}},
			},
		},
	}
	f.CreateAndWaitForReady(t, groupRepo, 2*time.Minute)
	e2e.AssertReady(t, groupRepo)
	e2e.AssertSynced(t, groupRepo)

	got, err := f.FetchMavenGroupRepo(groupName)
	if err != nil {
		t.Fatalf("fetching maven group repo from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("maven group repo %q not found in Nexus", groupName)
	}
	if got.Name != groupName {
		t.Errorf("repo name = %q, want %q", got.Name, groupName)
	}
}
