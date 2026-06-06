package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// DockerHandler handles Docker repository operations.
type DockerHandler struct{}

// SupportedTypes returns the repository types supported by DockerHandler.
func (h *DockerHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Docker repository exists and is up to date.
func (h *DockerHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(ctx, name, client.Repository().GetDockerHosted, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(ctx, name, client.Repository().GetDockerProxy, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(ctx, name, client.Repository().GetDockerGroup, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Docker repository of the given type.
func (h *DockerHandler) Create(ctx context.Context, client nexus.Client, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().CreateDockerHosted(ctx, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().CreateDockerProxy(ctx, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().CreateDockerGroup(ctx, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// Update updates an existing Docker repository of the given type.
func (h *DockerHandler) Update(ctx context.Context, client nexus.Client, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().UpdateDockerHosted(ctx, name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().UpdateDockerProxy(ctx, name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().UpdateDockerGroup(ctx, name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// Delete removes a Docker repository of the given type.
func (h *DockerHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().DeleteDockerHosted(ctx, name)
	case repoTypeProxy:
		return client.Repository().DeleteDockerProxy(ctx, name)
	case repoTypeGroup:
		return client.Repository().DeleteDockerGroup(ctx, name)
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// generateHosted builds a DockerHostedRepository from the CR spec.
func (h *DockerHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.DockerHostedRepository {
	return repository.DockerHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateDockerHostedStorage(repoCR),
		Docker:  generateDockerConfig(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a DockerProxyRepository from the CR spec.
func (h *DockerHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.DockerProxyRepository {
	return repository.DockerProxyRepository{
		Name:          repoCR.Spec.ForProvider.Name,
		Online:        getOnline(repoCR),
		Storage:       generateProxyStorage(repoCR),
		Docker:        generateDockerConfig(repoCR),
		Proxy:         generateProxyConfig(repoCR),
		NegativeCache: generateNegativeCache(repoCR),
		HTTPClient:    generateHTTPClient(ctx, repoCR),
		DockerProxy: repository.DockerProxy{
			IndexType: repository.DockerProxyIndexTypeHub,
		},
	}
}

// generateGroup builds a DockerGroupRepository from the CR spec.
func (h *DockerHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.DockerGroupRepository {
	return repository.DockerGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Docker:  generateDockerConfig(repoCR),
		Group:   generateGroupDeployConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the hosted repository matches the CR spec.
func (h *DockerHandler) isHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.DockerHostedRepository) bool {
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
func (h *DockerHandler) isHostedStorageUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.DockerHostedRepository) bool {
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
func (h *DockerHandler) isHostedDockerConfigUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.DockerHostedRepository) bool {
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
func (h *DockerHandler) isProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.DockerProxyRepository) bool {
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
func (h *DockerHandler) isGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.DockerGroupRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if repoCR.Spec.ForProvider.Group != nil {
		if !utils.StringSlicesEqual(repo.Group.MemberNames, repoCR.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}
