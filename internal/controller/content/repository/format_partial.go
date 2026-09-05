package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// This file holds the formats that Nexus only offers for some repository
// types, and that add no settings of their own. Asking for a type a format
// does not support is rejected by the dispatcher with a message naming the
// types it does support.

// init registers the Helm repository format, which has no group type.
func init() {
	registerFormat("helm",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.HelmHostedRepository] { return svc.Helm.Hosted },
			buildHelmHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.HelmProxyRepository] { return svc.Helm.Proxy },
			buildHelmProxy,
		),
	)
}

// buildHelmHosted renders a Helm hosted repository payload.
func buildHelmHosted(desired desiredRepo) schema.HelmHostedRepository {
	return schema.HelmHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildHelmProxy renders a Helm proxy repository payload.
func buildHelmProxy(desired desiredRepo) schema.HelmProxyRepository {
	return schema.HelmProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// init registers the Conan repository format, which has no group type.
func init() {
	registerFormat("conan",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.ConanHostedRepository] { return svc.Conan.Hosted },
			buildConanHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.ConanProxyRepository] { return svc.Conan.Proxy },
			buildConanProxy,
		),
	)
}

// buildConanHosted renders a Conan hosted repository payload.
func buildConanHosted(desired desiredRepo) schema.ConanHostedRepository {
	return schema.ConanHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildConanProxy renders a Conan proxy repository payload.
func buildConanProxy(desired desiredRepo) schema.ConanProxyRepository {
	return schema.ConanProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// init registers the Go repository format, which has no hosted type.
func init() {
	registerFormat("go",
		proxy(
			func(svc *nexusAPI) repoAPI[schema.GoProxyRepository] { return svc.Go.Proxy },
			buildGoProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.GoGroupRepository] { return svc.Go.Group },
			buildGoGroup,
		),
	)
}

// buildGoProxy renders a Go proxy repository payload.
func buildGoProxy(desired desiredRepo) schema.GoProxyRepository {
	return schema.GoProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// buildGoGroup renders a Go group repository payload.
func buildGoGroup(desired desiredRepo) schema.GoGroupRepository {
	return schema.GoGroupRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.storage(), Group: desired.group(),
	}
}

// init registers the Git LFS repository format, which is hosted only.
func init() {
	registerFormat("gitlfs",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.GitLfsHostedRepository] { return svc.GitLfs.Hosted },
			buildGitLfsHosted,
		),
	)
}

// buildGitLfsHosted renders a Git LFS hosted repository payload.
func buildGitLfsHosted(desired desiredRepo) schema.GitLfsHostedRepository {
	return schema.GitLfsHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// init registers the CocoaPods repository format, which is proxy only.
func init() {
	registerFormat("cocoapods",
		proxy(
			func(svc *nexusAPI) repoAPI[schema.CocoapodsProxyRepository] { return svc.Cocoapods.Proxy },
			buildCocoapodsProxy,
		),
	)
}

// buildCocoapodsProxy renders a CocoaPods proxy repository payload.
func buildCocoapodsProxy(desired desiredRepo) schema.CocoapodsProxyRepository {
	return schema.CocoapodsProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// init registers the Conda repository format, which is proxy only.
func init() {
	registerFormat("conda",
		proxy(
			func(svc *nexusAPI) repoAPI[schema.CondaProxyRepository] { return svc.Conda.Proxy },
			buildCondaProxy,
		),
	)
}

// buildCondaProxy renders a Conda proxy repository payload.
func buildCondaProxy(desired desiredRepo) schema.CondaProxyRepository {
	return schema.CondaProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// init registers the Hugging Face repository format, which is proxy only.
func init() {
	registerFormat("huggingface",
		proxy(
			func(svc *nexusAPI) repoAPI[schema.HuggingfaceProxyRepository] { return svc.Huggingface.Proxy },
			buildHuggingfaceProxy,
		),
	)
}

// buildHuggingfaceProxy renders a Hugging Face proxy repository payload.
func buildHuggingfaceProxy(desired desiredRepo) schema.HuggingfaceProxyRepository {
	return schema.HuggingfaceProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// init registers the Eclipse P2 repository format, which is proxy only.
func init() {
	registerFormat("p2",
		proxy(
			func(svc *nexusAPI) repoAPI[schema.P2ProxyRepository] { return svc.P2.Proxy },
			buildP2Proxy,
		),
	)
}

// buildP2Proxy renders a P2 proxy repository payload.
func buildP2Proxy(desired desiredRepo) schema.P2ProxyRepository {
	return schema.P2ProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}
