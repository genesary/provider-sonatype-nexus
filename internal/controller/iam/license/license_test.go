package license

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	nexus "github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	licensefake "github.com/genesary/provider-sonatype-nexus/internal/fake"
)

// newTestLicense creates a License CR backed by a Kubernetes secret.
func newTestLicense(name, secretName, secretNS, secretKey string) *iamv1alpha1.License {
	return &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.LicenseSpec{
			ForProvider: iamv1alpha1.LicenseParameters{
				LicenseSecretRef: &xpv2.SecretKeySelector{
					SecretReference: xpv2.SecretReference{
						Name:      secretName,
						Namespace: secretNS,
					},
					Key: secretKey,
				},
			},
		},
	}
}

// newTestLicenseWithEndpoint creates a License CR that fetches from a URL.
func newTestLicenseWithEndpoint(name, endpointURL string) *iamv1alpha1.License {
	return &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.LicenseSpec{
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
			},
		},
	}
}

// newTestLicenseNoSource creates a License CR with no license source.
func newTestLicenseNoSource(name string) *iamv1alpha1.License {
	return &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       iamv1alpha1.LicenseSpec{},
	}
}

// newTestScheme registers all types needed by the license controller tests.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := iamv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(iam): %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus): %v", err)
	}

	err = corev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(core): %v", err)
	}

	return s
}

// newLicenseSecret creates a Kubernetes Secret holding license bytes.
func newLicenseSecret(name, namespace, key string, data []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{key: data},
	}
}

// testLicenseData is license content used across controller tests.
var testLicenseData = []byte("fake-nexus-license-binary")

// TestObserve_WrongType verifies Observe errors on a non-License resource.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: licensefake.NewMockLicenseClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Fatal("Observe() nil resource should return error")
	}
}

// TestObserve_DeletionTimestamp tests Observe when CR has deletion timestamp.
func TestObserve_DeletionTimestamp(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	cr := newTestLicense("test-license", "sec", "default", "k")
	cr.DeletionTimestamp = &now

	e := &external{client: licensefake.NewMockLicenseClient()}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() deletion timestamp returned unexpected error: %v", err)
	}

	if obs.ResourceExists {
		t.Error("Observe() ResourceExists = true, want false for deleting CR")
	}
}

// TestObserve_NoLicenseSource tests Observe with no source configured.
func TestObserve_NoLicenseSource(t *testing.T) {
	t.Parallel()

	cr := newTestLicenseNoSource("test-license")

	e := &external{
		client: licensefake.NewMockLicenseClient(),
		kube:   crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build(),
	}

	_, err := e.Observe(context.Background(), cr)
	if err == nil {
		t.Fatal("Observe() with no license source should return error")
	}
}

// TestObserve_SecretNotFound verifies Observe errors when secret is missing.
func TestObserve_SecretNotFound(t *testing.T) {
	t.Parallel()

	cr := newTestLicense("test-license", "missing-secret", "default", "k")

	e := &external{
		client: licensefake.NewMockLicenseClient(),
		kube:   crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build(),
	}

	_, err := e.Observe(context.Background(), cr)
	if err == nil {
		t.Fatal("Observe() with missing secret should return error")
	}
}

// TestObserve_NoLicense tests Observe when Nexus has no license installed.
func TestObserve_NoLicense(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.GetLicenseFn = func(_ context.Context) (*iamclient.LicenseInfo, error) {
		return nil, iamclient.ErrNoLicense
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() with ErrNoLicense returned unexpected error: %v", err)
	}

	if obs.ResourceExists {
		t.Error("Observe() ResourceExists = true, want false when no license installed")
	}
}

// TestObserve_GetLicenseError verifies Observe propagates Nexus API errors.
func TestObserve_GetLicenseError(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.GetLicenseFn = func(_ context.Context) (*iamclient.LicenseInfo, error) {
		return nil, errors.New("nexus unreachable")
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Observe(context.Background(), cr)
	if err == nil {
		t.Fatal("Observe() expected error on Nexus API failure, got nil")
	}
}

// TestObserve_ExistsAndUpToDate tests Observe when license is up to date.
func TestObserve_ExistsAndUpToDate(t *testing.T) {
	t.Parallel()

	hash := iamclient.HashLicense(testLicenseData)

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.GetLicenseFn = func(_ context.Context) (*iamclient.LicenseInfo, error) {
		return &iamclient.LicenseInfo{Fingerprint: "fp-abc"}, nil
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")
	cr.Status.AtProvider.InstalledHash = hash
	cr.Status.AtProvider.Fingerprint = "fp-abc"

	e := &external{client: mc, kube: kubeClient}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true")
	}

	if !obs.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = false, want true when hash matches")
	}
}

