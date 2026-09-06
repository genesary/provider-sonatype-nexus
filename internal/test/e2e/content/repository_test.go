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
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// reconcileTimeout bounds how long a spec change may take to reach Nexus.
const reconcileTimeout = 2 * time.Minute

// updateRepository re-reads the Repository and applies mutate to its spec,
// retrying on conflict.
func updateRepository(t *testing.T, f *e2e.Framework, repo *contentv1alpha1.Repository, mutate func(spec *contentv1alpha1.RepositoryParameters)) {
	t.Helper()

	ctx := context.Background()
	key := client.ObjectKeyFromObject(repo)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := f.Kube.Get(ctx, key, repo); err != nil {
			return err
		}
		mutate(&repo.Spec.ForProvider)
		return f.Kube.Update(ctx, repo)
	})
	if err != nil {
		t.Fatalf("updating %s: %v", repo.Name, err)
	}
}

// TestAptHostedRepositoryUpdate checks that a change to an APT specific field
// reaches Nexus. Observation used to compare only the repository name and its
// online flag, so every other field silently never updated.
func TestAptHostedRepositoryUpdate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-apt-hosted-update"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:       repoName,
				Format:     "apt",
				Type:       "hosted",
				Storage:    &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				Apt:        &contentv1alpha1.AptConfig{Distribution: ptrTo("bookworm")},
				AptSigning: &contentv1alpha1.AptSigningConfig{Keypair: "e2e-test-keypair"},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	got, err := f.FetchAptHostedRepo(repoName)
	if err != nil || got == nil {
		t.Fatalf("fetching apt hosted repo from Nexus: %v", err)
	}
	if got.Apt.Distribution != "bookworm" {
		t.Fatalf("distribution after create = %q, want %q", got.Apt.Distribution, "bookworm")
	}

	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Apt.Distribution = ptrTo("trixie")
		spec.Storage.StrictContentTypeValidation = ptrTo(false)
	})

	err = f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, err := f.FetchAptHostedRepo(repoName)
		if err != nil || observed == nil {
			return false, err
		}
		return observed.Apt.Distribution == "trixie" && !observed.Storage.StrictContentTypeValidation, nil
	})
	if err != nil {
		observed, _ := f.FetchAptHostedRepo(repoName)
		t.Fatalf("apt hosted repo did not converge on the updated spec: %v\n  nexus reports: %+v", err, observed)
	}
}

// TestCargoRepositoryUpdate checks that changes to the shared and to the Cargo
// specific parts of the spec both reach Nexus.
func TestCargoRepositoryUpdate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const repoName = "e2e-test-cargo-proxy-update"

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: repoName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:    repoName,
				Format:  "cargo",
				Type:    "proxy",
				Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
				Proxy: &contentv1alpha1.ProxyConfig{
					RemoteURL:     "https://crates.io",
					ContentMaxAge: ptrTo(int32(1440)),
				},
				Cargo: &contentv1alpha1.CargoConfig{RequireAuthentication: ptrTo(false)},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)

	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Proxy.ContentMaxAge = ptrTo(int32(60))
		spec.Cargo.RequireAuthentication = ptrTo(true)
	})

	err := f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, err := f.FetchCargoProxyRepo(repoName)
		if err != nil || observed == nil {
			return false, err
		}
		return observed.ContentMaxAge == 60 && observed.RequireAuthentication, nil
	})
	if err != nil {
		observed, _ := f.FetchCargoProxyRepo(repoName)
		t.Fatalf("cargo proxy repo did not converge on the updated spec: %v\n  nexus reports: %+v", err, observed)
	}
}

// TestRepositoryNameDiffersFromObjectName checks that the repository the
// provider manages is the one named by spec.forProvider.name.
// crossplane-runtime seeds crossplane.io/external-name from metadata.name, so
// when the two differ the provider must not go looking for a repository named
// after the Kubernetes object.
func TestRepositoryNameDiffersFromObjectName(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const (
		objectName = "e2e-test-object-name"
		repoName   = "e2e-test-nexus-name"
	)

	repo := &contentv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: objectName, Namespace: "default"},
		Spec: contentv1alpha1.RepositorySpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
			},
			ForProvider: contentv1alpha1.RepositoryParameters{
				Name:    repoName,
				Format:  "cargo",
				Type:    "hosted",
				Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
			},
		},
	}

	f.CreateAndWaitForReady(t, repo, reconcileTimeout)
	e2e.AssertReady(t, repo)
	e2e.AssertSynced(t, repo)
	e2e.AssertExternalName(t, repo, repoName)

	got, err := f.FetchCargoHostedRepo(repoName)
	if err != nil || got == nil {
		t.Fatalf("fetching cargo hosted repo %q from Nexus: %v", repoName, err)
	}

	if repo.Status.AtProvider.URL == nil || *repo.Status.AtProvider.URL == "" {
		t.Error("status.atProvider.url is not reported")
	}

	// A second reconcile must observe the same repository rather than trying
	// to create it again.
	updateRepository(t, f, repo, func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Online = ptrTo(false)
	})

	err = f.WaitForNexus(context.Background(), reconcileTimeout, func() (bool, error) {
		observed, err := f.FetchCargoHostedRepo(repoName)
		if err != nil || observed == nil {
			return false, err
		}
		return !observed.Online, nil
	})
	if err != nil {
		t.Fatalf("cargo hosted repo did not converge on online=false: %v", err)
	}
}

