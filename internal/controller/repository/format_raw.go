package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// RawHandler handles Raw repository operations.
type RawHandler struct{}

// SupportedTypes returns the repository types supported by RawHandler.
func (h *RawHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Raw repository exists and is up to date.
func (h *RawHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(ctx, name, client.Repository().GetRawHosted, h.isHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(ctx, name, client.Repository().GetRawProxy, h.isProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(ctx, name, client.Repository().GetRawGroup, h.isGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Raw repository of the given type.
func (h *RawHandler) Create(ctx context.Context, client nexus.Client, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().CreateRawHosted(ctx, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().CreateRawProxy(ctx, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().CreateRawGroup(ctx, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

// Update updates an existing Raw repository of the given type.
func (h *RawHandler) Update(ctx context.Context, client nexus.Client, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().UpdateRawHosted(ctx, name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Repository().UpdateRawProxy(ctx, name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Repository().UpdateRawGroup(ctx, name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

// Delete removes a Raw repository of the given type.
func (h *RawHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Repository().DeleteRawHosted(ctx, name)
	case repoTypeProxy:
		return client.Repository().DeleteRawProxy(ctx, name)
	case repoTypeGroup:
		return client.Repository().DeleteRawGroup(ctx, name)
	}

	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

// generateHosted builds a RawHostedRepository from the CR spec.
func (h *RawHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.RawHostedRepository {
	return repository.RawHostedRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateHostedStorage(repoCR),
		Cleanup: generateCleanup(repoCR),
	}
}

// generateProxy builds a RawProxyRepository from the CR spec.
func (h *RawHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.RawProxyRepository {
	return repository.RawProxyRepository{
		Name:          repoCR.Spec.ForProvider.Name,
		Online:        getOnline(repoCR),
		Storage:       generateProxyStorage(repoCR),
		Proxy:         generateProxyConfig(repoCR),
		NegativeCache: generateNegativeCache(repoCR),
		HTTPClient:    generateHTTPClient(ctx, repoCR),
	}
}

// generateGroup builds a RawGroupRepository from the CR spec.
func (h *RawHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.RawGroupRepository {
	return repository.RawGroupRepository{
		Name:    repoCR.Spec.ForProvider.Name,
		Online:  getOnline(repoCR),
		Storage: generateProxyStorage(repoCR),
		Group:   generateGroupConfig(repoCR),
	}
}

// isHostedUpToDate reports whether the raw hosted repository matches the
// CR spec.
func (h *RawHandler) isHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RawHostedRepository) bool {
	return isSimpleHostedUpToDate(
		repoCR,
		repo.Online,
		repo.Storage.BlobStoreName,
		repo.Storage.WritePolicy,
	)
}

// isProxyUpToDate reports whether the proxy repository matches the CR spec.
func (h *RawHandler) isProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RawProxyRepository) bool {
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
func (h *RawHandler) isGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RawGroupRepository) bool {
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
