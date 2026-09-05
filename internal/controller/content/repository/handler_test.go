package repository

import (
	"errors"
	"slices"
	"testing"

	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// fakeAPI is a repoAPI backed by an in-memory repository.
type fakeAPI struct {
	repo   *schema.RawHostedRepository
	getErr error
	// updated records the payload the last Update call submitted.
	updated *schema.RawHostedRepository
}

// Get implements repoAPI.
func (f *fakeAPI) Get(_ string) (*schema.RawHostedRepository, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.repo, nil
}

// Create implements repoAPI.
func (f *fakeAPI) Create(repo schema.RawHostedRepository) error {
	f.repo = &repo

	return nil
}

// Update implements repoAPI.
func (f *fakeAPI) Update(_ string, repo schema.RawHostedRepository) error {
	f.updated = &repo
	f.repo = &repo

	return nil
}

// Delete implements repoAPI.
func (f *fakeAPI) Delete(_ string) error {
	f.repo = nil

	return nil
}

// newRawOps builds typedOps for raw hosted repositories backed by api.
func newRawOps(api *fakeAPI) typedOps[schema.RawHostedRepository] {
	return typedOps[schema.RawHostedRepository]{
		api:   func(_ *nexusAPI) repoAPI[schema.RawHostedRepository] { return api },
		build: buildRawHosted,
	}
}

// newRawDesired builds the desired state of a raw hosted repository.
func newRawDesired(blobStore string) desiredRepo {
	return desiredRepo{repo: &repositoryv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "raw-hosted"},
		Spec: repositoryv1alpha1.RepositorySpec{
			ForProvider: repositoryv1alpha1.RepositoryParameters{
				Name:    "raw-hosted",
				Format:  "raw",
				Type:    repoTypeHosted,
				Storage: &repositoryv1alpha1.RepositoryStorage{BlobStoreName: blobStore},
			},
		},
	}}
}

// TestTypedOpsObserveDistinguishesMissingFromFailing checks that a missing
// repository is reported as absent while any other failure is returned. A read
// failure reported as "absent" would make Crossplane recreate a repository
// that already exists.
func TestTypedOpsObserveDistinguishesMissingFromFailing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		api        *fakeAPI
		wantExists bool
		wantErr    bool
	}{
		{
			name: "Missing",
			api:  &fakeAPI{getErr: errors.New("could not read repository 'raw-hosted': HTTP: 404, Repository not found")},
		},
		{
			name:    "Unreachable",
			api:     &fakeAPI{getErr: errors.New("dial tcp: connection refused")},
			wantErr: true,
		},
		{
			name:    "Unauthorized",
			api:     &fakeAPI{getErr: errors.New("could not read repository 'raw-hosted': HTTP: 401, ")},
			wantErr: true,
		},
		{
			name:       "Present",
			api:        &fakeAPI{repo: &schema.RawHostedRepository{Name: "raw-hosted"}},
			wantExists: true,
		},
		{
			name: "NilWithoutError",
			api:  &fakeAPI{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state, err := newRawOps(tt.api).Observe(nil, "raw-hosted", newRawDesired("default"))

			if (err != nil) != tt.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && state.exists != tt.wantExists {
				t.Errorf("Observe() exists = %v, want %v", state.exists, tt.wantExists)
			}
		})
	}
}

