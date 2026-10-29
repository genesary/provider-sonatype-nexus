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
			buildMavenGroup,
		),
	)
}

// buildMavenHosted renders a Maven hosted repository payload.
func buildMavenHosted(desired desiredRepo) v395.MavenHostedRepositoryApiRequest {
	return v395.MavenHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
		Maven:   desired.maven(),
	}
}

// buildMavenProxy renders a Maven proxy repository payload.
func buildMavenProxy(desired desiredRepo) v395.MavenProxyRepositoryApiRequest {
	return v395.MavenProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Maven:           desired.maven(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClientWithPreemptiveAuth(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// buildMavenGroup renders a Maven group repository payload. The Maven block is
// left out on purpose: Nexus does not accept it for group repositories.
func buildMavenGroup(desired desiredRepo) v395.MavenGroupRepositoryApiRequest {
	return v395.MavenGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
