package blobstore

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
	"github.com/genesary/provider-sonatype-nexus/test/mocks"
)

// newTestScheme builds a runtime.Scheme with corev1 types registered.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name         string
		cr           *repositoryv1alpha1.BlobStore
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "FileNotFound",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "FileExistsAndUpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
						Path: &testPath,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
					return &blobstore.File{
						Name: "test-blobstore",
						Path: testPath,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "FileExistsButOutdated",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
						Path: &testPath,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
					return &blobstore.File{
						Name: "test-blobstore",
						Path: "/different/path",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "S3NotFound",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: "test-bucket",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetS3Fn = func(ctx context.Context, name string) (*blobstore.S3, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "S3ExistsAndUpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: "test-bucket",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetS3Fn = func(ctx context.Context, name string) (*blobstore.S3, error) {
					return &blobstore.S3{
						Name: "test-s3-blobstore",
						BucketConfiguration: blobstore.S3BucketConfiguration{
							Bucket: blobstore.S3Bucket{
								Name: "test-bucket",
							},
						},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "GetFileError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
					return nil, errors.New("connection error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "AzureBlobStoreNotFound",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetAzureFn = func(ctx context.Context, name string) (*blobstore.Azure, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "AzureBlobStoreExistsUpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetAzureFn = func(ctx context.Context, name string) (*blobstore.Azure, error) {
					return &blobstore.Azure{
						Name: "test-azure-blobstore",
						BucketConfiguration: blobstore.AzureBucketConfiguration{
							AccountName:   "myaccount",
							ContainerName: "nexus-blobs",
							Authentication: blobstore.AzureBucketConfigurationAuthentication{
								AuthenticationMethod: blobstore.AzureAuthenticationMethodManagedIdentity,
							},
						},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "AzureBlobStoreExistsNotUpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "newaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetAzureFn = func(ctx context.Context, name string) (*blobstore.Azure, error) {
					return &blobstore.Azure{
						Name: "test-azure-blobstore",
						BucketConfiguration: blobstore.AzureBucketConfiguration{
							AccountName:   "oldaccount",
							ContainerName: "nexus-blobs",
							Authentication: blobstore.AzureBucketConfigurationAuthentication{
								AuthenticationMethod: blobstore.AzureAuthenticationMethodManagedIdentity,
							},
						},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetAzureError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.GetAzureFn = func(ctx context.Context, name string) (*blobstore.Azure, error) {
					return nil, errors.New("server error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			obs, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name      string
		cr        *repositoryv1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "CreateFileBlobStore",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-file-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-file-blobstore",
						Type: "File",
						Path: &testPath,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.CreateFileFn = func(ctx context.Context, bs *blobstore.File) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.CreateFileCalls) != 1 {
					t.Errorf("Expected 1 CreateFile call, got %d", len(mc.MockBlobStore.CreateFileCalls))
				}

				if mc.MockBlobStore.CreateFileCalls[0].Name != "test-file-blobstore" {
					t.Errorf("CreateFile called with wrong name: %s", mc.MockBlobStore.CreateFileCalls[0].Name)
				}
			},
		},
		{
			name: "CreateS3BlobStore",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: "my-bucket",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.CreateS3Fn = func(ctx context.Context, bs *blobstore.S3) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.CreateS3Calls) != 1 {
					t.Errorf("Expected 1 CreateS3 call, got %d", len(mc.MockBlobStore.CreateS3Calls))
				}

				if mc.MockBlobStore.CreateS3Calls[0].Name != "test-s3-blobstore" {
					t.Errorf("CreateS3 called with wrong name: %s", mc.MockBlobStore.CreateS3Calls[0].Name)
				}
			},
		},
		{
			name: "CreateFileBlobStoreError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-file-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-file-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.CreateFileFn = func(ctx context.Context, bs *blobstore.File) error {
					return errors.New("create error")
				}
			},
			wantErr: true,
		},
		{
			name: "CreateAzureBlobStoreManagedIdentity",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.CreateAzureFn = func(ctx context.Context, bs *blobstore.Azure) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.CreateAzureCalls) != 1 {
					t.Errorf("Expected 1 CreateAzure call, got %d", len(mc.MockBlobStore.CreateAzureCalls))
				}

				got := mc.MockBlobStore.CreateAzureCalls[0]
				if got.BucketConfiguration.AccountName != "myaccount" {
					t.Errorf("CreateAzure AccountName = %q, want %q", got.BucketConfiguration.AccountName, "myaccount")
				}
			},
		},
		{
			name: "CreateAzureBlobStoreError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.CreateAzureFn = func(ctx context.Context, bs *blobstore.Azure) error {
					return errors.New("nexus error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name      string
		cr        *repositoryv1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "UpdateFileBlobStore",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-file-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-file-blobstore",
						Type: "File",
						Path: &testPath,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.UpdateFileFn = func(ctx context.Context, name string, bs *blobstore.File) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.UpdateFileCalls) != 1 {
					t.Errorf("Expected 1 UpdateFile call, got %d", len(mc.MockBlobStore.UpdateFileCalls))
				}
			},
		},
		{
			name: "UpdateS3BlobStore",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-s3-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-s3-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: "my-bucket",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.UpdateS3Fn = func(ctx context.Context, name string, bs *blobstore.S3) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.UpdateS3Calls) != 1 {
					t.Errorf("Expected 1 UpdateS3 call, got %d", len(mc.MockBlobStore.UpdateS3Calls))
				}
			},
		},
		{
			name: "UpdateFileBlobStoreError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-file-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-file-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.UpdateFileFn = func(ctx context.Context, name string, bs *blobstore.File) error {
					return errors.New("update error")
				}
			},
			wantErr: true,
		},
		{
			name: "UpdateAzureBlobStoreManagedIdentity",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-azure-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-azure-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.UpdateAzureFn = func(ctx context.Context, name string, bs *blobstore.Azure) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.UpdateAzureCalls) != 1 {
					t.Errorf("Expected 1 UpdateAzure call, got %d", len(mc.MockBlobStore.UpdateAzureCalls))
				}
			},
		},
		{
			name: "UpdateAzureBlobStoreError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-azure-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-azure-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-azure-blobstore",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "myaccount",
							ContainerName:        "nexus-blobs",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.UpdateAzureFn = func(ctx context.Context, name string, bs *blobstore.Azure) error {
					return errors.New("nexus error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *repositoryv1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "DeleteBlobStoreSuccess",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.DeleteFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockBlobStore.DeleteCalls) != 1 {
					t.Errorf("Expected 1 Delete call, got %d", len(mc.MockBlobStore.DeleteCalls))
				}

				if mc.MockBlobStore.DeleteCalls[0] != "test-blobstore" {
					t.Errorf("Delete called with wrong name: %s", mc.MockBlobStore.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteBlobStoreNotFound",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.DeleteFn = func(ctx context.Context, name string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false, // Not found is not an error for delete
		},
		{
			name: "DeleteBlobStoreError",
			cr: &repositoryv1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockBlobStore.DeleteFn = func(ctx context.Context, name string) error {
					return errors.New("connection error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestIsNotFound tests the isNotFound function.
func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NilError",
			err:  nil,
			want: false,
		},
		{
			name: "404Error",
			err:  errors.New("404 not found"),
			want: true,
		},
		{
			name: "NotFoundError",
			err:  errors.New("resource not found"),
			want: true,
		},
		{
			name: "DoesNotExistError",
			err:  errors.New("resource does not exist"),
			want: true,
		},
		{
			name: "OtherError",
			err:  errors.New("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := helpers.IsNotFound(tt.err); got != tt.want {
				t.Errorf("helpers.IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateFileBlobStore tests the generateFileBlobStore function.
func TestGenerateFileBlobStore(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"
	quotaType := "spaceRemainingQuota"
	quotaLimit := int64(1000000)

	tests := []struct {
		name string
		cr   *repositoryv1alpha1.BlobStore
		want *blobstore.File
	}{
		{
			name: "BasicFileBlobStore",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
					},
				},
			},
			want: &blobstore.File{
				Name: "test-blobstore",
			},
		},
		{
			name: "FileBlobStoreWithPath",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
						Path: &testPath,
					},
				},
			},
			want: &blobstore.File{
				Name: "test-blobstore",
				Path: testPath,
			},
		},
		{
			name: "FileBlobStoreWithQuota",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
			},
			want: &blobstore.File{
				Name: "test-blobstore",
				SoftQuota: &blobstore.SoftQuota{
					Type:  quotaType,
					Limit: quotaLimit,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateFileBlobStore(tt.cr)
			if got.Name != tt.want.Name {
				t.Errorf("generateFileBlobStore() Name = %v, want %v", got.Name, tt.want.Name)
			}

			if got.Path != tt.want.Path {
				t.Errorf("generateFileBlobStore() Path = %v, want %v", got.Path, tt.want.Path)
			}

			if tt.want.SoftQuota == nil {
				return
			}

			if got.SoftQuota == nil {
				t.Error("generateFileBlobStore() SoftQuota is nil, want non-nil")

				return
			}

			if got.SoftQuota.Type != tt.want.SoftQuota.Type {
				t.Errorf("generateFileBlobStore() SoftQuota.Type = %v, want %v", got.SoftQuota.Type, tt.want.SoftQuota.Type)
			}

			if got.SoftQuota.Limit != tt.want.SoftQuota.Limit {
				t.Errorf("generateFileBlobStore() SoftQuota.Limit = %v, want %v", got.SoftQuota.Limit, tt.want.SoftQuota.Limit)
			}
		})
	}
}

// TestCreate_AzureWithAccountKey tests Create with Azure ACCOUNTKEY auth, resolving the secret.
func TestCreate_AzureWithAccountKey(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"accountKey": []byte("super-secret-key"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	mc := mocks.NewMockClient()
	mc.MockBlobStore.CreateAzureFn = func(ctx context.Context, bs *blobstore.Azure) error {
		if bs.BucketConfiguration.Authentication.AccountKey != "super-secret-key" {
			t.Errorf("CreateAzure AccountKey = %q, want %q",
				bs.BucketConfiguration.Authentication.AccountKey, "super-secret-key")
		}

		return nil
	}

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-azure-blobstore",
				Type: "Azure",
				AzureConfig: &repositoryv1alpha1.AzureConfig{
					AccountName:          "myaccount",
					ContainerName:        "nexus-blobs",
					AuthenticationMethod: "ACCOUNTKEY",
					AccountKeySecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "azure-secret",
							Namespace: "default",
						},
						Key: "accountKey",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Errorf("Create() with account key returned unexpected error: %v", err)
	}
}

// TestCreate_AzureSecretNotFound tests Create returns error when the account key secret is missing.
func TestCreate_AzureSecretNotFound(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	mc := mocks.NewMockClient()

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-azure-blobstore"},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-azure-blobstore",
				Type: "Azure",
				AzureConfig: &repositoryv1alpha1.AzureConfig{
					AccountName:          "myaccount",
					ContainerName:        "nexus-blobs",
					AuthenticationMethod: "ACCOUNTKEY",
					AccountKeySecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "missing-secret",
							Namespace: "default",
						},
						Key: "accountKey",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Error("Create() with missing secret should return error, got nil")
	}
}

// TestUpdate_AzureWithAccountKey tests Update with Azure ACCOUNTKEY auth, resolving the secret.
func TestUpdate_AzureWithAccountKey(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"accountKey": []byte("super-secret-key"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	mc := mocks.NewMockClient()
	mc.MockBlobStore.UpdateAzureFn = func(ctx context.Context, name string, bs *blobstore.Azure) error {
		return nil
	}

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-azure-blobstore",
			Annotations: map[string]string{
				"crossplane.io/external-name": "test-azure-blobstore",
			},
		},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-azure-blobstore",
				Type: "Azure",
				AzureConfig: &repositoryv1alpha1.AzureConfig{
					AccountName:          "myaccount",
					ContainerName:        "nexus-blobs",
					AuthenticationMethod: "ACCOUNTKEY",
					AccountKeySecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "azure-secret",
							Namespace: "default",
						},
						Key: "accountKey",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Errorf("Update() with account key returned unexpected error: %v", err)
	}
}

