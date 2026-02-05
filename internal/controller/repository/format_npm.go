package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/utils"
)

// NpmHandler handles npm repository operations.
type NpmHandler struct{}

func (h *NpmHandler) SupportedTypes() []string {
	return []string{"hosted", "proxy", "group"}
}

func (h *NpmHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetNpmHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := client.Repository().GetNpmProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isProxyUpToDate(cr, repo)
	case "group":
		repo, err := client.Repository().GetNpmGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, h.isGroupUpToDate(cr, repo)
	}
	return false, false
}

func (h *NpmHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateNpmHosted(ctx, h.generateHosted(cr))
	case "proxy":
		return client.Repository().CreateNpmProxy(ctx, h.generateProxy(cr))
	case "group":
		return client.Repository().CreateNpmGroup(ctx, h.generateGroup(cr))
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

func (h *NpmHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateNpmHosted(ctx, name, h.generateHosted(cr))
	case "proxy":
		return client.Repository().UpdateNpmProxy(ctx, name, h.generateProxy(cr))
	case "group":
		return client.Repository().UpdateNpmGroup(ctx, name, h.generateGroup(cr))
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

func (h *NpmHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteNpmHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteNpmProxy(ctx, name)
	case "group":
		return client.Repository().DeleteNpmGroup(ctx, name)
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

func (h *NpmHandler) generateHosted(cr *v1alpha1.Repository) repository.NpmHostedRepository {
	return repository.NpmHostedRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateHostedStorage(cr),
		Cleanup: generateCleanup(cr),
	}
}

func (h *NpmHandler) generateProxy(cr *v1alpha1.Repository) repository.NpmProxyRepository {
	return repository.NpmProxyRepository{
		Name:          cr.Spec.ForProvider.Name,
		Online:        getOnline(cr),
		Storage:       generateProxyStorage(cr),
		Proxy:         generateProxyConfig(cr),
		NegativeCache: generateNegativeCache(cr),
		HTTPClient:    generateHTTPClient(cr),
	}
}

func (h *NpmHandler) generateGroup(cr *v1alpha1.Repository) repository.NpmGroupRepository {
	return repository.NpmGroupRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateProxyStorage(cr),
		Group:   generateGroupDeployConfig(cr),
	}
}

func (h *NpmHandler) isHostedUpToDate(cr *v1alpha1.Repository, repo *repository.NpmHostedRepository) bool {
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

func (h *NpmHandler) isProxyUpToDate(cr *v1alpha1.Repository, repo *repository.NpmProxyRepository) bool {
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

func (h *NpmHandler) isGroupUpToDate(cr *v1alpha1.Repository, repo *repository.NpmGroupRepository) bool {
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
