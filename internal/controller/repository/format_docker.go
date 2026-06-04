package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// DockerHandler handles Docker repository operations.
type DockerHandler struct{}

func (h *DockerHandler) SupportedTypes() []string {
	return []string{"hosted", "proxy", "group"}
}

func (h *DockerHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetDockerHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := client.Repository().GetDockerProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isProxyUpToDate(cr, repo)
	case "group":
		repo, err := client.Repository().GetDockerGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, h.isGroupUpToDate(cr, repo)
	}

	return false, false
}

func (h *DockerHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateDockerHosted(ctx, h.generateHosted(cr))
	case "proxy":
		return client.Repository().CreateDockerProxy(ctx, h.generateProxy(ctx, cr))
	case "group":
		return client.Repository().CreateDockerGroup(ctx, h.generateGroup(cr))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

func (h *DockerHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateDockerHosted(ctx, name, h.generateHosted(cr))
	case "proxy":
		return client.Repository().UpdateDockerProxy(ctx, name, h.generateProxy(ctx, cr))
	case "group":
		return client.Repository().UpdateDockerGroup(ctx, name, h.generateGroup(cr))
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

func (h *DockerHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteDockerHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteDockerProxy(ctx, name)
	case "group":
		return client.Repository().DeleteDockerGroup(ctx, name)
	}

	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

func (h *DockerHandler) generateHosted(cr *v1alpha1.Repository) repository.DockerHostedRepository {
	return repository.DockerHostedRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateDockerHostedStorage(cr),
		Docker:  generateDockerConfig(cr),
		Cleanup: generateCleanup(cr),
	}
}

func (h *DockerHandler) generateProxy(ctx context.Context, cr *v1alpha1.Repository) repository.DockerProxyRepository {
	return repository.DockerProxyRepository{
		Name:          cr.Spec.ForProvider.Name,
		Online:        getOnline(cr),
		Storage:       generateProxyStorage(cr),
		Docker:        generateDockerConfig(cr),
		Proxy:         generateProxyConfig(cr),
		NegativeCache: generateNegativeCache(cr),
		HTTPClient:    generateHTTPClient(ctx, cr),
		DockerProxy: repository.DockerProxy{
			IndexType: repository.DockerProxyIndexTypeHub,
		},
	}
}

func (h *DockerHandler) generateGroup(cr *v1alpha1.Repository) repository.DockerGroupRepository {
	return repository.DockerGroupRepository{
		Name:    cr.Spec.ForProvider.Name,
		Online:  getOnline(cr),
		Storage: generateProxyStorage(cr),
		Docker:  generateDockerConfig(cr),
		Group:   generateGroupDeployConfig(cr),
	}
}

func (h *DockerHandler) isHostedUpToDate(cr *v1alpha1.Repository, repo *repository.DockerHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}

		if cr.Spec.ForProvider.Storage.WritePolicy != nil &&
			string(repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	if cr.Spec.ForProvider.Docker != nil {
		if cr.Spec.ForProvider.Docker.ForceBasicAuth != nil &&
			repo.ForceBasicAuth != *cr.Spec.ForProvider.Docker.ForceBasicAuth {
			return false
		}

		if cr.Spec.ForProvider.Docker.V1Enabled != nil &&
			repo.V1Enabled != *cr.Spec.ForProvider.Docker.V1Enabled {
			return false
		}
	}

	return true
}

func (h *DockerHandler) isProxyUpToDate(cr *v1alpha1.Repository, repo *repository.DockerProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

func (h *DockerHandler) isGroupUpToDate(cr *v1alpha1.Repository, repo *repository.DockerGroupRepository) bool {
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