// TestUpdate_AzureSecretNotFound tests Update returns error when the account key secret is missing.
func TestUpdate_AzureSecretNotFound(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	mc := mocks.NewMockClient()

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-azure-blobstore",
			Annotations: map[string]string{
				"crossplane.io/external-name": "test-azure-blobstore",
			},
		},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-azure-blobstore",
				Type: "Azure",
				AzureConfig: &repositoryv1alpha1.AzureConfig{
					AccountName:          "myaccount",
					ContainerName:        "nexus-blobs",
					AuthenticationMethod: "ACCOUNTKEY",
					AccountKeySecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "missing-secret",
							Namespace: "default",
						},
						Key: "accountKey",
					},
				},
			},
		},
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Update(context.Background(), cr)
	if err == nil {
		t.Error("Update() with missing secret should return error, got nil")
	}
}

// TestGenerateAzureBlobStore tests the generateAzureBlobStore method.
func TestGenerateAzureBlobStore(t *testing.T) {
	t.Parallel()

	quotaType := "spaceUsedQuota"
	quotaLimit := int64(1073741824)

	tests := []struct {
		name     string
		cr       *repositoryv1alpha1.BlobStore
		wantName string
		wantAuth string
		wantCont string
		wantAcct string
	}{
		{
			name: "ManagedIdentity",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "azure-store",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "storageacct",
							ContainerName:        "container",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
					},
				},
			},
			wantName: "azure-store",
			wantAuth: "MANAGEDIDENTITY",
			wantCont: "container",
			wantAcct: "storageacct",
		},
		{
			name: "WithSoftQuota",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "azure-store",
						Type: "Azure",
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "storageacct",
							ContainerName:        "container",
							AuthenticationMethod: "MANAGEDIDENTITY",
						},
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
			},
			wantName: "azure-store",
			wantAuth: "MANAGEDIDENTITY",
			wantCont: "container",
			wantAcct: "storageacct",
		},
		{
			name: "NoAzureConfig",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "azure-store",
						Type: "Azure",
					},
				},
			},
			wantName: "azure-store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: mocks.NewMockClient()}

			got, err := e.generateAzureBlobStore(context.Background(), tt.cr)
			if err != nil {
				t.Fatalf("generateAzureBlobStore() unexpected error: %v", err)
			}

			if got.Name != tt.wantName {
				t.Errorf("generateAzureBlobStore() Name = %q, want %q", got.Name, tt.wantName)
			}

			if tt.cr.Spec.ForProvider.AzureConfig == nil {
				return
			}

			if string(got.BucketConfiguration.Authentication.AuthenticationMethod) != tt.wantAuth {
				t.Errorf("generateAzureBlobStore() AuthMethod = %q, want %q",
					got.BucketConfiguration.Authentication.AuthenticationMethod, tt.wantAuth)
			}

			if got.BucketConfiguration.ContainerName != tt.wantCont {
				t.Errorf("generateAzureBlobStore() ContainerName = %q, want %q",
					got.BucketConfiguration.ContainerName, tt.wantCont)
			}

			if got.BucketConfiguration.AccountName != tt.wantAcct {
				t.Errorf("generateAzureBlobStore() AccountName = %q, want %q",
					got.BucketConfiguration.AccountName, tt.wantAcct)
			}
		})
	}
}

