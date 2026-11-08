package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the OCI repository format.
func init() {
	registerFormat("oci",
		hosted(
			get[v395.OciHostedApiRepository]((*nexusAPI).GetOciHostedRepository),
			create((*nexusAPI).CreateOciHostedRepository, v395.ApiCreateOciHostedRepositoryRequest.OciHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateOciHostedRepository, v395.ApiUpdateOciHostedRepositoryRequest.OciHostedRepositoryApiRequest),
			buildOciHosted,
		),
		proxy(
			get[v395.OciProxyApiRepository]((*nexusAPI).GetOciProxyRepository),
			create((*nexusAPI).CreateOciProxyRepository, v395.ApiCreateOciProxyRepositoryRequest.OciProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateOciProxyRepository, v395.ApiUpdateOciProxyRepositoryRequest.OciProxyRepositoryApiRequest),
			buildOciProxy,
		),
		group(
			get[v395.OciGroupApiRepository]((*nexusAPI).GetOciGroupRepository),
			create((*nexusAPI).CreateOciGroupRepository, v395.ApiCreateOciGroupRepositoryRequest.OciGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateOciGroupRepository, v395.ApiUpdateOciGroupRepositoryRequest.OciGroupRepositoryApiRequest),
			buildOciGroup,
		),
	)
}

// buildOciHosted renders an OCI hosted repository payload.
func buildOciHosted(desired desiredRepo) v395.OciHostedRepositoryApiRequest {
	repo := buildContainerHosted[v395.OciHostedRepositoryApiRequest](desired)
	repo.Oci = desired.oci()
	repo.Cosign = desired.cosign()

	return repo
}

// buildOciProxy renders an OCI proxy repository payload.
func buildOciProxy(desired desiredRepo) v395.OciProxyRepositoryApiRequest {
	repo := buildProxy[v395.OciProxyRepositoryApiRequest](desired)
	repo.Oci = desired.oci()
	repo.OciProxy = desired.ociProxy()
	repo.Cosign = desired.cosign()

	return repo
}

// buildOciGroup renders an OCI group repository payload.
func buildOciGroup(desired desiredRepo) v395.OciGroupRepositoryApiRequest {
	repo := buildDeployGroup[v395.OciGroupRepositoryApiRequest](desired)
	repo.Oci = desired.oci()
	repo.Cosign = desired.cosign()

	return repo
}
