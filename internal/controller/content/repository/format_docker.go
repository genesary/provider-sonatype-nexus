package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Docker repository format.
func init() {
	registerFormat("docker",
		hosted(
			get[v395.DockerHostedApiRepository]((*nexusAPI).GetDockerHostedRepository),
			create((*nexusAPI).CreateDockerHostedRepository, v395.ApiCreateDockerHostedRepositoryRequest.DockerHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateDockerHostedRepository, v395.ApiUpdateDockerHostedRepositoryRequest.DockerHostedRepositoryApiRequest),
			buildDockerHosted,
		),
		proxy(
			get[v395.DockerProxyApiRepository]((*nexusAPI).GetDockerProxyRepository),
			create((*nexusAPI).CreateDockerProxyRepository, v395.ApiCreateDockerProxyRepositoryRequest.DockerProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateDockerProxyRepository, v395.ApiUpdateDockerProxyRepositoryRequest.DockerProxyRepositoryApiRequest),
			buildDockerProxy,
		),
		group(
			get[v395.DockerGroupApiRepository]((*nexusAPI).GetDockerGroupRepository),
			create((*nexusAPI).CreateDockerGroupRepository, v395.ApiCreateDockerGroupRepositoryRequest.DockerGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateDockerGroupRepository, v395.ApiUpdateDockerGroupRepositoryRequest.DockerGroupRepositoryApiRequest),
			buildDockerGroup,
		),
	)
}

// buildDockerHosted renders a Docker hosted repository payload.
func buildDockerHosted(desired desiredRepo) v395.DockerHostedRepositoryApiRequest {
	return v395.DockerHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.dockerHostedStorage(),
		Docker:  desired.docker(),
		Cleanup: desired.cleanup(),
	}
}

// buildDockerProxy renders a Docker proxy repository payload.
func buildDockerProxy(desired desiredRepo) v395.DockerProxyRepositoryApiRequest {
	return v395.DockerProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Docker:          desired.docker(),
		DockerProxy:     desired.dockerProxy(),
		Cleanup:         desired.cleanup(),
	}
}

// buildDockerGroup renders a Docker group repository payload.
func buildDockerGroup(desired desiredRepo) v395.DockerGroupRepositoryApiRequest {
	return v395.DockerGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.groupDeploy(),
		Docker:  desired.docker(),
	}
}