// TestPopulateAzureBlobStoreObservation tests the populateAzureBlobStoreObservation function.
func TestPopulateAzureBlobStoreObservation(t *testing.T) {
	t.Parallel()

	cr := &repositoryv1alpha1.BlobStore{}
	az := &blobstore.Azure{
		Name: "azure-store",
		BucketConfiguration: blobstore.AzureBucketConfiguration{
			AccountName:   "storageacct",
			ContainerName: "container",
			Authentication: blobstore.AzureBucketConfigurationAuthentication{
				AuthenticationMethod: blobstore.AzureAuthenticationMethodManagedIdentity,
			},
		},
		SoftQuota: &blobstore.SoftQuota{
			Type:  "spaceUsedQuota",
			Limit: int64(1073741824),
		},
	}

	populateAzureBlobStoreObservation(cr, az)

	if cr.Status.AtProvider.AzureAccountName == nil || *cr.Status.AtProvider.AzureAccountName != "storageacct" {
		t.Errorf("AzureAccountName = %v, want %q", cr.Status.AtProvider.AzureAccountName, "storageacct")
	}

	if cr.Status.AtProvider.AzureContainerName == nil || *cr.Status.AtProvider.AzureContainerName != "container" {
		t.Errorf("AzureContainerName = %v, want %q", cr.Status.AtProvider.AzureContainerName, "container")
	}

	if cr.Status.AtProvider.AzureAuthenticationMethod == nil || *cr.Status.AtProvider.AzureAuthenticationMethod != "MANAGEDIDENTITY" {
		t.Errorf("AzureAuthenticationMethod = %v, want %q", cr.Status.AtProvider.AzureAuthenticationMethod, "MANAGEDIDENTITY")
	}

	if cr.Status.AtProvider.SoftQuotaType == nil || *cr.Status.AtProvider.SoftQuotaType != "spaceUsedQuota" {
		t.Errorf("SoftQuotaType = %v, want %q", cr.Status.AtProvider.SoftQuotaType, "spaceUsedQuota")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: mocks.NewMockClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}

// TestObserve_WrongType tests that Observe returns error for non-BlobStore resource.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: mocks.NewMockClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() expected error for nil managed resource, got nil")
	}
}

