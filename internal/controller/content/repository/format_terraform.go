package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Terraform repository format.
func init() {
	registerFormat("terraform",
		hosted(
			get[v395.TerraformHostedRepositoryApiRequest]((*nexusAPI).GetTerraformHostedRepository),
			create((*nexusAPI).CreateTerraformHostedRepository, v395.ApiCreateTerraformHostedRepositoryRequest.TerraformHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateTerraformHostedRepository, v395.ApiUpdateTerraformHostedRepositoryRequest.TerraformHostedRepositoryApiRequest),
			buildTerraformHosted,
		),
		proxy(
			get[v395.TerraformProxyApiRepository]((*nexusAPI).GetTerraformProxyRepository),
			create((*nexusAPI).CreateTerraformProxyRepository, v395.ApiCreateTerraformProxyRepositoryRequest.TerraformProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateTerraformProxyRepository, v395.ApiUpdateTerraformProxyRepositoryRequest.TerraformProxyRepositoryApiRequest),
			buildPreemptiveProxy,
		),
		group(
			get[v395.TerraformGroupApiRepository]((*nexusAPI).GetTerraformGroupRepository),
			create((*nexusAPI).CreateTerraformGroupRepository, v395.ApiCreateTerraformGroupRepositoryRequest.TerraformGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateTerraformGroupRepository, v395.ApiUpdateTerraformGroupRepositoryRequest.TerraformGroupRepositoryApiRequest),
			buildTerraformGroup,
		),
	)
}

// buildTerraformHosted renders a Terraform hosted repository payload.
func buildTerraformHosted(desired desiredRepo) v395.TerraformHostedRepositoryApiRequest {
	repo := buildHosted[v395.TerraformHostedRepositoryApiRequest](desired)
	repo.TerraformSigning = desired.terraformSigning()

	return repo
}

// Terraform proxy repositories are registered with the shared builder: the
// payload model has a terraform block, but Nexus ignores it and always reports
// requireAuthentication as false, which would show as drift forever.

// buildTerraformGroup renders a Terraform group repository payload.
func buildTerraformGroup(desired desiredRepo) v395.TerraformGroupRepositoryApiRequest {
	repo := buildGroup[v395.TerraformGroupRepositoryApiRequest](desired)
	repo.Terraform = desired.terraform()

	return repo
}
