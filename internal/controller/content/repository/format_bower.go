package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the Bower repository format.
func init() {
	registerFormat("bower",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.BowerHostedRepository] { return svc.Bower.Hosted },
			buildBowerHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.BowerProxyRepository] { return svc.Bower.Proxy },
			buildBowerProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.BowerGroupRepository] { return svc.Bower.Group },
			buildBowerGroup,
		),
	)
}

// buildBowerHosted renders a Bower hosted repository payload.
func buildBowerHosted(desired desiredRepo) schema.BowerHostedRepository {
	return schema.BowerHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildBowerProxy renders a Bower proxy repository payload.
func buildBowerProxy(desired desiredRepo) schema.BowerProxyRepository {
	return schema.BowerProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		Bower:         desired.bower(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildBowerGroup renders a Bower group repository payload.
func buildBowerGroup(desired desiredRepo) schema.BowerGroupRepository {
	return schema.BowerGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
