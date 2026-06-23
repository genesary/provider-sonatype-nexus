package repository

import (
	"context"

	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	repositorySchema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// DockerHandler handles Docker repository operations.
type DockerHandler struct{}

// SupportedTypes returns the repository types supported by DockerHandler.
func (h *DockerHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Docker repository exists and is up to date.
func (h *DockerHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Docker.Hosted.Get, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Docker.Proxy.Get, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Docker.Group.Get, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Docker repository of the given type.
func (h *DockerHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Docker.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Docker.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Docker.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// Update updates an existing Docker repository of the given type.
func (h *DockerHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Docker.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Docker.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Docker.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// Delete removes a Docker repository of the given type.
func (h *DockerHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Docker.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Docker.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Docker.Group.Delete(name)
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// generateHosted builds a DockerHostedRepository from the CR spec.
func (h *DockerHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repositorySchema.DockerHostedRepository {
	return repositorySchema.DockerHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateDockerHostedStorage(repoCR),
		Docker:  generateDockerConfig(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a DockerProxyRepository from the CR spec.
func (h *DockerHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repositorySchema.DockerProxyRepository {
	return repositorySchema.DockerProxyRepository{
		Name:          repoCR.Spec.ForProvider.Name,
		Online:        getOnline(repoCR),
		Storage:       generateProxyStorage(repoCR),
		Docker:        generateDockerConfig(repoCR),
		Proxy:         generateProxyConfig(repoCR),
		NegativeCache: generateNegativeCache(repoCR),
		HTTPClient:    generateHTTPClient(ctx, repoCR),
		DockerProxy: repositorySchema.DockerProxy{
			IndexType: repositorySchema.DockerProxyIndexTypeHub,
		},
	}
}

// generateGroup builds a DockerGroupRepository from the CR spec.
func (h *DockerHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repositorySchema.DockerGroupRepository {
	return repositorySchema.DockerGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Docker:  generateDockerConfig(repoCR),
		Group:   generateGroupDeployConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the hosted repository matches the CR spec.
func (h *DockerHandler) isHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repositorySchema.DockerHostedRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if !h.isHostedStorageUpToDate(repoCR, repo) {
		return false
	}

	if !h.isHostedDockerConfigUpToDate(repoCR, repo) {
		return false
	}

	return true
}

// isHostedStorageUpToDate checks if the storage fields are up to date.
func (h *DockerHandler) isHostedStorageUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repositorySchema.DockerHostedRepository) bool {
	if repoCR.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != repoCR.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}

		if repoCR.Spec.ForProvider.Storage.WritePolicy != nil &&
			string(repo.Storage.WritePolicy) != *repoCR.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	return true
}

// isHostedDockerConfigUpToDate checks if the Docker-specific fields are
// up to date.
func (h *DockerHandler) isHostedDockerConfigUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repositorySchema.DockerHostedRepository) bool {
	if repoCR.Spec.ForProvider.Docker != nil {
		if repoCR.Spec.ForProvider.Docker.ForceBasicAuth != nil &&
			repo.ForceBasicAuth != *repoCR.Spec.ForProvider.Docker.ForceBasicAuth {
			return false
		}

		if repoCR.Spec.ForProvider.Docker.V1Enabled != nil &&
			repo.V1Enabled != *repoCR.Spec.ForProvider.Docker.V1Enabled {
			return false
		}
	}

	return true
}

// isProxyUpToDate reports whether the proxy repository matches the CR spec.
func (h *DockerHandler) isProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repositorySchema.DockerProxyRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if repoCR.Spec.ForProvider.Proxy != nil {
		if repo.RemoteURL != repoCR.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

// isGroupUpToDate reports whether the group repository matches the CR spec.
func (h *DockerHandler) isGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repositorySchema.DockerGroupRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if repoCR.Spec.ForProvider.Group != nil {
		if !helpers.AreStringSlicesEqual(repo.Group.MemberNames, repoCR.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}
