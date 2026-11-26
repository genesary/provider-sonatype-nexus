package repository

import (
	"context"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// secretRef selects key of the Secret named name.
func secretRef(name, key string) *xpv2.LocalSecretKeySelector {
	return &xpv2.LocalSecretKeySelector{LocalSecretReference: xpv2.LocalSecretReference{Name: name}, Key: key}
}

// TestBuildersApplyRepositorySettings checks every repository setting the
// API exposes reaches the payload of the formats and types that take it.
func TestBuildersApplyRepositorySettings(t *testing.T) {
	t.Parallel()

	desired := newDesired("any", repoTypeProxy)
	desired.httpBearerToken = "token"
	desired.ecrSecretAccessKey = "secret-key"
	desired.ecrSessionToken = "session"
	desired.repo.Spec.ForProvider = repositoryv1alpha1.RepositoryParameters{
		Name:        "repo",
		Storage:     &repositoryv1alpha1.RepositoryStorage{BlobStoreName: "default", LatestPolicy: new(true)},
		Component:   &repositoryv1alpha1.ComponentConfig{ProprietaryComponents: new(true)},
		Firewall:    &repositoryv1alpha1.FirewallConfig{Mode: "AUDIT"},
		Replication: &repositoryv1alpha1.ReplicationConfig{PreemptivePullEnabled: true, AssetPathRegex: new("^/org/.*")},
		Proxy:       &repositoryv1alpha1.ProxyConfig{RemoteURL: "https://remote.example.com", PreserveEncodedCharacters: new(true)},
		HTTPClient: &repositoryv1alpha1.HTTPClientConfig{
			Authentication: &repositoryv1alpha1.HTTPClientAuthentication{
				Type:                 new("bearerToken"),
				BearerTokenSecretRef: secretRef("upstream", "token"),
				Preemptive:           new(true),
			},
		},
		DockerProxy: &repositoryv1alpha1.ContainerProxyConfig{
			EcrAuth: &repositoryv1alpha1.EcrAuthConfig{
				Enabled:     true,
				AwsRegion:   new("eu-west-3"),
				RegistryID:  new("123456789012"),
				AccessKeyID: new("AKIAIOSFODNN7EXAMPLE"),
			},
		},
		Apt:        &repositoryv1alpha1.AptConfig{Distribution: new("bookworm"), EnforceDistribution: new(true)},
		AptSigning: &repositoryv1alpha1.AptSigningConfig{Keypair: "pgp"},
		NugetProxy: &repositoryv1alpha1.NugetProxyConfig{
			AllowAnonymousSymbolAccess: new(false),
			SymbolServerURL:            new("https://symbols.nuget.org/download/symbols"),
		},
		Raw:        &repositoryv1alpha1.RawConfig{ContentDisposition: new("INLINE")},
		ConanProxy: &repositoryv1alpha1.ConanProxyConfig{ConanVersion: new("V1")},
		Pypi:       &repositoryv1alpha1.PypiProxyConfig{IndexPath: new("")},
		Xet: &repositoryv1alpha1.XetConfig{
			XetCasURL:                new("https://cas.example.com"),
			XetTransferHostAllowList: []string{"transfer.example.com"},
		},
	}

	if got := buildDockerHosted(desired); !got.Storage.GetLatestPolicy() || !got.Component.GetProprietaryComponents() {
		t.Errorf("buildDockerHosted() = %+v, want the latest policy and proprietary components", got)
	}

	if got := buildOciHosted(desired); !got.Storage.GetLatestPolicy() {
		t.Error("buildOciHosted() dropped storage.latestPolicy")
	}

	if got := buildHosted[v395.NpmHostedRepositoryApiRequest](desired); !got.Component.GetProprietaryComponents() {
		t.Error("buildHosted() dropped component.proprietaryComponents")
	}

	rawProxy := buildRawProxy(desired)
	if rawProxy.Firewall.GetMode() != "AUDIT" {
		t.Errorf("buildProxy() firewall = %+v, want AUDIT", rawProxy.Firewall)
	}

	if !rawProxy.Replication.PreemptivePullEnabled || rawProxy.Replication.GetAssetPathRegex() != "^/org/.*" {
		t.Errorf("buildProxy() replication = %+v, want the pre-emptive pull", rawProxy.Replication)
	}

	if !rawProxy.Proxy.GetPreserveEncodedCharacters() {
		t.Error("buildProxy() dropped proxy.preserveEncodedCharacters")
	}

	if got := rawProxy.HttpClient.Authentication; got.GetBearerToken() != "token" || got.Preemptive != nil {
		t.Errorf("buildProxy() authentication = %+v, want the bearer token and no pre-emptive flag", got)
	}

	if got := rawProxy.Raw.GetContentDisposition(); got != "INLINE" {
		t.Errorf("buildRawProxy() contentDisposition = %q, want INLINE", got)
	}

	if buildRawHosted(desired).Raw == nil || buildRawGroup(desired).Raw == nil {
		t.Error("buildRawHosted() or buildRawGroup() dropped the raw block")
	}

	if got := buildMavenProxy(desired).HttpClient.Authentication; !got.GetPreemptive() || got.GetBearerToken() != "token" {
		t.Errorf("buildMavenProxy() authentication = %+v, want pre-emptive bearer token authentication", got)
	}

	ecr := buildDockerProxy(desired).DockerProxy.EcrAuth
	if ecr == nil || !ecr.GetEnabled() || ecr.GetAwsRegion() != "eu-west-3" || ecr.GetRegistryId() != "123456789012" ||
		ecr.GetAccessKeyId() != "AKIAIOSFODNN7EXAMPLE" || ecr.GetSecretAccessKey() != "secret-key" || ecr.GetSessionToken() != "session" {
		t.Errorf("buildDockerProxy() ecrAuth = %+v, want the spec and resolved credentials", ecr)
	}

	aptProxy := buildAptProxy(desired)
	if !aptProxy.Apt.GetEnforceDistribution() || aptProxy.AptSigning.GetKeypair() != "pgp" {
		t.Errorf("buildAptProxy() = %+v, want the enforced distribution and the signing key", aptProxy)
	}

	nuget := buildNugetProxy(desired).NugetProxy
	if nuget.GetAllowAnonymousSymbolAccess() || nuget.GetSymbolServerUrl() != "https://symbols.nuget.org/download/symbols" {
		t.Errorf("buildNugetProxy() nugetProxy = %+v, want the symbol server settings", nuget)
	}

	if got := buildConanProxy(desired).ConanProxy; got.GetConanVersion() != "V1" {
		t.Errorf("buildConanProxy() conanProxy = %+v, want V1", got)
	}

	pypi := buildPypiProxy(desired)
	if got := pypi.Pypi; got == nil || got.IndexPath == nil || *got.IndexPath != "" {
		t.Errorf("buildPypiProxy() pypi = %+v, want the root index path", got)
	}

	if !pypi.HttpClient.Authentication.GetPreemptive() {
		t.Error("buildPypiProxy() dropped httpClient.authentication.preemptive")
	}

	if got := buildHuggingFaceProxy(desired).Xet; got.GetXetCasUrl() != "https://cas.example.com" || len(got.XetTransferHostAllowList) != 1 {
		t.Errorf("buildHuggingFaceProxy() xet = %+v, want the spec settings", got)
	}
}

// TestBuildersSendResettableDefaults checks the defaults sent for the
// settings Nexus resets when an update leaves them out, and that the blocks
// Nexus keeps or removes on its own are left out when the spec declares none.
func TestBuildersSendResettableDefaults(t *testing.T) {
	t.Parallel()

	desired := newDesired("any", repoTypeProxy)
	desired.repo.Spec.ForProvider.HTTPClient = &repositoryv1alpha1.HTTPClientConfig{
		Authentication: &repositoryv1alpha1.HTTPClientAuthentication{Type: new("username"), Username: new("user")},
	}

	dockerHosted := buildDockerHosted(desired)
	if dockerHosted.Storage.LatestPolicy == nil || *dockerHosted.Storage.LatestPolicy {
		t.Errorf("buildDockerHosted() latestPolicy = %v, want false", dockerHosted.Storage.LatestPolicy)
	}

	if dockerHosted.Component == nil || dockerHosted.Component.ProprietaryComponents == nil || *dockerHosted.Component.ProprietaryComponents {
		t.Errorf("buildDockerHosted() component = %+v, want proprietaryComponents false", dockerHosted.Component)
	}

	rawProxy := buildRawProxy(desired)
	if rawProxy.Proxy.PreserveEncodedCharacters == nil || *rawProxy.Proxy.PreserveEncodedCharacters {
		t.Errorf("buildRawProxy() preserveEncodedCharacters = %v, want false", rawProxy.Proxy.PreserveEncodedCharacters)
	}

	if rawProxy.Firewall != nil || rawProxy.Replication != nil || rawProxy.Raw != nil {
		t.Errorf("buildRawProxy() = %+v, want no firewall, replication or raw block", rawProxy)
	}

	if rawProxy.HttpClient.Authentication.BearerToken != nil {
		t.Error("buildRawProxy() sent a bearer token the spec does not reference")
	}

	if got := buildMavenProxy(desired).HttpClient.Authentication; got.Preemptive == nil || *got.Preemptive {
		t.Errorf("buildMavenProxy() preemptive = %v, want false", got.Preemptive)
	}

	if got := buildDockerProxy(desired).DockerProxy.EcrAuth; got != nil {
		t.Errorf("buildDockerProxy() ecrAuth = %+v, want none", got)
	}

	aptProxy := buildAptProxy(desired)
	if aptProxy.Apt.EnforceDistribution == nil || *aptProxy.Apt.EnforceDistribution || aptProxy.AptSigning != nil {
		t.Errorf("buildAptProxy() = %+v, want enforceDistribution false and no signing key", aptProxy)
	}

	if got := buildNugetProxy(desired).NugetProxy; !got.GetAllowAnonymousSymbolAccess() || got.SymbolServerUrl != nil {
		t.Errorf("buildNugetProxy() nugetProxy = %+v, want anonymous symbol access and no symbol server", got)
	}

	if got := buildConanProxy(desired).ConanProxy; got != nil {
		t.Errorf("buildConanProxy() conanProxy = %+v, want none: Nexus cannot change the version", got)
	}

	if got := buildPypiProxy(desired).Pypi; got.GetIndexPath() != defaultPypiIndexPath {
		t.Errorf("buildPypiProxy() indexPath = %q, want %q", got.GetIndexPath(), defaultPypiIndexPath)
	}

	if got := buildHuggingFaceProxy(desired).Xet; got != nil {
		t.Errorf("buildHuggingFaceProxy() xet = %+v, want none", got)
	}
}

// TestIsUpToDateRepositorySettingQuirks checks drift detection on the
// settings Nexus does not report the way they were sent.
func TestIsUpToDateRepositorySettingQuirks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		desired  map[string]any
		observed map[string]any
		want     bool
	}{
		{
			name:     "UnreportedReplication",
			desired:  map[string]any{"replication": map[string]any{"preemptivePullEnabled": false}},
			observed: map[string]any{},
			want:     true,
		},
		{
			name:     "ReportedReplicationDrift",
			desired:  map[string]any{"replication": map[string]any{"preemptivePullEnabled": true}},
			observed: map[string]any{"replication": map[string]any{"preemptivePullEnabled": false}},
		},
		{
			name:     "UnreportedXet",
			desired:  map[string]any{"xet": map[string]any{"xetCasUrl": "https://cas.example.com"}},
			observed: map[string]any{"xet": nil},
			want:     true,
		},
		{
			name:     "UnreportedComponent",
			desired:  map[string]any{"component": map[string]any{"proprietaryComponents": false}},
			observed: map[string]any{"component": nil},
			want:     true,
		},
		{
			name:     "ReportedComponentDrift",
			desired:  map[string]any{"component": map[string]any{"proprietaryComponents": false}},
			observed: map[string]any{"component": map[string]any{"proprietaryComponents": true}},
		},
		{
			name:     "RootIndexPathDrift",
			desired:  map[string]any{"pypi": map[string]any{"indexPath": ""}},
			observed: map[string]any{"pypi": map[string]any{"indexPath": "/simple"}},
		},
		{
			name:     "RootIndexPath",
			desired:  map[string]any{"pypi": map[string]any{"indexPath": ""}},
			observed: map[string]any{"pypi": map[string]any{"indexPath": ""}},
			want:     true,
		},
		{
			name: "MaskedEcrCredentials",
			desired: map[string]any{"dockerProxy": map[string]any{"ecrAuth": map[string]any{
				"enabled": true, "accessKeyId": "AKIAIOSFODNN7EXAMPLE", "secretAccessKey": "secret", "sessionToken": "token",
			}}},
			observed: map[string]any{"dockerProxy": map[string]any{"ecrAuth": map[string]any{
				"enabled": true, "accessKeyId": "****************MPLE",
			}}},
			want: true,
		},
		{
			name:     "EcrDisabledOutOfBand",
			desired:  map[string]any{"dockerProxy": map[string]any{"ecrAuth": map[string]any{"enabled": true}}},
			observed: map[string]any{"dockerProxy": map[string]any{"ecrAuth": nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, _, err := isUpToDate(tt.desired, tt.observed)
			if err != nil {
				t.Fatalf("isUpToDate() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("isUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// newSecretKube returns a fake Kubernetes client serving objs.
func newSecretKube(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()

	err := repositoryv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(content) failed: %v", err)
	}

	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// TestCredentialsFingerprintIsStable checks that the fingerprint of a
// Repository using none of the Secrets added after the password is the one
// recorded before they existed, so that upgrading the provider does not
// re-apply every Repository's credentials.
func TestCredentialsFingerprintIsStable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cr := newTestRepo("maven-proxy", "maven2", repoTypeProxy)
	cr.Namespace = "default"
	cr.Spec.ForProvider.HTTPClient = &repositoryv1alpha1.HTTPClientConfig{
		Authentication: &repositoryv1alpha1.HTTPClientAuthentication{
			Type:              new("username"),
			Username:          new("upstream"),
			PasswordSecretRef: secretRef("upstream", "password"),
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "upstream", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte("password")},
	}
	kube := newSecretKube(t, secret)

	// legacySpec is the spec the fingerprint was computed from before the
	// bearer token and ECR authentication were supported.
	legacySpec := struct {
		Authentication   *repositoryv1alpha1.HTTPClientAuthentication `json:"authentication,omitempty"`
		AptSigning       *repositoryv1alpha1.AptSigningConfig         `json:"aptSigning,omitempty"`
		YumSigning       *repositoryv1alpha1.YumSigningConfig         `json:"yumSigning,omitempty"`
		AlpineSigning    *repositoryv1alpha1.AlpineSigningConfig      `json:"alpineSigning,omitempty"`
		TerraformSigning *repositoryv1alpha1.TerraformSigningConfig   `json:"terraformSigning,omitempty"`
	}{Authentication: cr.Spec.ForProvider.HTTPClient.Authentication}

	want, err := nexus.CredentialsFingerprint(ctx, kube, "default", legacySpec, cr.Spec.ForProvider.HTTPClient.Authentication.PasswordSecretRef)
	if err != nil {
		t.Fatalf("legacy fingerprint: %v", err)
	}

	got, err := newTestExternal(nil).credentialsFingerprint(ctx, kube, cr)
	if err != nil {
		t.Fatalf("credentialsFingerprint() error = %v", err)
	}

	if got != want {
		t.Errorf("credentialsFingerprint() = %q, want the legacy fingerprint %q", got, want)
	}
}

// TestBearerTokenAndEcrCredentials checks that the bearer token and AWS ECR
// Secrets are resolved into the payload, and that rotating them is reported as
// drift even though Nexus never reports them.
func TestBearerTokenAndEcrCredentials(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cr := newTestRepo("docker-proxy", "docker", repoTypeProxy)
	cr.Namespace = "default"
	cr.Spec.ForProvider.HTTPClient = &repositoryv1alpha1.HTTPClientConfig{
		Authentication: &repositoryv1alpha1.HTTPClientAuthentication{
			Type:                 new("bearerToken"),
			BearerTokenSecretRef: secretRef("upstream", "token"),
		},
	}
	cr.Spec.ForProvider.DockerProxy = &repositoryv1alpha1.ContainerProxyConfig{
		EcrAuth: &repositoryv1alpha1.EcrAuthConfig{
			Enabled:                  true,
			AccessKeyID:              new("AKIAIOSFODNN7EXAMPLE"),
			SecretAccessKeySecretRef: secretRef("aws", "secretAccessKey"),
			SessionTokenSecretRef:    secretRef("aws", "sessionToken"),
		},
	}

	upstream := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "upstream", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("bearer")},
	}
	aws := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "aws", Namespace: "default"},
		Data:       map[string][]byte{"secretAccessKey": []byte("secret-key"), "sessionToken": []byte("session")},
	}
	kube := newSecretKube(t, cr, upstream, aws)

	e := newTestExternal(&mockHandler{exists: true, upToDate: true})
	e.kube = kube

	desired, err := e.desiredState(ctx, kube, cr)
	if err != nil {
		t.Fatalf("desiredState() error = %v", err)
	}

	payload := buildDockerProxy(desired)
	if got := payload.HttpClient.Authentication.GetBearerToken(); got != "bearer" {
		t.Errorf("bearer token = %q, want the Secret value", got)
	}

	if got := payload.DockerProxy.EcrAuth; got.GetSecretAccessKey() != "secret-key" || got.GetSessionToken() != "session" {
		t.Errorf("ecrAuth = %+v, want the Secret values", got)
	}

	_, err = e.Update(ctx, cr)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	assertUpToDate := func(step string, want bool) {
		t.Helper()

		obs, observeErr := e.Observe(ctx, cr)
		if observeErr != nil {
			t.Fatalf("%s: Observe() error = %v", step, observeErr)
		}

		if obs.ResourceUpToDate != want {
			t.Fatalf("%s: ResourceUpToDate = %v, want %v", step, obs.ResourceUpToDate, want)
		}
	}

	assertUpToDate("after update", true)

	for _, rotate := range []struct {
		step   string
		secret *corev1.Secret
		key    string
	}{
		{"bearer token rotation", upstream, "token"},
		{"ECR secret access key rotation", aws, "secretAccessKey"},
	} {
		rotate.secret.Data[rotate.key] = []byte("rotated")

		err = kube.Update(ctx, rotate.secret)
		if err != nil {
			t.Fatalf("%s: %v", rotate.step, err)
		}

		assertUpToDate("after "+rotate.step, false)

		_, err = e.Update(ctx, cr)
		if err != nil {
			t.Fatalf("%s: Update() error = %v", rotate.step, err)
		}

		assertUpToDate("after update for "+rotate.step, true)
	}

	cr.Spec.ForProvider.DockerProxy.EcrAuth.AccessKeyID = new("AKIAI44QH8DHBEXAMPLE")
	assertUpToDate("after access key ID change", false)

	cr.Spec.ForProvider.HTTPClient.Authentication.BearerTokenSecretRef = secretRef("missing", "token")

	_, err = e.Update(ctx, cr)
	if err == nil {
		t.Error("Update() with a missing bearer token Secret returned no error")
	}
}
