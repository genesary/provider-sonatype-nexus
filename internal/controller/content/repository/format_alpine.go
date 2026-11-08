package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"
)

// init registers the Alpine repository format.
func init() {
	registerFormat("alpine",
		hosted(
			get[v395.AlpineHostedApiRepository]((*nexusAPI).GetAlpineHostedRepository),
			create((*nexusAPI).CreateAlpineHostedRepository, v395.ApiCreateAlpineHostedRepositoryRequest.AlpineHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateAlpineHostedRepository, v395.ApiUpdateAlpineHostedRepositoryRequest.AlpineHostedRepositoryApiRequest),
			buildAlpineHosted,
		),
		proxy(
			get[v395.AlpineProxyApiRepository]((*nexusAPI).GetAlpineProxyRepository),
			create((*nexusAPI).CreateAlpineProxyRepository, v395.ApiCreateAlpineProxyRepositoryRequest.AlpineProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateAlpineProxyRepository, v395.ApiUpdateAlpineProxyRepositoryRequest.AlpineProxyRepositoryApiRequest),
			buildAlpineProxy,
		),
		group(
			get[v395.AlpineGroupApiRepository]((*nexusAPI).GetAlpineGroupRepository),
			create((*nexusAPI).CreateAlpineGroupRepository, v395.ApiCreateAlpineGroupRepositoryRequest.AlpineGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateAlpineGroupRepository, v395.ApiUpdateAlpineGroupRepositoryRequest.AlpineGroupRepositoryApiRequest),
			buildAlpineGroup,
		),
	)
}

// buildAlpineHosted renders an alpine hosted repository payload. Nexus only
// signs a hosted repository's index when the spec gives it a key.
func buildAlpineHosted(desired desiredRepo) v395.AlpineHostedRepositoryApiRequest {
	repo := buildHosted[v395.AlpineHostedRepositoryApiRequest](desired)
	repo.AlpineSigning = desired.alpineSigning()

	return repo
}

// buildAlpineProxy renders an alpine proxy repository payload.
func buildAlpineProxy(desired desiredRepo) v395.AlpineProxyRepositoryApiRequest {
	repo := buildProxy[v395.AlpineProxyRepositoryApiRequest](desired)
	repo.AlpineSigning = ptr.Deref(desired.alpineSigning(), v395.AlpineSigningRepositoriesAttributes{})

	return repo
}

// buildAlpineGroup renders an alpine group repository payload.
func buildAlpineGroup(desired desiredRepo) v395.AlpineGroupRepositoryApiRequest {
	repo := buildGroup[v395.AlpineGroupRepositoryApiRequest](desired)
	repo.AlpineSigning = ptr.Deref(desired.alpineSigning(), v395.AlpineSigningRepositoriesAttributes{})

	return repo
}
