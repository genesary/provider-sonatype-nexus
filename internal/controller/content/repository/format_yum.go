package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the Yum repository format.
func init() {
	registerFormat("yum",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.YumHostedRepository] { return svc.Yum.Hosted },
			buildYumHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.YumProxyRepository] { return svc.Yum.Proxy },
			buildYumProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.YumGroupRepository] { return svc.Yum.Group },
			buildYumGroup,
		),
	)
}

// buildYumHosted renders a Yum hosted repository payload.
func buildYumHosted(desired desiredRepo) schema.YumHostedRepository {
	return schema.YumHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Yum:     desired.yum(),
		Cleanup: desired.cleanup(),
	}
}

// buildYumProxy renders a Yum proxy repository payload.
func buildYumProxy(desired desiredRepo) schema.YumProxyRepository {
	return schema.YumProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
		YumSigning:    desired.yumSigning(),
	}
}

// buildYumGroup renders a Yum group repository payload.
func buildYumGroup(desired desiredRepo) schema.YumGroupRepository {
	return schema.YumGroupRepository{
		Name:       desired.name(),
		Online:     desired.online(),
		Storage:    desired.storage(),
		Group:      desired.group(),
		YumSigning: desired.yumSigning(),
	}
}
