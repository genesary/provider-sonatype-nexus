package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	pkgerrors "github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// fakeAPI is an in-memory Nexus serving a single raw hosted repository.
type fakeAPI struct {
	repo   *v395.RawHostedApiRepository
	getErr error
	// updated records the payload the last update submitted.
	updated *v395.RawHostedRepositoryApiRequest
}

// store records repo the way Nexus reports it back.
func (f *fakeAPI) store(repo v395.RawHostedRepositoryApiRequest) {
	f.repo = &v395.RawHostedApiRepository{
		Name:    repo.Name,
		Online:  repo.Online,
		Storage: repo.Storage,
		Cleanup: repo.Cleanup,
		Format:  "raw",
		Type:    repoTypeHosted,
		Url:     "https://nexus.example.com/repository/" + repo.Name,
	}
}

// newRawOps builds typedOps for raw hosted repositories backed by api.
func newRawOps(api *fakeAPI) typedOps[v395.RawHostedRepositoryApiRequest, v395.RawHostedApiRepository] {
	return typedOps[v395.RawHostedRepositoryApiRequest, v395.RawHostedApiRepository]{
		get: func(_ context.Context, _ *nexusAPI, _ string) (*v395.RawHostedApiRepository, error) {
			if api.getErr != nil {
				return nil, api.getErr
			}

			return api.repo, nil
		},
		create: func(_ context.Context, _ *nexusAPI, _ string, repo v395.RawHostedRepositoryApiRequest) error {
			api.store(repo)

			return nil
		},
		update: func(_ context.Context, _ *nexusAPI, _ string, repo v395.RawHostedRepositoryApiRequest) error {
			api.updated = &repo
			api.store(repo)

			return nil
		},
		build: buildHosted[v395.RawHostedRepositoryApiRequest],
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
			api:  &fakeAPI{getErr: pkgerrors.Wrap(nexus.ErrNotFound, "404 Not Found")},
		},
		{
			name:    "Unreachable",
			api:     &fakeAPI{getErr: errors.New("dial tcp: connection refused")},
			wantErr: true,
		},
		{
			name:    "Unauthorized",
			api:     &fakeAPI{getErr: errors.New("401 Unauthorized")},
			wantErr: true,
		},
		{
			name:       "Present",
			api:        &fakeAPI{repo: &v395.RawHostedApiRepository{Name: "raw-hosted"}},
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

			state, err := newRawOps(tt.api).Observe(context.Background(), nil, "raw-hosted", newRawDesired("default"))

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

	err := ops.Create(context.Background(), nil, newRawDesired("default"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	state, err := ops.Observe(context.Background(), nil, "raw-hosted", newRawDesired("default"))
	if err != nil || !state.exists || !state.upToDate {
		t.Fatalf("Observe() after create = %+v (err %v), want exists and up to date", state, err)
	}

	if got := generateRepositoryObservation(state.fields, "https://nexus.example.com/repository/raw-hosted"); got.BlobStoreName != "default" {
		t.Errorf("observation blobStoreName = %q, want %q", got.BlobStoreName, "default")
	}

	state, err = ops.Observe(context.Background(), nil, "raw-hosted", newRawDesired("other-blobstore"))
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if state.upToDate {
		t.Fatal("Observe() upToDate = true, want false: the blob store changed")
	}

	err = ops.Update(context.Background(), nil, "raw-hosted", newRawDesired("other-blobstore"))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if api.updated == nil || api.updated.Storage.BlobStoreName != "other-blobstore" {
		t.Fatalf("Update() submitted %+v, want blobStoreName other-blobstore", api.updated)
	}

	state, err = ops.Observe(context.Background(), nil, "raw-hosted", newRawDesired("other-blobstore"))
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

	err := handler.Delete(context.Background(), nil, "lfs", repoTypeGroup)
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

	all := []string{repoTypeGroup, repoTypeHosted, repoTypeProxy}
	wantTypes := map[string][]string{
		"alpine":        all,
		"ansiblegalaxy": all,
		"apt":           {repoTypeHosted, repoTypeProxy},
		"cargo":         all,
		"cocoapods":     {repoTypeProxy},
		"composer":      all,
		"conan":         all,
		"conda":         all,
		"docker":        all,
		"gitlfs":        {repoTypeHosted},
		"go":            all,
		"helm":          all,
		"huggingface":   {repoTypeProxy},
		"maven2":        all,
		"npm":           all,
		"nuget":         all,
		"oci":           all,
		"p2":            {repoTypeProxy},
		"pub":           all,
		"pypi":          all,
		"r":             all,
		"raw":           all,
		"rubygems":      all,
		"swift":         all,
		"terraform":     all,
		"yum":           all,
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
				Cargo:       &repositoryv1alpha1.RequireAuthenticationConfig{RequireAuthentication: new(true)},
				YumSigning:  &repositoryv1alpha1.YumSigningConfig{Keypair: new("key")},
				RoutingRule: new("block-all"),
				DockerProxy: &repositoryv1alpha1.ContainerProxyConfig{
					IndexType:          new("CUSTOM"),
					IndexURL:           new("https://index.example.com"),
					CacheForeignLayers: new(true),
				},
			},
		},
	}}

	if got := buildCargoProxy(desired); !got.Cargo.GetRequireAuthentication() {
		t.Error("buildCargoProxy() dropped cargo.requireAuthentication")
	}

	if got := buildCargoGroup(desired); !got.Cargo.GetRequireAuthentication() {
		t.Error("buildCargoGroup() dropped cargo.requireAuthentication")
	}

	if got := buildYumProxy(desired); got.YumSigning == nil || got.YumSigning.Keypair != "key" {
		t.Error("buildYumProxy() dropped yumSigning.keypair")
	}

	if got := buildAptProxy(desired); ptr.Deref(got.RoutingRuleName, "") != "block-all" {
		t.Error("buildAptProxy() dropped routingRule")
	}

	dockerProxy := buildDockerProxy(desired).DockerProxy
	if dockerProxy.IndexType != "CUSTOM" {
		t.Errorf("buildDockerProxy() indexType = %q, want CUSTOM", dockerProxy.IndexType)
	}

	if ptr.Deref(dockerProxy.IndexUrl, "") != "https://index.example.com" {
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

	hostedRepo := buildHosted[v395.RawHostedRepositoryApiRequest](desired)
	if hostedRepo.Storage.BlobStoreName != defaultBlobStoreName || !hostedRepo.Online {
		t.Errorf("buildHosted() = %+v, want the default blob store and online", hostedRepo.Storage)
	}

	if hostedRepo.Cleanup != nil {
		t.Error("buildHosted() set a cleanup block the spec does not declare")
	}

	proxyRepo := buildProxy[v395.RawProxyRepositoryApiRequest](desired)
	if proxyRepo.Proxy.ContentMaxAge != defaultMaxAge || proxyRepo.Proxy.MetadataMaxAge != defaultMaxAge {
		t.Errorf("buildProxy() proxy = %+v, want the default max ages", proxyRepo.Proxy)
	}

	if !proxyRepo.NegativeCache.Enabled || proxyRepo.NegativeCache.TimeToLive != defaultNegativeCacheTTL {
		t.Errorf("buildProxy() negativeCache = %+v, want enabled with the default TTL", proxyRepo.NegativeCache)
	}

	if !proxyRepo.HttpClient.GetAutoBlock() || proxyRepo.HttpClient.GetBlocked() {
		t.Errorf("buildProxy() httpClient = %+v, want autoBlock and not blocked", proxyRepo.HttpClient)
	}

	nugetRepo := buildNugetProxy(desired).NugetProxy
	if nugetRepo.GetQueryCacheItemMaxAge() != defaultNugetQueryCacheItemMaxAge || nugetRepo.GetNugetVersion() != defaultNugetVersion {
		t.Errorf("buildNugetProxy() nugetProxy = %+v, want the CRD defaults", nugetRepo)
	}
}

// TestFormatBindingsReachTheirEndpoints drives every registered format and
// repository type through a fake Nexus, and checks each operation reaches the
// endpoint of that format and type with that repository's payload. The
// bindings are type checked, but nothing else stops a type from being wired
// to the endpoints of another one.
func TestFormatBindingsReachTheirEndpoints(t *testing.T) {
	t.Parallel()

	for _, format := range supportedFormats() {
		for _, repoType := range getHandler(format).SupportedTypes() {
			t.Run(format+"/"+repoType, func(t *testing.T) {
				t.Parallel()

				recorder := &requestRecorder{}
				server := httptest.NewServer(recorder)
				t.Cleanup(server.Close)

				client, err := nexus.NewClient(nexus.Credentials{URL: server.URL})
				if err != nil {
					t.Fatalf("NewClient() error = %v", err)
				}

				api := client.RepositoryManagementAPI
				handler := getHandler(format)
				desired := newDesired(format, repoType)
				name := desired.name()

				segment := format
				if format == "maven2" {
					segment = "maven"
				}

				endpoint := "/service/rest/v1/repositories/" + segment + "/" + repoType

				state, err := handler.Observe(context.Background(), api, name, repoType, desired)
				if err != nil || state.exists {
					t.Errorf("Observe() = %+v (err %v), want a missing repository", state, err)
				}

				recorder.expect(t, http.MethodGet, endpoint+"/"+name, "")

				err = handler.Create(context.Background(), api, repoType, desired)
				if err != nil {
					t.Errorf("Create() error = %v", err)
				}

				recorder.expect(t, http.MethodPost, endpoint, name)

				err = handler.Update(context.Background(), api, name, repoType, desired)
				if err != nil {
					t.Errorf("Update() error = %v", err)
				}

				recorder.expect(t, http.MethodPut, endpoint+"/"+name, name)

				err = handler.Delete(context.Background(), api, name, repoType)
				if err != nil {
					t.Errorf("Delete() error = %v", err)
				}

				recorder.expect(t, http.MethodDelete, "/service/rest/v1/repositories/"+name, "")
			})
		}
	}
}

// requestRecorder is a fake Nexus that records the last request it served. It
// reports every repository as missing and accepts every write.
type requestRecorder struct {
	mu     sync.Mutex
	method string
	path   string
	body   map[string]any
}

// ServeHTTP records the request and answers like Nexus would.
func (r *requestRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.method, r.path, r.body = req.Method, req.URL.Path, nil
	_ = json.NewDecoder(req.Body).Decode(&r.body)

	switch req.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusNotFound)
	case http.MethodPost:
		w.WriteHeader(http.StatusCreated)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// expect checks the last request against the method and path an operation
// should have used, and, for a write, that it submitted the named repository.
func (r *requestRecorder) expect(t *testing.T, method, path, name string) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.method != method || r.path != path {
		t.Errorf("request = %s %s, want %s %s", r.method, r.path, method, path)
	}

	if name != "" && r.body["name"] != name {
		t.Errorf("%s %s submitted name %v, want %q", method, path, r.body["name"], name)
	}
}

