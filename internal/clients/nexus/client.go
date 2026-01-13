// Package nexus provides a client interface for Sonatype Nexus Repository Manager.
package nexus

import (
	"context"
	"encoding/json"

	"github.com/datadrivers/go-nexus-client/nexus3"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
)

// Credentials contains the credentials for connecting to Nexus.
type Credentials struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// Client is an interface for interacting with the Nexus API.
type Client interface {
	BlobStore() BlobStoreService
	Repository() RepositoryService
	Security() SecurityService
}

// BlobStoreService provides methods for managing blob stores.
type BlobStoreService interface {
	GetFile(ctx context.Context, name string) (*blobstore.File, error)
	GetS3(ctx context.Context, name string) (*blobstore.S3, error)
	CreateFile(ctx context.Context, bs *blobstore.File) error
	CreateS3(ctx context.Context, bs *blobstore.S3) error
	UpdateFile(ctx context.Context, name string, bs *blobstore.File) error
	UpdateS3(ctx context.Context, name string, bs *blobstore.S3) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]blobstore.Generic, error)
}

// RepositoryService provides methods for managing repositories.
type RepositoryService interface {
	// Maven
	GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error)
	CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error
	UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error
	DeleteMavenHosted(ctx context.Context, name string) error

	GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error)
	CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error
	UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error
	DeleteMavenProxy(ctx context.Context, name string) error

	GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error)
	CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error
	UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error
	DeleteMavenGroup(ctx context.Context, name string) error

	// Docker
	GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error)
	CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error
	UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error
	DeleteDockerHosted(ctx context.Context, name string) error

	GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error)
	CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error
	UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error
	DeleteDockerProxy(ctx context.Context, name string) error

	GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error)
	CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error
	UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error
	DeleteDockerGroup(ctx context.Context, name string) error

	// npm
	GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error)
	CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error
	UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error
	DeleteNpmHosted(ctx context.Context, name string) error

	GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error)
	CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error
	UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error
	DeleteNpmProxy(ctx context.Context, name string) error

	GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error)
	CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error
	UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error
	DeleteNpmGroup(ctx context.Context, name string) error

	// Raw
	GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error)
	CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error
	UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error
	DeleteRawHosted(ctx context.Context, name string) error

	GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error)
	CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error
	UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error
	DeleteRawProxy(ctx context.Context, name string) error

	GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error)
	CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error
	UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error
	DeleteRawGroup(ctx context.Context, name string) error
}

// SecurityService provides methods for managing security resources.
type SecurityService interface {
	GetUser(ctx context.Context, id string) (*security.User, error)
	CreateUser(ctx context.Context, user security.User) error
	UpdateUser(ctx context.Context, id string, user security.User) error
	DeleteUser(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, id string, password string) error

	GetRole(ctx context.Context, id string) (*security.Role, error)
	CreateRole(ctx context.Context, role security.Role) error
	UpdateRole(ctx context.Context, id string, role security.Role) error
	DeleteRole(ctx context.Context, id string) error
}

// nexusClientWrapper implements the Client interface.
type nexusClientWrapper struct {
	client *nexus3.NexusClient
}

// blobStoreService implements BlobStoreService.
type blobStoreService struct {
	client *nexus3.NexusClient
}

// repositoryService implements RepositoryService.
type repositoryService struct {
	client *nexus3.NexusClient
}

// securityService implements SecurityService.
type securityService struct {
	client *nexus3.NexusClient
}

// NewClient creates a new Nexus client from the provided credentials.
func NewClient(creds Credentials) (Client, error) {
	cfg := client.Config{
		URL:      creds.URL,
		Username: creds.Username,
		Password: creds.Password,
		Insecure: creds.Insecure,
	}

	nc := nexus3.NewClient(cfg)
	if nc == nil {
		return nil, errors.New("failed to create Nexus client")
	}

	return &nexusClientWrapper{client: nc}, nil
}

