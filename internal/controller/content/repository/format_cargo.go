package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// init registers the Cargo repository format.
func init() {
	registerFormat("cargo",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.CargoHostedRepository] { return svc.Cargo.Hosted },
			buildCargoHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.CargoProxyRepository] { return svc.Cargo.Proxy },
			buildCargoProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.CargoGroupRepository] { return svc.Cargo.Group },
			buildCargoGroup,
		),
	)
}

// buildCargoHosted renders a Cargo hosted repository payload.
func buildCargoHosted(desired desiredRepo) schema.CargoHostedRepository {
	return schema.CargoHostedRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildCargoProxy renders a Cargo proxy repository payload.
func buildCargoProxy(desired desiredRepo) schema.CargoProxyRepository {
	return schema.CargoProxyRepository{
		Name:          desired.name(),
		Online:        desired.online(),
		Storage:       desired.storage(),
		Proxy:         desired.proxy(),
		NegativeCache: desired.negativeCache(),
		HTTPClient:    desired.httpClient(),
		Cargo:         desired.cargo(),
		RoutingRule:   desired.routingRule(),
		Cleanup:       desired.cleanup(),
	}
}

// buildCargoGroup renders a Cargo group repository payload.
func buildCargoGroup(desired desiredRepo) schema.CargoGroupRepository {
	return schema.CargoGroupRepository{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
		Cargo:   desired.cargo(),
	}
}
