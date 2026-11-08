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
	repo := buildContainerHosted[v395.DockerHostedRepositoryApiRequest](desired)
	repo.Docker = desired.docker()

	return repo
}

// buildDockerProxy renders a Docker proxy repository payload.
func buildDockerProxy(desired desiredRepo) v395.DockerProxyRepositoryApiRequest {
	repo := buildProxy[v395.DockerProxyRepositoryApiRequest](desired)
	repo.Docker = desired.docker()
	repo.DockerProxy = desired.dockerProxy()

	return repo
}

// buildDockerGroup renders a Docker group repository payload.
func buildDockerGroup(desired desiredRepo) v395.DockerGroupRepositoryApiRequest {
	repo := buildDeployGroup[v395.DockerGroupRepositoryApiRequest](desired)
	repo.Docker = desired.docker()

	return repo
}