// TestObserve_ExistsAndOutdated tests Observe when license hash differs.
func TestObserve_ExistsAndOutdated(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.GetLicenseFn = func(_ context.Context) (*iamclient.LicenseInfo, error) {
		return &iamclient.LicenseInfo{Fingerprint: "fp-abc"}, nil
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")
	cr.Status.AtProvider.InstalledHash = iamclient.HashLicense([]byte("old-license-data"))
	cr.Status.AtProvider.Fingerprint = "fp-abc"

	e := &external{client: mc, kube: kubeClient}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true")
	}

	if obs.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = true, want false when hash differs")
	}
}

// TestCreate_WrongType verifies Create errors on a non-License resource.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: licensefake.NewMockLicenseClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("Create() nil resource should return error")
	}
}

// TestCreate_NoLicenseSource tests Create with no source configured.
func TestCreate_NoLicenseSource(t *testing.T) {
	t.Parallel()

	cr := newTestLicenseNoSource("test-license")

	e := &external{
		client: licensefake.NewMockLicenseClient(),
		kube:   crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build(),
	}

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("Create() with no license source should return error")
	}
}

// TestCreate_InstallError verifies Create errors when InstallLicense fails.
func TestCreate_InstallError(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		return errors.New("installation rejected")
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("Create() expected error when InstallLicense fails, got nil")
	}
}

// TestCreate_Success tests Create with license from a Kubernetes secret.
func TestCreate_Success(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithStatusSubresource(&iamv1alpha1.License{}).
		WithObjects(secret).
		Build()

	var installedData []byte

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, data []byte) error {
		installedData = data

		return nil
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if !bytes.Equal(installedData, testLicenseData) {
		t.Errorf("Create() installed %v, want %v", installedData, testLicenseData)
	}
}

// TestCreate_WithEndpoint tests Create with an HTTP endpoint source.
func TestCreate_WithEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(testLicenseData)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		Build()

	var installedData []byte

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, data []byte) error {
		installedData = data

		return nil
	}

	cr := newTestLicenseWithEndpoint("test-license", server.URL)

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() with endpoint returned unexpected error: %v", err)
	}

	if !bytes.Equal(installedData, testLicenseData) {
		t.Errorf("Create() installed %v, want %v", installedData, testLicenseData)
	}
}

// TestCreate_WithEndpointAndCache tests Create returns license bytes in
// ConnectionDetails.
func TestCreate_WithEndpointAndCache(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testLicenseData)
	}))
	defer server.Close()

	kubeClient := crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		return nil
	}

	endpointURL := server.URL
	cr := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test-license"},
		Spec: iamv1alpha1.LicenseSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv2.LocalSecretReference{
					Name: "license-cache",
				},
			},
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	creation, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() with endpoint+cache returned unexpected error: %v", err)
	}

	if !bytes.Equal(creation.ConnectionDetails[licenseSecretCacheKey], testLicenseData) {
		t.Error("Create() ConnectionDetails does not contain the downloaded license")
	}
}

// TestCreate_EndpointFallbackToCache tests Create with cache fallback.
func TestCreate_EndpointFallbackToCache(t *testing.T) {
	t.Parallel()

	cachedData := []byte("cached-license-data")

	cacheSecret := newLicenseSecret("license-cache", "default", licenseSecretCacheKey, cachedData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(cacheSecret).
		Build()

	var installedData []byte

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, data []byte) error {
		installedData = data

		return nil
	}

	endpointURL := "http://127.0.0.1:1" // unreachable port
	cr := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test-license"},
		Spec: iamv1alpha1.LicenseSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv2.LocalSecretReference{
					Name: "license-cache",
				},
			},
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() with endpoint fallback returned unexpected error: %v", err)
	}

	if !bytes.Equal(installedData, cachedData) {
		t.Errorf("Create() installed %v, want cached %v", installedData, cachedData)
	}
}

// TestCreate_EndpointFailsNoCacheFails tests Create when all sources fail.
func TestCreate_EndpointFailsNoCacheFails(t *testing.T) {
	t.Parallel()

	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		Build()

	mc := licensefake.NewMockLicenseClient()

	endpointURL := "http://127.0.0.1:1" // unreachable
	cr := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test-license"},
		Spec: iamv1alpha1.LicenseSpec{
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("Create() expected error when endpoint unavailable and no cache, got nil")
	}
}

// TestCreate_WithBasicAuthEndpoint tests Create with basic auth credentials.
func TestCreate_WithBasicAuthEndpoint(t *testing.T) {
	t.Parallel()

	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testLicenseData)
	}))
	defer server.Close()

	usernameSecret := newLicenseSecret("ep-creds", "default", "username", []byte("myuser"))
	passwordSecret := newLicenseSecret("ep-creds-pw", "default", "password", []byte("mypass"))
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(usernameSecret, passwordSecret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		return nil
	}

	endpointURL := server.URL
	cr := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test-license"},
		Spec: iamv1alpha1.LicenseSpec{
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
				EndpointCredentials: &iamv1alpha1.LicenseEndpointCredentials{
					UsernameSecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "ep-creds",
							Namespace: "default",
						},
						Key: "username",
					},
					PasswordSecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "ep-creds-pw",
							Namespace: "default",
						},
						Key: "password",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() with basic auth returned unexpected error: %v", err)
	}

	if gotAuth == "" {
		t.Error("Create() did not send Authorization header for basic auth")
	}
}