// TestCreate_WrongType tests that Create returns error for non-BlobStore resource.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: mocks.NewMockClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() expected error for nil managed resource, got nil")
	}
}

// TestUpdate_WrongType tests that Update returns error for non-BlobStore resource.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: mocks.NewMockClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() expected error for nil managed resource, got nil")
	}
}

// TestDelete_WrongType tests that Delete returns error for non-BlobStore resource.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: mocks.NewMockClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() expected error for nil managed resource, got nil")
	}
}

// TestObserveTypedBlobStore_NilResult tests that observeTypedBlobStore handles nil result from getter.
func TestObserveTypedBlobStore_NilResult(t *testing.T) {
	t.Parallel()

	mc := mocks.NewMockClient()
	mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
		return nil, nil
	}

	e := &external{client: mc}

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-blobstore",
				Type: "File",
			},
		},
	}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if obs.ResourceExists {
		t.Error("Observe() ResourceExists should be false for nil result")
	}
}

// TestObserve_PopulateStats tests that Observe populates blob store stats when List returns data.
func TestObserve_PopulateStats(t *testing.T) {
	t.Parallel()

	mc := mocks.NewMockClient()
	mc.MockBlobStore.GetFileFn = func(ctx context.Context, name string) (*blobstore.File, error) {
		return &blobstore.File{Name: "test-blobstore"}, nil
	}
	mc.MockBlobStore.ListFn = func(ctx context.Context) ([]blobstore.Generic, error) {
		return []blobstore.Generic{
			{
				Name:                  "test-blobstore",
				AvailableSpaceInBytes: 512000000,
				TotalSizeInBytes:      1073741824,
				BlobCount:             42,
			},
			{
				Name: "other-store",
			},
		}, nil
	}

	e := &external{client: mc}

	cr := &repositoryv1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
		Spec: repositoryv1alpha1.BlobStoreSpec{
			ForProvider: repositoryv1alpha1.BlobStoreParameters{
				Name: "test-blobstore",
				Type: "File",
			},
		},
	}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists should be true")
	}

	if cr.Status.AtProvider.BlobCount == nil || *cr.Status.AtProvider.BlobCount != 42 {
		t.Errorf("Observe() BlobCount = %v, want 42", cr.Status.AtProvider.BlobCount)
	}

	if cr.Status.AtProvider.TotalSizeInBytes == nil || *cr.Status.AtProvider.TotalSizeInBytes != 1073741824 {
		t.Errorf("Observe() TotalSizeInBytes = %v, want 1073741824", cr.Status.AtProvider.TotalSizeInBytes)
	}
}

