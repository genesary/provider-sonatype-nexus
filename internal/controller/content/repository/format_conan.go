package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Conan repository format.
func init() {
	registerFormat("conan",
		hosted(
			get[v395.ConanHostedApiRepository]((*nexusAPI).GetConanHostedRepository),
			create((*nexusAPI).CreateConanHostedRepository, v395.ApiCreateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateConanHostedRepository, v395.ApiUpdateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.ConanProxyApiRepository]((*nexusAPI).GetConanProxyRepository),
			create((*nexusAPI).CreateConanProxyRepository, v395.ApiCreateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateConanProxyRepository, v395.ApiUpdateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			buildConanProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetConanGroupRepository),
			create((*nexusAPI).CreateConanGroupRepository, v395.ApiCreateConanGroupRepositoryRequest.ConanGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateConanGroupRepository, v395.ApiUpdateConanGroupRepositoryRequest.ConanGroupRepositoryApiRequest),
			buildDeployGroup,
		),
	)
}

// buildConanProxy renders a Conan proxy repository payload.
func buildConanProxy(desired desiredRepo) v395.ConanProxyRepositoryApiRequest {
	repo := buildProxy[v395.ConanProxyRepositoryApiRequest](desired)
	repo.ConanProxy = desired.conanProxy()

	return repo
}
