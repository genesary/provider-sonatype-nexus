package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// This file holds the formats that support all three repository types and add
// no settings of their own: their payloads are made up entirely of the blocks
// shared by every format. Only the Go type of the payload differs between
// them, which is why the builders repeat the same field list.

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
	return v395.RawHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildRawProxy renders a raw proxy repository payload.
func buildRawProxy(desired desiredRepo) v395.RawProxyRepositoryApiRequest {
	return v395.RawProxyRepositoryApiRequest{
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

// buildRawGroup renders a raw group repository payload.
func buildRawGroup(desired desiredRepo) v395.RawGroupRepositoryApiRequest {
	return v395.RawGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}

// init registers the PyPI repository format.
func init() {
	registerFormat("pypi",
		hosted(
			get[v395.PypiHostedApiRepository]((*nexusAPI).GetPypiHostedRepository),
			create((*nexusAPI).CreatePypiHostedRepository, v395.ApiCreatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiHostedRepository, v395.ApiUpdatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			buildPypiHosted,
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
			buildPypiGroup,
		),
	)
}

// buildPypiHosted renders a PyPI hosted repository payload.
func buildPypiHosted(desired desiredRepo) v395.PypiHostedRepositoryApiRequest {
	return v395.PypiHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildPypiProxy renders a PyPI proxy repository payload.
func buildPypiProxy(desired desiredRepo) v395.PypiProxyRepositoryApiRequest {
	return v395.PypiProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      new(desired.httpClientWithPreemptiveAuth()),
		RoutingRuleName: desired.routingRule(),
		Cleanup:         desired.cleanup(),
	}
}

// buildPypiGroup renders a PyPI group repository payload.
func buildPypiGroup(desired desiredRepo) v395.PypiGroupRepositoryApiRequest {
	return v395.PypiGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.groupDeploy(),
	}
}

// init registers the RubyGems repository format.
func init() {
	registerFormat("rubygems",
		hosted(
			get[v395.RubyGemsHostedApiRepository]((*nexusAPI).GetRubygemsHostedRepository),
			create((*nexusAPI).CreateRubygemsHostedRepository, v395.ApiCreateRubygemsHostedRepositoryRequest.RubyGemsHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsHostedRepository, v395.ApiUpdateRubygemsHostedRepositoryRequest.RubyGemsHostedRepositoryApiRequest),
			buildRubyGemsHosted,
		),
		proxy(
			get[v395.RubyGemsProxyApiRepository]((*nexusAPI).GetRubygemsProxyRepository),
			create((*nexusAPI).CreateRubygemsProxyRepository, v395.ApiCreateRubygemsProxyRepositoryRequest.RubyGemsProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsProxyRepository, v395.ApiUpdateRubygemsProxyRepositoryRequest.RubyGemsProxyRepositoryApiRequest),
			buildRubyGemsProxy,
		),
		group(
			get[v395.RubyGemsGroupApiRepository]((*nexusAPI).GetRubygemsGroupRepository),
			create((*nexusAPI).CreateRubygemsGroupRepository, v395.ApiCreateRubygemsGroupRepositoryRequest.RubyGemsGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsGroupRepository, v395.ApiUpdateRubygemsGroupRepositoryRequest.RubyGemsGroupRepositoryApiRequest),
			buildRubyGemsGroup,
		),
	)
}

// buildRubyGemsHosted renders a RubyGems hosted repository payload.
func buildRubyGemsHosted(desired desiredRepo) v395.RubyGemsHostedRepositoryApiRequest {
	return v395.RubyGemsHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildRubyGemsProxy renders a RubyGems proxy repository payload.
func buildRubyGemsProxy(desired desiredRepo) v395.RubyGemsProxyRepositoryApiRequest {
	return v395.RubyGemsProxyRepositoryApiRequest{
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

// buildRubyGemsGroup renders a RubyGems group repository payload.
func buildRubyGemsGroup(desired desiredRepo) v395.RubyGemsGroupRepositoryApiRequest {
	return v395.RubyGemsGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}

// init registers the R repository format.
func init() {
	registerFormat("r",
		hosted(
			get[v395.RHostedApiRepository]((*nexusAPI).GetRHostedRepository),
			create((*nexusAPI).CreateRHostedRepository, v395.ApiCreateRHostedRepositoryRequest.RHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRHostedRepository, v395.ApiUpdateRHostedRepositoryRequest.RHostedRepositoryApiRequest),
			buildRHosted,
		),
		proxy(
			get[v395.RProxyApiRepository]((*nexusAPI).GetRProxyRepository),
			create((*nexusAPI).CreateRProxyRepository, v395.ApiCreateRProxyRepositoryRequest.RProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRProxyRepository, v395.ApiUpdateRProxyRepositoryRequest.RProxyRepositoryApiRequest),
			buildRProxy,
		),
		group(
			get[v395.RGroupApiRepository]((*nexusAPI).GetRGroupRepository),
			create((*nexusAPI).CreateRGroupRepository, v395.ApiCreateRGroupRepositoryRequest.RGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRGroupRepository, v395.ApiUpdateRGroupRepositoryRequest.RGroupRepositoryApiRequest),
			buildRGroup,
		),
	)
}

// buildRHosted renders a R hosted repository payload.
func buildRHosted(desired desiredRepo) v395.RHostedRepositoryApiRequest {
	return v395.RHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildRProxy renders a R proxy repository payload.
func buildRProxy(desired desiredRepo) v395.RProxyRepositoryApiRequest {
	return v395.RProxyRepositoryApiRequest{
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

// buildRGroup renders a R group repository payload.
func buildRGroup(desired desiredRepo) v395.RGroupRepositoryApiRequest {
	return v395.RGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
