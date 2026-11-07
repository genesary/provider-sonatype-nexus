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

// buildAptHosted renders a APT hosted repository payload.
func buildAptHosted(desired desiredRepo) v395.AptHostedRepositoryApiRequest {
	return v395.AptHostedRepositoryApiRequest{
		Name:       desired.name(),
		Online:     desired.online(),
		Storage:    desired.hostedStorage(),
		Cleanup:    desired.cleanup(),
		Apt:        desired.apt(),
		AptSigning: desired.aptSigning(),
	}
}

// buildAptProxy renders a APT proxy repository payload.
func buildAptProxy(desired desiredRepo) v395.AptProxyRepositoryApiRequest {
	return v395.AptProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Apt:             desired.aptProxy(),
		Cleanup:         desired.cleanup(),
	}
}
