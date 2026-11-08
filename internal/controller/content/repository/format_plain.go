package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// This file holds the formats that add no settings of their own: their
// payloads are made up entirely of the blocks shared by every format, so they
// are registered with the shared builders of payload.go. Nexus offers some of
// them for only some repository types; asking for a type a format does not
// support is rejected by the dispatcher with a message naming the types it
// does support.

// init registers the Ansible Galaxy repository format.
func init() {
	registerFormat("ansiblegalaxy",
		hosted(
			get[v395.AnsibleGalaxyHostedApiRepository]((*nexusAPI).GetAnsiblegalaxyHostedRepository),
			create((*nexusAPI).CreateAnsiblegalaxyHostedRepository, v395.ApiCreateAnsiblegalaxyHostedRepositoryRequest.AnsibleGalaxyHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateAnsiblegalaxyHostedRepository, v395.ApiUpdateAnsiblegalaxyHostedRepositoryRequest.AnsibleGalaxyHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.AnsibleGalaxyProxyApiRepository]((*nexusAPI).GetAnsiblegalaxyProxyRepository),
			create((*nexusAPI).CreateAnsiblegalaxyProxyRepository, v395.ApiCreateAnsiblegalaxyProxyRepositoryRequest.AnsibleGalaxyProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateAnsiblegalaxyProxyRepository, v395.ApiUpdateAnsiblegalaxyProxyRepositoryRequest.AnsibleGalaxyProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.AnsibleGalaxyGroupApiRepository]((*nexusAPI).GetAnsiblegalaxyGroupRepository),
			create((*nexusAPI).CreateAnsiblegalaxyGroupRepository, v395.ApiCreateAnsiblegalaxyGroupRepositoryRequest.AnsibleGalaxyGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateAnsiblegalaxyGroupRepository, v395.ApiUpdateAnsiblegalaxyGroupRepositoryRequest.AnsibleGalaxyGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the CocoaPods repository format.
func init() {
	registerFormat("cocoapods",
		proxy(
			get[v395.CocoapodsProxyApiRepository]((*nexusAPI).GetCocoapodsProxyRepository),
			create((*nexusAPI).CreateCocoapodsProxyRepository, v395.ApiCreateCocoapodsProxyRepositoryRequest.CocoapodsProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateCocoapodsProxyRepository, v395.ApiUpdateCocoapodsProxyRepositoryRequest.CocoapodsProxyRepositoryApiRequest),
			buildProxy,
		),
	)
}

// init registers the Composer repository format.
func init() {
	registerFormat("composer",
		hosted(
			get[v395.ComposerHostedApiRepository]((*nexusAPI).GetComposerHostedRepository),
			create((*nexusAPI).CreateComposerHostedRepository, v395.ApiCreateComposerHostedRepositoryRequest.ComposerHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateComposerHostedRepository, v395.ApiUpdateComposerHostedRepositoryRequest.ComposerHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.ComposerProxyApiRepository]((*nexusAPI).GetComposerProxyRepository),
			create((*nexusAPI).CreateComposerProxyRepository, v395.ApiCreateComposerProxyRepositoryRequest.ComposerProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateComposerProxyRepository, v395.ApiUpdateComposerProxyRepositoryRequest.ComposerProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.ComposerGroupApiRepository]((*nexusAPI).GetComposerGroupRepository),
			create((*nexusAPI).CreateComposerGroupRepository, v395.ApiCreateComposerGroupRepositoryRequest.ComposerGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateComposerGroupRepository, v395.ApiUpdateComposerGroupRepositoryRequest.ComposerGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the Conan repository format.
func init() {
	registerFormat("conan",
		hosted(
			get[v395.ConanHostedApiRepository]((*nexusAPI).GetConanHostedRepository),
			create((*nexusAPI).CreateConanHostedRepository, v395.ApiCreateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateConanHostedRepository, v395.ApiUpdateConanHostedRepositoryRequest.ConanHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.ConanProxyApiRepository]((*nexusAPI).GetConanProxyRepository),
			create((*nexusAPI).CreateConanProxyRepository, v395.ApiCreateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateConanProxyRepository, v395.ApiUpdateConanProxyRepositoryRequest.ConanProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetConanGroupRepository),
			create((*nexusAPI).CreateConanGroupRepository, v395.ApiCreateConanGroupRepositoryRequest.ConanGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateConanGroupRepository, v395.ApiUpdateConanGroupRepositoryRequest.ConanGroupRepositoryApiRequest),
			buildDeployGroup,
		),
	)
}

// init registers the Conda repository format.
func init() {
	registerFormat("conda",
		hosted(
			get[v395.CondaHostedApiRepository]((*nexusAPI).GetCondaHostedRepository),
			create((*nexusAPI).CreateCondaHostedRepository, v395.ApiCreateCondaHostedRepositoryRequest.CondaHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateCondaHostedRepository, v395.ApiUpdateCondaHostedRepositoryRequest.CondaHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.CondaProxyApiRepository]((*nexusAPI).GetCondaProxyRepository),
			create((*nexusAPI).CreateCondaProxyRepository, v395.ApiCreateCondaProxyRepositoryRequest.CondaProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateCondaProxyRepository, v395.ApiUpdateCondaProxyRepositoryRequest.CondaProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.SimpleApiGroupRepository]((*nexusAPI).GetCondaGroupRepository),
			create((*nexusAPI).CreateCondaGroupRepository, v395.ApiCreateCondaGroupRepositoryRequest.CondaGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateCondaGroupRepository, v395.ApiUpdateCondaGroupRepositoryRequest.CondaGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the Git LFS repository format.
func init() {
	registerFormat("gitlfs",
		hosted(
			get[v395.GitLfsHostedApiRepository]((*nexusAPI).GetGitlfsHostedRepository),
			create((*nexusAPI).CreateGitlfsHostedRepository, v395.ApiCreateGitlfsHostedRepositoryRequest.GitLfsHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateGitlfsHostedRepository, v395.ApiUpdateGitlfsHostedRepositoryRequest.GitLfsHostedRepositoryApiRequest),
			buildHosted,
		),
	)
}

// init registers the Go repository format.
func init() {
	registerFormat("go",
		hosted(
			get[v395.GolangHostedApiRepository]((*nexusAPI).GetGoHostedRepository),
			create((*nexusAPI).CreateGoHostedRepository, v395.ApiCreateGoHostedRepositoryRequest.GolangHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateGoHostedRepository, v395.ApiUpdateGoHostedRepositoryRequest.GolangHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.GolangProxyApiRepository]((*nexusAPI).GetGoProxyRepository),
			create((*nexusAPI).CreateGoProxyRepository, v395.ApiCreateGoProxyRepositoryRequest.GolangProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateGoProxyRepository, v395.ApiUpdateGoProxyRepositoryRequest.GolangProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.GolangGroupApiRepository]((*nexusAPI).GetGoGroupRepository),
			create((*nexusAPI).CreateGoGroupRepository, v395.ApiCreateGoGroupRepositoryRequest.GolangGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateGoGroupRepository, v395.ApiUpdateGoGroupRepositoryRequest.GolangGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the Helm repository format.
func init() {
	registerFormat("helm",
		hosted(
			get[v395.HelmHostedApiRepository]((*nexusAPI).GetHelmHostedRepository),
			create((*nexusAPI).CreateHelmHostedRepository, v395.ApiCreateHelmHostedRepositoryRequest.HelmHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateHelmHostedRepository, v395.ApiUpdateHelmHostedRepositoryRequest.HelmHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.HelmProxyApiRepository]((*nexusAPI).GetHelmProxyRepository),
			create((*nexusAPI).CreateHelmProxyRepository, v395.ApiCreateHelmProxyRepositoryRequest.HelmProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateHelmProxyRepository, v395.ApiUpdateHelmProxyRepositoryRequest.HelmProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.HelmGroupApiRepository]((*nexusAPI).GetHelmGroupRepository),
			create((*nexusAPI).CreateHelmGroupRepository, v395.ApiCreateHelmGroupRepositoryRequest.HelmGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateHelmGroupRepository, v395.ApiUpdateHelmGroupRepositoryRequest.HelmGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the Hugging Face repository format.
func init() {
	registerFormat("huggingface",
		proxy(
			get[v395.HuggingFaceProxyApiRepository]((*nexusAPI).GetHuggingfaceProxyRepository),
			create((*nexusAPI).CreateHuggingfaceProxyRepository, v395.ApiCreateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateHuggingfaceProxyRepository, v395.ApiUpdateHuggingfaceProxyRepositoryRequest.HuggingFaceProxyRepositoryApiRequest),
			buildProxy,
		),
	)
}

// init registers the npm repository format.
func init() {
	registerFormat("npm",
		hosted(
			get[v395.NpmHostedApiRepository]((*nexusAPI).GetNpmHostedRepository),
			create((*nexusAPI).CreateNpmHostedRepository, v395.ApiCreateNpmHostedRepositoryRequest.NpmHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmHostedRepository, v395.ApiUpdateNpmHostedRepositoryRequest.NpmHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.NpmProxyApiRepository]((*nexusAPI).GetNpmProxyRepository),
			create((*nexusAPI).CreateNpmProxyRepository, v395.ApiCreateNpmProxyRepositoryRequest.NpmProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmProxyRepository, v395.ApiUpdateNpmProxyRepositoryRequest.NpmProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetNpmGroupRepository),
			create((*nexusAPI).CreateNpmGroupRepository, v395.ApiCreateNpmGroupRepositoryRequest.NpmGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateNpmGroupRepository, v395.ApiUpdateNpmGroupRepositoryRequest.NpmGroupRepositoryApiRequest),
			buildDeployGroup,
		),
	)
}

// init registers the P2 repository format.
func init() {
	registerFormat("p2",
		proxy(
			get[v395.P2ProxyApiRepository]((*nexusAPI).GetP2ProxyRepository),
			create((*nexusAPI).CreateP2ProxyRepository, v395.ApiCreateP2ProxyRepositoryRequest.P2ProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateP2ProxyRepository, v395.ApiUpdateP2ProxyRepositoryRequest.P2ProxyRepositoryApiRequest),
			buildProxy,
		),
	)
}

// init registers the Pub repository format.
func init() {
	registerFormat("pub",
		hosted(
			get[v395.PubHostedApiRepository]((*nexusAPI).GetPubHostedRepository),
			create((*nexusAPI).CreatePubHostedRepository, v395.ApiCreatePubHostedRepositoryRequest.PubHostedRepositoryApiRequest),
			update((*nexusAPI).UpdatePubHostedRepository, v395.ApiUpdatePubHostedRepositoryRequest.PubHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.PubProxyApiRepository]((*nexusAPI).GetPubProxyRepository),
			create((*nexusAPI).CreatePubProxyRepository, v395.ApiCreatePubProxyRepositoryRequest.PubProxyRepositoryApiRequest),
			update((*nexusAPI).UpdatePubProxyRepository, v395.ApiUpdatePubProxyRepositoryRequest.PubProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.PubGroupApiRepository]((*nexusAPI).GetPubGroupRepository),
			create((*nexusAPI).CreatePubGroupRepository, v395.ApiCreatePubGroupRepositoryRequest.PubGroupRepositoryApiRequest),
			update((*nexusAPI).UpdatePubGroupRepository, v395.ApiUpdatePubGroupRepositoryRequest.PubGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the PyPI repository format.
func init() {
	registerFormat("pypi",
		hosted(
			get[v395.PypiHostedApiRepository]((*nexusAPI).GetPypiHostedRepository),
			create((*nexusAPI).CreatePypiHostedRepository, v395.ApiCreatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiHostedRepository, v395.ApiUpdatePypiHostedRepositoryRequest.PypiHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.PyPiProxyApiRepository]((*nexusAPI).GetPypiProxyRepository),
			create((*nexusAPI).CreatePypiProxyRepository, v395.ApiCreatePypiProxyRepositoryRequest.PypiProxyRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiProxyRepository, v395.ApiUpdatePypiProxyRepositoryRequest.PypiProxyRepositoryApiRequest),
			buildPreemptiveProxy,
		),
		group(
			get[v395.SimpleApiGroupDeployRepository]((*nexusAPI).GetPypiGroupRepository),
			create((*nexusAPI).CreatePypiGroupRepository, v395.ApiCreatePypiGroupRepositoryRequest.PypiGroupRepositoryApiRequest),
			update((*nexusAPI).UpdatePypiGroupRepository, v395.ApiUpdatePypiGroupRepositoryRequest.PypiGroupRepositoryApiRequest),
			buildDeployGroup,
		),
	)
}

// init registers the R repository format.
func init() {
	registerFormat("r",
		hosted(
			get[v395.RHostedApiRepository]((*nexusAPI).GetRHostedRepository),
			create((*nexusAPI).CreateRHostedRepository, v395.ApiCreateRHostedRepositoryRequest.RHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRHostedRepository, v395.ApiUpdateRHostedRepositoryRequest.RHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.RProxyApiRepository]((*nexusAPI).GetRProxyRepository),
			create((*nexusAPI).CreateRProxyRepository, v395.ApiCreateRProxyRepositoryRequest.RProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRProxyRepository, v395.ApiUpdateRProxyRepositoryRequest.RProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.RGroupApiRepository]((*nexusAPI).GetRGroupRepository),
			create((*nexusAPI).CreateRGroupRepository, v395.ApiCreateRGroupRepositoryRequest.RGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRGroupRepository, v395.ApiUpdateRGroupRepositoryRequest.RGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the raw repository format.
func init() {
	registerFormat("raw",
		hosted(
			get[v395.RawHostedApiRepository]((*nexusAPI).GetRawHostedRepository),
			create((*nexusAPI).CreateRawHostedRepository, v395.ApiCreateRawHostedRepositoryRequest.RawHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRawHostedRepository, v395.ApiUpdateRawHostedRepositoryRequest.RawHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.RawProxyApiRepository]((*nexusAPI).GetRawProxyRepository),
			create((*nexusAPI).CreateRawProxyRepository, v395.ApiCreateRawProxyRepositoryRequest.RawProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRawProxyRepository, v395.ApiUpdateRawProxyRepositoryRequest.RawProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.RawGroupApiRepository]((*nexusAPI).GetRawGroupRepository),
			create((*nexusAPI).CreateRawGroupRepository, v395.ApiCreateRawGroupRepositoryRequest.RawGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRawGroupRepository, v395.ApiUpdateRawGroupRepositoryRequest.RawGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}

// init registers the RubyGems repository format.
func init() {
	registerFormat("rubygems",
		hosted(
			get[v395.RubyGemsHostedApiRepository]((*nexusAPI).GetRubygemsHostedRepository),
			create((*nexusAPI).CreateRubygemsHostedRepository, v395.ApiCreateRubygemsHostedRepositoryRequest.RubyGemsHostedRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsHostedRepository, v395.ApiUpdateRubygemsHostedRepositoryRequest.RubyGemsHostedRepositoryApiRequest),
			buildHosted,
		),
		proxy(
			get[v395.RubyGemsProxyApiRepository]((*nexusAPI).GetRubygemsProxyRepository),
			create((*nexusAPI).CreateRubygemsProxyRepository, v395.ApiCreateRubygemsProxyRepositoryRequest.RubyGemsProxyRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsProxyRepository, v395.ApiUpdateRubygemsProxyRepositoryRequest.RubyGemsProxyRepositoryApiRequest),
			buildProxy,
		),
		group(
			get[v395.RubyGemsGroupApiRepository]((*nexusAPI).GetRubygemsGroupRepository),
			create((*nexusAPI).CreateRubygemsGroupRepository, v395.ApiCreateRubygemsGroupRepositoryRequest.RubyGemsGroupRepositoryApiRequest),
			update((*nexusAPI).UpdateRubygemsGroupRepository, v395.ApiUpdateRubygemsGroupRepositoryRequest.RubyGemsGroupRepositoryApiRequest),
			buildGroup,
		),
	)
}