// TestCreate_WithBearerTokenEndpoint tests Create with bearer token auth.
func TestCreate_WithBearerTokenEndpoint(t *testing.T) {
	t.Parallel()

	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testLicenseData)
	}))
	defer server.Close()

	tokenSecret := newLicenseSecret("ep-token", "default", "token", []byte("my-bearer-token"))
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(tokenSecret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		return nil
	}

	endpointURL := server.URL
	cr := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: "test-license"},
		Spec: iamv1alpha1.LicenseSpec{
			ForProvider: iamv1alpha1.LicenseParameters{
				EndpointURL: &endpointURL,
				EndpointCredentials: &iamv1alpha1.LicenseEndpointCredentials{
					PasswordSecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "ep-token",
							Namespace: "default",
						},
						Key: "token",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() with bearer token returned unexpected error: %v", err)
	}

	if gotAuth != "Bearer my-bearer-token" {
		t.Errorf("Create() Authorization = %q, want %q", gotAuth, "Bearer my-bearer-token")
	}
}

// TestUpdate_WrongType verifies Update errors on a non-License resource.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: licensefake.NewMockLicenseClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("Update() nil resource should return error")
	}
}

// TestUpdate_Success verifies Update reinstalls the license.
func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	var reinstalled bool

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		reinstalled = true

		return nil
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if !reinstalled {
		t.Error("Update() did not call InstallLicense")
	}
}

// TestUpdate_Error verifies Update propagates InstallLicense errors.
func TestUpdate_Error(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "license.lic", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	mc := licensefake.NewMockLicenseClient()
	mc.InstallLicenseFn = func(_ context.Context, _ []byte) error {
		return errors.New("reinstall failed")
	}

	cr := newTestLicense("test-license", "license-secret", "default", "license.lic")

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Update(context.Background(), cr)
	if err == nil {
		t.Fatal("Update() expected error from InstallLicense, got nil")
	}
}

// TestDelete_WrongType verifies Delete errors on a non-License resource.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: licensefake.NewMockLicenseClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Fatal("Delete() nil resource should return error")
	}
}

// TestDelete_Success verifies Delete removes the license from Nexus.
func TestDelete_Success(t *testing.T) {
	t.Parallel()

	var deleted bool

	mc := licensefake.NewMockLicenseClient()
	mc.DeleteLicenseFn = func(_ context.Context) error {
		deleted = true

		return nil
	}

	cr := newTestLicense("test-license", "sec", "default", "k")

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if !deleted {
		t.Error("Delete() did not call DeleteLicense")
	}
}

// TestDelete_Error verifies Delete propagates DeleteLicense errors.
func TestDelete_Error(t *testing.T) {
	t.Parallel()

	mc := licensefake.NewMockLicenseClient()
	mc.DeleteLicenseFn = func(_ context.Context) error {
		return errors.New("cannot remove license")
	}

	cr := newTestLicense("test-license", "sec", "default", "k")

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err == nil {
		t.Fatal("Delete() expected error from DeleteLicense, got nil")
	}
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: licensefake.NewMockLicenseClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnect_WrongType verifies Connect errors on a non-License resource.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Fatal("Connect() nil resource should return error")
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestLicense("test-license", "sec", "default", "k")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Fatal("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests Connect without ProviderConfig.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestLicense("test-license", "sec", "default", "k")
	cr.UID = types.UID("test-uid-9876")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Fatal("Connect() should fail when ProviderConfig is absent from the store")
	}
}

// TestGetSecretBytes_MissingKey tests GetSecretBytes with a missing key.
func TestGetSecretBytes_MissingKey(t *testing.T) {
	t.Parallel()

	secret := newLicenseSecret("license-secret", "default", "other-key", testLicenseData)
	kubeClient := crfake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret).
		Build()

	sel := &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "license-secret",
			Namespace: "default",
		},
		Key: "missing-key",
	}

	_, err := nexus.GetSecretBytes(context.Background(), kubeClient, sel)
	if err == nil {
		t.Fatal("GetSecretBytes() expected error for missing key, got nil")
	}
}

// TestEndpointRequest_NonOKStatus tests doEndpointRequest on 403 response.
func TestEndpointRequest_NonOKStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	kubeClient := crfake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	e := &external{kube: kubeClient}

	_, err := e.doEndpointRequest(context.Background(), server.URL, nil)
	if err == nil {
		t.Fatal("doEndpointRequest() expected error on 403, got nil")
	}
}