// GetCredentialsFromSecret extracts Nexus credentials from a Kubernetes secret.
func GetCredentialsFromSecret(ctx context.Context, kube kubeclient.Client, pc *v1alpha1.ProviderConfig) (Credentials, error) {
	var creds Credentials

	if pc.Spec.Credentials.Source != "Secret" {
		return creds, errors.New("only Secret source is supported")
	}

	if pc.Spec.Credentials.SecretRef == nil {
		return creds, errors.New("secretRef is required when source is Secret")
	}

	secret := &corev1.Secret{}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      pc.Spec.Credentials.SecretRef.Name,
		Namespace: pc.Spec.Credentials.SecretRef.Namespace,
	}, secret)
	if err != nil {
		return creds, errors.Wrap(err, "failed to get credentials secret")
	}

	key := "credentials"
	if pc.Spec.Credentials.SecretRef.Key != "" {
		key = pc.Spec.Credentials.SecretRef.Key
	}

	data, ok := secret.Data[key]
	if !ok {
		return creds, errors.Errorf("secret does not contain key %q", key)
	}

	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, errors.Wrap(err, "failed to unmarshal credentials")
	}

	return creds, nil
}

// BlobStore returns the BlobStoreService.
func (c *nexusClientWrapper) BlobStore() BlobStoreService {
	return &blobStoreService{client: c.client}
}

// Repository returns the RepositoryService.
func (c *nexusClientWrapper) Repository() RepositoryService {
	return &repositoryService{client: c.client}
}

// Security returns the SecurityService.
func (c *nexusClientWrapper) Security() SecurityService {
	return &securityService{client: c.client}
}

// BlobStoreService implementations

// GetFile gets a File blob store by name.
func (s *blobStoreService) GetFile(ctx context.Context, name string) (*blobstore.File, error) {
	return s.client.BlobStore.File.Get(name)
}

// GetS3 gets an S3 blob store by name.
func (s *blobStoreService) GetS3(ctx context.Context, name string) (*blobstore.S3, error) {
	return s.client.BlobStore.S3.Get(name)
}

// CreateFile creates a File blob store.
func (s *blobStoreService) CreateFile(ctx context.Context, bs *blobstore.File) error {
	return s.client.BlobStore.File.Create(bs)
}

// CreateS3 creates an S3 blob store.
func (s *blobStoreService) CreateS3(ctx context.Context, bs *blobstore.S3) error {
	return s.client.BlobStore.S3.Create(bs)
}

// UpdateFile updates a File blob store.
func (s *blobStoreService) UpdateFile(ctx context.Context, name string, bs *blobstore.File) error {
	return s.client.BlobStore.File.Update(name, bs)
}

// UpdateS3 updates an S3 blob store.
func (s *blobStoreService) UpdateS3(ctx context.Context, name string, bs *blobstore.S3) error {
	return s.client.BlobStore.S3.Update(name, bs)
}

// Delete deletes a blob store by name.
func (s *blobStoreService) Delete(ctx context.Context, name string) error {
	return s.client.BlobStore.Delete(name)
}

// List lists all blob stores.
func (s *blobStoreService) List(ctx context.Context) ([]blobstore.Generic, error) {
	return s.client.BlobStore.List()
}

// RepositoryService implementations - Maven

// GetMavenHosted gets a Maven hosted repository.
func (s *repositoryService) GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
	return s.client.Repository.Maven.Hosted.Get(name)
}

// CreateMavenHosted creates a Maven hosted repository.
func (s *repositoryService) CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error {
	return s.client.Repository.Maven.Hosted.Create(repo)
}

// UpdateMavenHosted updates a Maven hosted repository.
func (s *repositoryService) UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
	return s.client.Repository.Maven.Hosted.Update(name, repo)
}

// DeleteMavenHosted deletes a Maven hosted repository.
func (s *repositoryService) DeleteMavenHosted(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Hosted.Delete(name)
}

// GetMavenProxy gets a Maven proxy repository.
func (s *repositoryService) GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error) {
	return s.client.Repository.Maven.Proxy.Get(name)
}

// CreateMavenProxy creates a Maven proxy repository.
func (s *repositoryService) CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error {
	return s.client.Repository.Maven.Proxy.Create(repo)
}

// UpdateMavenProxy updates a Maven proxy repository.
func (s *repositoryService) UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error {
	return s.client.Repository.Maven.Proxy.Update(name, repo)
}

