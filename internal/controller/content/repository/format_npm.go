package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the npm repository format.
func init() {
	registerFormat("npm",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.NpmHostedRepository] { return svc.Npm.Hosted },
			buildNpmHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.NpmProxyRepository] { return svc.Npm.Proxy },
			buildNpmProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.NpmGroupRepository] { return svc.Npm.Group },
			buildNpmGroup,
		),
	)
}

// buildNpmHosted renders an npm hosted repository payload.
func buildNpmHosted(desired desiredRepo) schema.NpmHostedRepository {
	return schema.NpmHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildNpmProxy renders an npm proxy repository payload.
func buildNpmProxy(desired desiredRepo) schema.NpmProxyRepository {
	return schema.NpmProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		Npm:           desired.npm(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildNpmGroup renders an npm group repository payload.
func buildNpmGroup(desired desiredRepo) schema.NpmGroupRepository {
	return schema.NpmGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.groupDeploy(),
	}
}