// TestRepositoryObservedState checks that status.atProvider reports what Nexus
// actually holds, for each of the three repository types. The observation is
// built format-agnostically, so covering the three types covers every format.
func TestRepositoryObservedState(t *testing.T) {
	f := e2e.New(t)

	const (
		hostedName = "e2e-observed-maven-hosted"
		proxyName  = "e2e-observed-maven-proxy"
		groupName  = "e2e-observed-maven-group"
		remoteURL  = "https://repo1.maven.org/maven2/"
	)

	newRepo := func(name, repoType string, mutate func(spec *contentv1alpha1.RepositoryParameters)) *contentv1alpha1.Repository {
		repo := &contentv1alpha1.Repository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: contentv1alpha1.RepositorySpec{
				ManagedResourceSpec: xpv2.ManagedResourceSpec{
					ProviderConfigReference: newProviderConfigRef(f.ProviderConfigName),
				},
				ForProvider: contentv1alpha1.RepositoryParameters{
					Name:    name,
					Format:  "maven2",
					Type:    repoType,
					Storage: &contentv1alpha1.RepositoryStorage{BlobStoreName: defaultBlobStore},
					Maven:   &contentv1alpha1.MavenConfig{VersionPolicy: ptrTo("RELEASE"), LayoutPolicy: ptrTo("STRICT")},
				},
			},
		}
		mutate(&repo.Spec.ForProvider)

		return repo
	}

	hosted := newRepo(hostedName, "hosted", func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Storage.WritePolicy = ptrTo("ALLOW")
	})
	f.CreateAndWaitForReady(t, hosted, reconcileTimeout)

	proxy := newRepo(proxyName, "proxy", func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Proxy = &contentv1alpha1.ProxyConfig{RemoteURL: remoteURL}
	})
	f.CreateAndWaitForReady(t, proxy, reconcileTimeout)

	group := newRepo(groupName, "group", func(spec *contentv1alpha1.RepositoryParameters) {
		spec.Maven = nil
		spec.Group = &contentv1alpha1.GroupConfig{MemberNames: []string{hostedName, proxyName}}
	})
	f.CreateAndWaitForReady(t, group, reconcileTimeout)

	// Fields every repository reports, whatever its format and type.
	for _, repo := range []*contentv1alpha1.Repository{hosted, proxy, group} {
		observed := repo.Status.AtProvider

		if observed.URL == nil || *observed.URL == "" {
			t.Errorf("%s: atProvider.url is not reported", repo.Name)
		}

		if observed.Name != repo.Spec.ForProvider.Name {
			t.Errorf("%s: atProvider.name = %q, want %q", repo.Name, observed.Name, repo.Spec.ForProvider.Name)
		}

		if observed.Online == nil || !*observed.Online {
			t.Errorf("%s: atProvider.online = %v, want true", repo.Name, observed.Online)
		}

		if observed.BlobStoreName != defaultBlobStore {
			t.Errorf("%s: atProvider.blobStoreName = %q, want %q", repo.Name, observed.BlobStoreName, defaultBlobStore)
		}

		if observed.StrictContentTypeValidation == nil {
			t.Errorf("%s: atProvider.strictContentTypeValidation is not reported", repo.Name)
		}
	}

	// Fields that only make sense for one repository type.
	if got := hosted.Status.AtProvider.WritePolicy; got != "ALLOW" {
		t.Errorf("hosted: atProvider.writePolicy = %q, want ALLOW", got)
	}

	if got := proxy.Status.AtProvider.RemoteURL; got != remoteURL {
		t.Errorf("proxy: atProvider.remoteUrl = %q, want %q", got, remoteURL)
	}

	if got := group.Status.AtProvider.MemberNames; len(got) != 2 {
		t.Errorf("group: atProvider.memberNames = %v, want both members", got)
	}

	if got := hosted.Status.AtProvider.RemoteURL; got != "" {
		t.Errorf("hosted: atProvider.remoteUrl = %q, want it left empty", got)
	}

	if got := proxy.Status.AtProvider.MemberNames; got != nil {
		t.Errorf("proxy: atProvider.memberNames = %v, want it left empty", got)
	}
}
