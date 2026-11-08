package blobstore

import (
	"context"
	"errors"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// Fixtures of the Azure blob store tests.
const (
	// testAzureName is the name of the Azure blob store.
	testAzureName = "test-azure-blobstore"
	// testAccountName is the Azure storage account.
	testAccountName = "mystorageaccount"
	// testContainerName is the Azure container.
	testContainerName = "nexus-blobs"
	// testAccountKey is the account key the Secret initially holds.
	testAccountKey = "initial-account-key"
	// testKeySecretName is the name of the account key Secret.
	testKeySecretName = "azure-storage-secret"
	// testKeySecretKey is the key of the account key in its Secret.
	testKeySecretKey = "accountKey"
	// testNamespace is the namespace of the BlobStore and its Secret.
	testNamespace = "default"
	// testAuthAccountKey is the account key authentication method.
	testAuthAccountKey = "ACCOUNTKEY"
)

// expectError reports whether err is the error a test case expects: when
// wantErr is set, err must contain it and wrap wantIs if set, and the case
// ends there; otherwise err must be nil.
func expectError(t *testing.T, op string, err error, wantErr string, wantIs error) bool {
	t.Helper()

	if wantErr == "" {
		if err != nil {
			t.Fatalf("%s() unexpected error: %v", op, err)
		}

		return false
	}

	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("%s() error = %v, want it to contain %q", op, err, wantErr)
	}

	if wantIs != nil && !errors.Is(err, wantIs) {
		t.Fatalf("%s() error = %v, want it to wrap %v", op, err, wantIs)
	}

	return true
}

// mockAzureClient is a mock implementing instance.BlobStoreAzureClient.
type mockAzureClient struct {
	GetFn       func(name string) (*v395.AzureBlobStoreApiModel, error)
	CreateFn    func(bs v395.AzureBlobStoreApiModel) error
	UpdateFn    func(name string, bs v395.AzureBlobStoreApiModel) error
	CreateCalls []v395.AzureBlobStoreApiModel
	UpdateCalls []v395.AzureBlobStoreApiModel
}

// Get delegates to GetFn; returns errNotConfigured if GetFn is nil.
func (m *mockAzureClient) Get(_ context.Context, name string) (*v395.AzureBlobStoreApiModel, error) {
	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errNotConfigured
}

// Create records the call and delegates to CreateFn if set.
func (m *mockAzureClient) Create(_ context.Context, bs v395.AzureBlobStoreApiModel) error {
	m.CreateCalls = append(m.CreateCalls, bs)
	if m.CreateFn != nil {
		return m.CreateFn(bs)
	}

	return nil
}

// Update records the call and delegates to UpdateFn if set.
func (m *mockAzureClient) Update(_ context.Context, name string, bs v395.AzureBlobStoreApiModel) error {
	m.UpdateCalls = append(m.UpdateCalls, bs)
	if m.UpdateFn != nil {
		return m.UpdateFn(name, bs)
	}

	return nil
}

// newTestScheme returns a runtime.Scheme with the types the controller reads.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{instancev1alpha1.AddToScheme, corev1.AddToScheme} {
		err := add(s)
		if err != nil {
			t.Fatalf("AddToScheme failed: %v", err)
		}
	}

	return s
}

// newAccountKeySecret returns the Secret holding the Azure account key.
func newAccountKeySecret(key string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testKeySecretName, Namespace: testNamespace},
		Data:       map[string][]byte{testKeySecretKey: []byte(key)},
	}
}

// withAccountKeySecret returns the account key Secret as the only object, or
// no object when present is false.
func withAccountKeySecret(present bool) []client.Object {
	if !present {
		return nil
	}

	return []client.Object{newAccountKeySecret(testAccountKey)}
}

