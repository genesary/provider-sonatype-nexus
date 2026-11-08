package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// This file renders the blocks every format shares. The generated client
// gives each format and repository type its own payload type, but those types
// agree on the setters of the shared blocks. The builders below are written
// once against those setters, so a format only adds its own settings on top,
// and a format without any is registered with a shared builder directly.
//
// Go cannot infer a type parameter from the parameter of a constraint's
// method, so each variant of a shared block (the HTTP client with pre-emptive
// authentication, the group with a writable member, the storage with a latest
// policy) has a constraint and a builder of its own.

// basePayload holds the setters every repository payload has.
type basePayload interface {
	SetName(v string)
	SetOnline(v bool)
}

// cleanupPayload holds the setter of the cleanup policies, which hosted and
// proxy repositories have.
type cleanupPayload interface {
	SetCleanup(v v395.CleanupPolicyAttributes)
}

// hostedPayload is satisfied by the payload of a hosted repository.
type hostedPayload[T any] interface {
	*T
	basePayload
	cleanupPayload
	SetStorage(v v395.HostedStorageAttributes)
}

// containerHostedPayload is satisfied by the payload of a hosted repository
// whose storage also carries a latest tag policy: Docker and OCI.
type containerHostedPayload[T any] interface {
	*T
	basePayload
	cleanupPayload
	SetStorage(v v395.DockerHostedStorageAttributes)
}

// proxyBasePayload holds the setters of every proxy repository payload, other
// than the HTTP client.
type proxyBasePayload interface {
	basePayload
	cleanupPayload
	SetStorage(v v395.StorageAttributes)
	SetProxy(v v395.ProxyAttributes)
	SetNegativeCache(v v395.NegativeCacheAttributes)
	SetRoutingRuleName(v string)
}

// proxyPayload is satisfied by the payload of a proxy repository.
type proxyPayload[T any] interface {
	*T
	proxyBasePayload
	SetHttpClient(v v395.HttpClientAttributes)
}

// preemptiveProxyPayload is satisfied by the payload of a proxy repository
// whose HTTP client also accepts pre-emptive authentication.
type preemptiveProxyPayload[T any] interface {
	*T
	proxyBasePayload
	SetHttpClient(v v395.HttpClientAttributesWithPreemptiveAuth)
}

// groupBasePayload holds the setters of every group repository payload, other
// than the membership.
type groupBasePayload interface {
	basePayload
	SetStorage(v v395.StorageAttributes)
}

// groupPayload is satisfied by the payload of a group repository.
type groupPayload[T any] interface {
	*T
	groupBasePayload
	SetGroup(v v395.GroupAttributes)
}

// deployGroupPayload is satisfied by the payload of a group repository whose
// format also supports a writable member.
type deployGroupPayload[T any] interface {
	*T
	groupBasePayload
	SetGroup(v v395.GroupDeployAttributes)
}

// buildHosted renders the shared blocks of a hosted repository payload.
func buildHosted[T any, P hostedPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setBase(payload, desired)
	setCleanup(payload, desired)
	payload.SetStorage(desired.hostedStorage())

	return repo
}

// buildContainerHosted renders the shared blocks of a Docker or OCI hosted
// repository payload.
func buildContainerHosted[T any, P containerHostedPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setBase(payload, desired)
	setCleanup(payload, desired)
	payload.SetStorage(desired.containerHostedStorage())

	return repo
}

// buildProxy renders the shared blocks of a proxy repository payload.
func buildProxy[T any, P proxyPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setProxyBase(payload, desired)
	payload.SetHttpClient(desired.httpClient())

	return repo
}

// buildPreemptiveProxy renders the shared blocks of a proxy repository
// payload whose HTTP client accepts pre-emptive authentication.
func buildPreemptiveProxy[T any, P preemptiveProxyPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setProxyBase(payload, desired)
	payload.SetHttpClient(desired.httpClientWithPreemptiveAuth())

	return repo
}

// buildGroup renders the shared blocks of a group repository payload.
func buildGroup[T any, P groupPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setBase(payload, desired)
	payload.SetStorage(desired.storage())
	payload.SetGroup(desired.group())

	return repo
}

// buildDeployGroup renders the shared blocks of a group repository payload
// whose format supports a writable member.
func buildDeployGroup[T any, P deployGroupPayload[T]](desired desiredRepo) T {
	var repo T

	payload := P(&repo)
	setBase(payload, desired)
	payload.SetStorage(desired.storage())
	payload.SetGroup(desired.groupDeploy())

	return repo
}

// setBase sets the name and online state every repository payload has.
func setBase(payload basePayload, desired desiredRepo) {
	payload.SetName(desired.name())
	payload.SetOnline(desired.online())
}

// setCleanup sets the cleanup policies, leaving the block out when the spec
// declares none.
func setCleanup(payload cleanupPayload, desired desiredRepo) {
	cleanup := desired.cleanup()
	if cleanup != nil {
		payload.SetCleanup(*cleanup)
	}
}

// setProxyBase sets every shared block of a proxy repository payload but the
// HTTP client, leaving the routing rule out when the spec assigns none.
func setProxyBase(payload proxyBasePayload, desired desiredRepo) {
	setBase(payload, desired)
	setCleanup(payload, desired)
	payload.SetStorage(desired.storage())
	payload.SetProxy(desired.proxy())
	payload.SetNegativeCache(desired.negativeCache())

	routingRule := desired.routingRule()
	if routingRule != nil {
		payload.SetRoutingRuleName(*routingRule)
	}
}