// DeleteMavenProxy deletes a Maven proxy repository.
func (s *repositoryService) DeleteMavenProxy(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Proxy.Delete(name)
}

// GetMavenGroup gets a Maven group repository.
func (s *repositoryService) GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error) {
	return s.client.Repository.Maven.Group.Get(name)
}

// CreateMavenGroup creates a Maven group repository.
func (s *repositoryService) CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error {
	return s.client.Repository.Maven.Group.Create(repo)
}

// UpdateMavenGroup updates a Maven group repository.
func (s *repositoryService) UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error {
	return s.client.Repository.Maven.Group.Update(name, repo)
}

// DeleteMavenGroup deletes a Maven group repository.
func (s *repositoryService) DeleteMavenGroup(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Group.Delete(name)
}

// RepositoryService implementations - Docker

// GetDockerHosted gets a Docker hosted repository.
func (s *repositoryService) GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error) {
	return s.client.Repository.Docker.Hosted.Get(name)
}

// CreateDockerHosted creates a Docker hosted repository.
func (s *repositoryService) CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error {
	return s.client.Repository.Docker.Hosted.Create(repo)
}

// UpdateDockerHosted updates a Docker hosted repository.
func (s *repositoryService) UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error {
	return s.client.Repository.Docker.Hosted.Update(name, repo)
}

// DeleteDockerHosted deletes a Docker hosted repository.
func (s *repositoryService) DeleteDockerHosted(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Hosted.Delete(name)
}

// GetDockerProxy gets a Docker proxy repository.
func (s *repositoryService) GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error) {
	return s.client.Repository.Docker.Proxy.Get(name)
}

// CreateDockerProxy creates a Docker proxy repository.
func (s *repositoryService) CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error {
	return s.client.Repository.Docker.Proxy.Create(repo)
}

// UpdateDockerProxy updates a Docker proxy repository.
func (s *repositoryService) UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error {
	return s.client.Repository.Docker.Proxy.Update(name, repo)
}

// DeleteDockerProxy deletes a Docker proxy repository.
func (s *repositoryService) DeleteDockerProxy(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Proxy.Delete(name)
}

// GetDockerGroup gets a Docker group repository.
func (s *repositoryService) GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error) {
	return s.client.Repository.Docker.Group.Get(name)
}

// CreateDockerGroup creates a Docker group repository.
func (s *repositoryService) CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error {
	return s.client.Repository.Docker.Group.Create(repo)
}

// UpdateDockerGroup updates a Docker group repository.
func (s *repositoryService) UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error {
	return s.client.Repository.Docker.Group.Update(name, repo)
}

// DeleteDockerGroup deletes a Docker group repository.
func (s *repositoryService) DeleteDockerGroup(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Group.Delete(name)
}

// RepositoryService implementations - npm

// GetNpmHosted gets an npm hosted repository.
func (s *repositoryService) GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error) {
	return s.client.Repository.Npm.Hosted.Get(name)
}

// CreateNpmHosted creates an npm hosted repository.
func (s *repositoryService) CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error {
	return s.client.Repository.Npm.Hosted.Create(repo)
}

// UpdateNpmHosted updates an npm hosted repository.
func (s *repositoryService) UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error {
	return s.client.Repository.Npm.Hosted.Update(name, repo)
}

// DeleteNpmHosted deletes an npm hosted repository.
func (s *repositoryService) DeleteNpmHosted(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Hosted.Delete(name)
}

// GetNpmProxy gets an npm proxy repository.
func (s *repositoryService) GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error) {
	return s.client.Repository.Npm.Proxy.Get(name)
}

// CreateNpmProxy creates an npm proxy repository.
func (s *repositoryService) CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error {
	return s.client.Repository.Npm.Proxy.Create(repo)
}

// UpdateNpmProxy updates an npm proxy repository.
func (s *repositoryService) UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error {
	return s.client.Repository.Npm.Proxy.Update(name, repo)
}

// DeleteNpmProxy deletes an npm proxy repository.
func (s *repositoryService) DeleteNpmProxy(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Proxy.Delete(name)
}