// newDesired builds the desired state of a repository of format and type with
// only the settings the CRD requires.
func newDesired(format, repoType string) desiredRepo {
	params := repositoryv1alpha1.RepositoryParameters{
		Name:   format + "-" + repoType,
		Format: format,
		Type:   repoType,
	}

	return desiredRepo{repo: &repositoryv1alpha1.Repository{Spec: repositoryv1alpha1.RepositorySpec{ForProvider: params}}}
}

// TestNewFormatBuilders checks the settings specific to the formats the
// generated client added reach their payloads, with their defaults.
func TestNewFormatBuilders(t *testing.T) {
	t.Parallel()

	empty := newDesired("any", repoTypeProxy)

	configured := newDesired("any", repoTypeProxy)
	configured.repo.Spec.ForProvider = repositoryv1alpha1.RepositoryParameters{
		Name:             "repo",
		AlpineSigning:    &repositoryv1alpha1.AlpineSigningConfig{Keypair: "rsa", Passphrase: new("secret")},
		TerraformSigning: &repositoryv1alpha1.TerraformSigningConfig{Keypair: "pgp"},
		Swift:            &repositoryv1alpha1.RequireAuthenticationConfig{RequireAuthentication: new(true)},
		Terraform:        &repositoryv1alpha1.RequireAuthenticationConfig{RequireAuthentication: new(true)},
		Oci:              &repositoryv1alpha1.ContainerConfig{PathEnabled: new(true), HTTPPort: new(int32(8082))},
		OciProxy:         &repositoryv1alpha1.ContainerProxyConfig{IndexType: new("CUSTOM"), IndexURL: new("https://index.example.com")},
		Cosign:           &repositoryv1alpha1.CosignConfig{Enforcement: "KEYLESS", IssuerRegex: new("^https://token.actions.githubusercontent.com$")},
	}

	if got := buildAlpineHosted(empty); got.AlpineSigning != nil {
		t.Error("buildAlpineHosted() set a signing key the spec does not declare")
	}

	if got := buildAlpineProxy(configured).AlpineSigning; got.Keypair != "rsa" || ptr.Deref(got.Passphrase, "") != "secret" {
		t.Errorf("buildAlpineProxy() alpineSigning = %+v, want the spec key", got)
	}

	if got := buildAlpineGroup(configured).AlpineSigning; got.Keypair != "rsa" {
		t.Errorf("buildAlpineGroup() alpineSigning = %+v, want the spec key", got)
	}

	if got := buildTerraformHosted(configured).TerraformSigning; got.Keypair != "pgp" || got.Passphrase != nil {
		t.Errorf("buildTerraformHosted() terraformSigning = %+v, want the spec key", got)
	}

	if buildSwiftProxy(empty).Swift.GetRequireAuthentication() || buildTerraformGroup(empty).Terraform.GetRequireAuthentication() {
		t.Error("requireAuthentication defaults to true, want false")
	}

	if !buildSwiftProxy(configured).Swift.GetRequireAuthentication() {
		t.Error("buildSwiftProxy() dropped swift.requireAuthentication")
	}

	if !buildTerraformGroup(configured).Terraform.GetRequireAuthentication() {
		t.Error("buildTerraformGroup() dropped terraform.requireAuthentication")
	}

	// Nexus resets a block a PUT leaves out, so the defaults are always sent
	// and compared, which catches settings changed outside of the provider.
	if got := buildOciHosted(empty); !got.Oci.ForceBasicAuth || got.Oci.V1Enabled || got.Cosign.GetEnforcement() != "NONE" {
		t.Errorf("buildOciHosted() = %+v, want forceBasicAuth, no V1 API and no cosign enforcement", got)
	}

	if got := buildOciProxy(empty).OciProxy; ptr.Deref(got.IndexType, "") != defaultOciIndexType || got.IndexUrl != nil {
		t.Errorf("buildOciProxy() ociProxy = %+v, want the REGISTRY index Nexus defaults to", got)
	}

	if got := buildDockerProxy(empty).DockerProxy; got.IndexType != defaultDockerIndexType {
		t.Errorf("buildDockerProxy() dockerProxy = %+v, want the HUB index Nexus defaults to", got)
	}

	ociProxy := buildOciProxy(configured)
	if !ptr.Deref(ociProxy.Oci.PathEnabled, false) || ptr.Deref(ociProxy.Oci.HttpPort, 0) != 8082 {
		t.Errorf("buildOciProxy() oci = %+v, want the spec settings", ociProxy.Oci)
	}

	if ptr.Deref(ociProxy.OciProxy.IndexType, "") != "CUSTOM" || ptr.Deref(ociProxy.OciProxy.IndexUrl, "") != "https://index.example.com" {
		t.Errorf("buildOciProxy() ociProxy = %+v, want the custom index", ociProxy.OciProxy)
	}

	if got := buildOciGroup(configured).Cosign; got.Enforcement != "KEYLESS" || got.IssuerRegex == nil {
		t.Errorf("buildOciGroup() cosign = %+v, want the issuer regex enforced", got)
	}
}

