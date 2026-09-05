package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the NuGet repository format.
func init() {
	registerFormat("nuget",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.NugetHostedRepository] { return svc.Nuget.Hosted },
			buildNugetHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.NugetProxyRepository] { return svc.Nuget.Proxy },
			buildNugetProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.NugetGroupRepository] { return svc.Nuget.Group },
			buildNugetGroup,
		),
	)
}

// buildNugetHosted renders a NuGet hosted repository payload.
func buildNugetHosted(desired desiredRepo) schema.NugetHostedRepository {
	return schema.NugetHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildNugetProxy renders a NuGet proxy repository payload.
func buildNugetProxy(desired desiredRepo) schema.NugetProxyRepository {
	return schema.NugetProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		NugetProxy:    desired.nugetProxy(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildNugetGroup renders a NuGet group repository payload.
func buildNugetGroup(desired desiredRepo) schema.NugetGroupRepository {
	return schema.NugetGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
