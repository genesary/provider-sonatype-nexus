package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the raw repository format.
func init() {
	registerFormat("raw",
		hosted(
			get[v395.RawHostedApiRepository]((*nexusAPI).GetRawHostedRepository),
			create((*nexusAPI).CreateRawHostedRepository, v395.ApiCreateRawHostedRepositoryRequest.RawHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRawHostedRepository, v395.ApiUpdateRawHostedRepositoryRequest.RawHostedRepositoryApiRequest),
			buildRawHosted,
		),
		proxy(
			get[v395.RawProxyApiRepository]((*nexusAPI).GetRawProxyRepository),
			create((*nexusAPI).CreateRawProxyRepository, v395.ApiCreateRawProxyRepositoryRequest.RawProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRawProxyRepository, v395.ApiUpdateRawProxyRepositoryRequest.RawProxyRepositoryApiRequest),
			buildRawProxy,
		),
		group(
			get[v395.RawGroupApiRepository]((*nexusAPI).GetRawGroupRepository),
			create((*nexusAPI).CreateRawGroupRepository, v395.ApiCreateRawGroupRepositoryRequest.RawGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRawGroupRepository, v395.ApiUpdateRawGroupRepositoryRequest.RawGroupRepositoryApiRequest),
			buildRawGroup,
		),
	)
}

// buildRawHosted renders a raw hosted repository payload.
func buildRawHosted(desired desiredRepo) v395.RawHostedRepositoryApiRequest {
	repo := buildHosted[v395.RawHostedRepositoryApiRequest](desired)
	repo.Raw = desired.raw()

	return repo
}

// buildRawProxy renders a raw proxy repository payload.
func buildRawProxy(desired desiredRepo) v395.RawProxyRepositoryApiRequest {
	repo := buildProxy[v395.RawProxyRepositoryApiRequest](desired)
	repo.Raw = desired.raw()

	return repo
}

// buildRawGroup renders a raw group repository payload.
func buildRawGroup(desired desiredRepo) v395.RawGroupRepositoryApiRequest {
	repo := buildGroup[v395.RawGroupRepositoryApiRequest](desired)
	repo.Raw = desired.raw()

	return repo
}
