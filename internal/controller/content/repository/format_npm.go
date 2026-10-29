package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the npm repository format.
func init() {
	registerFormat("npm",
		hosted(
			get[v395.NpmHostedApiRepository]((*nexusAPI).GetNpmHostedRepository),
			create((*nexusAPI).CreateNpmHostedRepository, v395.ApiCreateNpmHostedRepositoryRequest.NpmHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmHostedRepository, v395.ApiUpdateNpmHostedRepositoryRequest.NpmHostedRepositoryApiRequest),
			buildNpmHosted,
		),
		proxy(
			get[v395.NpmProxyApiRepository]((*nexusAPI).GetNpmProxyRepository),
			create((*nexusAPI).CreateNpmProxyRepository, v395.ApiCreateNpmProxyRepositoryRequest.NpmProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmProxyRepository, v395.ApiUpdateNpmProxyRepositoryRequest.NpmProxyRepositoryApiRequest),
			buildNpmProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetNpmGroupRepository),
			create((*nexusAPI).CreateNpmGroupRepository, v395.ApiCreateNpmGroupRepositoryRequest.NpmGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmGroupRepository, v395.ApiUpdateNpmGroupRepositoryRequest.NpmGroupRepositoryApiRequest),
			buildNpmGroup,
		),
	)
}

// buildNpmHosted renders a npm hosted repository payload.
func buildNpmHosted(desired desiredRepo) v395.NpmHostedRepositoryApiRequest {
	return v395.NpmHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildNpmProxy renders a npm proxy repository payload. The spec's npm block
// is not submitted: Nexus moved those settings to the firewall configuration.
func buildNpmProxy(desired desiredRepo) v395.NpmProxyRepositoryApiRequest {
	return v395.NpmProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// buildNpmGroup renders a npm group repository payload.
func buildNpmGroup(desired desiredRepo) v395.NpmGroupRepositoryApiRequest {
	return v395.NpmGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.groupDeploy(),
	}
}
