package blobstore

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/test/mocks"
)

func TestObserve(t *testing.T) {
	testPath := "/data/blobs/test"

	tests := []struct {
		name           string
		cr             *v1alpha1.BlobStore
		mockSetup      func(*mocks.MockClient)
		wantExists     bool
		wantUpToDate   bool
		wantErr        bool
	}{
		{
			name: "FileNotFound",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &v1alpha1.S3Config{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &v1alpha1.S3Config{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestCreate(t *testing.T) {
	testPath := "/data/blobs/test"

	tests := []struct {
		name      string
		cr        *v1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "CreateFileBlobStore",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-file-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-s3-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &v1alpha1.S3Config{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test-file-blobstore"},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestUpdate(t *testing.T) {
	testPath := "/data/blobs/test"

	tests := []struct {
		name      string
		cr        *v1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "UpdateFileBlobStore",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-file-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
				if len(mc.MockBlobStore.UpdateFileCalls) != 1 {
					t.Errorf("Expected 1 UpdateFile call, got %d", len(mc.MockBlobStore.UpdateFileCalls))
				}
			},
		},
		{
			name: "UpdateS3BlobStore",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-s3-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-s3-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
						Name: "test-s3-blobstore",
						Type: "S3",
						S3Config: &v1alpha1.S3Config{
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
				if len(mc.MockBlobStore.UpdateS3Calls) != 1 {
					t.Errorf("Expected 1 UpdateS3 call, got %d", len(mc.MockBlobStore.UpdateS3Calls))
				}
			},
		},
		{
			name: "UpdateFileBlobStoreError",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-file-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.BlobStore
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "DeleteBlobStoreSuccess",
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-blobstore",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-blobstore",
					},
				},
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			err := e.Delete(context.Background(), tt.cr)

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

func TestIsNotFound(t *testing.T) {
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
			if got := isNotFound(tt.err); got != tt.want {
				t.Errorf("isNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateFileBlobStore(t *testing.T) {
	testPath := "/data/blobs/test"
	quotaType := "spaceRemainingQuota"
	quotaLimit := int64(1000000)

	tests := []struct {
		name string
		cr   *v1alpha1.BlobStore
		want *blobstore.File
	}{
		{
			name: "BasicFileBlobStore",
			cr: &v1alpha1.BlobStore{
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
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
			cr: &v1alpha1.BlobStore{
				Spec: v1alpha1.BlobStoreSpec{
					ForProvider: v1alpha1.BlobStoreParameters{
						Name: "test-blobstore",
						Type: "File",
						SoftQuota: &v1alpha1.SoftQuota{
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
			got := generateFileBlobStore(tt.cr)
			if got.Name != tt.want.Name {
				t.Errorf("generateFileBlobStore() Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Path != tt.want.Path {
				t.Errorf("generateFileBlobStore() Path = %v, want %v", got.Path, tt.want.Path)
			}
			if tt.want.SoftQuota != nil {
				if got.SoftQuota == nil {
					t.Error("generateFileBlobStore() SoftQuota is nil, want non-nil")
				} else {
					if got.SoftQuota.Type != tt.want.SoftQuota.Type {
						t.Errorf("generateFileBlobStore() SoftQuota.Type = %v, want %v", got.SoftQuota.Type, tt.want.SoftQuota.Type)
					}
					if got.SoftQuota.Limit != tt.want.SoftQuota.Limit {
						t.Errorf("generateFileBlobStore() SoftQuota.Limit = %v, want %v", got.SoftQuota.Limit, tt.want.SoftQuota.Limit)
					}
				}
			}
		})
	}
}