// newAzureBlobStoreCR returns an Azure BlobStore authenticating with the
// account key Secret.
func newAzureBlobStoreCR() *instancev1alpha1.BlobStore {
	cr := newBlobStoreCR(testAzureName, blobStoreTypeAzure)
	cr.Namespace = testNamespace
	cr.Spec.ForProvider.AzureConfig = &instancev1alpha1.AzureConfig{
		AccountName:          testAccountName,
		ContainerName:        testContainerName,
		AuthenticationMethod: testAuthAccountKey,
		AccountKeySecretRef: &xpv2.LocalSecretKeySelector{
			LocalSecretReference: xpv2.LocalSecretReference{Name: testKeySecretName},
			Key:                  testKeySecretKey,
		},
	}

	return cr
}

// newObservedAzureBlobStore returns the Azure blob store Nexus reports for
// newAzureBlobStoreCR. Like other Nexus secrets, the account key is not
// reported.
func newObservedAzureBlobStore() *v395.AzureBlobStoreApiModel {
	return &v395.AzureBlobStoreApiModel{
		Name: testAzureName,
		BucketConfiguration: v395.AzureBlobStoreApiBucketConfiguration{
			AccountName:   testAccountName,
			ContainerName: testContainerName,
			Authentication: v395.AzureBlobStoreApiAuthentication{
				AuthenticationMethod: testAuthAccountKey,
			},
		},
	}
}

// newAzureTestExternal returns an *external for Azure blob stores, reading
// Secrets from objs.
func newAzureTestExternal(t *testing.T, azure *mockAzureClient, objs ...client.Object) (*external, client.Client) {
	t.Helper()

	builder := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(objs...)
	for _, obj := range objs {
		if bs, ok := obj.(*instancev1alpha1.BlobStore); ok {
			builder = builder.WithStatusSubresource(bs)
		}
	}

	kube := builder.Build()

	e := newTestExternal(nil, nil, nil)
	e.azureClient = azure
	e.kube = kube

	return e, kube
}

// TestAzureCredentialRotation verifies that rotating the account key Secret
// is reported as drift and re-applied to Nexus, even though Nexus never
// returns the account key.
func TestAzureCredentialRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cr := newAzureBlobStoreCR()
	secret := newAccountKeySecret(testAccountKey)

	var created bool

	azure := &mockAzureClient{
		GetFn: func(_ string) (*v395.AzureBlobStoreApiModel, error) {
			if !created {
				return nil, nexus.ErrNotFound
			}

			return newObservedAzureBlobStore(), nil
		},
		CreateFn: func(_ v395.AzureBlobStoreApiModel) error {
			created = true

			return nil
		},
	}

	e, kube := newAzureTestExternal(t, azure, cr, secret)

	observe := func(step string, wantExists, wantUpToDate bool) {
		t.Helper()

		obs, err := e.Observe(ctx, cr)
		if err != nil {
			t.Fatalf("%s: Observe() unexpected error: %v", step, err)
		}

		if obs.ResourceExists != wantExists || obs.ResourceUpToDate != wantUpToDate {
			t.Fatalf("%s: Observe() = exists %v, up to date %v; want %v, %v",
				step, obs.ResourceExists, obs.ResourceUpToDate, wantExists, wantUpToDate)
		}
	}

	observe("before create", false, false)

	_, err := e.Create(ctx, cr)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.CredentialsFingerprint == "" {
		t.Fatal("Create() recorded no credentials fingerprint")
	}

	observe("after create", true, true)

	secret.Data[testKeySecretKey] = []byte("rotated-account-key")

	err = kube.Update(ctx, secret)
	if err != nil {
		t.Fatalf("rotating secret: %v", err)
	}

	observe("after rotation", true, false)

	_, err = e.Update(ctx, cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	observe("after update", true, true)

	if len(azure.CreateCalls) != 1 || len(azure.UpdateCalls) != 1 {
		t.Fatalf("Create calls = %d, Update calls = %d; want 1 and 1", len(azure.CreateCalls), len(azure.UpdateCalls))
	}

	applied := []string{
		azure.CreateCalls[0].BucketConfiguration.Authentication.GetAccountKey(),
		azure.UpdateCalls[0].BucketConfiguration.Authentication.GetAccountKey(),
	}
	if applied[0] != testAccountKey || applied[1] != "rotated-account-key" {
		t.Fatalf("applied account keys = %v, want [%s rotated-account-key]", applied, testAccountKey)
	}
}

