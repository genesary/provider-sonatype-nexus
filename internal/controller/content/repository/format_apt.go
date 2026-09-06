package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the APT repository format, which has no group type.
func init() {
	registerFormat("apt",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.AptHostedRepository] { return svc.Apt.Hosted },
			buildAptHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.AptProxyRepository] { return svc.Apt.Proxy },
			buildAptProxy,
		),
	)
}

// buildAptHosted renders an APT hosted repository payload.
func buildAptHosted(desired desiredRepo) schema.AptHostedRepository {
	return schema.AptHostedRepository{
		Name:       desired.name(),
		Online:     desired.online(),
		Storage:    desired.hostedStorage(),
		Apt:        desired.apt(),
		AptSigning: desired.aptSigning(),
		Cleanup:    desired.cleanup(),
	}
}

// buildAptProxy renders an APT proxy repository payload.
func buildAptProxy(desired desiredRepo) schema.AptProxyRepository {
	return schema.AptProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		Apt:           desired.aptProxy(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}
