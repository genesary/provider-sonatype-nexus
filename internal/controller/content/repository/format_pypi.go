package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the PyPI repository format.
func init() {
	registerFormat("pypi",
		hosted(
			get[v395.PypiHostedApiRepository]((*nexusAPI).GetPypiHostedRepository),
			create((*nexusAPI).CreatePypiHostedRepository, v395.ApiCreatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiHostedRepository, v395.ApiUpdatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.PyPiProxyApiRepository]((*nexusAPI).GetPypiProxyRepository),
			create((*nexusAPI).CreatePypiProxyRepository, v395.ApiCreatePypiProxyRepositoryRequest.PypiProxyRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiProxyRepository, v395.ApiUpdatePypiProxyRepositoryRequest.PypiProxyRepositoryApiRequest),
			buildPypiProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetPypiGroupRepository),
			create((*nexusAPI).CreatePypiGroupRepository, v395.ApiCreatePypiGroupRepositoryRequest.PypiGroupRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiGroupRepository, v395.ApiUpdatePypiGroupRepositoryRequest.PypiGroupRepositoryApiRequest),
			buildDeployGroup,
		),
	)
}

// buildPypiProxy renders a PyPI proxy repository payload.
func buildPypiProxy(desired desiredRepo) v395.PypiProxyRepositoryApiRequest {
	repo := buildPreemptiveProxy[v395.PypiProxyRepositoryApiRequest](desired)
	repo.Pypi = desired.pypiProxy()

	return repo
}