// TestObserveAzure covers the Azure branches of Observe.
func TestObserveAzure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutateCR     func(cr *instancev1alpha1.BlobStore)
		mutateObs    func(obs *v395.AzureBlobStoreApiModel)
		getErr       error
		noSecret     bool
		wantExists   bool
		wantUpToDate bool
		wantErr      string
	}{
		"NotFound": {
			getErr: nexus.ErrNotFound,
		},
		"GetError": {
			getErr:  errors.New("boom"),
			wantErr: errGetBlobStore,
		},
		"UpToDate": {
			wantExists:   true,
			wantUpToDate: true,
		},
		"AccountNameDrift": {
			mutateObs:  func(obs *v395.AzureBlobStoreApiModel) { obs.BucketConfiguration.AccountName = "other" },
			wantExists: true,
		},
		"ContainerNameDrift": {
			mutateObs:  func(obs *v395.AzureBlobStoreApiModel) { obs.BucketConfiguration.ContainerName = "other" },
			wantExists: true,
		},
		"AuthenticationMethodDrift": {
			mutateObs: func(obs *v395.AzureBlobStoreApiModel) {
				obs.BucketConfiguration.Authentication.AuthenticationMethod = "MANAGEDIDENTITY"
			},
			wantExists: true,
		},
		"PreSignedURLDrift": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.Spec.ForProvider.AzureConfig.PreSignedURLEnabled = new(true)
			},
			wantExists: true,
		},
		"PreSignedURLUnsetIgnored": {
			mutateObs: func(obs *v395.AzureBlobStoreApiModel) {
				obs.BucketConfiguration.PreSignedUrlEnabled = new(true)
			},
			wantExists:   true,
			wantUpToDate: true,
		},
		"PreSignedURLMatches": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.Spec.ForProvider.AzureConfig.PreSignedURLEnabled = new(true)
			},
			mutateObs: func(obs *v395.AzureBlobStoreApiModel) {
				obs.BucketConfiguration.PreSignedUrlEnabled = new(true)
			},
			wantExists:   true,
			wantUpToDate: true,
		},
		"SoftQuotaDrift": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.Spec.ForProvider.SoftQuota = &instancev1alpha1.SoftQuota{
					Type:  new("spaceUsedQuota"),
					Limit: new(int64(1024)),
				}
			},
			wantExists: true,
		},
		"NoKeyFingerprintedIsUpToDate": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.Spec.ForProvider.AzureConfig.AuthenticationMethod = "MANAGEDIDENTITY"
				cr.Spec.ForProvider.AzureConfig.AccountKeySecretRef = nil
				cr.Status.AtProvider.CredentialsFingerprint = ""
			},
			mutateObs: func(obs *v395.AzureBlobStoreApiModel) {
				obs.BucketConfiguration.Authentication.AuthenticationMethod = "MANAGEDIDENTITY"
			},
			wantExists:   true,
			wantUpToDate: true,
		},
		"MissingSecret": {
			noSecret: true,
			wantErr:  errFingerprint,
		},
		"MissingSecretWhileDeleting": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.DeletionTimestamp = new(metav1.Now())
				cr.Finalizers = []string{"test"}
			},
			noSecret:     true,
			wantExists:   true,
			wantUpToDate: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cr := newAzureBlobStoreCR()

			e, _ := newAzureTestExternal(t, &mockAzureClient{}, withAccountKeySecret(!tc.noSecret)...)

			// Record the fingerprint of the current Secret as applied.
			fingerprint, err := azureCredentialsFingerprint(ctx, e.kube, cr)
			if err == nil {
				cr.Status.AtProvider.CredentialsFingerprint = fingerprint
			}

			if tc.mutateCR != nil {
				tc.mutateCR(cr)
			}

			e.azureClient = &mockAzureClient{
				GetFn: func(_ string) (*v395.AzureBlobStoreApiModel, error) {
					if tc.getErr != nil {
						return nil, tc.getErr
					}

					obs := newObservedAzureBlobStore()
					if tc.mutateObs != nil {
						tc.mutateObs(obs)
					}

					return obs, nil
				},
			}

			got, err := e.Observe(ctx, cr)
			if expectError(t, "Observe", err, tc.wantErr, nil) {
				return
			}

			if got.ResourceExists != tc.wantExists || got.ResourceUpToDate != tc.wantUpToDate {
				t.Fatalf("Observe() = exists %v, up to date %v; want %v, %v",
					got.ResourceExists, got.ResourceUpToDate, tc.wantExists, tc.wantUpToDate)
			}
		})
	}
}