// TestGenerateS3BlobStore tests the generateS3BlobStore function with all optional fields.
func TestGenerateS3BlobStore(t *testing.T) {
	t.Parallel()

	region := "us-east-1"
	prefix := "nexus/"
	endpoint := "https://s3.example.com"
	expiry := int32(30)
	forcePathStyle := true
	assumeRole := "arn:aws:iam::123:role/nexus"
	quotaType := "spaceUsedQuota"
	quotaLimit := int64(1073741824)

	tests := []struct {
		name        string
		cr          *repositoryv1alpha1.BlobStore
		wantBucket  string
		wantRegion  string
		wantPrefix  string
		wantExpiry  int32
	}{
		{
			name: "FullS3Config",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "s3-store",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket:         "my-bucket",
							Region:         &region,
							Prefix:         &prefix,
							ExpirationDays: &expiry,
							Endpoint:       &endpoint,
							ForcePathStyle: &forcePathStyle,
							AssumeRole:     &assumeRole,
						},
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
			},
			wantBucket: "my-bucket",
			wantRegion: region,
			wantPrefix: prefix,
			wantExpiry: expiry,
		},
		{
			name: "BareS3Config",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "s3-store",
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: "bare-bucket",
						},
					},
				},
			},
			wantBucket: "bare-bucket",
		},
		{
			name: "NoS3Config",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						Name: "s3-store",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateS3BlobStore(tt.cr)
			if got.Name != tt.cr.Spec.ForProvider.Name {
				t.Errorf("generateS3BlobStore() Name = %q, want %q", got.Name, tt.cr.Spec.ForProvider.Name)
			}

			if tt.wantBucket == "" {
				return
			}

			if got.BucketConfiguration.Bucket.Name != tt.wantBucket {
				t.Errorf("generateS3BlobStore() BucketName = %q, want %q",
					got.BucketConfiguration.Bucket.Name, tt.wantBucket)
			}

			if tt.wantRegion != "" && got.BucketConfiguration.Bucket.Region != tt.wantRegion {
				t.Errorf("generateS3BlobStore() Region = %q, want %q",
					got.BucketConfiguration.Bucket.Region, tt.wantRegion)
			}

			if tt.wantPrefix != "" && got.BucketConfiguration.Bucket.Prefix != tt.wantPrefix {
				t.Errorf("generateS3BlobStore() Prefix = %q, want %q",
					got.BucketConfiguration.Bucket.Prefix, tt.wantPrefix)
			}

			if tt.wantExpiry != 0 && got.BucketConfiguration.Bucket.Expiration != tt.wantExpiry {
				t.Errorf("generateS3BlobStore() Expiry = %d, want %d",
					got.BucketConfiguration.Bucket.Expiration, tt.wantExpiry)
			}
		})
	}
}

