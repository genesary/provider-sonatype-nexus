package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// This file holds the formats that Nexus only offers for some repository
// types, and that add no settings of their own. Asking for a type a format
// does not support is rejected by the dispatcher with a message naming the
// types it does support.

// init registers the Helm repository format.
func init() {
	registerFormat("helm",
		hosted(
			get[v395.HelmHostedApiRepository]((*nexusAPI).GetHelmHostedRepository),
			create((*nexusAPI).CreateHelmHostedRepository, v395.ApiCreateHelmHostedRepositoryRequest.HelmHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateHelmHostedRepository, v395.ApiUpdateHelmHostedRepositoryRequest.HelmHostedRepositoryApiRequest),
			buildHelmHosted,
		),
		proxy(
			get[v395.HelmProxyApiRepository]((*nexusAPI).GetHelmProxyRepository),
			create((*nexusAPI).CreateHelmProxyRepository, v395.ApiCreateHelmProxyRepositoryRequest.HelmProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateHelmProxyRepository, v395.ApiUpdateHelmProxyRepositoryRequest.HelmProxyRepositoryApiRequest),
			buildHelmProxy,
		),
	)
}

// buildHelmHosted renders a Helm hosted repository payload.
func buildHelmHosted(desired desiredRepo) v395.HelmHostedRepositoryApiRequest {
	return v395.HelmHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildHelmProxy renders a Helm proxy repository payload.
func buildHelmProxy(desired desiredRepo) v395.HelmProxyRepositoryApiRequest {
	return v395.HelmProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// init registers the Conan repository format.
func init() {
	registerFormat("conan",
		hosted(
			get[v395.ConanHostedApiRepository]((*nexusAPI).GetConanHostedRepository),
			create((*nexusAPI).CreateConanHostedRepository, v395.ApiCreateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateConanHostedRepository, v395.ApiUpdateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			buildConanHosted,
		),
		proxy(
			get[v395.ConanProxyApiRepository]((*nexusAPI).GetConanProxyRepository),
			create((*nexusAPI).CreateConanProxyRepository, v395.ApiCreateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateConanProxyRepository, v395.ApiUpdateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			buildConanProxy,
		),
	)
}

// buildConanHosted renders a Conan hosted repository payload.
func buildConanHosted(desired desiredRepo) v395.ConanHostedRepositoryApiRequest {
	return v395.ConanHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildConanProxy renders a Conan proxy repository payload.
func buildConanProxy(desired desiredRepo) v395.ConanProxyRepositoryApiRequest {
	return v395.ConanProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// init registers the Go repository format.
func init() {
	registerFormat("go",
		proxy(
			get[v395.GolangProxyApiRepository]((*nexusAPI).GetGoProxyRepository),
			create((*nexusAPI).CreateGoProxyRepository, v395.ApiCreateGoProxyRepositoryRequest.GolangProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateGoProxyRepository, v395.ApiUpdateGoProxyRepositoryRequest.GolangProxyRepositoryApiRequest),
			buildGolangProxy,
		),
		group(
			get[v395.GolangGroupApiRepository]((*nexusAPI).GetGoGroupRepository),
			create((*nexusAPI).CreateGoGroupRepository, v395.ApiCreateGoGroupRepositoryRequest.GolangGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateGoGroupRepository, v395.ApiUpdateGoGroupRepositoryRequest.GolangGroupRepositoryApiRequest),
			buildGolangGroup,
		),
	)
}

// buildGolangProxy renders a Go proxy repository payload.
func buildGolangProxy(desired desiredRepo) v395.GolangProxyRepositoryApiRequest {
	return v395.GolangProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// buildGolangGroup renders a Go group repository payload.
func buildGolangGroup(desired desiredRepo) v395.GolangGroupRepositoryApiRequest {
	return v395.GolangGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}

// init registers the Git LFS repository format.
func init() {
	registerFormat("gitlfs",
		hosted(
			get[v395.GitLfsHostedApiRepository]((*nexusAPI).GetGitlfsHostedRepository),
			create((*nexusAPI).CreateGitlfsHostedRepository, v395.ApiCreateGitlfsHostedRepositoryRequest.GitLfsHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateGitlfsHostedRepository, v395.ApiUpdateGitlfsHostedRepositoryRequest.GitLfsHostedRepositoryApiRequest),
			buildGitLfsHosted,
		),
	)
}

// buildGitLfsHosted renders a Git LFS hosted repository payload.
func buildGitLfsHosted(desired desiredRepo) v395.GitLfsHostedRepositoryApiRequest {
	return v395.GitLfsHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// init registers the CocoaPods repository format.
func init() {
	registerFormat("cocoapods",
		proxy(
			get[v395.CocoapodsProxyApiRepository]((*nexusAPI).GetCocoapodsProxyRepository),
			create((*nexusAPI).CreateCocoapodsProxyRepository, v395.ApiCreateCocoapodsProxyRepositoryRequest.CocoapodsProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateCocoapodsProxyRepository, v395.ApiUpdateCocoapodsProxyRepositoryRequest.CocoapodsProxyRepositoryApiRequest),
			buildCocoapodsProxy,
		),
	)
}

// buildCocoapodsProxy renders a CocoaPods proxy repository payload.
func buildCocoapodsProxy(desired desiredRepo) v395.CocoapodsProxyRepositoryApiRequest {
	return v395.CocoapodsProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// init registers the Conda repository format.
func init() {
	registerFormat("conda",
		proxy(
			get[v395.CondaProxyApiRepository]((*nexusAPI).GetCondaProxyRepository),
			create((*nexusAPI).CreateCondaProxyRepository, v395.ApiCreateCondaProxyRepositoryRequest.CondaProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateCondaProxyRepository, v395.ApiUpdateCondaProxyRepositoryRequest.CondaProxyRepositoryApiRequest),
			buildCondaProxy,
		),
	)
}

// buildCondaProxy renders a Conda proxy repository payload.
func buildCondaProxy(desired desiredRepo) v395.CondaProxyRepositoryApiRequest {
	return v395.CondaProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// init registers the Hugging Face repository format.
func init() {
	registerFormat("huggingface",
		proxy(
			get[v395.HuggingFaceProxyApiRepository]((*nexusAPI).GetHuggingfaceProxyRepository),
			create((*nexusAPI).CreateHuggingfaceProxyRepository, v395.ApiCreateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateHuggingfaceProxyRepository, v395.ApiUpdateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			buildHuggingFaceProxy,
		),
	)
}

// buildHuggingFaceProxy renders a Hugging Face proxy repository payload.
func buildHuggingFaceProxy(desired desiredRepo) v395.HuggingFaceProxyRepositoryApiRequest {
	return v395.HuggingFaceProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// init registers the P2 repository format.
func init() {
	registerFormat("p2",
		proxy(
			get[v395.P2ProxyApiRepository]((*nexusAPI).GetP2ProxyRepository),
			create((*nexusAPI).CreateP2ProxyRepository, v395.ApiCreateP2ProxyRepositoryRequest.P2ProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateP2ProxyRepository, v395.ApiUpdateP2ProxyRepositoryRequest.P2ProxyRepositoryApiRequest),
			buildP2Proxy,
		),
	)
}

// buildP2Proxy renders a P2 proxy repository payload.
func buildP2Proxy(desired desiredRepo) v395.P2ProxyRepositoryApiRequest {
	return v395.P2ProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}
