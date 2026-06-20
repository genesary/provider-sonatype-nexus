package nexus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
)

const blobstorePath = "/service/rest/v1/blobstores"

// TestBlobStoreService_GetFile_Success tests GetFile returns the File blob store on 200.
func TestBlobStoreService_GetFile_Success(t *testing.T) {
	t.Parallel()

	want := &blobstore.File{Name: "file-store", Path: "/data/blobs"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == blobstorePath+"/file/file-store" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.BlobStore().GetFile(context.Background(), "file-store")
	if err != nil {
		t.Fatalf("GetFile() unexpected error: %v", err)
	}

	if got.Path != want.Path {
		t.Errorf("GetFile() Path = %q, want %q", got.Path, want.Path)
	}
}

// TestBlobStoreService_GetFile_Error tests GetFile returns error on non-200.
func TestBlobStoreService_GetFile_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.BlobStore().GetFile(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetFile() expected error for non-200, got nil")
	}
}

// TestBlobStoreService_GetS3_Success tests GetS3 returns the S3 blob store on 200.
func TestBlobStoreService_GetS3_Success(t *testing.T) {
	t.Parallel()

	want := &blobstore.S3{
		Name: "s3-store",
		BucketConfiguration: blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{Name: "my-bucket"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == blobstorePath+"/s3/s3-store" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.BlobStore().GetS3(context.Background(), "s3-store")
	if err != nil {
		t.Fatalf("GetS3() unexpected error: %v", err)
	}

	if got.BucketConfiguration.Bucket.Name != want.BucketConfiguration.Bucket.Name {
		t.Errorf("GetS3() BucketName = %q, want %q", got.BucketConfiguration.Bucket.Name, want.BucketConfiguration.Bucket.Name)
	}
}

// TestBlobStoreService_GetS3_Error tests GetS3 returns error on non-200.
func TestBlobStoreService_GetS3_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.BlobStore().GetS3(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetS3() expected error for non-200, got nil")
	}
}

// TestBlobStoreService_GetAzure_Success tests GetAzure returns the Azure blob store on 200.
func TestBlobStoreService_GetAzure_Success(t *testing.T) {
	t.Parallel()

	want := &blobstore.Azure{
		Name: "azure-store",
		BucketConfiguration: blobstore.AzureBucketConfiguration{
			AccountName:   "myaccount",
			ContainerName: "nexus-blobs",
			Authentication: blobstore.AzureBucketConfigurationAuthentication{
				AuthenticationMethod: blobstore.AzureAuthenticationMethodManagedIdentity,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == blobstorePath+"/azure/azure-store" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.BlobStore().GetAzure(context.Background(), "azure-store")
	if err != nil {
		t.Fatalf("GetAzure() unexpected error: %v", err)
	}

	if got.BucketConfiguration.AccountName != want.BucketConfiguration.AccountName {
		t.Errorf("GetAzure() AccountName = %q, want %q", got.BucketConfiguration.AccountName, want.BucketConfiguration.AccountName)
	}
}

// TestBlobStoreService_GetAzure_Error tests GetAzure returns error on non-200.
func TestBlobStoreService_GetAzure_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.BlobStore().GetAzure(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetAzure() expected error for non-200, got nil")
	}
}

// TestBlobStoreService_CreateFile_Success tests CreateFile succeeds on 201.
func TestBlobStoreService_CreateFile_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.File{Name: "new-file-store", Path: "/data/blobs"}

	var received *blobstore.File

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == blobstorePath+"/file" {
			received = &blobstore.File{}
			_ = json.NewDecoder(r.Body).Decode(received)
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateFile(context.Background(), bs)
	if err != nil {
		t.Fatalf("CreateFile() unexpected error: %v", err)
	}

	if received == nil {
		t.Fatal("CreateFile() server did not receive request body")
	}

	if received.Name != bs.Name {
		t.Errorf("CreateFile() sent Name = %q, want %q", received.Name, bs.Name)
	}
}

// TestBlobStoreService_CreateFile_Error tests CreateFile returns error on non-2xx.
func TestBlobStoreService_CreateFile_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "failed")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateFile(context.Background(), &blobstore.File{Name: "x"})
	if err == nil {
		t.Fatal("CreateFile() expected error for non-2xx, got nil")
	}
}

// TestBlobStoreService_CreateS3_Success tests CreateS3 succeeds on 204.
func TestBlobStoreService_CreateS3_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.S3{
		Name: "new-s3-store",
		BucketConfiguration: blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{Name: "my-bucket"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == blobstorePath+"/s3" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateS3(context.Background(), bs)
	if err != nil {
		t.Fatalf("CreateS3() unexpected error: %v", err)
	}
}

// TestBlobStoreService_CreateS3_Error tests CreateS3 returns error on non-2xx.
func TestBlobStoreService_CreateS3_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateS3(context.Background(), &blobstore.S3{Name: "x"})
	if err == nil {
		t.Fatal("CreateS3() expected error for non-2xx, got nil")
	}
}

// TestBlobStoreService_CreateAzure_Success tests CreateAzure succeeds on 204.
func TestBlobStoreService_CreateAzure_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.Azure{
		Name: "new-azure-store",
		BucketConfiguration: blobstore.AzureBucketConfiguration{
			AccountName:   "myaccount",
			ContainerName: "nexus-blobs",
			Authentication: blobstore.AzureBucketConfigurationAuthentication{
				AuthenticationMethod: blobstore.AzureAuthenticationMethodManagedIdentity,
			},
		},
	}

	var received *blobstore.Azure

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == blobstorePath+"/azure" {
			received = &blobstore.Azure{}
			_ = json.NewDecoder(r.Body).Decode(received)
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateAzure(context.Background(), bs)
	if err != nil {
		t.Fatalf("CreateAzure() unexpected error: %v", err)
	}

	if received == nil {
		t.Fatal("CreateAzure() server did not receive request body")
	}

	if received.Name != bs.Name {
		t.Errorf("CreateAzure() sent Name = %q, want %q", received.Name, bs.Name)
	}
}

// TestBlobStoreService_CreateAzure_Error tests CreateAzure returns error on non-2xx.
func TestBlobStoreService_CreateAzure_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "failed")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().CreateAzure(context.Background(), &blobstore.Azure{Name: "x"})
	if err == nil {
		t.Fatal("CreateAzure() expected error for non-2xx, got nil")
	}
}

