// format_others.go contains handlers for formats with simpler patterns:
// - Formats with all three types (hosted, proxy, group): nuget, pypi, rubygems, yum, r, cargo, bower
// - Formats with partial support: apt, helm, go, gitlfs
// - Proxy-only formats: cocoapods, conan, conda

package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/utils"
)

// NugetHandler handles NuGet repository operations.
type NugetHandler struct{}

func (h *NugetHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *NugetHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetNugetHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetNugetProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetNugetGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *NugetHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateNugetHosted(ctx, repository.NugetHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		repo := repository.NugetProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.NugetProxy != nil {
			if cr.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge != nil {
				repo.QueryCacheItemMaxAge = int(*cr.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge)
			}

			if cr.Spec.ForProvider.NugetProxy.NugetVersion != nil {
				repo.NugetVersion = repository.NugetVersion(*cr.Spec.ForProvider.NugetProxy.NugetVersion)
			}
		}

		return client.Repository().CreateNugetProxy(ctx, repo)
	case "group":
		return client.Repository().CreateNugetGroup(ctx, repository.NugetGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

func (h *NugetHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateNugetHosted(ctx, name, repository.NugetHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		repo := repository.NugetProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.NugetProxy != nil {
			if cr.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge != nil {
				repo.QueryCacheItemMaxAge = int(*cr.Spec.ForProvider.NugetProxy.QueryCacheItemMaxAge)
			}

			if cr.Spec.ForProvider.NugetProxy.NugetVersion != nil {
				repo.NugetVersion = repository.NugetVersion(*cr.Spec.ForProvider.NugetProxy.NugetVersion)
			}
		}

		return client.Repository().UpdateNugetProxy(ctx, name, repo)
	case "group":
		return client.Repository().UpdateNugetGroup(ctx, name, repository.NugetGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

func (h *NugetHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteNugetHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteNugetProxy(ctx, name)
	case "group":
		return client.Repository().DeleteNugetGroup(ctx, name)
	}

	return errors.Errorf("unsupported nuget repository type: %s", repoType)
}

// PypiHandler handles PyPI repository operations.
type PypiHandler struct{}

func (h *PypiHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *PypiHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetPypiHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetPypiProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetPypiGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *PypiHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreatePypiHosted(ctx, repository.PypiHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().CreatePypiProxy(ctx, repository.PypiProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreatePypiGroup(ctx, repository.PypiGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

func (h *PypiHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdatePypiHosted(ctx, name, repository.PypiHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().UpdatePypiProxy(ctx, name, repository.PypiProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdatePypiGroup(ctx, name, repository.PypiGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

func (h *PypiHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeletePypiHosted(ctx, name)
	case "proxy":
		return client.Repository().DeletePypiProxy(ctx, name)
	case "group":
		return client.Repository().DeletePypiGroup(ctx, name)
	}

	return errors.Errorf("unsupported pypi repository type: %s", repoType)
}

// RubygemsHandler handles RubyGems repository operations.
type RubygemsHandler struct{}

func (h *RubygemsHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *RubygemsHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetRubygemsHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetRubygemsProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetRubygemsGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *RubygemsHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateRubygemsHosted(ctx, repository.RubyGemsHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().CreateRubygemsProxy(ctx, repository.RubyGemsProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreateRubygemsGroup(ctx, repository.RubyGemsGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

func (h *RubygemsHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateRubygemsHosted(ctx, name, repository.RubyGemsHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().UpdateRubygemsProxy(ctx, name, repository.RubyGemsProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdateRubygemsGroup(ctx, name, repository.RubyGemsGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

func (h *RubygemsHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteRubygemsHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteRubygemsProxy(ctx, name)
	case "group":
		return client.Repository().DeleteRubygemsGroup(ctx, name)
	}

	return errors.Errorf("unsupported rubygems repository type: %s", repoType)
}

// YumHandler handles Yum repository operations.
type YumHandler struct{}

func (h *YumHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *YumHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetYumHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetYumProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetYumGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *YumHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		repo := repository.YumHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)}
		if cr.Spec.ForProvider.Yum != nil {
			if cr.Spec.ForProvider.Yum.RepodataDepth != nil {
				repo.RepodataDepth = int(*cr.Spec.ForProvider.Yum.RepodataDepth)
			}

			if cr.Spec.ForProvider.Yum.DeployPolicy != nil {
				deployPolicy := repository.YumDeployPolicy(*cr.Spec.ForProvider.Yum.DeployPolicy)
				repo.DeployPolicy = &deployPolicy
			}
		}

		return client.Repository().CreateYumHosted(ctx, repo)
	case "proxy":
		return client.Repository().CreateYumProxy(ctx, repository.YumProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreateYumGroup(ctx, repository.YumGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

func (h *YumHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		repo := repository.YumHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)}
		if cr.Spec.ForProvider.Yum != nil {
			if cr.Spec.ForProvider.Yum.RepodataDepth != nil {
				repo.RepodataDepth = int(*cr.Spec.ForProvider.Yum.RepodataDepth)
			}

			if cr.Spec.ForProvider.Yum.DeployPolicy != nil {
				deployPolicy := repository.YumDeployPolicy(*cr.Spec.ForProvider.Yum.DeployPolicy)
				repo.DeployPolicy = &deployPolicy
			}
		}

		return client.Repository().UpdateYumHosted(ctx, name, repo)
	case "proxy":
		return client.Repository().UpdateYumProxy(ctx, name, repository.YumProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdateYumGroup(ctx, name, repository.YumGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

func (h *YumHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteYumHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteYumProxy(ctx, name)
	case "group":
		return client.Repository().DeleteYumGroup(ctx, name)
	}

	return errors.Errorf("unsupported yum repository type: %s", repoType)
}

// RHandler handles R repository operations.
type RHandler struct{}

func (h *RHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *RHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetRHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetRProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetRGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *RHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateRHosted(ctx, repository.RHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().CreateRProxy(ctx, repository.RProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreateRGroup(ctx, repository.RGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

func (h *RHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateRHosted(ctx, name, repository.RHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().UpdateRProxy(ctx, name, repository.RProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdateRGroup(ctx, name, repository.RGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

func (h *RHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteRHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteRProxy(ctx, name)
	case "group":
		return client.Repository().DeleteRGroup(ctx, name)
	}

	return errors.Errorf("unsupported r repository type: %s", repoType)
}

// CargoHandler handles Cargo repository operations.
type CargoHandler struct{}

func (h *CargoHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *CargoHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetCargoHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetCargoProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetCargoGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *CargoHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateCargoHosted(ctx, repository.CargoHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().CreateCargoProxy(ctx, repository.CargoProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreateCargoGroup(ctx, repository.CargoGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

func (h *CargoHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateCargoHosted(ctx, name, repository.CargoHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().UpdateCargoProxy(ctx, name, repository.CargoProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdateCargoGroup(ctx, name, repository.CargoGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

func (h *CargoHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteCargoHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteCargoProxy(ctx, name)
	case "group":
		return client.Repository().DeleteCargoGroup(ctx, name)
	}

	return errors.Errorf("unsupported cargo repository type: %s", repoType)
}

// BowerHandler handles Bower repository operations.
type BowerHandler struct{}

func (h *BowerHandler) SupportedTypes() []string { return []string{"hosted", "proxy", "group"} }

func (h *BowerHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetBowerHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetBowerProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetBowerGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *BowerHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateBowerHosted(ctx, repository.BowerHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		repo := repository.BowerProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.Bower != nil && cr.Spec.ForProvider.Bower.RewritePackageUrls != nil {
			repo.Bower = repository.Bower{RewritePackageUrls: *cr.Spec.ForProvider.Bower.RewritePackageUrls}
		}

		return client.Repository().CreateBowerProxy(ctx, repo)
	case "group":
		return client.Repository().CreateBowerGroup(ctx, repository.BowerGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

func (h *BowerHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateBowerHosted(ctx, name, repository.BowerHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		repo := repository.BowerProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.Bower != nil && cr.Spec.ForProvider.Bower.RewritePackageUrls != nil {
			repo.Bower = repository.Bower{RewritePackageUrls: *cr.Spec.ForProvider.Bower.RewritePackageUrls}
		}

		return client.Repository().UpdateBowerProxy(ctx, name, repo)
	case "group":
		return client.Repository().UpdateBowerGroup(ctx, name, repository.BowerGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

func (h *BowerHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteBowerHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteBowerProxy(ctx, name)
	case "group":
		return client.Repository().DeleteBowerGroup(ctx, name)
	}

	return errors.Errorf("unsupported bower repository type: %s", repoType)
}

// AptHandler handles APT repository operations.
type AptHandler struct{}

func (h *AptHandler) SupportedTypes() []string { return []string{"hosted", "proxy"} }

func (h *AptHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetAptHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetAptProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	}

	return false, false
}

func (h *AptHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		repo := repository.AptHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)}
		if cr.Spec.ForProvider.Apt != nil && cr.Spec.ForProvider.Apt.Distribution != nil {
			repo.Apt.Distribution = *cr.Spec.ForProvider.Apt.Distribution
		}

		if cr.Spec.ForProvider.AptSigning != nil {
			repo.AptSigning = repository.AptSigning{Keypair: cr.Spec.ForProvider.AptSigning.Keypair, Passphrase: cr.Spec.ForProvider.AptSigning.Passphrase}
		}

		return client.Repository().CreateAptHosted(ctx, repo)
	case "proxy":
		repo := repository.AptProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.Apt != nil {
			if cr.Spec.ForProvider.Apt.Distribution != nil {
				repo.Apt.Distribution = *cr.Spec.ForProvider.Apt.Distribution
			}

			if cr.Spec.ForProvider.Apt.Flat != nil {
				repo.Apt.Flat = *cr.Spec.ForProvider.Apt.Flat
			}
		}

		return client.Repository().CreateAptProxy(ctx, repo)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

func (h *AptHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		repo := repository.AptHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)}
		if cr.Spec.ForProvider.Apt != nil && cr.Spec.ForProvider.Apt.Distribution != nil {
			repo.Apt.Distribution = *cr.Spec.ForProvider.Apt.Distribution
		}

		if cr.Spec.ForProvider.AptSigning != nil {
			repo.AptSigning = repository.AptSigning{Keypair: cr.Spec.ForProvider.AptSigning.Keypair, Passphrase: cr.Spec.ForProvider.AptSigning.Passphrase}
		}

		return client.Repository().UpdateAptHosted(ctx, name, repo)
	case "proxy":
		repo := repository.AptProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)}
		if cr.Spec.ForProvider.Apt != nil {
			if cr.Spec.ForProvider.Apt.Distribution != nil {
				repo.Apt.Distribution = *cr.Spec.ForProvider.Apt.Distribution
			}

			if cr.Spec.ForProvider.Apt.Flat != nil {
				repo.Apt.Flat = *cr.Spec.ForProvider.Apt.Flat
			}
		}

		return client.Repository().UpdateAptProxy(ctx, name, repo)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

func (h *AptHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteAptHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteAptProxy(ctx, name)
	}

	return errors.Errorf("unsupported apt repository type: %s", repoType)
}

// HelmHandler handles Helm repository operations.
type HelmHandler struct{}

func (h *HelmHandler) SupportedTypes() []string { return []string{"hosted", "proxy"} }

func (h *HelmHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "hosted":
		repo, err := client.Repository().GetHelmHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	case "proxy":
		repo, err := client.Repository().GetHelmProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	}

	return false, false
}

func (h *HelmHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().CreateHelmHosted(ctx, repository.HelmHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().CreateHelmProxy(ctx, repository.HelmProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

func (h *HelmHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().UpdateHelmHosted(ctx, name, repository.HelmHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	case "proxy":
		return client.Repository().UpdateHelmProxy(ctx, name, repository.HelmProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

func (h *HelmHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "hosted":
		return client.Repository().DeleteHelmHosted(ctx, name)
	case "proxy":
		return client.Repository().DeleteHelmProxy(ctx, name)
	}

	return errors.Errorf("unsupported helm repository type: %s", repoType)
}

// GoHandler handles Go repository operations.
type GoHandler struct{}

func (h *GoHandler) SupportedTypes() []string { return []string{"proxy", "group"} }

func (h *GoHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	switch repoType {
	case "proxy":
		repo, err := client.Repository().GetGoProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	case "group":
		repo, err := client.Repository().GetGoGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicGroupUpToDate(cr, repo.Name, repo.Online, repo.MemberNames)
	}

	return false, false
}

func (h *GoHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "proxy":
		return client.Repository().CreateGoProxy(ctx, repository.GoProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().CreateGoGroup(ctx, repository.GoGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

func (h *GoHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "proxy":
		return client.Repository().UpdateGoProxy(ctx, name, repository.GoProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	case "group":
		return client.Repository().UpdateGoGroup(ctx, name, repository.GoGroupRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Group: generateGroupConfig(cr)})
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

func (h *GoHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	switch repoType {
	case "proxy":
		return client.Repository().DeleteGoProxy(ctx, name)
	case "group":
		return client.Repository().DeleteGoGroup(ctx, name)
	}

	return errors.Errorf("unsupported go repository type: %s", repoType)
}

// GitLfsHandler handles GitLFS repository operations.
type GitLfsHandler struct{}

func (h *GitLfsHandler) SupportedTypes() []string { return []string{"hosted"} }

func (h *GitLfsHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	if repoType == "hosted" {
		repo, err := client.Repository().GetGitLfsHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicHostedUpToDate(cr, repo.Name, repo.Online)
	}

	return false, false
}

func (h *GitLfsHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "hosted" {
		return client.Repository().CreateGitLfsHosted(ctx, repository.GitLfsHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

func (h *GitLfsHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "hosted" {
		return client.Repository().UpdateGitLfsHosted(ctx, name, repository.GitLfsHostedRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateHostedStorage(cr), Cleanup: generateCleanup(cr)})
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

func (h *GitLfsHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	if repoType == "hosted" {
		return client.Repository().DeleteGitLfsHosted(ctx, name)
	}

	return errors.Errorf("unsupported gitlfs repository type: %s", repoType)
}

// CocoapodsHandler handles Cocoapods repository operations.
type CocoapodsHandler struct{}

func (h *CocoapodsHandler) SupportedTypes() []string { return []string{"proxy"} }

func (h *CocoapodsHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	if repoType == "proxy" {
		repo, err := client.Repository().GetCocoapodsProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	}

	return false, false
}

func (h *CocoapodsHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().CreateCocoapodsProxy(ctx, repository.CocoapodsProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

func (h *CocoapodsHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().UpdateCocoapodsProxy(ctx, name, repository.CocoapodsProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

func (h *CocoapodsHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().DeleteCocoapodsProxy(ctx, name)
	}

	return errors.Errorf("unsupported cocoapods repository type: %s", repoType)
}

// ConanHandler handles Conan repository operations.
type ConanHandler struct{}

func (h *ConanHandler) SupportedTypes() []string { return []string{"proxy"} }

func (h *ConanHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	if repoType == "proxy" {
		repo, err := client.Repository().GetConanProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	}

	return false, false
}

func (h *ConanHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().CreateConanProxy(ctx, repository.ConanProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

func (h *ConanHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().UpdateConanProxy(ctx, name, repository.ConanProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

func (h *ConanHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().DeleteConanProxy(ctx, name)
	}

	return errors.Errorf("unsupported conan repository type: %s", repoType)
}

// CondaHandler handles Conda repository operations.
type CondaHandler struct{}

func (h *CondaHandler) SupportedTypes() []string { return []string{"proxy"} }

func (h *CondaHandler) Observe(ctx context.Context, client nexus.Client, name, repoType string, cr *v1alpha1.Repository) (bool, bool) {
	if repoType == "proxy" {
		repo, err := client.Repository().GetCondaProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}

		return true, isBasicProxyUpToDate(cr, repo.Name, repo.Online, repo.RemoteURL)
	}

	return false, false
}

func (h *CondaHandler) Create(ctx context.Context, client nexus.Client, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().CreateCondaProxy(ctx, repository.CondaProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

func (h *CondaHandler) Update(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Repository, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().UpdateCondaProxy(ctx, name, repository.CondaProxyRepository{Name: cr.Spec.ForProvider.Name, Online: getOnline(cr), Storage: generateProxyStorage(cr), Proxy: generateProxyConfig(cr), NegativeCache: generateNegativeCache(cr), HTTPClient: generateHTTPClient(ctx, cr)})
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

func (h *CondaHandler) Delete(ctx context.Context, client nexus.Client, name, repoType string) error {
	if repoType == "proxy" {
		return client.Repository().DeleteCondaProxy(ctx, name)
	}

	return errors.Errorf("unsupported conda repository type: %s", repoType)
}

// Helper functions for up-to-date checks

func isBasicHostedUpToDate(cr *v1alpha1.Repository, name string, online bool) bool {
	if cr.Spec.ForProvider.Name != name {
		return false
	}

	if cr.Spec.ForProvider.Online != nil && *cr.Spec.ForProvider.Online != online {
		return false
	}

	return true
}

func isBasicProxyUpToDate(cr *v1alpha1.Repository, name string, online bool, remoteURL string) bool {
	if cr.Spec.ForProvider.Name != name {
		return false
	}

	if cr.Spec.ForProvider.Online != nil && *cr.Spec.ForProvider.Online != online {
		return false
	}

	if cr.Spec.ForProvider.Proxy != nil && cr.Spec.ForProvider.Proxy.RemoteURL != remoteURL {
		return false
	}

	return true
}

func isBasicGroupUpToDate(cr *v1alpha1.Repository, name string, online bool, memberNames []string) bool {
	if cr.Spec.ForProvider.Name != name {
		return false
	}

	if cr.Spec.ForProvider.Online != nil && *cr.Spec.ForProvider.Online != online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !utils.StringSlicesEqual(cr.Spec.ForProvider.Group.MemberNames, memberNames) {
			return false
		}
	}

	return true
}

// stringSlicesEqual is kept for backward compatibility with tests.
func stringSlicesEqual(a, b []string) bool {
	return utils.StringSlicesEqual(a, b)
}