// TestSharedBuildersApplyOptionalBlocks checks the shared builders submit the
// cleanup policies and routing rule a spec declares, and the variants of the
// blocks a format picks.
func TestSharedBuildersApplyOptionalBlocks(t *testing.T) {
	t.Parallel()

	desired := newDesired("raw", repoTypeProxy)
	desired.httpPassword = "password"
	desired.repo.Spec.ForProvider.RoutingRule = new("block-all")
	desired.repo.Spec.ForProvider.Cleanup = &repositoryv1alpha1.RepositoryCleanup{}
	desired.repo.Spec.ForProvider.Group = &repositoryv1alpha1.GroupConfig{MemberNames: []string{"a"}, WritableMember: new("a")}
	desired.repo.Spec.ForProvider.HTTPClient = &repositoryv1alpha1.HTTPClientConfig{
		Authentication: &repositoryv1alpha1.HTTPClientAuthentication{Type: new("username"), Username: new("user")},
	}

	proxyRepo := buildProxy[v395.RawProxyRepositoryApiRequest](desired)
	if ptr.Deref(proxyRepo.RoutingRuleName, "") != "block-all" {
		t.Error("buildProxy() dropped routingRule")
	}

	if proxyRepo.Cleanup == nil || proxyRepo.Cleanup.PolicyNames == nil {
		t.Error("buildProxy() dropped an empty cleanup block, which clears the policies")
	}

	preemptive := buildPreemptiveProxy[v395.PypiProxyRepositoryApiRequest](desired)
	if preemptive.HttpClient == nil || preemptive.HttpClient.Authentication.GetPassword() != "password" {
		t.Errorf("buildPreemptiveProxy() httpClient = %+v, want the resolved password", preemptive.HttpClient)
	}

	if got := buildDeployGroup[v395.NpmGroupRepositoryApiRequest](desired).Group; ptr.Deref(got.WritableMember, "") != "a" {
		t.Errorf("buildDeployGroup() group = %+v, want writable member a", got)
	}

	if got := buildDockerHosted(desired).Storage; got.WritePolicy != defaultWritePolicy || got.BlobStoreName != defaultBlobStoreName {
		t.Errorf("buildDockerHosted() storage = %+v, want the hosted defaults", got)
	}
}
