package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the Maven repository format.
func init() {
	registerFormat("maven2",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.MavenHostedRepository] { return svc.Maven.Hosted },
			buildMavenHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.MavenProxyRepository] { return svc.Maven.Proxy },
			buildMavenProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.MavenGroupRepository] { return svc.Maven.Group },
			buildMavenGroup,
		),
	)
}

// buildMavenHosted renders a Maven hosted repository payload.
func buildMavenHosted(desired desiredRepo) schema.MavenHostedRepository {
	return schema.MavenHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Maven:   desired.maven(),
		Cleanup: desired.cleanup(),
	}
}

// buildMavenProxy renders a Maven proxy repository payload.
func buildMavenProxy(desired desiredRepo) schema.MavenProxyRepository {
	return schema.MavenProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Maven:         desired.maven(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClientWithPreemptiveAuth(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildMavenGroup renders a Maven group repository payload. The Maven block is
// left out on purpose: Nexus does not return it for group repositories.
func buildMavenGroup(desired desiredRepo) schema.MavenGroupRepository {
	return schema.MavenGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
	}
}
