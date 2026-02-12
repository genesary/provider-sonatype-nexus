package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// RawHandler handles Raw repository operations.
type RawHandler struct{}

func (h *RawHandler) SupportedTypes() []string {
	return []string{"hosted", "proxy", "group"}
}

func (h *RawHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetRawHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := client.Repository().GetRawProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isProxyUpToDate(cr, repo)
	case "group":
		repo, err := client.Repository().GetRawGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isGroupUpToDate(cr, repo)
	}
	return false, false
}

func (h *RawHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateRawHosted(ctx, h.generateHosted(cr))
	case "proxy":
		return client.Repository().CreateRawProxy(ctx, h.generateProxy(cr))
	case "group":
		return client.Repository().CreateRawGroup(ctx, h.generateGroup(cr))
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

func (h *RawHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateRawHosted(ctx, name, h.generateHosted(cr))
	case "proxy":
		return client.Repository().UpdateRawProxy(ctx, name, h.generateProxy(cr))
	case "group":
		return client.Repository().UpdateRawGroup(ctx, name, h.generateGroup(cr))
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

func (h *RawHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteRawHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteRawProxy(ctx, name)
	case "group":
		return client.Repository().DeleteRawGroup(ctx, name)
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

func (h *RawHandler) generateHosted(cr *v1alpha1.Repository) repository.RawHostedRepository {
	return repository.RawHostedRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateHostedStorage(cr),
		Cleanup: generateCleanup(cr),
	}
}

func (h *RawHandler) generateProxy(cr *v1alpha1.Repository) repository.RawProxyRepository {
	return repository.RawProxyRepository{
		Name:          cr.Spec.ForProvider.Name,
		Online:        getOnline(cr),
		Storage:       generateProxyStorage(cr),
		Proxy:         generateProxyConfig(cr),
		NegativeCache: generateNegativeCache(cr),
		HTTPClient:    generateHTTPClient(cr),
	}
}

func (h *RawHandler) generateGroup(cr *v1alpha1.Repository) repository.RawGroupRepository {
	return repository.RawGroupRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateProxyStorage(cr),
		Group:   generateGroupConfig(cr),
	}
}

func (h *RawHandler) isHostedUpToDate(cr *v1alpha1.Repository, repo *repository.RawHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}
	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil && repo.Storage.WritePolicy != nil &&
			string(*repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}
	return true
}

func (h *RawHandler) isProxyUpToDate(cr *v1alpha1.Repository, repo *repository.RawProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}
	if cr.Spec.ForProvider.Proxy != nil {
		if repo.Proxy.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}
	return true
}

func (h *RawHandler) isGroupUpToDate(cr *v1alpha1.Repository, repo *repository.RawGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}
	if cr.Spec.ForProvider.Group != nil {
		if !utils.StringSlicesEqual(repo.Group.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}
	return true
}
