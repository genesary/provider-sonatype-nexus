package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// init registers the Hugging Face repository format.
func init() {
	registerFormat("huggingface",
		proxy(
			get[v395.HuggingFaceProxyApiRepository]((*nexusAPI).GetHuggingfaceProxyRepository),
			create((*nexusAPI).CreateHuggingfaceProxyRepository, v395.ApiCreateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateHuggingfaceProxyRepository, v395.ApiUpdateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			buildHuggingFaceProxy,
		),
	)
}

// buildHuggingFaceProxy renders a Hugging Face proxy repository payload.
func buildHuggingFaceProxy(desired desiredRepo) v395.HuggingFaceProxyRepositoryApiRequest {
	repo := buildProxy[v395.HuggingFaceProxyRepositoryApiRequest](desired)
	repo.Xet = desired.xet()

	return repo
}
