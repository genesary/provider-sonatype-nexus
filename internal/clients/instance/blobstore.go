package instance

import (
	blobstoreSchema "github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// BlobStoreClient provides methods for managing Nexus capabilities.
type BlobStoreClient interface {
	Delete(name string) error
	GetQuotaStatus(name string) (*blobstoreSchema.QuotaStatus, error)
	List() ([]blobstoreSchema.Generic, error)
}

// BlobStoreS3Client provides methods for managing Nexus S3 blobstores.
type BlobStoreS3Client interface {
	Create(bs *blobstoreSchema.S3) error
	Delete(name string) error
	Get(name string) (*blobstoreSchema.S3, error)
	GetQuotaStatus(name string) (*blobstoreSchema.QuotaStatus, error)
	Update(name string, bs *blobstoreSchema.S3) error
}

// BlobStoreAzureClient provides methods for managing Nexus Azure blobstores.
type BlobStoreAzureClient interface {
	Create(bs *blobstoreSchema.Azure) error
	Delete(name string) error
	Get(name string) (*blobstoreSchema.Azure, error)
	GetQuotaStatus(name string) (*blobstoreSchema.QuotaStatus, error)
	Update(name string, bs *blobstoreSchema.Azure) error
}

// BlobStoreFileClient provides methods for managing Nexus File blobstores.
type BlobStoreFileClient interface {
	Create(bs *blobstoreSchema.File) error
	Delete(name string) error
	Get(name string) (*blobstoreSchema.File, error)
	GetQuotaStatus(name string) (*blobstoreSchema.QuotaStatus, error)
	Update(name string, bs *blobstoreSchema.File) error
}

// BlobStoreGroupClient provides methods for managing Nexus Group blobstores.
type BlobStoreGroupClient interface {
	Create(bs *blobstoreSchema.Group) error
	Delete(name string) error
	Get(name string) (*blobstoreSchema.Group, error)
	GetQuotaStatus(name string) (*blobstoreSchema.QuotaStatus, error)
	Update(name string, bs *blobstoreSchema.Group) error
}

// NewBlobStoreClient returns a BlobStoreClient backed by a live Nexus
// connection.
func NewBlobStoreClient(creds nexus.Credentials) (BlobStoreClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.BlobStore, nil
}

// NewBlobStoreS3Client returns a BlobStoreS3Client backed by a live Nexus.
func NewBlobStoreS3Client(creds nexus.Credentials) (BlobStoreS3Client, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.BlobStore.S3, nil
}

// NewBlobStoreAzureClient returns a BlobStoreAzureClient
// backed by a live Nexus.
func NewBlobStoreAzureClient(creds nexus.Credentials) (BlobStoreAzureClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.BlobStore.Azure, nil
}

// NewBlobStoreFileClient returns a BlobStoreFileClient backed by a live Nexus.
func NewBlobStoreFileClient(creds nexus.Credentials) (BlobStoreFileClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.BlobStore.File, nil
}

// NewBlobStoreGroupClient returns a BlobStoreGroupClient
// backed by a live Nexus.
func NewBlobStoreGroupClient(creds nexus.Credentials) (BlobStoreGroupClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.BlobStore.Group, nil
}
