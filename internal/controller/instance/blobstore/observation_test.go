package blobstore

import (
	"context"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"

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
		GetFn: func(_ string) (*blobstore.File, error) {
			return &blobstore.File{
				Name:      "test-blobstore",
				SoftQuota: &blobstore.SoftQuota{Type: "spaceRemainingQuota", Limit: 1024},
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
	file.GetFn = func(_ string) (*blobstore.File, error) {
		return &blobstore.File{Name: "test-blobstore"}, nil
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
		GetFn: func(_ string) (*blobstore.File, error) {
			return &blobstore.File{
				Name:      "test-blobstore",
				SoftQuota: &blobstore.SoftQuota{Type: "spaceRemainingQuota", Limit: 1024},
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

// TestObserveReportsBucketExpirationDrift tests that the S3 bucket expiration
// the provider submits is compared against the observed one.
func TestObserveReportsBucketExpirationDrift(t *testing.T) {
	t.Parallel()

	expiration := int32(7)

	cr := newBlobStoreCR("test-s3-blobstore", "S3")
	cr.Spec.ForProvider.S3Config = &instancev1alpha1.S3Config{
		Bucket:         "test-bucket",
		ExpirationDays: &expiration,
	}

	s3Client := &mockS3Client{
		GetFn: func(_ string) (*blobstore.S3, error) {
			return &blobstore.S3{
				Name: "test-s3-blobstore",
				BucketConfiguration: blobstore.S3BucketConfiguration{
					Bucket: blobstore.S3Bucket{Name: "test-bucket", Expiration: 3},
				},
			}, nil
		},
	}

	observation, err := newTestExternal(nil, s3Client, nil).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if observation.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = true, want false for a differing bucket expiration")
	}

	if cr.Status.AtProvider.BucketExpirationDays == nil || *cr.Status.AtProvider.BucketExpirationDays != 3 {
		t.Errorf("AtProvider.BucketExpirationDays = %v, want 3", cr.Status.AtProvider.BucketExpirationDays)
	}
}

// TestObserveRecordsNameTypeAndStats tests that the observation reports the
// blob store identity along with the stats read from the listing endpoint.
func TestObserveRecordsNameTypeAndStats(t *testing.T) {
	t.Parallel()

	cr := newBlobStoreCR("test-blobstore", "File")

	file := &mockFileClient{
		GetFn: func(_ string) (*blobstore.File, error) {
			return &blobstore.File{Name: "test-blobstore", Path: "/data/blobs/test"}, nil
		},
	}

	generic := &mockGenericClient{
		ListFn: func() ([]blobstore.Generic, error) {
			return []blobstore.Generic{{
				Name:                  "test-blobstore",
				Type:                  "File",
				AvailableSpaceInBytes: 42,
				TotalSizeInBytes:      100,
				BlobCount:             7,
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
