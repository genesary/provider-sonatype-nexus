package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Swift repository format.
func init() {
	registerFormat("swift",
		hosted(
			get[v395.SwiftHostedApiRepository]((*nexusAPI).GetSwiftHostedRepository),
			create((*nexusAPI).CreateSwiftHostedRepository, v395.ApiCreateSwiftHostedRepositoryRequest.SwiftHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateSwiftHostedRepository, v395.ApiUpdateSwiftHostedRepositoryRequest.SwiftHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.SwiftProxyApiRepository]((*nexusAPI).GetSwiftProxyRepository),
			create((*nexusAPI).CreateSwiftProxyRepository, v395.ApiCreateSwiftProxyRepositoryRequest.SwiftProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateSwiftProxyRepository, v395.ApiUpdateSwiftProxyRepositoryRequest.SwiftProxyRepositoryApiRequest),
			buildSwiftProxy,
		),
		group(
			get[v395.SwiftGroupApiRepository]((*nexusAPI).GetSwiftGroupRepository),
			create((*nexusAPI).CreateSwiftGroupRepository, v395.ApiCreateSwiftGroupRepositoryRequest.SwiftGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateSwiftGroupRepository, v395.ApiUpdateSwiftGroupRepositoryRequest.SwiftGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// buildSwiftProxy renders a Swift proxy repository payload.
func buildSwiftProxy(desired desiredRepo) v395.SwiftProxyRepositoryApiRequest {
	repo := buildProxy[v395.SwiftProxyRepositoryApiRequest](desired)
	repo.Swift = desired.swift()

	return repo
}