// TestBlobStoreService_UpdateFile_Success tests UpdateFile succeeds on 204.
func TestBlobStoreService_UpdateFile_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.File{Name: "file-store", Path: "/new/path"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == blobstorePath+"/file/file-store" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateFile(context.Background(), "file-store", bs)
	if err != nil {
		t.Fatalf("UpdateFile() unexpected error: %v", err)
	}
}

// TestBlobStoreService_UpdateFile_Error tests UpdateFile returns error on non-204.
func TestBlobStoreService_UpdateFile_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "failed")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateFile(context.Background(), "store", &blobstore.File{Name: "store"})
	if err == nil {
		t.Fatal("UpdateFile() expected error for non-204, got nil")
	}
}

// TestBlobStoreService_UpdateS3_Success tests UpdateS3 succeeds on 204.
func TestBlobStoreService_UpdateS3_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.S3{
		Name: "s3-store",
		BucketConfiguration: blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{Name: "my-bucket"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == blobstorePath+"/s3/s3-store" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateS3(context.Background(), "s3-store", bs)
	if err != nil {
		t.Fatalf("UpdateS3() unexpected error: %v", err)
	}
}

// TestBlobStoreService_UpdateS3_Error tests UpdateS3 returns error on non-204.
func TestBlobStoreService_UpdateS3_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateS3(context.Background(), "s3-store", &blobstore.S3{Name: "s3-store"})
	if err == nil {
		t.Fatal("UpdateS3() expected error for non-204, got nil")
	}
}

// TestBlobStoreService_UpdateAzure_Success tests UpdateAzure succeeds on 204.
func TestBlobStoreService_UpdateAzure_Success(t *testing.T) {
	t.Parallel()

	bs := &blobstore.Azure{
		Name: "azure-store",
		BucketConfiguration: blobstore.AzureBucketConfiguration{
			AccountName:   "myaccount",
			ContainerName: "nexus-blobs",
			Authentication: blobstore.AzureBucketConfigurationAuthentication{
				AuthenticationMethod: blobstore.AzureAuthenticationMethodAccountKey,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == blobstorePath+"/azure/azure-store" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateAzure(context.Background(), "azure-store", bs)
	if err != nil {
		t.Fatalf("UpdateAzure() unexpected error: %v", err)
	}
}

// TestBlobStoreService_UpdateAzure_Error tests UpdateAzure returns error on non-204.
func TestBlobStoreService_UpdateAzure_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().UpdateAzure(context.Background(), "azure-store", &blobstore.Azure{Name: "azure-store"})
	if err == nil {
		t.Fatal("UpdateAzure() expected error for non-204, got nil")
	}
}

// TestBlobStoreService_Delete_Success tests Delete succeeds on 204.
func TestBlobStoreService_Delete_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == blobstorePath+"/old-store" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().Delete(context.Background(), "old-store")
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}
}

// TestBlobStoreService_Delete_Error tests Delete returns error on non-204.
func TestBlobStoreService_Delete_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.BlobStore().Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error for non-204, got nil")
	}
}

// TestBlobStoreService_List_Success tests List returns blob stores on 200.
func TestBlobStoreService_List_Success(t *testing.T) {
	t.Parallel()

	want := []blobstore.Generic{
		{Name: "store-a", Type: "File"},
		{Name: "store-b", Type: "S3"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == blobstorePath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.BlobStore().List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}

	if len(got) != len(want) {
		t.Errorf("List() len = %d, want %d", len(got), len(want))
	}
}

// TestBlobStoreService_List_Error tests List returns error on non-200.
func TestBlobStoreService_List_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.BlobStore().List(context.Background())
	if err == nil {
		t.Fatal("List() expected error for non-200, got nil")
	}
}