// TestPopulateFileBlobStoreObservation tests populateFileBlobStoreObservation with soft quota.
func TestPopulateFileBlobStoreObservation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		file      *blobstore.File
		wantPath  string
		wantQuota bool
	}{
		{
			name: "WithQuota",
			file: &blobstore.File{
				Path: "/data/blobs",
				SoftQuota: &blobstore.SoftQuota{
					Type:  "spaceUsedQuota",
					Limit: int64(1073741824),
				},
			},
			wantPath:  "/data/blobs",
			wantQuota: true,
		},
		{
			name:     "NoQuota",
			file:     &blobstore.File{Path: "/data"},
			wantPath: "/data",
		},
		{
			name: "EmptyPath",
			file: &blobstore.File{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := &repositoryv1alpha1.BlobStore{}
			populateFileBlobStoreObservation(cr, tt.file)

			if tt.wantPath != "" {
				if cr.Status.AtProvider.Path == nil || *cr.Status.AtProvider.Path != tt.wantPath {
					t.Errorf("Path = %v, want %q", cr.Status.AtProvider.Path, tt.wantPath)
				}
			}

			if tt.wantQuota {
				if cr.Status.AtProvider.SoftQuotaType == nil {
					t.Error("SoftQuotaType should be set")
				}

				if cr.Status.AtProvider.SoftQuotaLimit == nil {
					t.Error("SoftQuotaLimit should be set")
				}
			}
		})
	}
}

