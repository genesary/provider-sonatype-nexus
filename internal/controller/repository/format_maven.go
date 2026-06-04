package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// MavenHandler handles Maven repository operations.
type MavenHandler struct{}

func (h *MavenHandler) SupportedTypes() []string {
	return []string{"hosted", "proxy", "group"}
}

func (h *MavenHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetMavenHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := client.Repository().GetMavenProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isProxyUpToDate(cr, repo)
	case "group":
		repo, err := client.Repository().GetMavenGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isGroupUpToDate(cr, repo)
	}

	return false, false
}

func (h *MavenHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateMavenHosted(ctx, h.generateHosted(cr))
	case "proxy":
		return client.Repository().CreateMavenProxy(ctx, h.generateProxy(ctx, cr))
	case "group":
		return client.Repository().CreateMavenGroup(ctx, h.generateGroup(cr))
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

func (h *MavenHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateMavenHosted(ctx, name, h.generateHosted(cr))
	case "proxy":
		return client.Repository().UpdateMavenProxy(ctx, name, h.generateProxy(ctx, cr))
	case "group":
		return client.Repository().UpdateMavenGroup(ctx, name, h.generateGroup(cr))
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

func (h *MavenHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteMavenHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteMavenProxy(ctx, name)
	case "group":
		return client.Repository().DeleteMavenGroup(ctx, name)
	}

	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

func (h *MavenHandler) generateHosted(cr *v1alpha1.Repository) repository.MavenHostedRepository {
	return repository.MavenHostedRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateHostedStorage(cr),
		Maven:   generateMavenConfig(cr),
		Cleanup: generateCleanup(cr),
	}
}

func (h *MavenHandler) generateProxy(ctx context.Context, cr *v1alpha1.Repository) repository.MavenProxyRepository {
	return repository.MavenProxyRepository{
		Name:          cr.Spec.ForProvider.Name,
		Online:        getOnline(cr),
		Storage:       generateProxyStorage(cr),
		Maven:         generateMavenConfig(cr),
		Proxy:         generateProxyConfig(cr),
		NegativeCache: generateNegativeCache(cr),
		HTTPClient:    generateHTTPClientWithPreemptiveAuth(ctx, cr),
	}
}

func (h *MavenHandler) generateGroup(cr *v1alpha1.Repository) repository.MavenGroupRepository {
	return repository.MavenGroupRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateProxyStorage(cr),
		Group:   generateGroupConfig(cr),
	}
}

func (h *MavenHandler) isHostedUpToDate(cr *v1alpha1.Repository, repo *repository.MavenHostedRepository) bool {
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

	if cr.Spec.ForProvider.Maven != nil {
		if cr.Spec.ForProvider.Maven.VersionPolicy != nil &&
			string(repo.VersionPolicy) != *cr.Spec.ForProvider.Maven.VersionPolicy {
			return false
		}

		if cr.Spec.ForProvider.Maven.LayoutPolicy != nil &&
			string(repo.LayoutPolicy) != *cr.Spec.ForProvider.Maven.LayoutPolicy {
			return false
		}
	}

	return true
}

func (h *MavenHandler) isProxyUpToDate(cr *v1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	if cr.Spec.ForProvider.Maven != nil {
		if cr.Spec.ForProvider.Maven.VersionPolicy != nil &&
			string(repo.VersionPolicy) != *cr.Spec.ForProvider.Maven.VersionPolicy {
			return false
		}

		if cr.Spec.ForProvider.Maven.LayoutPolicy != nil &&
			string(repo.LayoutPolicy) != *cr.Spec.ForProvider.Maven.LayoutPolicy {
			return false
		}
	}

	return true
}

func (h *MavenHandler) isGroupUpToDate(cr *v1alpha1.Repository, repo *repository.MavenGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !utils.StringSlicesEqual(repo.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}
