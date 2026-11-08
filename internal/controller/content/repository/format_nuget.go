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
			buildHosted,
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
			buildGroup,
		),
	)
}

// buildNugetProxy renders a NuGet proxy repository payload.
func buildNugetProxy(desired desiredRepo) v395.NugetProxyRepositoryApiRequest {
	repo := buildProxy[v395.NugetProxyRepositoryApiRequest](desired)
	repo.NugetProxy = desired.nugetProxy()

	return repo
}