// GetNpmGroup gets an npm group repository.
func (s *repositoryService) GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error) {
	return s.client.Repository.Npm.Group.Get(name)
}

// CreateNpmGroup creates an npm group repository.
func (s *repositoryService) CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error {
	return s.client.Repository.Npm.Group.Create(repo)
}

// UpdateNpmGroup updates an npm group repository.
func (s *repositoryService) UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error {
	return s.client.Repository.Npm.Group.Update(name, repo)
}

// DeleteNpmGroup deletes an npm group repository.
func (s *repositoryService) DeleteNpmGroup(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Group.Delete(name)
}

// RepositoryService implementations - Raw

// GetRawHosted gets a Raw hosted repository.
func (s *repositoryService) GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error) {
	return s.client.Repository.Raw.Hosted.Get(name)
}

// CreateRawHosted creates a Raw hosted repository.
func (s *repositoryService) CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error {
	return s.client.Repository.Raw.Hosted.Create(repo)
}

// UpdateRawHosted updates a Raw hosted repository.
func (s *repositoryService) UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error {
	return s.client.Repository.Raw.Hosted.Update(name, repo)
}

// DeleteRawHosted deletes a Raw hosted repository.
func (s *repositoryService) DeleteRawHosted(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Hosted.Delete(name)
}

// GetRawProxy gets a Raw proxy repository.
func (s *repositoryService) GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error) {
	return s.client.Repository.Raw.Proxy.Get(name)
}

// CreateRawProxy creates a Raw proxy repository.
func (s *repositoryService) CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error {
	return s.client.Repository.Raw.Proxy.Create(repo)
}

// UpdateRawProxy updates a Raw proxy repository.
func (s *repositoryService) UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error {
	return s.client.Repository.Raw.Proxy.Update(name, repo)
}

// DeleteRawProxy deletes a Raw proxy repository.
func (s *repositoryService) DeleteRawProxy(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Proxy.Delete(name)
}

// GetRawGroup gets a Raw group repository.
func (s *repositoryService) GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error) {
	return s.client.Repository.Raw.Group.Get(name)
}

// CreateRawGroup creates a Raw group repository.
func (s *repositoryService) CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error {
	return s.client.Repository.Raw.Group.Create(repo)
}

// UpdateRawGroup updates a Raw group repository.
func (s *repositoryService) UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error {
	return s.client.Repository.Raw.Group.Update(name, repo)
}

// DeleteRawGroup deletes a Raw group repository.
func (s *repositoryService) DeleteRawGroup(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Group.Delete(name)
}

// SecurityService implementations

// GetUser gets a user by ID.
func (s *securityService) GetUser(ctx context.Context, id string) (*security.User, error) {
	return s.client.Security.User.Get(id, nil)
}

// CreateUser creates a new user.
func (s *securityService) CreateUser(ctx context.Context, user security.User) error {
	return s.client.Security.User.Create(user)
}

// UpdateUser updates an existing user.
func (s *securityService) UpdateUser(ctx context.Context, id string, user security.User) error {
	return s.client.Security.User.Update(id, user)
}

// DeleteUser deletes a user by ID.
func (s *securityService) DeleteUser(ctx context.Context, id string) error {
	return s.client.Security.User.Delete(id)
}

// ChangePassword changes a user's password.
func (s *securityService) ChangePassword(ctx context.Context, id string, password string) error {
	return s.client.Security.User.ChangePassword(id, password)
}

// GetRole gets a role by ID.
func (s *securityService) GetRole(ctx context.Context, id string) (*security.Role, error) {
	return s.client.Security.Role.Get(id)
}

// CreateRole creates a new role.
func (s *securityService) CreateRole(ctx context.Context, role security.Role) error {
	return s.client.Security.Role.Create(role)
}

// UpdateRole updates an existing role.
func (s *securityService) UpdateRole(ctx context.Context, id string, role security.Role) error {
	return s.client.Security.Role.Update(id, role)
}

// DeleteRole deletes a role by ID.
func (s *securityService) DeleteRole(ctx context.Context, id string) error {
	return s.client.Security.Role.Delete(id)
}
