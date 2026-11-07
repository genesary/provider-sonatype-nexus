package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Cargo repository format.
func init() {
	registerFormat("cargo",
		hosted(
			get[v395.CargoHostedApiRepository]((*nexusAPI).GetCargoHostedRepository),
			create((*nexusAPI).CreateCargoHostedRepository, v395.ApiCreateCargoHostedRepositoryRequest.CargoHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateCargoHostedRepository, v395.ApiUpdateCargoHostedRepositoryRequest.CargoHostedRepositoryApiRequest),
			buildCargoHosted,
		),
		proxy(
			get[v395.CargoProxyApiRepository]((*nexusAPI).GetCargoProxyRepository),
			create((*nexusAPI).CreateCargoProxyRepository, v395.ApiCreateCargoProxyRepositoryRequest.CargoProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateCargoProxyRepository, v395.ApiUpdateCargoProxyRepositoryRequest.CargoProxyRepositoryApiRequest),
			buildCargoProxy,
		),
		group(
			get[v395.CargoGroupApiRepository]((*nexusAPI).GetCargoGroupRepository),
			create((*nexusAPI).CreateCargoGroupRepository, v395.ApiCreateCargoGroupRepositoryRequest.CargoGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateCargoGroupRepository, v395.ApiUpdateCargoGroupRepositoryRequest.CargoGroupRepositoryApiRequest),
			buildCargoGroup,
		),
	)
}

// buildCargoHosted renders a Cargo hosted repository payload.
func buildCargoHosted(desired desiredRepo) v395.CargoHostedRepositoryApiRequest {
	return v395.CargoHostedRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.hostedStorage(),
		Cleanup: desired.cleanup(),
	}
}

// buildCargoProxy renders a Cargo proxy repository payload.
func buildCargoProxy(desired desiredRepo) v395.CargoProxyRepositoryApiRequest {
	return v395.CargoProxyRepositoryApiRequest{
		Name:            desired.name(),
		Online:          desired.online(),
		Storage:         desired.storage(),
		Proxy:           desired.proxy(),
		NegativeCache:   desired.negativeCache(),
		HttpClient:      desired.httpClient(),
		RoutingRuleName: desired.routingRule(),
		Cargo:           desired.cargo(),
		Cleanup:         desired.cleanup(),
	}
}

// buildCargoGroup renders a Cargo group repository payload.
func buildCargoGroup(desired desiredRepo) v395.CargoGroupRepositoryApiRequest {
	return v395.CargoGroupRepositoryApiRequest{
		Name:    desired.name(),
		Online:  desired.online(),
		Storage: desired.storage(),
		Group:   desired.group(),
		Cargo:   desired.cargo(),
	}
}
