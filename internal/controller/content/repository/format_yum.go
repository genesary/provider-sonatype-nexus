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
	return v395.YumHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
		Yum:     desired.yum(),
	}
}

// buildYumProxy renders a Yum proxy repository payload.
func buildYumProxy(desired desiredRepo) v395.YumProxyRepositoryApiRequest {
	return v395.YumProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
		YumSigning:      desired.yumSigning(),
	}
}

// buildYumGroup renders a Yum group repository payload.
func buildYumGroup(desired desiredRepo) v395.YumGroupRepositoryApiRequest {
	return v395.YumGroupRepositoryApiRequest{
		Name:       desired.name(),
		Online:     desired.online(),
		Storage:    desired.storage(),
		Group:      desired.group(),
		YumSigning: desired.yumSigning(),
	}
}
