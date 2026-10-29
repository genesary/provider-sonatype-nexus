package blobstore

import (
	"context"
	"errors"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// errNotConfigured is returned by mock functions that have not been configured.
var errNotConfigured = errors.New("mock function not configured")

// mockGenericClient is a mock implementing instance.BlobStoreClient.
type mockGenericClient struct {
	DeleteFn    func(name string) error
	ListFn      func() ([]v395.GenericBlobStoreApiResponse, error)
	DeleteCalls []string
}

// Delete records the call and delegates to DeleteFn if set.
func (m *mockGenericClient) Delete(_ context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)
	if m.DeleteFn != nil {
		return m.DeleteFn(name)
	}

	return nil
}

// List delegates to ListFn if set, otherwise returns an empty list.
func (m *mockGenericClient) List(_ context.Context) ([]v395.GenericBlobStoreApiResponse, error) {
	if m.ListFn != nil {
		return m.ListFn()
	}

	return []v395.GenericBlobStoreApiResponse{}, nil
}

// fileCall captures arguments to a file-client Create or Update call.
type fileCall struct {
	Name string
	Path string
}

// mockFileClient is a mock implementing instance.BlobStoreFileClient.
type mockFileClient struct {
	GetFn       func(name string) (*v395.FileBlobStoreApiModel, error)
	CreateFn    func(bs v395.FileBlobStoreApiCreateRequest) error
	UpdateFn    func(name string, bs v395.FileBlobStoreApiUpdateRequest) error
	CreateCalls []fileCall
	UpdateCalls []fileCall
}

// Get delegates to GetFn; returns errNotConfigured if GetFn is nil.
func (m *mockFileClient) Get(_ context.Context, name string) (*v395.FileBlobStoreApiModel, error) {
	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errNotConfigured
}

// Create records the call and delegates to CreateFn if set.
func (m *mockFileClient) Create(_ context.Context, bs v395.FileBlobStoreApiCreateRequest) error {
	m.CreateCalls = append(m.CreateCalls, fileCall{Name: bs.Name, Path: bs.Path})
	if m.CreateFn != nil {
		return m.CreateFn(bs)
	}

	return nil
}

// Update records the call and delegates to UpdateFn if set.
func (m *mockFileClient) Update(_ context.Context, name string, bs v395.FileBlobStoreApiUpdateRequest) error {
	m.UpdateCalls = append(m.UpdateCalls, fileCall{Name: name, Path: bs.Path})
	if m.UpdateFn != nil {
		return m.UpdateFn(name, bs)
	}

	return nil
}

// s3Call captures arguments to an S3-client Create or Update call.
type s3Call struct {
	Name string
	BS   v395.S3BlobStoreApiModel
}

// mockS3Client is a mock implementing instance.BlobStoreS3Client.
type mockS3Client struct {
	GetFn       func(name string) (*v395.S3BlobStoreApiModel, error)
	CreateFn    func(bs v395.S3BlobStoreApiModel) error
	UpdateFn    func(name string, bs v395.S3BlobStoreApiModel) error
	CreateCalls []s3Call
	UpdateCalls []s3Call
}

// Get delegates to GetFn; returns errNotConfigured if GetFn is nil.
func (m *mockS3Client) Get(_ context.Context, name string) (*v395.S3BlobStoreApiModel, error) {
	if m.GetFn != nil {
		return m.GetFn(name)
	}

	return nil, errNotConfigured
}

// Create records the call and delegates to CreateFn if set.
func (m *mockS3Client) Create(_ context.Context, bs v395.S3BlobStoreApiModel) error {
	m.CreateCalls = append(m.CreateCalls, s3Call{Name: bs.Name, BS: bs})
	if m.CreateFn != nil {
		return m.CreateFn(bs)
	}

	return nil
}

// Update records the call and delegates to UpdateFn if set.
func (m *mockS3Client) Update(_ context.Context, name string, bs v395.S3BlobStoreApiModel) error {
	m.UpdateCalls = append(m.UpdateCalls, s3Call{Name: name, BS: bs})
	if m.UpdateFn != nil {
		return m.UpdateFn(name, bs)
	}

	return nil
}

// newTestExternal builds an *external with typed mocks, using empty mocks for
// any nil argument.
func newTestExternal(file *mockFileClient, s3 *mockS3Client, generic *mockGenericClient) *external {
	if file == nil {
		file = &mockFileClient{}
	}

	if s3 == nil {
		s3 = &mockS3Client{}
	}

	if generic == nil {
		generic = &mockGenericClient{}
	}

	return &external{
		fileClient:    file,
		s3Client:      s3,
		genericClient: generic,
	}
}

