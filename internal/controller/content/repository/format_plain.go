package repository

import (
	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// This file holds the formats that support all three repository types and add
// no settings of their own: their payloads are made up entirely of the blocks
// shared by every format. Only the Go type of the payload differs between
// them, which is why the builders repeat the same field list.

// init registers the Raw repository format.
func init() {
	registerFormat("raw",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.RawHostedRepository] { return svc.Raw.Hosted },
			buildRawHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.RawProxyRepository] { return svc.Raw.Proxy },
			buildRawProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.RawGroupRepository] { return svc.Raw.Group },
			buildRawGroup,
		),
	)
}

// buildRawHosted renders a Raw hosted repository payload.
func buildRawHosted(desired desiredRepo) schema.RawHostedRepository {
	return schema.RawHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildRawProxy renders a Raw proxy repository payload.
func buildRawProxy(desired desiredRepo) schema.RawProxyRepository {
	return schema.RawProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// buildRawGroup renders a Raw group repository payload.
func buildRawGroup(desired desiredRepo) schema.RawGroupRepository {
	return schema.RawGroupRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.storage(), Group: desired.group(),
	}
}

// init registers the PyPI repository format.
func init() {
	registerFormat("pypi",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.PypiHostedRepository] { return svc.Pypi.Hosted },
			buildPypiHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.PypiProxyRepository] { return svc.Pypi.Proxy },
			buildPypiProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.PypiGroupRepository] { return svc.Pypi.Group },
			buildPypiGroup,
		),
	)
}

// buildPypiHosted renders a PyPI hosted repository payload.
func buildPypiHosted(desired desiredRepo) schema.PypiHostedRepository {
	return schema.PypiHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildPypiProxy renders a PyPI proxy repository payload.
func buildPypiProxy(desired desiredRepo) schema.PypiProxyRepository {
	return schema.PypiProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// buildPypiGroup renders a PyPI group repository payload.
func buildPypiGroup(desired desiredRepo) schema.PypiGroupRepository {
	return schema.PypiGroupRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.storage(), Group: desired.group(),
	}
}

// init registers the RubyGems repository format.
func init() {
	registerFormat("rubygems",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.RubyGemsHostedRepository] { return svc.RubyGems.Hosted },
			buildRubyGemsHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.RubyGemsProxyRepository] { return svc.RubyGems.Proxy },
			buildRubyGemsProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.RubyGemsGroupRepository] { return svc.RubyGems.Group },
			buildRubyGemsGroup,
		),
	)
}

// buildRubyGemsHosted renders a RubyGems hosted repository payload.
func buildRubyGemsHosted(desired desiredRepo) schema.RubyGemsHostedRepository {
	return schema.RubyGemsHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildRubyGemsProxy renders a RubyGems proxy repository payload.
func buildRubyGemsProxy(desired desiredRepo) schema.RubyGemsProxyRepository {
	return schema.RubyGemsProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// buildRubyGemsGroup renders a RubyGems group repository payload.
func buildRubyGemsGroup(desired desiredRepo) schema.RubyGemsGroupRepository {
	return schema.RubyGemsGroupRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.storage(), Group: desired.group(),
	}
}

// init registers the R repository format.
func init() {
	registerFormat("r",
		hosted(
			func(svc *nexusAPI) repoAPI[schema.RHostedRepository] { return svc.R.Hosted },
			buildRHosted,
		),
		proxy(
			func(svc *nexusAPI) repoAPI[schema.RProxyRepository] { return svc.R.Proxy },
			buildRProxy,
		),
		group(
			func(svc *nexusAPI) repoAPI[schema.RGroupRepository] { return svc.R.Group },
			buildRGroup,
		),
	)
}

// buildRHosted renders an R hosted repository payload.
func buildRHosted(desired desiredRepo) schema.RHostedRepository {
	return schema.RHostedRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.hostedStorage(), Cleanup: desired.cleanup(),
	}
}

// buildRProxy renders an R proxy repository payload.
func buildRProxy(desired desiredRepo) schema.RProxyRepository {
	return schema.RProxyRepository{
		Name: desired.name(), Online: desired.online(), Storage: desired.storage(),
		Proxy: desired.proxy(), NegativeCache: desired.negativeCache(), HTTPClient: desired.httpClient(),
		RoutingRule: desired.routingRule(), Cleanup: desired.cleanup(),
	}
}

// buildRGroup renders an R group repository payload.
func buildRGroup(desired desiredRepo) schema.RGroupRepository {
	return schema.RGroupRepository{
		Name: desired.name(), Online: desired.online(),
		Storage: desired.storage(), Group: desired.group(),
	}
}
