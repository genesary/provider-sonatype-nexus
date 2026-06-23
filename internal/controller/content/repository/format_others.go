// Package repository contains handlers for all repository format types.
// format_others.go contains handlers for formats with simpler patterns:
//   - Formats with all three types (hosted, proxy, group): nuget, pypi,
//     rubygems, yum, r, cargo, bower
//   - Formats with partial support: apt, helm, go, gitlfs
//   - Proxy-only formats: cocoapods, conan, conda

package repository

import (
	"context"

	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// NugetHandler handles NuGet repository operations.
type NugetHandler struct{}

// SupportedTypes returns the repository types supported by NugetHandler.
func (h *NugetHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the NuGet repository exists and is up to date.
func (h *NugetHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Nuget.Hosted.Get, isNugetHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Nuget.Proxy.Get, isNugetProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Nuget.Group.Get, isNugetGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new NuGet repository of the given type.
func (h *NugetHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Nuget.Hosted.Create(repository.NugetHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		repo := repository.NugetProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.NugetProxy != nil {
			if repoCR.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge != nil {
				repo.QueryCacheItemMaxAge = int(*repoCR.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge)
			}

			if repoCR.Spec.ForProvider.NugetProxy.NugetVersion != nil {
				repo.NugetVersion = repository.NugetVersion(*repoCR.Spec.ForProvider.NugetProxy.NugetVersion)
			}
		}

		return client.Nuget.Proxy.Create(repo)
	case repoTypeGroup:
		return client.Nuget.Group.Create(repository.NugetGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

// Update updates an existing NuGet repository of the given type.
func (h *NugetHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Nuget.Hosted.Update(name, repository.NugetHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		repo := repository.NugetProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.NugetProxy != nil {
			if repoCR.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge != nil {
				repo.QueryCacheItemMaxAge = int(*repoCR.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge)
			}

			if repoCR.Spec.ForProvider.NugetProxy.NugetVersion != nil {
				repo.NugetVersion = repository.NugetVersion(*repoCR.Spec.ForProvider.NugetProxy.NugetVersion)
			}
		}

		return client.Nuget.Proxy.Update(name, repo)
	case repoTypeGroup:
		return client.Nuget.Group.Update(name, repository.NugetGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

// Delete removes a NuGet repository of the given type.
func (h *NugetHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Nuget.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Nuget.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Nuget.Group.Delete(name)
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

// PypiHandler handles PyPI repository operations.
type PypiHandler struct{}

// SupportedTypes returns the repository types supported by PypiHandler.
func (h *PypiHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the PyPI repository exists and is up to date.
func (h *PypiHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Pypi.Hosted.Get, isPypiHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Pypi.Proxy.Get, isPypiProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Pypi.Group.Get, isPypiGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new PyPI repository of the given type.
func (h *PypiHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Pypi.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Pypi.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Pypi.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

// Update updates an existing PyPI repository of the given type.
func (h *PypiHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Pypi.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Pypi.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Pypi.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

// Delete removes a PyPI repository of the given type.
func (h *PypiHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Pypi.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Pypi.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Pypi.Group.Delete(name)
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

// generateHosted builds a PypiHostedRepository from the CR spec.
func (h *PypiHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.PypiHostedRepository {
	return repository.PypiHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
}

// generateProxy builds a PypiProxyRepository from the CR spec.
func (h *PypiHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.PypiProxyRepository {
	return repository.PypiProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
}

// generateGroup builds a PypiGroupRepository from the CR spec.
func (h *PypiHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.PypiGroupRepository {
	return repository.PypiGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)}
}

// RubygemsHandler handles RubyGems repository operations.
type RubygemsHandler struct{}

// SupportedTypes returns the repository types supported by RubygemsHandler.
func (h *RubygemsHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the RubyGems repository exists and is up to date.
func (h *RubygemsHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.RubyGems.Hosted.Get, isRubygemsHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.RubyGems.Proxy.Get, isRubygemsProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.RubyGems.Group.Get, isRubygemsGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new RubyGems repository of the given type.
func (h *RubygemsHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.RubyGems.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.RubyGems.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.RubyGems.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

// Update updates an existing RubyGems repository of the given type.
func (h *RubygemsHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.RubyGems.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.RubyGems.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.RubyGems.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

// Delete removes a RubyGems repository of the given type.
func (h *RubygemsHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.RubyGems.Hosted.Delete(name)
	case repoTypeProxy:
		return client.RubyGems.Proxy.Delete(name)
	case repoTypeGroup:
		return client.RubyGems.Group.Delete(name)
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

// generateHosted builds a RubyGemsHostedRepository from the CR spec.
func (h *RubygemsHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.RubyGemsHostedRepository {
	return repository.RubyGemsHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
}

// generateProxy builds a RubyGemsProxyRepository from the CR spec.
func (h *RubygemsHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.RubyGemsProxyRepository {
	return repository.RubyGemsProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
}

// generateGroup builds a RubyGemsGroupRepository from the CR spec.
func (h *RubygemsHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.RubyGemsGroupRepository {
	return repository.RubyGemsGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)}
}

// YumHandler handles Yum repository operations.
type YumHandler struct{}

// SupportedTypes returns the repository types supported by YumHandler.
func (h *YumHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Yum repository exists and is up to date.
func (h *YumHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Yum.Hosted.Get, isYumHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Yum.Proxy.Get, isYumProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Yum.Group.Get, isYumGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Yum repository of the given type.
func (h *YumHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		repo := repository.YumHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
		if repoCR.Spec.ForProvider.Yum != nil {
			if repoCR.Spec.ForProvider.Yum.RepodataDepth != nil {
				repo.RepodataDepth = int(*repoCR.Spec.ForProvider.Yum.RepodataDepth)
			}

			if repoCR.Spec.ForProvider.Yum.DeployPolicy != nil {
				deployPolicy := repository.YumDeployPolicy(*repoCR.Spec.ForProvider.Yum.DeployPolicy)
				repo.DeployPolicy = &deployPolicy
			}
		}

		return client.Yum.Hosted.Create(repo)
	case repoTypeProxy:
		return client.Yum.Proxy.Create(repository.YumProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	case repoTypeGroup:
		return client.Yum.Group.Create(repository.YumGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

// Update updates an existing Yum repository of the given type.
func (h *YumHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		repo := repository.YumHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
		if repoCR.Spec.ForProvider.Yum != nil {
			if repoCR.Spec.ForProvider.Yum.RepodataDepth != nil {
				repo.RepodataDepth = int(*repoCR.Spec.ForProvider.Yum.RepodataDepth)
			}

			if repoCR.Spec.ForProvider.Yum.DeployPolicy != nil {
				deployPolicy := repository.YumDeployPolicy(*repoCR.Spec.ForProvider.Yum.DeployPolicy)
				repo.DeployPolicy = &deployPolicy
			}
		}

		return client.Yum.Hosted.Update(name, repo)
	case repoTypeProxy:
		return client.Yum.Proxy.Update(name, repository.YumProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	case repoTypeGroup:
		return client.Yum.Group.Update(name, repository.YumGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

// Delete removes a Yum repository of the given type.
func (h *YumHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Yum.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Yum.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Yum.Group.Delete(name)
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

// RHandler handles R repository operations.
type RHandler struct{}

// SupportedTypes returns the repository types supported by RHandler.
func (h *RHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the R repository exists and is up to date.
func (h *RHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.R.Hosted.Get, isRHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.R.Proxy.Get, isRProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.R.Group.Get, isRGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new R repository of the given type.
func (h *RHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.R.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.R.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.R.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

// Update updates an existing R repository of the given type.
func (h *RHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.R.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.R.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.R.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

// Delete removes an R repository of the given type.
func (h *RHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.R.Hosted.Delete(name)
	case repoTypeProxy:
		return client.R.Proxy.Delete(name)
	case repoTypeGroup:
		return client.R.Group.Delete(name)
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

// generateHosted builds a RHostedRepository from the CR spec.
func (h *RHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.RHostedRepository {
	return repository.RHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
}

// generateProxy builds a RProxyRepository from the CR spec.
func (h *RHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.RProxyRepository {
	return repository.RProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
}

// generateGroup builds a RGroupRepository from the CR spec.
func (h *RHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.RGroupRepository {
	return repository.RGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)}
}

// CargoHandler handles Cargo repository operations.
type CargoHandler struct{}

// SupportedTypes returns the repository types supported by CargoHandler.
func (h *CargoHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Cargo repository exists and is up to date.
func (h *CargoHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Cargo.Hosted.Get, isCargoHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Cargo.Proxy.Get, isCargoProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Cargo.Group.Get, isCargoGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Cargo repository of the given type.
func (h *CargoHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Cargo.Hosted.Create(h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Cargo.Proxy.Create(h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Cargo.Group.Create(h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

// Update updates an existing Cargo repository of the given type.
func (h *CargoHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Cargo.Hosted.Update(name, h.generateHosted(repoCR))
	case repoTypeProxy:
		return client.Cargo.Proxy.Update(name, h.generateProxy(ctx, repoCR))
	case repoTypeGroup:
		return client.Cargo.Group.Update(name, h.generateGroup(repoCR))
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

// Delete removes a Cargo repository of the given type.
func (h *CargoHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Cargo.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Cargo.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Cargo.Group.Delete(name)
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

// generateHosted builds a CargoHostedRepository from the CR spec.
func (h *CargoHandler) generateHosted(repoCR *repositoryv1alpha1.Repository) repository.CargoHostedRepository {
	return repository.CargoHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
}

// generateProxy builds a CargoProxyRepository from the CR spec.
func (h *CargoHandler) generateProxy(ctx context.Context, repoCR *repositoryv1alpha1.Repository) repository.CargoProxyRepository {
	return repository.CargoProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
}

// generateGroup builds a CargoGroupRepository from the CR spec.
func (h *CargoHandler) generateGroup(repoCR *repositoryv1alpha1.Repository) repository.CargoGroupRepository {
	return repository.CargoGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)}
}

// BowerHandler handles Bower repository operations.
type BowerHandler struct{}

// SupportedTypes returns the repository types supported by BowerHandler.
func (h *BowerHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// Observe checks whether the Bower repository exists and is up to date.
func (h *BowerHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Bower.Hosted.Get, isBowerHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Bower.Proxy.Get, isBowerProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Bower.Group.Get, isBowerGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Bower repository of the given type.
func (h *BowerHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Bower.Hosted.Create(repository.BowerHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		repo := repository.BowerProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.Bower != nil && repoCR.Spec.ForProvider.Bower.RewritePackageUrls != nil {
			repo.Bower = repository.Bower{RewritePackageUrls: *repoCR.Spec.ForProvider.Bower.RewritePackageUrls}
		}

		return client.Bower.Proxy.Create(repo)
	case repoTypeGroup:
		return client.Bower.Group.Create(repository.BowerGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

// Update updates an existing Bower repository of the given type.
func (h *BowerHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Bower.Hosted.Update(name, repository.BowerHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		repo := repository.BowerProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.Bower != nil && repoCR.Spec.ForProvider.Bower.RewritePackageUrls != nil {
			repo.Bower = repository.Bower{RewritePackageUrls: *repoCR.Spec.ForProvider.Bower.RewritePackageUrls}
		}

		return client.Bower.Proxy.Update(name, repo)
	case repoTypeGroup:
		return client.Bower.Group.Update(name, repository.BowerGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

// Delete removes a Bower repository of the given type.
func (h *BowerHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Bower.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Bower.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Bower.Group.Delete(name)
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

// AptHandler handles APT repository operations.
type AptHandler struct{}

// SupportedTypes returns the repository types supported by AptHandler.
func (h *AptHandler) SupportedTypes() []string { return []string{repoTypeHosted, repoTypeProxy} }

// Observe checks whether the APT repository exists and is up to date.
func (h *AptHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Apt.Hosted.Get, isAptHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Apt.Proxy.Get, isAptProxyUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new APT repository of the given type.
func (h *AptHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		repo := repository.AptHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
		if repoCR.Spec.ForProvider.Apt != nil && repoCR.Spec.ForProvider.Apt.Distribution != nil {
			repo.Apt.Distribution = *repoCR.Spec.ForProvider.Apt.Distribution
		}

		if repoCR.Spec.ForProvider.AptSigning != nil {
			repo.AptSigning = repository.AptSigning{Keypair: repoCR.Spec.ForProvider.AptSigning.Keypair, Passphrase: repoCR.Spec.ForProvider.AptSigning.Passphrase}
		}

		return client.Apt.Hosted.Create(repo)
	case repoTypeProxy:
		repo := repository.AptProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.Apt != nil {
			if repoCR.Spec.ForProvider.Apt.Distribution != nil {
				repo.Apt.Distribution = *repoCR.Spec.ForProvider.Apt.Distribution
			}

			if repoCR.Spec.ForProvider.Apt.Flat != nil {
				repo.Apt.Flat = *repoCR.Spec.ForProvider.Apt.Flat
			}
		}

		return client.Apt.Proxy.Create(repo)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

// Update updates an existing APT repository of the given type.
func (h *AptHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		repo := repository.AptHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)}
		if repoCR.Spec.ForProvider.Apt != nil && repoCR.Spec.ForProvider.Apt.Distribution != nil {
			repo.Apt.Distribution = *repoCR.Spec.ForProvider.Apt.Distribution
		}

		if repoCR.Spec.ForProvider.AptSigning != nil {
			repo.AptSigning = repository.AptSigning{Keypair: repoCR.Spec.ForProvider.AptSigning.Keypair, Passphrase: repoCR.Spec.ForProvider.AptSigning.Passphrase}
		}

		return client.Apt.Hosted.Update(name, repo)
	case repoTypeProxy:
		repo := repository.AptProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)}
		if repoCR.Spec.ForProvider.Apt != nil {
			if repoCR.Spec.ForProvider.Apt.Distribution != nil {
				repo.Apt.Distribution = *repoCR.Spec.ForProvider.Apt.Distribution
			}

			if repoCR.Spec.ForProvider.Apt.Flat != nil {
				repo.Apt.Flat = *repoCR.Spec.ForProvider.Apt.Flat
			}
		}

		return client.Apt.Proxy.Update(name, repo)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

// Delete removes an APT repository of the given type.
func (h *AptHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Apt.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Apt.Proxy.Delete(name)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

// HelmHandler handles Helm repository operations.
type HelmHandler struct{}

// SupportedTypes returns the repository types supported by HelmHandler.
func (h *HelmHandler) SupportedTypes() []string { return []string{repoTypeHosted, repoTypeProxy} }

// Observe checks whether the Helm repository exists and is up to date.
func (h *HelmHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeHosted:
		return observeRepo(name, client.Helm.Hosted.Get, isHelmHostedUpToDate, repoCR)
	case repoTypeProxy:
		return observeRepo(name, client.Helm.Proxy.Get, isHelmProxyUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Helm repository of the given type.
func (h *HelmHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Helm.Hosted.Create(repository.HelmHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		return client.Helm.Proxy.Create(repository.HelmProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

// Update updates an existing Helm repository of the given type.
func (h *HelmHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Helm.Hosted.Update(name, repository.HelmHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	case repoTypeProxy:
		return client.Helm.Proxy.Update(name, repository.HelmProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

// Delete removes a Helm repository of the given type.
func (h *HelmHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeHosted:
		return client.Helm.Hosted.Delete(name)
	case repoTypeProxy:
		return client.Helm.Proxy.Delete(name)
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

// GoHandler handles Go repository operations.
type GoHandler struct{}

// SupportedTypes returns the repository types supported by GoHandler.
func (h *GoHandler) SupportedTypes() []string { return []string{repoTypeProxy, repoTypeGroup} }

// Observe checks whether the Go repository exists and is up to date.
func (h *GoHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	switch repoType {
	case repoTypeProxy:
		return observeRepo(name, client.Go.Proxy.Get, isGoProxyUpToDate, repoCR)
	case repoTypeGroup:
		return observeRepo(name, client.Go.Group.Get, isGoGroupUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Go repository of the given type.
func (h *GoHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeProxy:
		return client.Go.Proxy.Create(repository.GoProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	case repoTypeGroup:
		return client.Go.Group.Create(repository.GoGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

// Update updates an existing Go repository of the given type.
func (h *GoHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	switch repoType {
	case repoTypeProxy:
		return client.Go.Proxy.Update(name, repository.GoProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	case repoTypeGroup:
		return client.Go.Group.Update(name, repository.GoGroupRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Group: generateGroupConfig(repoCR)})
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

// Delete removes a Go repository of the given type.
func (h *GoHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	switch repoType {
	case repoTypeProxy:
		return client.Go.Proxy.Delete(name)
	case repoTypeGroup:
		return client.Go.Group.Delete(name)
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

// GitLfsHandler handles GitLFS repository operations.
type GitLfsHandler struct{}

// SupportedTypes returns the repository types supported by GitLfsHandler.
func (h *GitLfsHandler) SupportedTypes() []string { return []string{repoTypeHosted} }

// Observe checks whether the GitLFS repository exists and is up to date.
func (h *GitLfsHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	if repoType == repoTypeHosted {
		return observeRepo(name, client.GitLfs.Hosted.Get, isGitLfsHostedUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new GitLFS repository of the given type.
func (h *GitLfsHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeHosted {
		return client.GitLfs.Hosted.Create(repository.GitLfsHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

// Update updates an existing GitLFS repository of the given type.
func (h *GitLfsHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeHosted {
		return client.GitLfs.Hosted.Update(name, repository.GitLfsHostedRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateHostedStorage(repoCR), Cleanup: generateCleanup(repoCR)})
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

// Delete removes a GitLFS repository of the given type.
func (h *GitLfsHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	if repoType == repoTypeHosted {
		return client.GitLfs.Hosted.Delete(name)
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

// CocoapodsHandler handles Cocoapods repository operations.
type CocoapodsHandler struct{}

// SupportedTypes returns the repository types supported by CocoapodsHandler.
func (h *CocoapodsHandler) SupportedTypes() []string { return []string{repoTypeProxy} }

// Observe checks whether the Cocoapods repository exists and is up to date.
func (h *CocoapodsHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	if repoType == repoTypeProxy {
		return observeRepo(name, client.Cocoapods.Proxy.Get, isCocoapodsProxyUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Cocoapods repository of the given type.
func (h *CocoapodsHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Cocoapods.Proxy.Create(repository.CocoapodsProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

// Update updates an existing Cocoapods repository of the given type.
func (h *CocoapodsHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Cocoapods.Proxy.Update(name, repository.CocoapodsProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

// Delete removes a Cocoapods repository of the given type.
func (h *CocoapodsHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Cocoapods.Proxy.Delete(name)
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

// ConanHandler handles Conan repository operations.
type ConanHandler struct{}

// SupportedTypes returns the repository types supported by ConanHandler.
func (h *ConanHandler) SupportedTypes() []string { return []string{repoTypeProxy} }

// Observe checks whether the Conan repository exists and is up to date.
func (h *ConanHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	if repoType == repoTypeProxy {
		return observeRepo(name, client.Conan.Proxy.Get, isConanProxyUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Conan repository of the given type.
func (h *ConanHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conan.Proxy.Create(repository.ConanProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

// Update updates an existing Conan repository of the given type.
func (h *ConanHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conan.Proxy.Update(name, repository.ConanProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

// Delete removes a Conan repository of the given type.
func (h *ConanHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conan.Proxy.Delete(name)
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

// CondaHandler handles Conda repository operations.
type CondaHandler struct{}

// SupportedTypes returns the repository types supported by CondaHandler.
func (h *CondaHandler) SupportedTypes() []string { return []string{repoTypeProxy} }

// Observe checks whether the Conda repository exists and is up to date.
func (h *CondaHandler) Observe(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string, repoCR *repositoryv1alpha1.Repository) (exists, upToDate bool) {
	if repoType == repoTypeProxy {
		return observeRepo(name, client.Conda.Proxy.Get, isCondaProxyUpToDate, repoCR)
	}

	return false, false
}

// Create creates a new Conda repository of the given type.
func (h *CondaHandler) Create(ctx context.Context, client *pkgrepository.RepositoryService, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conda.Proxy.Create(repository.CondaProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

// Update updates an existing Conda repository of the given type.
func (h *CondaHandler) Update(ctx context.Context, client *pkgrepository.RepositoryService, name string, repoCR *repositoryv1alpha1.Repository, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conda.Proxy.Update(name, repository.CondaProxyRepository{Name: repoCR.Spec.ForProvider.Name, Online: getOnline(repoCR), Storage: generateProxyStorage(repoCR), Proxy: generateProxyConfig(repoCR), NegativeCache: generateNegativeCache(repoCR), HTTPClient: generateHTTPClient(ctx, repoCR)})
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

// Delete removes a Conda repository of the given type.
func (h *CondaHandler) Delete(ctx context.Context, client *pkgrepository.RepositoryService, name, repoType string) error {
	if repoType == repoTypeProxy {
		return client.Conda.Proxy.Delete(name)
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

// isNugetHostedUpToDate checks if a Nuget hosted repository is up to date.
func isNugetHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NugetHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isNugetProxyUpToDate checks if a Nuget proxy repository is up to date.
func isNugetProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NugetProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isNugetGroupUpToDate checks if a Nuget group repository is up to date.
func isNugetGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.NugetGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isPypiHostedUpToDate checks if a Pypi hosted repository is up to date.
func isPypiHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.PypiHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isPypiProxyUpToDate checks if a Pypi proxy repository is up to date.
func isPypiProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.PypiProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isPypiGroupUpToDate checks if a Pypi group repository is up to date.
func isPypiGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.PypiGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isRubygemsHostedUpToDate checks if a Rubygems hosted repository is up
// to date.
func isRubygemsHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RubyGemsHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isRubygemsProxyUpToDate checks if a Rubygems proxy repository is up to date.
func isRubygemsProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RubyGemsProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isRubygemsGroupUpToDate checks if a Rubygems group repository is up to date.
func isRubygemsGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RubyGemsGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isYumHostedUpToDate checks if a Yum hosted repository is up to date.
func isYumHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.YumHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isYumProxyUpToDate checks if a Yum proxy repository is up to date.
func isYumProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.YumProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isYumGroupUpToDate checks if a Yum group repository is up to date.
func isYumGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.YumGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isRHostedUpToDate checks if a R hosted repository is up to date.
func isRHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isRProxyUpToDate checks if a R proxy repository is up to date.
func isRProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isRGroupUpToDate checks if a R group repository is up to date.
func isRGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.RGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isCargoHostedUpToDate checks if a Cargo hosted repository is up to date.
func isCargoHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.CargoHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isCargoProxyUpToDate checks if a Cargo proxy repository is up to date.
func isCargoProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.CargoProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isCargoGroupUpToDate checks if a Cargo group repository is up to date.
func isCargoGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.CargoGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isBowerHostedUpToDate checks if a Bower hosted repository is up to date.
func isBowerHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.BowerHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isBowerProxyUpToDate checks if a Bower proxy repository is up to date.
func isBowerProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.BowerProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isBowerGroupUpToDate checks if a Bower group repository is up to date.
func isBowerGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.BowerGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isAptHostedUpToDate checks if a Apt hosted repository is up to date.
func isAptHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.AptHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isAptProxyUpToDate checks if a Apt proxy repository is up to date.
func isAptProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.AptProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isHelmHostedUpToDate checks if a Helm hosted repository is up to date.
func isHelmHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.HelmHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isHelmProxyUpToDate checks if a Helm proxy repository is up to date.
func isHelmProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.HelmProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isGoProxyUpToDate checks if a Go proxy repository is up to date.
func isGoProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.GoProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isGoGroupUpToDate checks if a Go group repository is up to date.
func isGoGroupUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.GoGroupRepository) bool {
	return isBasicGroupUpToDate(repoCR, repo.Name, repo.Online, repo.MemberNames)
}

// isGitLfsHostedUpToDate checks if a GitLfs hosted repository is up to date.
func isGitLfsHostedUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.GitLfsHostedRepository) bool {
	return isBasicHostedUpToDate(repoCR, repo.Name, repo.Online)
}

// isCocoapodsProxyUpToDate checks if a Cocoapods proxy repository is up
// to date.
func isCocoapodsProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.CocoapodsProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isConanProxyUpToDate checks if a Conan proxy repository is up to date.
func isConanProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.ConanProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isCondaProxyUpToDate checks if a Conda proxy repository is up to date.
func isCondaProxyUpToDate(repoCR *repositoryv1alpha1.Repository, repo *repository.CondaProxyRepository) bool {
	return isBasicProxyUpToDate(repoCR, repo.Name, repo.Online, repo.RemoteURL)
}

// isBasicHostedUpToDate checks if a basic hosted repository is up to date
// by comparing the name and online status with the desired state from cr.
func isBasicHostedUpToDate(repoCR *repositoryv1alpha1.Repository, name string, online bool) bool {
	if repoCR.Spec.ForProvider.Name != name {
		return false
	}

	if repoCR.Spec.ForProvider.Online != nil && *repoCR.Spec.ForProvider.Online != online {
		return false
	}

	return true
}

// isBasicProxyUpToDate checks if a basic proxy repository is up to date by
// comparing name, online status, and remote URL with the desired state from cr.
func isBasicProxyUpToDate(repoCR *repositoryv1alpha1.Repository, name string, online bool, remoteURL string) bool {
	if repoCR.Spec.ForProvider.Name != name {
		return false
	}

	if repoCR.Spec.ForProvider.Online != nil && *repoCR.Spec.ForProvider.Online != online {
		return false
	}

	if repoCR.Spec.ForProvider.Proxy != nil && repoCR.Spec.ForProvider.Proxy.RemoteURL != remoteURL {
		return false
	}

	return true
}

// isBasicGroupUpToDate checks if a basic group repository is up to date by
// comparing name, online status, and member names with the desired state.
func isBasicGroupUpToDate(repoCR *repositoryv1alpha1.Repository, name string, online bool, memberNames []string) bool {
	if repoCR.Spec.ForProvider.Name != name {
		return false
	}

	if repoCR.Spec.ForProvider.Online != nil && *repoCR.Spec.ForProvider.Online != online {
		return false
	}

	if repoCR.Spec.ForProvider.Group != nil {
		if !helpers.AreStringSlicesEqual(repoCR.Spec.ForProvider.Group.MemberNames, memberNames) {
			return false
		}
	}

	return true
}
