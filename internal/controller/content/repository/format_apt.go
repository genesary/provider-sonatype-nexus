package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the APT repository format.
func init() {
	registerFormat("apt",
		hosted(
			get[v395.AptHostedApiRepository]((*nexusAPI).GetAptHostedRepository),
			create((*nexusAPI).CreateAptHostedRepository, v395.ApiCreateAptHostedRepositoryRequest.AptHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateAptHostedRepository, v395.ApiUpdateAptHostedRepositoryRequest.AptHostedRepositoryApiRequest),
			buildAptHosted,
		),
		proxy(
			get[v395.AptProxyApiRepository]((*nexusAPI).GetAptProxyRepository),
			create((*nexusAPI).CreateAptProxyRepository, v395.ApiCreateAptProxyRepositoryRequest.AptProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateAptProxyRepository, v395.ApiUpdateAptProxyRepositoryRequest.AptProxyRepositoryApiRequest),
			buildAptProxy,
		),
	)
}

// buildAptHosted renders an APT hosted repository payload.
func buildAptHosted(desired desiredRepo) v395.AptHostedRepositoryApiRequest {
	repo := buildHosted[v395.AptHostedRepositoryApiRequest](desired)
	repo.Apt = desired.apt()
	repo.AptSigning = desired.aptSigning()

	return repo
}

// buildAptProxy renders an APT proxy repository payload.
func buildAptProxy(desired desiredRepo) v395.AptProxyRepositoryApiRequest {
	repo := buildProxy[v395.AptProxyRepositoryApiRequest](desired)
	repo.Apt = desired.aptProxy()

	return repo
}