// TestObserveAzureRecordsObservation verifies that the observation reflects
// the Azure blob store Nexus reported.
func TestObserveAzureRecordsObservation(t *testing.T) {
	t.Parallel()

	cr := newAzureBlobStoreCR()
	cr.Status.AtProvider.CredentialsFingerprint = "applied"

	observed := newObservedAzureBlobStore()
	observed.BucketConfiguration.PreSignedUrlEnabled = new(true)
	observed.SoftQuota = &v395.BlobStoreApiSoftQuota{Type: "spaceUsedQuota", Limit: new(int64(10))}

	e, _ := newAzureTestExternal(t, &mockAzureClient{
		GetFn: func(_ string) (*v395.AzureBlobStoreApiModel, error) { return observed, nil },
	}, newAccountKeySecret(testAccountKey))

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	obs := cr.Status.AtProvider
	if obs.Name != testAzureName ||
		ptr.Deref(obs.AccountName, "") != testAccountName ||
		ptr.Deref(obs.ContainerName, "") != testContainerName ||
		ptr.Deref(obs.AuthenticationMethod, "") != testAuthAccountKey ||
		!ptr.Deref(obs.PreSignedURLEnabled, false) ||
		ptr.Deref(obs.SoftQuotaType, "") != "spaceUsedQuota" ||
		ptr.Deref(obs.SoftQuotaLimit, 0) != 10 {
		t.Errorf("Observe() observation = %+v", obs)
	}

	if obs.CredentialsFingerprint != "applied" {
		t.Errorf("Observe() CredentialsFingerprint = %q, want the applied one carried over", obs.CredentialsFingerprint)
	}
}

