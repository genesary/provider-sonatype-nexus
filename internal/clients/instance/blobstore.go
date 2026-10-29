package instance

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// BlobStoreClient provides the operations shared by every Nexus blob store
// type.
type BlobStoreClient interface {
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]v395.GenericBlobStoreApiResponse, error)
}

// BlobStoreFileClient provides methods for managing Nexus File blobstores.
type BlobStoreFileClient interface {
	Create(ctx context.Context, bs v395.FileBlobStoreApiCreateRequest) error
	Get(ctx context.Context, name string) (*v395.FileBlobStoreApiModel, error)
	Update(ctx context.Context, name string, bs v395.FileBlobStoreApiUpdateRequest) error
}

// BlobStoreS3Client provides methods for managing Nexus S3 blobstores.
type BlobStoreS3Client interface {
	Create(ctx context.Context, bs v395.S3BlobStoreApiModel) error
	Get(ctx context.Context, name string) (*v395.S3BlobStoreApiModel, error)
	Update(ctx context.Context, name string, bs v395.S3BlobStoreApiModel) error
}

// blobStoreClient implements BlobStoreClient over the Nexus API.
type blobStoreClient struct {
	api *v395.BlobStoreAPIService
}

// fileBlobStoreClient implements BlobStoreFileClient over the Nexus API.
type fileBlobStoreClient struct {
	api *v395.BlobStoreAPIService
}

// s3BlobStoreClient implements BlobStoreS3Client over the Nexus API.
type s3BlobStoreClient struct {
	api *v395.BlobStoreAPIService
}

// NewBlobStoreClients returns the generic, File and S3 blob store clients,
// all backed by one live Nexus connection.
func NewBlobStoreClients(creds nexus.Credentials) (BlobStoreClient, BlobStoreFileClient, BlobStoreS3Client, error) {
	client, err := nexus.NewClient(creds)
	if err != nil {
		return nil, nil, nil, err
	}

	return &blobStoreClient{api: client.BlobStoreAPI},
		&fileBlobStoreClient{api: client.BlobStoreAPI},
		&s3BlobStoreClient{api: client.BlobStoreAPI},
		nil
}

// Delete deletes the blob store with the given name, whatever its type.
func (c *blobStoreClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteBlobstores(ctx, name).Execute())
}

// List returns every blob store, with its type and usage statistics.
func (c *blobStoreClient) List(ctx context.Context) ([]v395.GenericBlobStoreApiResponse, error) {
	stores, resp, err := c.api.ListBlobstores(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return stores, nil
}

// Create creates a File blob store.
func (c *fileBlobStoreClient) Create(ctx context.Context, bs v395.FileBlobStoreApiCreateRequest) error {
	return nexus.APIError(c.api.CreateBlobstoresFile(ctx).FileBlobStoreApiCreateRequest(bs).Execute())
}

// Get returns the File blob store with the given name.
func (c *fileBlobStoreClient) Get(ctx context.Context, name string) (*v395.FileBlobStoreApiModel, error) {
	bs, resp, err := c.api.GetBlobstoresFile(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return bs, nil
}

// Update updates the File blob store with the given name.
func (c *fileBlobStoreClient) Update(ctx context.Context, name string, bs v395.FileBlobStoreApiUpdateRequest) error {
	return nexus.APIError(c.api.UpdateBlobstoresFile(ctx, name).FileBlobStoreApiUpdateRequest(bs).Execute())
}

// Create creates an S3 blob store.
func (c *s3BlobStoreClient) Create(ctx context.Context, bs v395.S3BlobStoreApiModel) error {
	return nexus.APIError(c.api.CreateS3BlobStore(ctx).S3BlobStoreApiModel(bs).Execute())
}

// Get returns the S3 blob store with the given name.
func (c *s3BlobStoreClient) Get(ctx context.Context, name string) (*v395.S3BlobStoreApiModel, error) {
	bs, resp, err := c.api.GetS3BlobStore(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return bs, nil
}

// Update updates the S3 blob store with the given name.
func (c *s3BlobStoreClient) Update(ctx context.Context, name string, bs v395.S3BlobStoreApiModel) error {
	return nexus.APIError(c.api.UpdateS3BlobStore(ctx, name).S3BlobStoreApiModel(bs).Execute())
}
