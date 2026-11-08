package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Yum repository format.
func init() {
	registerFormat("yum",
		hosted(
			get[v395.YumHostedApiRepository]((*nexusAPI).GetYumHostedRepository),
			create((*nexusAPI).CreateYumHostedRepository, v395.ApiCreateYumHostedRepositoryRequest.YumHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateYumHostedRepository, v395.ApiUpdateYumHostedRepositoryRequest.YumHostedRepositoryApiRequest),
			buildYumHosted,
		),
		proxy(
			get[v395.YumProxyApiRepository]((*nexusAPI).GetYumProxyRepository),
			create((*nexusAPI).CreateYumProxyRepository, v395.ApiCreateYumProxyRepositoryRequest.YumProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateYumProxyRepository, v395.ApiUpdateYumProxyRepositoryRequest.YumProxyRepositoryApiRequest),
			buildYumProxy,
		),
		group(
			get[v395.YumGroupApiRepository]((*nexusAPI).GetYumGroupRepository),
			create((*nexusAPI).CreateYumGroupRepository, v395.ApiCreateYumGroupRepositoryRequest.YumGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateYumGroupRepository, v395.ApiUpdateYumGroupRepositoryRequest.YumGroupRepositoryApiRequest),
			buildYumGroup,
		),
	)
}

// buildYumHosted renders a Yum hosted repository payload.
func buildYumHosted(desired desiredRepo) v395.YumHostedRepositoryApiRequest {
	repo := buildHosted[v395.YumHostedRepositoryApiRequest](desired)
	repo.Yum = desired.yum()

	return repo
}

// buildYumProxy renders a Yum proxy repository payload.
func buildYumProxy(desired desiredRepo) v395.YumProxyRepositoryApiRequest {
	repo := buildProxy[v395.YumProxyRepositoryApiRequest](desired)
	repo.YumSigning = desired.yumSigning()

	return repo
}

// buildYumGroup renders a Yum group repository payload.
func buildYumGroup(desired desiredRepo) v395.YumGroupRepositoryApiRequest {
	repo := buildGroup[v395.YumGroupRepositoryApiRequest](desired)
	repo.YumSigning = desired.yumSigning()

	return repo
}
