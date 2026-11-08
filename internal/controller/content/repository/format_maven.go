package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Maven repository format.
func init() {
	registerFormat("maven2",
		hosted(
			get[v395.MavenHostedApiRepository]((*nexusAPI).GetMavenHostedRepository),
			create((*nexusAPI).CreateMavenHostedRepository, v395.ApiCreateMavenHostedRepositoryRequest.MavenHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateMavenHostedRepository, v395.ApiUpdateMavenHostedRepositoryRequest.MavenHostedRepositoryApiRequest),
			buildMavenHosted,
		),
		proxy(
			get[v395.MavenProxyApiRepository]((*nexusAPI).GetMavenProxyRepository),
			create((*nexusAPI).CreateMavenProxyRepository, v395.ApiCreateMavenProxyRepositoryRequest.MavenProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateMavenProxyRepository, v395.ApiUpdateMavenProxyRepositoryRequest.MavenProxyRepositoryApiRequest),
			buildMavenProxy,
		),
		group(
			get[v395.MavenGroupApiRepository]((*nexusAPI).GetMavenGroupRepository),
			create((*nexusAPI).CreateMavenGroupRepository, v395.ApiCreateMavenGroupRepositoryRequest.MavenGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateMavenGroupRepository, v395.ApiUpdateMavenGroupRepositoryRequest.MavenGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// buildMavenHosted renders a Maven hosted repository payload.
func buildMavenHosted(desired desiredRepo) v395.MavenHostedRepositoryApiRequest {
	repo := buildHosted[v395.MavenHostedRepositoryApiRequest](desired)
	repo.Maven = desired.maven()

	return repo
}

// buildMavenProxy renders a Maven proxy repository payload.
//
// Maven group repositories are registered with the shared group builder: the
// Maven block is left out on purpose, as Nexus does not accept it for groups.
func buildMavenProxy(desired desiredRepo) v395.MavenProxyRepositoryApiRequest {
	repo := buildPreemptiveProxy[v395.MavenProxyRepositoryApiRequest](desired)
	repo.Maven = desired.maven()

	return repo
}
