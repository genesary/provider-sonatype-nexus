package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the NuGet repository format.
func init() {
	registerFormat("nuget",
		hosted(
			get[v395.NugetHostedApiRepository]((*nexusAPI).GetNugetHostedRepository),
			create((*nexusAPI).CreateNugetHostedRepository, v395.ApiCreateNugetHostedRepositoryRequest.NugetHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateNugetHostedRepository, v395.ApiUpdateNugetHostedRepositoryRequest.NugetHostedRepositoryApiRequest),
			buildNugetHosted,
		),
		proxy(
			get[v395.NugetProxyApiRepository]((*nexusAPI).GetNugetProxyRepository),
			create((*nexusAPI).CreateNugetProxyRepository, v395.ApiCreateNugetProxyRepositoryRequest.NugetProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateNugetProxyRepository, v395.ApiUpdateNugetProxyRepositoryRequest.NugetProxyRepositoryApiRequest),
			buildNugetProxy,
		),
		group(
			get[v395.NugetGroupApiRepository]((*nexusAPI).GetNugetGroupRepository),
			create((*nexusAPI).CreateNugetGroupRepository, v395.ApiCreateNugetGroupRepositoryRequest.NugetGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateNugetGroupRepository, v395.ApiUpdateNugetGroupRepositoryRequest.NugetGroupRepositoryApiRequest),
			buildNugetGroup,
		),
	)
}

// buildNugetHosted renders a NuGet hosted repository payload.
func buildNugetHosted(desired desiredRepo) v395.NugetHostedRepositoryApiRequest {
	return v395.NugetHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildNugetProxy renders a NuGet proxy repository payload.
func buildNugetProxy(desired desiredRepo) v395.NugetProxyRepositoryApiRequest {
	return v395.NugetProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		NugetProxy:      desired.nugetProxy(),
		Cleanup:         desired.cleanup(),
	}
}

// buildNugetGroup renders a NuGet group repository payload.
func buildNugetGroup(desired desiredRepo) v395.NugetGroupRepositoryApiRequest {
	return v395.NugetGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