// TestCreateAzure covers the Azure branches of Create.
func TestCreateAzure(t *testing.T) {
	t.Parallel()

	proErr := errors.Join(instance.ErrProEditionRequired, errors.New("402 Payment Required"))

	tests := map[string]struct {
		mutateCR  func(cr *instancev1alpha1.BlobStore)
		createErr error
		noSecret  bool
		wantErr   string
		wantIs    error
		check     func(t *testing.T, bs v395.AzureBlobStoreApiModel)
	}{
		"AccountKey": {
			check: func(t *testing.T, bs v395.AzureBlobStoreApiModel) {
				t.Helper()

				bucket := bs.BucketConfiguration
				if bs.Name != testAzureName || bucket.AccountName != testAccountName ||
					bucket.ContainerName != testContainerName ||
					bucket.Authentication.AuthenticationMethod != testAuthAccountKey ||
					bucket.Authentication.GetAccountKey() != testAccountKey {
					t.Errorf("Create() sent %+v", bs)
				}

				if bucket.PreSignedUrlEnabled != nil || bs.SoftQuota != nil {
					t.Errorf("Create() sent unset optional fields: %+v", bs)
				}
			},
		},
		"ManagedIdentityWithOptions": {
			mutateCR: func(cr *instancev1alpha1.BlobStore) {
				cr.Spec.ForProvider.AzureConfig.AuthenticationMethod = "MANAGEDIDENTITY"
				cr.Spec.ForProvider.AzureConfig.AccountKeySecretRef = nil
				cr.Spec.ForProvider.AzureConfig.PreSignedURLEnabled = new(true)
				cr.Spec.ForProvider.SoftQuota = &instancev1alpha1.SoftQuota{
					Type:  new("spaceUsedQuota"),
					Limit: new(int64(1024)),
				}
			},
			noSecret: true,
			check: func(t *testing.T, bs v395.AzureBlobStoreApiModel) {
				t.Helper()

				bucket := bs.BucketConfiguration
				if bucket.Authentication.AuthenticationMethod != "MANAGEDIDENTITY" || bucket.Authentication.AccountKey != nil {
					t.Errorf("Create() sent authentication %+v", bucket.Authentication)
				}

				if !ptr.Deref(bucket.PreSignedUrlEnabled, false) {
					t.Error("Create() did not enable pre-signed URLs")
				}

				if bs.SoftQuota == nil || bs.SoftQuota.Type != "spaceUsedQuota" || bs.SoftQuota.GetLimit() != 1024 {
					t.Errorf("Create() sent soft quota %+v", bs.SoftQuota)
				}
			},
		},
		"MissingSecret": {
			noSecret: true,
			wantErr:  errFingerprint,
		},
		"ProEditionRequired": {
			createErr: proErr,
			wantErr:   errCreateBlobStore,
			wantIs:    instance.ErrProEditionRequired,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cr := newAzureBlobStoreCR()
			if tc.mutateCR != nil {
				tc.mutateCR(cr)
			}

			objs := append([]client.Object{cr}, withAccountKeySecret(!tc.noSecret)...)

			azure := &mockAzureClient{
				CreateFn: func(_ v395.AzureBlobStoreApiModel) error { return tc.createErr },
			}
			e, _ := newAzureTestExternal(t, azure, objs...)

			_, err := e.Create(context.Background(), cr)
			if expectError(t, "Create", err, tc.wantErr, tc.wantIs) {
				return
			}

			if len(azure.CreateCalls) != 1 {
				t.Fatalf("Create calls = %d, want 1", len(azure.CreateCalls))
			}

			tc.check(t, azure.CreateCalls[0])

			if got := cr.GetAnnotations()["crossplane.io/external-name"]; got != testAzureName {
				t.Errorf("external name = %q, want %q", got, testAzureName)
			}
		})
	}
}

// TestUpdateAzure covers the Azure branches of Update.
func TestUpdateAzure(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		cr := newAzureBlobStoreCR()
		azure := &mockAzureClient{}
		e, _ := newAzureTestExternal(t, azure, cr, newAccountKeySecret(testAccountKey))

		_, err := e.Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if len(azure.UpdateCalls) != 1 || azure.UpdateCalls[0].BucketConfiguration.Authentication.GetAccountKey() != testAccountKey {
			t.Fatalf("Update() calls = %+v", azure.UpdateCalls)
		}

		if cr.Status.AtProvider.CredentialsFingerprint == "" {
			t.Error("Update() recorded no credentials fingerprint")
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		cr := newAzureBlobStoreCR()
		azure := &mockAzureClient{
			UpdateFn: func(_ string, _ v395.AzureBlobStoreApiModel) error { return errors.New("boom") },
		}
		e, _ := newAzureTestExternal(t, azure, cr, newAccountKeySecret(testAccountKey))

		_, err := e.Update(context.Background(), cr)
		if err == nil || !strings.Contains(err.Error(), errUpdateBlobStore) {
			t.Fatalf("Update() error = %v, want it to contain %q", err, errUpdateBlobStore)
		}

		if cr.Status.AtProvider.CredentialsFingerprint != "" {
			t.Error("Update() recorded a fingerprint although the update failed")
		}
	})

	t.Run("MissingSecretKey", func(t *testing.T) {
		t.Parallel()

		cr := newAzureBlobStoreCR()
		secret := newAccountKeySecret(testAccountKey)
		secret.Data = map[string][]byte{"other": []byte("x")}
		azure := &mockAzureClient{}
		e, _ := newAzureTestExternal(t, azure, cr, secret)

		_, err := e.Update(context.Background(), cr)
		if err == nil || !strings.Contains(err.Error(), errGetAccountKey) {
			t.Fatalf("Update() error = %v, want it to contain %q", err, errGetAccountKey)
		}

		if len(azure.UpdateCalls) != 0 {
			t.Error("Update() called Nexus without the account key")
		}
	})
}