// newBlobStoreCR creates a minimal BlobStore CR for testing.
func newBlobStoreCR(name, bsType string) *instancev1alpha1.BlobStore {
	return &instancev1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"crossplane.io/external-name": name,
			},
		},
		Spec: instancev1alpha1.BlobStoreSpec{
			ForProvider: instancev1alpha1.BlobStoreParameters{
				Name: name,
				Type: bsType,
			},
		},
	}
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name         string
		cr           *instancev1alpha1.BlobStore
		file         *mockFileClient
		s3           *mockS3Client
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "FileNotFound",
			cr:   newBlobStoreCR("test-blobstore", "File"),
			file: &mockFileClient{
				GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
					return nil, nexus.ErrNotFound
				},
			},
		},
		{
			name: "FileExistsAndUpToDate",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-blobstore", "File")
				cr.Spec.ForProvider.Path = &testPath

				return cr
			}(),
			file: &mockFileClient{
				GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
					return &v395.FileBlobStoreApiModel{Path: testPath}, nil
				},
			},
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name: "S3NotFound",
			cr:   newBlobStoreCR("test-s3-blobstore", "S3"),
			s3: &mockS3Client{
				GetFn: func(_ string) (*v395.S3BlobStoreApiModel, error) {
					return nil, nexus.ErrNotFound
				},
			},
		},
		{
			name: "S3ExistsAndUpToDate",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-s3-blobstore", "S3")
				cr.Spec.ForProvider.S3Config = &instancev1alpha1.S3Config{Bucket: "test-bucket"}

				return cr
			}(),
			s3: &mockS3Client{
				GetFn: func(_ string) (*v395.S3BlobStoreApiModel, error) {
					return &v395.S3BlobStoreApiModel{
						Name: "test-s3-blobstore",
						BucketConfiguration: v395.S3BlobStoreApiBucketConfiguration{
							Bucket: v395.S3BlobStoreApiBucket{Name: "test-bucket"},
						},
					}, nil
				},
			},
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name: "GetFileError",
			cr:   newBlobStoreCR("test-blobstore", "File"),
			file: &mockFileClient{
				GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
					return nil, errors.New("connection error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestExternal(tt.file, tt.s3, nil)
			obs, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if obs.ResourceExists != tt.wantExists {
					t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
				}

				if obs.ResourceUpToDate != tt.wantUpToDate {
					t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
				}
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name     string
		cr       *instancev1alpha1.BlobStore
		file     *mockFileClient
		s3       *mockS3Client
		wantErr  bool
		validate func(t *testing.T, file *mockFileClient, s3 *mockS3Client)
	}{
		{
			name: "CreateFileBlobStore",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-file-blobstore", "File")
				cr.Spec.ForProvider.Path = &testPath

				return cr
			}(),
			file: &mockFileClient{},
			validate: func(t *testing.T, file *mockFileClient, _ *mockS3Client) {
				t.Helper()

				if len(file.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(file.CreateCalls))
				}

				if len(file.CreateCalls) > 0 && file.CreateCalls[0].Name != "test-file-blobstore" {
					t.Errorf("Create called with wrong name: %s", file.CreateCalls[0].Name)
				}
			},
		},
		{
			name: "CreateS3BlobStore",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-s3-blobstore", "S3")
				cr.Spec.ForProvider.S3Config = &instancev1alpha1.S3Config{Bucket: "my-bucket"}

				return cr
			}(),
			s3: &mockS3Client{},
			validate: func(t *testing.T, _ *mockFileClient, s3 *mockS3Client) {
				t.Helper()

				if len(s3.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(s3.CreateCalls))
				}

				if len(s3.CreateCalls) > 0 && s3.CreateCalls[0].Name != "test-s3-blobstore" {
					t.Errorf("Create called with wrong name: %s", s3.CreateCalls[0].Name)
				}
			},
		},
		{
			name: "CreateFileError",
			cr:   newBlobStoreCR("test-file-blobstore", "File"),
			file: &mockFileClient{
				CreateFn: func(_ v395.FileBlobStoreApiCreateRequest) error { return errors.New("create error") },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file := tt.file
			s3 := tt.s3
			e := newTestExternal(file, s3, nil)
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, file, s3)
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	testPath := "/data/blobs/test"

	tests := []struct {
		name     string
		cr       *instancev1alpha1.BlobStore
		file     *mockFileClient
		s3       *mockS3Client
		wantErr  bool
		validate func(t *testing.T, file *mockFileClient, s3 *mockS3Client)
	}{
		{
			name: "UpdateFileBlobStore",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-file-blobstore", "File")
				cr.Spec.ForProvider.Path = &testPath

				return cr
			}(),
			file: &mockFileClient{},
			validate: func(t *testing.T, file *mockFileClient, _ *mockS3Client) {
				t.Helper()

				if len(file.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(file.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateS3BlobStore",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-s3-blobstore", "S3")
				cr.Spec.ForProvider.S3Config = &instancev1alpha1.S3Config{Bucket: "my-bucket"}

				return cr
			}(),
			s3: &mockS3Client{},
			validate: func(t *testing.T, _ *mockFileClient, s3 *mockS3Client) {
				t.Helper()

				if len(s3.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(s3.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateFileError",
			cr:   newBlobStoreCR("test-file-blobstore", "File"),
			file: &mockFileClient{
				UpdateFn: func(_ string, _ v395.FileBlobStoreApiUpdateRequest) error { return errors.New("update error") },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file := tt.file
			s3 := tt.s3
			e := newTestExternal(file, s3, nil)
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, file, s3)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cr       *instancev1alpha1.BlobStore
		generic  *mockGenericClient
		wantErr  bool
		validate func(t *testing.T, generic *mockGenericClient)
	}{
		{
			name:    "DeleteBlobStoreSuccess",
			cr:      newBlobStoreCR("test-blobstore", "File"),
			generic: &mockGenericClient{},
			validate: func(t *testing.T, generic *mockGenericClient) {
				t.Helper()

				if len(generic.DeleteCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(generic.DeleteCalls))
				}

				if len(generic.DeleteCalls) > 0 && generic.DeleteCalls[0] != "test-blobstore" {
					t.Errorf("Delete called with wrong name: %s", generic.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteBlobStoreNotFound",
			cr:   newBlobStoreCR("test-blobstore", "File"),
			generic: &mockGenericClient{
				DeleteFn: func(_ string) error { return nexus.ErrNotFound },
			},
		},
		{
			name: "DeleteBlobStoreError",
			cr:   newBlobStoreCR("test-blobstore", "File"),
			generic: &mockGenericClient{
				DeleteFn: func(_ string) error { return errors.New("connection error") },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			generic := tt.generic
			e := newTestExternal(nil, nil, generic)
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, generic)
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
		cr   *instancev1alpha1.BlobStore
		want v395.FileBlobStoreApiCreateRequest
	}{
		{
			name: "FileBlobStoreNoPath",
			cr:   newBlobStoreCR("test-blobstore", "File"),
			want: v395.FileBlobStoreApiCreateRequest{Name: "test-blobstore"},
		},
		{
			name: "FileBlobStoreWithPath",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-blobstore", "File")
				cr.Spec.ForProvider.Path = &testPath

				return cr
			}(),
			want: v395.FileBlobStoreApiCreateRequest{Name: "test-blobstore", Path: testPath},
		},
		{
			name: "FileBlobStoreWithQuota",
			cr: func() *instancev1alpha1.BlobStore {
				cr := newBlobStoreCR("test-blobstore", "File")
				cr.Spec.ForProvider.SoftQuota = &instancev1alpha1.SoftQuota{
					Type:  &quotaType,
					Limit: &quotaLimit,
				}

				return cr
			}(),
			want: v395.FileBlobStoreApiCreateRequest{
				Name:      "test-blobstore",
				SoftQuota: &v395.BlobStoreApiSoftQuota{Type: quotaType, Limit: &quotaLimit},
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
				t.Error("generateFileBlobStore() SoftQuota nil, want non-nil")

				return
			}

			if got.SoftQuota.Type != tt.want.SoftQuota.Type {
				t.Errorf("generateFileBlobStore() SoftQuota.Type = %v, want %v",
					got.SoftQuota.Type, tt.want.SoftQuota.Type)
			}

			if got.SoftQuota.GetLimit() != tt.want.SoftQuota.GetLimit() {
				t.Errorf("generateFileBlobStore() SoftQuota.Limit = %v, want %v",
					got.SoftQuota.GetLimit(), tt.want.SoftQuota.GetLimit())
			}
		})
	}
}