// TestPopulateS3BlobStoreObservation tests populateS3BlobStoreObservation.
func TestPopulateS3BlobStoreObservation(t *testing.T) {
	t.Parallel()

	region := "us-east-1"
	prefix := "nexus/"

	tests := []struct {
		name       string
		s3         *blobstore.S3
		wantBucket string
		wantRegion string
		wantPrefix string
		wantQuota  bool
	}{
		{
			name: "Full",
			s3: &blobstore.S3{
				BucketConfiguration: blobstore.S3BucketConfiguration{
					Bucket: blobstore.S3Bucket{
						Name:   "my-bucket",
						Region: region,
						Prefix: prefix,
					},
				},
				SoftQuota: &blobstore.SoftQuota{
					Type:  "spaceUsedQuota",
					Limit: int64(1073741824),
				},
			},
			wantBucket: "my-bucket",
			wantRegion: region,
			wantPrefix: prefix,
			wantQuota:  true,
		},
		{
			name: "BucketOnly",
			s3: &blobstore.S3{
				BucketConfiguration: blobstore.S3BucketConfiguration{
					Bucket: blobstore.S3Bucket{Name: "my-bucket"},
				},
			},
			wantBucket: "my-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := &repositoryv1alpha1.BlobStore{}
			populateS3BlobStoreObservation(cr, tt.s3)

			if cr.Status.AtProvider.BucketName == nil || *cr.Status.AtProvider.BucketName != tt.wantBucket {
				t.Errorf("BucketName = %v, want %q", cr.Status.AtProvider.BucketName, tt.wantBucket)
			}

			if tt.wantRegion != "" {
				if cr.Status.AtProvider.BucketRegion == nil || *cr.Status.AtProvider.BucketRegion != tt.wantRegion {
					t.Errorf("BucketRegion = %v, want %q", cr.Status.AtProvider.BucketRegion, tt.wantRegion)
				}
			}

			if tt.wantPrefix != "" {
				if cr.Status.AtProvider.BucketPrefix == nil || *cr.Status.AtProvider.BucketPrefix != tt.wantPrefix {
					t.Errorf("BucketPrefix = %v, want %q", cr.Status.AtProvider.BucketPrefix, tt.wantPrefix)
				}
			}

			if tt.wantQuota {
				if cr.Status.AtProvider.SoftQuotaType == nil {
					t.Error("SoftQuotaType should be set")
				}
			}
		})
	}
}