// TestTypedOpsRoundTrip checks that a created repository observes as up to
// date, that a spec change is detected, and that Update submits it.
func TestTypedOpsRoundTrip(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{}
	ops := newRawOps(api)

	err := ops.Create(nil, newRawDesired("default"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	state, err := ops.Observe(nil, "raw-hosted", newRawDesired("default"))
	if err != nil || !state.exists || !state.upToDate {
		t.Fatalf("Observe() after create = %+v (err %v), want exists and up to date", state, err)
	}

	if got := generateRepositoryObservation(state.fields, "https://nexus.example.com/repository/raw-hosted"); got.BlobStoreName != "default" {
		t.Errorf("observation blobStoreName = %q, want %q", got.BlobStoreName, "default")
	}

	state, err = ops.Observe(nil, "raw-hosted", newRawDesired("other-blobstore"))
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if state.upToDate {
		t.Fatal("Observe() upToDate = true, want false: the blob store changed")
	}

	err = ops.Update(nil, "raw-hosted", newRawDesired("other-blobstore"))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if api.updated == nil || api.updated.Storage.BlobStoreName != "other-blobstore" {
		t.Fatalf("Update() submitted %+v, want blobStoreName other-blobstore", api.updated)
	}

	state, err = ops.Observe(nil, "raw-hosted", newRawDesired("other-blobstore"))
	if err != nil || !state.upToDate {
		t.Errorf("Observe() after update = %+v (err %v), want up to date", state, err)
	}
}

// TestDispatcherRejectsUnsupportedType checks the error raised for a
// format/type combination Nexus does not offer.
func TestDispatcherRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	handler := getHandler("gitlfs")
	if handler == nil {
		t.Fatal("gitlfs is not registered")
	}

	err := handler.Delete(nil, "lfs", repoTypeGroup)
	if err == nil {
		t.Fatal("Delete() on an unsupported type returned no error")
	}

	if !slices.Equal(handler.SupportedTypes(), []string{repoTypeHosted}) {
		t.Errorf("SupportedTypes() = %v, want [hosted]", handler.SupportedTypes())
	}
}

// TestRegisteredFormats checks the registry covers every format the CRD
// accepts, and that each format declares the types Nexus offers for it.
func TestRegisteredFormats(t *testing.T) {
	t.Parallel()

	wantTypes := map[string][]string{
		"apt":         {repoTypeHosted, repoTypeProxy},
		"bower":       {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"cargo":       {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"cocoapods":   {repoTypeProxy},
		"conan":       {repoTypeHosted, repoTypeProxy},
		"conda":       {repoTypeProxy},
		"docker":      {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"gitlfs":      {repoTypeHosted},
		"go":          {repoTypeGroup, repoTypeProxy},
		"helm":        {repoTypeHosted, repoTypeProxy},
		"huggingface": {repoTypeProxy},
		"maven2":      {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"npm":         {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"nuget":       {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"p2":          {repoTypeProxy},
		"pypi":        {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"r":           {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"raw":         {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"rubygems":    {repoTypeGroup, repoTypeHosted, repoTypeProxy},
		"yum":         {repoTypeGroup, repoTypeHosted, repoTypeProxy},
	}

	for format, types := range wantTypes {
		handler := getHandler(format)
		if handler == nil {
			t.Errorf("format %q is not registered", format)

			continue
		}

		if !slices.Equal(handler.SupportedTypes(), types) {
			t.Errorf("format %q supports %v, want %v", format, handler.SupportedTypes(), types)
		}
	}

	for _, format := range supportedFormats() {
		_, expected := wantTypes[format]
		if !expected {
			t.Errorf("format %q is registered but not expected by this test", format)
		}
	}

	if getHandler("does-not-exist") != nil {
		t.Error("getHandler() returned a handler for an unknown format")
	}
}

// TestBuildersApplyFormatSpecificSettings checks the format specific spec
// fields reach the payload. These were silently dropped before.
func TestBuildersApplyFormatSpecificSettings(t *testing.T) {
	t.Parallel()

	desired := desiredRepo{repo: &repositoryv1alpha1.Repository{
		Spec: repositoryv1alpha1.RepositorySpec{
			ForProvider: repositoryv1alpha1.RepositoryParameters{
				Name:        "repo",
				Cargo:       &repositoryv1alpha1.CargoConfig{RequireAuthentication: new(true)},
				Npm:         &repositoryv1alpha1.NpmConfig{RemoveQuarantined: new(true)},
				YumSigning:  &repositoryv1alpha1.YumSigningConfig{Keypair: new("key")},
				RoutingRule: new("block-all"),
				DockerProxy: &repositoryv1alpha1.DockerProxyConfig{
					IndexType:          new("CUSTOM"),
					IndexURL:           new("https://index.example.com"),
					CacheForeignLayers: new(true),
				},
			},
		},
	}}

	if got := buildCargoProxy(desired); !got.RequireAuthentication {
		t.Error("buildCargoProxy() dropped cargo.requireAuthentication")
	}

	if got := buildCargoGroup(desired); !got.RequireAuthentication {
		t.Error("buildCargoGroup() dropped cargo.requireAuthentication")
	}

	if got := buildNpmProxy(desired); got.Npm == nil || !got.RemoveQuarantined {
		t.Error("buildNpmProxy() dropped npm.removeQuarantined")
	}

	if got := buildYumProxy(desired); got.YumSigning == nil || ptr.Deref(got.Keypair, "") != "key" {
		t.Error("buildYumProxy() dropped yumSigning.keypair")
	}

	if got := buildAptProxy(desired); ptr.Deref(got.RoutingRule, "") != "block-all" {
		t.Error("buildAptProxy() dropped routingRule")
	}

	dockerProxy := buildDockerProxy(desired)
	if dockerProxy.IndexType != schema.DockerProxyIndexTypeCustom {
		t.Errorf("buildDockerProxy() indexType = %q, want CUSTOM", dockerProxy.IndexType)
	}

	if ptr.Deref(dockerProxy.IndexURL, "") != "https://index.example.com" {
		t.Error("buildDockerProxy() dropped dockerProxy.indexUrl")
	}

	if !ptr.Deref(dockerProxy.CacheForeignLayers, false) {
		t.Error("buildDockerProxy() dropped dockerProxy.cacheForeignLayers")
	}
}

// TestBuildersApplyDefaults checks the defaults used when the spec leaves a
// shared block out.
func TestBuildersApplyDefaults(t *testing.T) {
	t.Parallel()

	desired := desiredRepo{repo: &repositoryv1alpha1.Repository{
		Spec: repositoryv1alpha1.RepositorySpec{
			ForProvider: repositoryv1alpha1.RepositoryParameters{Name: "repo"},
		},
	}}

	hostedRepo := buildRawHosted(desired)
	if hostedRepo.Storage.BlobStoreName != defaultBlobStoreName || !hostedRepo.Online {
		t.Errorf("buildRawHosted() = %+v, want the default blob store and online", hostedRepo.Storage)
	}

	if hostedRepo.Cleanup != nil {
		t.Error("buildRawHosted() set a cleanup block the spec does not declare")
	}

	proxyRepo := buildRawProxy(desired)
	if proxyRepo.ContentMaxAge != defaultMaxAge || proxyRepo.MetadataMaxAge != defaultMaxAge {
		t.Errorf("buildRawProxy() proxy = %+v, want the default max ages", proxyRepo.Proxy)
	}

	if !proxyRepo.Enabled || proxyRepo.TTL != defaultNegativeCacheTTL {
		t.Errorf("buildRawProxy() negativeCache = %+v, want enabled with the default TTL", proxyRepo.NegativeCache)
	}

	if !proxyRepo.AutoBlock || proxyRepo.Blocked {
		t.Errorf("buildRawProxy() httpClient = %+v, want autoBlock and not blocked", proxyRepo.HTTPClient)
	}

	nugetRepo := buildNugetProxy(desired)
	if nugetRepo.QueryCacheItemMaxAge != defaultNugetQueryCacheItemMaxAge || nugetRepo.NugetVersion != schema.NugetVersion3 {
		t.Errorf("buildNugetProxy() nugetProxy = %+v, want the CRD defaults", nugetRepo.NugetProxy)
	}
}
