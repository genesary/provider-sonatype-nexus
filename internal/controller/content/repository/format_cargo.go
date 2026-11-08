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
			buildHosted,
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

// buildCargoProxy renders a Cargo proxy repository payload.
func buildCargoProxy(desired desiredRepo) v395.CargoProxyRepositoryApiRequest {
	repo := buildProxy[v395.CargoProxyRepositoryApiRequest](desired)
	repo.Cargo = desired.cargo()

	return repo
}

// buildCargoGroup renders a Cargo group repository payload.
func buildCargoGroup(desired desiredRepo) v395.CargoGroupRepositoryApiRequest {
	repo := buildGroup[v395.CargoGroupRepositoryApiRequest](desired)
	repo.Cargo = desired.cargo()

	return repo
}
