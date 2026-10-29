package blobstore

import (
	"context"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// newFileBlobStoreWithQuota returns a File blob store CR whose spec asks for
// the given soft quota.
func newFileBlobStoreWithQuota(name, quotaType string, quotaLimit int64) *instancev1alpha1.BlobStore {
	cr := newBlobStoreCR(name, "File")
	cr.Spec.ForProvider.SoftQuota = &instancev1alpha1.SoftQuota{
		Type:  &quotaType,
		Limit: &quotaLimit,
	}

	return cr
}

// TestObserveReplacesStaleObservation tests that a setting removed from the
// blob store out of band is dropped from the observation, so that the up to
// date check no longer matches the spec against a value Nexus no longer holds.
func TestObserveReplacesStaleObservation(t *testing.T) {
	t.Parallel()

	cr := newFileBlobStoreWithQuota("test-blobstore", "spaceRemainingQuota", 1024)

	file := &mockFileClient{
		GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
			return &v395.FileBlobStoreApiModel{
				SoftQuota: &v395.BlobStoreApiSoftQuota{Type: "spaceRemainingQuota", Limit: new(int64(1024))},
			}, nil
		},
	}

	ext := newTestExternal(file, nil, nil)

	observation, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if !observation.ResourceUpToDate {
		t.Fatalf("Observe() ResourceUpToDate = false, want true for a matching quota")
	}

	// The quota is dropped from Nexus behind the provider's back.
	file.GetFn = func(_ string) (*v395.FileBlobStoreApiModel, error) {
		return &v395.FileBlobStoreApiModel{}, nil
	}

	observation, err = ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if cr.Status.AtProvider.SoftQuotaType != nil || cr.Status.AtProvider.SoftQuotaLimit != nil {
		t.Errorf("AtProvider still reports the removed quota: %+v", cr.Status.AtProvider)
	}

	if observation.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = true, want false after the quota was removed")
	}
}

// TestObserveReportsQuotaRemovedFromSpec tests that dropping the soft quota
// from the spec is reported as drift: the payload builders omit the block,
// which is how Nexus is told to remove the quota.
func TestObserveReportsQuotaRemovedFromSpec(t *testing.T) {
	t.Parallel()

	cr := newBlobStoreCR("test-blobstore", "File")

	file := &mockFileClient{
		GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
			return &v395.FileBlobStoreApiModel{
				SoftQuota: &v395.BlobStoreApiSoftQuota{Type: "spaceRemainingQuota", Limit: new(int64(1024))},
			}, nil
		},
	}

	observation, err := newTestExternal(file, nil, nil).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if observation.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = true, want false when the spec asks for no quota")
	}
}

// TestObserveIgnoresBucketExpiration tests that the deprecated S3 bucket
// expiration takes no part in drift detection: Nexus neither accepts nor
// reports it, so comparing it would report drift forever.
func TestObserveIgnoresBucketExpiration(t *testing.T) {
	t.Parallel()

	expiration := int32(7)

	cr := newBlobStoreCR("test-s3-blobstore", "S3")
	cr.Spec.ForProvider.S3Config = &instancev1alpha1.S3Config{
		Bucket:         "test-bucket",
		ExpirationDays: &expiration, //nolint:staticcheck // asserts the deprecated field is ignored
	}

	s3Client := &mockS3Client{
		GetFn: func(_ string) (*v395.S3BlobStoreApiModel, error) {
			return &v395.S3BlobStoreApiModel{
				Name: "test-s3-blobstore",
				BucketConfiguration: v395.S3BlobStoreApiBucketConfiguration{
					Bucket: v395.S3BlobStoreApiBucket{Name: "test-bucket", Region: "us-east-1"},
				},
			}, nil
		},
	}

	observation, err := newTestExternal(nil, s3Client, nil).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if !observation.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = false, want true when only the expiration differs")
	}

	//nolint:staticcheck // asserts the deprecated field is never set
	if cr.Status.AtProvider.BucketExpirationDays != nil {
		t.Errorf("AtProvider.BucketExpirationDays = %v, want nil", cr.Status.AtProvider.BucketExpirationDays) //nolint:staticcheck // see above
	}
}

// TestObserveRecordsNameTypeAndStats tests that the observation reports the
// blob store identity along with the stats read from the listing endpoint.
func TestObserveRecordsNameTypeAndStats(t *testing.T) {
	t.Parallel()

	cr := newBlobStoreCR("test-blobstore", "File")

	file := &mockFileClient{
		GetFn: func(_ string) (*v395.FileBlobStoreApiModel, error) {
			return &v395.FileBlobStoreApiModel{Path: "/data/blobs/test"}, nil
		},
	}

	generic := &mockGenericClient{
		ListFn: func() ([]v395.GenericBlobStoreApiResponse, error) {
			return []v395.GenericBlobStoreApiResponse{{
				Name:                  new("test-blobstore"),
				Type:                  new("File"),
				AvailableSpaceInBytes: new(int64(42)),
				TotalSizeInBytes:      new(int64(100)),
				BlobCount:             new(int64(7)),
			}}, nil
		},
	}

	_, err := newTestExternal(file, nil, generic).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	observed := cr.Status.AtProvider

	if observed.Name != "test-blobstore" {
		t.Errorf("AtProvider.Name = %q, want test-blobstore", observed.Name)
	}

	if observed.Type != "File" {
		t.Errorf("AtProvider.Type = %q, want File", observed.Type)
	}

	if observed.BlobCount == nil || *observed.BlobCount != 7 {
		t.Errorf("AtProvider.BlobCount = %v, want 7", observed.BlobCount)
	}

	if observed.Path == nil || *observed.Path != "/data/blobs/test" {
		t.Errorf("AtProvider.Path = %v, want /data/blobs/test", observed.Path)
	}
}