// TestIsSoftQuotaUpToDate tests the isSoftQuotaUpToDate function exhaustively.
func TestIsSoftQuotaUpToDate(t *testing.T) {
	t.Parallel()

	quotaType := "spaceUsedQuota"
	otherType := "spaceRemainingQuota"
	quotaLimit := int64(1073741824)
	otherLimit := int64(2147483648)

	tests := []struct {
		name string
		cr   *repositoryv1alpha1.BlobStore
		want bool
	}{
		{
			name: "NoSoftQuota",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{},
				},
			},
			want: true,
		},
		{
			name: "ObsNil",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{},
				},
			},
			want: false,
		},
		{
			name: "UpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						SoftQuotaType:  &quotaType,
						SoftQuotaLimit: &quotaLimit,
					},
				},
			},
			want: true,
		},
		{
			name: "TypeChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						SoftQuotaType:  &otherType,
						SoftQuotaLimit: &quotaLimit,
					},
				},
			},
			want: false,
		},
		{
			name: "LimitChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						SoftQuota: &repositoryv1alpha1.SoftQuota{
							Type:  &quotaType,
							Limit: &quotaLimit,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						SoftQuotaType:  &quotaType,
						SoftQuotaLimit: &otherLimit,
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isSoftQuotaUpToDate(tt.cr); got != tt.want {
				t.Errorf("isSoftQuotaUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsS3BucketConfigUpToDate tests isS3BucketConfigUpToDate exhaustively.
func TestIsS3BucketConfigUpToDate(t *testing.T) {
	t.Parallel()

	region := "us-east-1"
	otherRegion := "eu-west-1"
	prefix := "nexus/"
	otherPrefix := "other/"
	bucket := "my-bucket"
	otherBucket := "other-bucket"

	tests := []struct {
		name string
		cr   *repositoryv1alpha1.BlobStore
		want bool
	}{
		{
			name: "NoS3Config",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{},
				},
			},
			want: true,
		},
		{
			name: "BucketChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{Bucket: otherBucket},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName: &bucket,
					},
				},
			},
			want: false,
		},
		{
			name: "RegionChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: bucket,
							Region: &otherRegion,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName:   &bucket,
						BucketRegion: &region,
					},
				},
			},
			want: false,
		},
		{
			name: "PrefixChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: bucket,
							Prefix: &otherPrefix,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName:   &bucket,
						BucketPrefix: &prefix,
					},
				},
			},
			want: false,
		},
		{
			name: "UpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: bucket,
							Region: &region,
							Prefix: &prefix,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName:   &bucket,
						BucketRegion: &region,
						BucketPrefix: &prefix,
					},
				},
			},
			want: true,
		},
		{
			name: "RegionNilInObs",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: bucket,
							Region: &region,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName: &bucket,
					},
				},
			},
			want: false,
		},
		{
			name: "PrefixNilInObs",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						S3Config: &repositoryv1alpha1.S3Config{
							Bucket: bucket,
							Prefix: &prefix,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						BucketName: &bucket,
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isS3BucketConfigUpToDate(tt.cr); got != tt.want {
				t.Errorf("isS3BucketConfigUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsAzureBlobStoreUpToDate tests the isAzureBlobStoreUpToDate function.
func TestIsAzureBlobStoreUpToDate(t *testing.T) {
	t.Parallel()

	acct := "storageacct"
	container := "container"
	method := "MANAGEDIDENTITY"
	other := "othervalue"

	tests := []struct {
		name string
		cr   *repositoryv1alpha1.BlobStore
		want bool
	}{
		{
			name: "NoAzureConfig",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{},
				},
			},
			want: true,
		},
		{
			name: "UpToDate",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          acct,
							ContainerName:        container,
							AuthenticationMethod: method,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						AzureAccountName:          &acct,
						AzureContainerName:        &container,
						AzureAuthenticationMethod: &method,
					},
				},
			},
			want: true,
		},
		{
			name: "AccountNameChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          "newacct",
							ContainerName:        container,
							AuthenticationMethod: method,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						AzureAccountName:          &acct,
						AzureContainerName:        &container,
						AzureAuthenticationMethod: &method,
					},
				},
			},
			want: false,
		},
		{
			name: "ContainerChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          acct,
							ContainerName:        "newcontainer",
							AuthenticationMethod: method,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						AzureAccountName:          &acct,
						AzureContainerName:        &container,
						AzureAuthenticationMethod: &method,
					},
				},
			},
			want: false,
		},
		{
			name: "AuthMethodChanged",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          acct,
							ContainerName:        container,
							AuthenticationMethod: "ACCOUNTKEY",
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						AzureAccountName:          &acct,
						AzureContainerName:        &container,
						AzureAuthenticationMethod: &method,
					},
				},
			},
			want: false,
		},
		{
			name: "ObservationMissing",
			cr: &repositoryv1alpha1.BlobStore{
				Spec: repositoryv1alpha1.BlobStoreSpec{
					ForProvider: repositoryv1alpha1.BlobStoreParameters{
						AzureConfig: &repositoryv1alpha1.AzureConfig{
							AccountName:          acct,
							ContainerName:        container,
							AuthenticationMethod: method,
						},
					},
				},
				Status: repositoryv1alpha1.BlobStoreStatus{
					AtProvider: repositoryv1alpha1.BlobStoreObservation{
						AzureAccountName:          nil,
						AzureContainerName:        &other,
						AzureAuthenticationMethod: &other,
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isAzureBlobStoreUpToDate(tt.cr); got != tt.want {
				t.Errorf("isAzureBlobStoreUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
