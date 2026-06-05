package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// NpmHandler handles npm repository operations.
type NpmHandler struct{}

// SupportedTypes returns the repository types supported by NpmHandler.
func (h *NpmHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the npm repository exists and is up to date.
func (h *NpmHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, repoCR *v1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(ctx, name, client.Repository().GetNpmHosted, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(ctx, name, client.Repository().GetNpmProxy, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(ctx, name, client.Repository().GetNpmGroup, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new npm repository of the given type.
func (h *NpmHandler) Create(ctx context.Context, client nexus.Client, repoCR *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().CreateNpmHosted(ctx, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().CreateNpmProxy(ctx, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().CreateNpmGroup(ctx, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// Update updates an existing npm repository of the given type.
func (h *NpmHandler) Update(ctx context.Context, client nexus.Client, name string, repoCR *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().UpdateNpmHosted(ctx, name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().UpdateNpmProxy(ctx, name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().UpdateNpmGroup(ctx, name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// Delete removes an npm repository of the given type.
func (h *NpmHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().DeleteNpmHosted(ctx, name)
	case repoTypeProxy:
		return client.Repository().DeleteNpmProxy(ctx, name)
	case repoTypeGroup:
		return client.Repository().DeleteNpmGroup(ctx, name)
	}

	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// generateHosted builds a NpmHostedRepository from the CR spec.
func (h *NpmHandler) generateHosted(repoCR *v1alpha1.Repository) repository.NpmHostedRepository {
	return repository.NpmHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateHostedStorage(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a NpmProxyRepository from the CR spec.
func (h *NpmHandler) generateProxy(ctx context.Context, repoCR *v1alpha1.Repository) repository.NpmProxyRepository {
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
func (h *NpmHandler) generateGroup(repoCR *v1alpha1.Repository) repository.NpmGroupRepository {
	return repository.NpmGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Group:   generateGroupDeployConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the npm hosted repository matches the
// CR spec.
func (h *NpmHandler) isHostedUpToDate(repoCR *v1alpha1.Repository, repo *repository.NpmHostedRepository) bool {
	return isSimpleHostedUpToDate(
		repoCR,
		repo.Online,
		repo.Storage.BlobStoreName,
		repo.Storage.WritePolicy,
	)
}

// isProxyUpToDate reports whether the proxy repository matches the CR spec.
func (h *NpmHandler) isProxyUpToDate(repoCR *v1alpha1.Repository, repo *repository.NpmProxyRepository) bool {
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
func (h *NpmHandler) isGroupUpToDate(repoCR *v1alpha1.Repository, repo *repository.NpmGroupRepository) bool {
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
