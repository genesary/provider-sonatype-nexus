package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the Docker repository format.
func init() {
	registerFormat("docker",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.DockerHostedRepository] { return svc.Docker.Hosted },
			buildDockerHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.DockerProxyRepository] { return svc.Docker.Proxy },
			buildDockerProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.DockerGroupRepository] { return svc.Docker.Group },
			buildDockerGroup,
		),
	)
}

// buildDockerHosted renders a Docker hosted repository payload.
func buildDockerHosted(desired desiredRepo) schema.DockerHostedRepository {
	return schema.DockerHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.dockerHostedStorage(),
		Docker:  desired.docker(),
		Cleanup: desired.cleanup(),
	}
}

// buildDockerProxy renders a Docker proxy repository payload.
func buildDockerProxy(desired desiredRepo) schema.DockerProxyRepository {
	return schema.DockerProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		Docker:        desired.docker(),
		DockerProxy:   desired.dockerProxy(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildDockerGroup renders a Docker group repository payload.
func buildDockerGroup(desired desiredRepo) schema.DockerGroupRepository {
	return schema.DockerGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.groupDeploy(),
		Docker:  desired.docker(),
	}
}
