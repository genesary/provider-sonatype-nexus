package repository

import (
	"context"

	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// NpmHandler handles npm repository operations.
type NpmHandler struct{}

// SupportedTypes returns the repository types supported by NpmHandler.
func (h *NpmHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the npm repository exists and is up to date.
func (h *NpmHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Npm.Hosted.Get, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Npm.Proxy.Get, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Npm.Group.Get, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new npm repository of the given type.
func (h *NpmHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Npm.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Npm.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Npm.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// Update updates an existing npm repository of the given type.
func (h *NpmHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Npm.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Npm.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Npm.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// Delete removes an npm repository of the given type.
func (h *NpmHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Npm.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Npm.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Npm.Group.Delete(name)
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// generateHosted builds a NpmHostedRepository from the CR spec.
func (h *NpmHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.NpmHostedRepository {
	return repository.NpmHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateHostedStorage(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a NpmProxyRepository from the CR spec.
func (h *NpmHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.NpmProxyRepository {
	return repository.NpmProxyRepository{
		Name:          repoCR.Spec.ForProvider.Name,
		Online:        getOnline(repoCR),
		Storage:       generateProxyStorage(repoCR),
		Proxy:         generateProxyConfig(repoCR),
		NegativeCache: generateNegativeCache(repoCR),
		HTTPClient:    generateHTTPClient(ctx, repoCR),
	}
}

// generateGroup builds a NpmGroupRepository from the CR spec.
func (h *NpmHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.NpmGroupRepository {
	return repository.NpmGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Group:   generateGroupDeployConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the npm hosted repository matches the
// CR spec.
func (h *NpmHandler) isHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NpmHostedRepository) bool {
	return isSimpleHostedUpToDate(
		repoCR,
		repo.Online,
		repo.Storage.BlobStoreName,
		repo.Storage.WritePolicy,
	)
}

// isProxyUpToDate reports whether the proxy repository matches the CR spec.
func (h *NpmHandler) isProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NpmProxyRepository) bool {
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
func (h *NpmHandler) isGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NpmGroupRepository) bool {
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
