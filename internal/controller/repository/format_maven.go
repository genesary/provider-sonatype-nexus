package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// MavenHandler handles Maven repository operations.
type MavenHandler struct{}

// SupportedTypes returns the repository types supported by MavenHandler.
func (h *MavenHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Maven repository exists and is up to date.
func (h *MavenHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(ctx, name, client.Repository().GetMavenHosted, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(ctx, name, client.Repository().GetMavenProxy, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(ctx, name, client.Repository().GetMavenGroup, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Maven repository of the given type.
func (h *MavenHandler) Create(ctx context.Context, client nexus.Client, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().CreateMavenHosted(ctx, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().CreateMavenProxy(ctx, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().CreateMavenGroup(ctx, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

// Update updates an existing Maven repository of the given type.
func (h *MavenHandler) Update(ctx context.Context, client nexus.Client, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().UpdateMavenHosted(ctx, name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().UpdateMavenProxy(ctx, name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().UpdateMavenGroup(ctx, name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

// Delete removes a Maven repository of the given type.
func (h *MavenHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().DeleteMavenHosted(ctx, name)
	case repoTypeProxy:
		return client.Repository().DeleteMavenProxy(ctx, name)
	case repoTypeGroup:
		return client.Repository().DeleteMavenGroup(ctx, name)
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

// generateHosted builds a MavenHostedRepository from the CR spec.
func (h *MavenHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.MavenHostedRepository {
	return repository.MavenHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateHostedStorage(repoCR),
		Maven:   generateMavenConfig(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a MavenProxyRepository from the CR spec.
func (h *MavenHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.MavenProxyRepository {
	return repository.MavenProxyRepository{
		Name:          repoCR.Spec.ForProvider.Name,
		Online:        getOnline(repoCR),
		Storage:       generateProxyStorage(repoCR),
		Maven:         generateMavenConfig(repoCR),
		Proxy:         generateProxyConfig(repoCR),
		NegativeCache: generateNegativeCache(repoCR),
		HTTPClient:    generateHTTPClientWithPreemptiveAuth(ctx, repoCR),
	}
}

// generateGroup builds a MavenGroupRepository from the CR spec.
func (h *MavenHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.MavenGroupRepository {
	return repository.MavenGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Group:   generateGroupConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the hosted repository matches the CR spec.
func (h *MavenHandler) isHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenHostedRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if !h.isHostedStorageUpToDate(repoCR, repo) {
		return false
	}

	if !h.isHostedMavenConfigUpToDate(repoCR, repo) {
		return false
	}

	return true
}

// isHostedStorageUpToDate checks if the storage fields are up to date.
func (h *MavenHandler) isHostedStorageUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenHostedRepository) bool {
	if repoCR.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != repoCR.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}

		if repoCR.Spec.ForProvider.Storage.WritePolicy != nil && repo.Storage.WritePolicy != nil &&
			string(*repo.Storage.WritePolicy) != *repoCR.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	return true
}

// isMavenConfigUpToDate checks if Maven version/layout policy fields match.
func (h *MavenHandler) isMavenConfigUpToDate(repoCR *repositoryv1alpha1.Repository, versionPolicy, layoutPolicy string) bool {
	if repoCR.Spec.ForProvider.Maven != nil {
		if repoCR.Spec.ForProvider.Maven.VersionPolicy != nil &&
			versionPolicy != *repoCR.Spec.ForProvider.Maven.VersionPolicy {
			return false
		}

		if repoCR.Spec.ForProvider.Maven.LayoutPolicy != nil &&
			layoutPolicy != *repoCR.Spec.ForProvider.Maven.LayoutPolicy {
			return false
		}
	}

	return true
}

// isHostedMavenConfigUpToDate checks if the Maven-specific fields are
// up to date for a hosted repository.
func (h *MavenHandler) isHostedMavenConfigUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenHostedRepository) bool {
	return h.isMavenConfigUpToDate(repoCR, string(repo.VersionPolicy), string(repo.LayoutPolicy))
}

// isProxyUpToDate reports whether the proxy repository matches the CR spec.
func (h *MavenHandler) isProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if !h.isProxyStorageUpToDate(repoCR, repo) {
		return false
	}

	if !h.isProxyRemoteURLUpToDate(repoCR, repo) {
		return false
	}

	if !h.isProxyMavenConfigUpToDate(repoCR, repo) {
		return false
	}

	return true
}

// isProxyStorageUpToDate checks if the storage fields are up to date for a
// proxy repository.
func (h *MavenHandler) isProxyStorageUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	if repoCR.Spec.ForProvider.Storage != nil {
		if repo.BlobStoreName != repoCR.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
	}

	return true
}

// isProxyRemoteURLUpToDate checks if the remote URL is up to date.
func (h *MavenHandler) isProxyRemoteURLUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	if repoCR.Spec.ForProvider.Proxy != nil {
		if repo.RemoteURL != repoCR.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

// isProxyMavenConfigUpToDate checks if the Maven-specific fields are up to
// date for a proxy repository.
func (h *MavenHandler) isProxyMavenConfigUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	return h.isMavenConfigUpToDate(repoCR, string(repo.VersionPolicy), string(repo.LayoutPolicy))
}

// isGroupUpToDate reports whether the group repository matches the CR spec.
func (h *MavenHandler) isGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.MavenGroupRepository) bool {
	if repoCR.Spec.ForProvider.Online != nil && repo.Online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if repoCR.Spec.ForProvider.Group != nil {
		if !utils.StringSlicesEqual(repo.MemberNames, repoCR.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}
