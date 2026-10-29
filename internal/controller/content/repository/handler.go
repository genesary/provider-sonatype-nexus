package repository

import (
	"context"
	"net/http"
	"slices"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// nexusAPI is the generated Nexus repository management API. Format bindings
// pick the operations they need out of it.
type nexusAPI = v395.RepositoryManagementAPIService

// getFunc reads one repository of a format and repository type, returning
// what Nexus reported for it.
type getFunc[T any] func(ctx context.Context, api *nexusAPI, name string) (*T, error)

// writeFunc creates or updates one repository of a format and repository type
// from its payload. Creation ignores the name.
type writeFunc[T any] func(ctx context.Context, api *nexusAPI, name string, repo T) error

// buildFunc renders the desired state of a Repository as the payload Nexus
// expects for one format and repository type.
type buildFunc[T any] func(desired desiredRepo) T

// getRequest is a generated request reading a repository.
type getRequest[T any] interface {
	Execute() (*T, *http.Response, error)
}

// writeRequest is a generated request creating or updating a repository.
type writeRequest interface {
	Execute() (*http.Response, error)
}

// get adapts a generated repository getter, such as
// (*nexusAPI).GetMavenHostedRepository, to a getFunc.
func get[T any, R getRequest[T]](start func(*nexusAPI, context.Context, string) R) getFunc[T] {
	return func(ctx context.Context, api *nexusAPI, name string) (*T, error) {
		repo, resp, err := start(api, ctx, name).Execute()

		return repo, nexus.APIError(resp, err)
	}
}

// create adapts a generated repository creation request and its payload
// setter, such as (*nexusAPI).CreateMavenHostedRepository and
// v395.ApiCreateMavenHostedRepositoryRequest.MavenHostedRepositoryApiRequest,
// to a writeFunc.
func create[T any, R writeRequest](start func(*nexusAPI, context.Context) R, body func(R, T) R) writeFunc[T] {
	return func(ctx context.Context, api *nexusAPI, _ string, repo T) error {
		return nexus.APIError(body(start(api, ctx), repo).Execute())
	}
}

// update adapts a generated repository update request and its payload setter
// to a writeFunc.
func update[T any, R writeRequest](start func(*nexusAPI, context.Context, string) R, body func(R, T) R) writeFunc[T] {
	return func(ctx context.Context, api *nexusAPI, name string, repo T) error {
		return nexus.APIError(body(start(api, ctx, name), repo).Execute())
	}
}

// repoState is what an Observe call learned about a repository.
type repoState struct {
	// exists reports whether the repository is present in Nexus.
	exists bool
	// upToDate reports whether the repository already matches the spec.
	upToDate bool
	// fields is the repository exactly as Nexus reported it, rendered as a
	// generic JSON object so that the observed state can be recorded without
	// knowing the concrete format type. It is nil when the repository does
	// not exist.
	fields map[string]any
}

// typeOps performs the external operations for one format and repository type.
type typeOps interface {
	// Observe reports whether the repository exists and matches the spec.
	Observe(ctx context.Context, api *nexusAPI, name string, desired desiredRepo) (repoState, error)
	// Create creates the repository in Nexus.
	Create(ctx context.Context, api *nexusAPI, desired desiredRepo) error
	// Update reconciles the repository in Nexus with the spec.
	Update(ctx context.Context, api *nexusAPI, name string, desired desiredRepo) error
}

// typedOps implements typeOps for a single Nexus repository type. It holds the
// only things that differ between formats: the endpoints serving the type, and
// how the spec turns into that type's payload. Everything else - error
// handling, not-found detection, drift detection - is shared.
//
// Nexus reads and writes a repository through different models, so the
// payload type Req and the reported type Resp differ; drift detection compares
// them as JSON.
type typedOps[Req, Resp any] struct {
	// get reads the repository.
	get getFunc[Resp]
	// create creates the repository.
	create writeFunc[Req]
	// update updates the repository.
	update writeFunc[Req]
	// build renders the spec as this format and repository type's payload.
	build buildFunc[Req]
}

// Observe reads the repository from Nexus and compares it with the spec. A
// missing repository is reported as not existing; every other failure is
// returned, so that a transient outage is never mistaken for a deleted
// repository.
func (o typedOps[Req, Resp]) Observe(ctx context.Context, api *nexusAPI, name string, desired desiredRepo) (repoState, error) {
	observed, err := o.get(ctx, api, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return repoState{}, nil
		}

		return repoState{}, err
	}

	if observed == nil {
		return repoState{}, nil
	}

	upToDate, fields, err := isUpToDate(o.build(desired), observed)
	if err != nil {
		return repoState{exists: true}, err
	}

	return repoState{exists: true, upToDate: upToDate, fields: fields}, nil
}

// Create creates the repository in Nexus.
func (o typedOps[Req, Resp]) Create(ctx context.Context, api *nexusAPI, desired desiredRepo) error {
	return o.create(ctx, api, "", o.build(desired))
}

// Update reconciles the repository in Nexus with the spec.
func (o typedOps[Req, Resp]) Update(ctx context.Context, api *nexusAPI, name string, desired desiredRepo) error {
	return o.update(ctx, api, name, o.build(desired))
}

// formatHandler performs the external operations for every repository type a
// Nexus repository format supports.
type formatHandler interface {
	// Observe reports whether the repository exists and matches the spec.
	Observe(ctx context.Context, api *nexusAPI, name, repoType string, desired desiredRepo) (repoState, error)
	// Create creates the repository in Nexus.
	Create(ctx context.Context, api *nexusAPI, repoType string, desired desiredRepo) error
	// Update reconciles the repository in Nexus with the spec.
	Update(ctx context.Context, api *nexusAPI, name, repoType string, desired desiredRepo) error
	// Delete removes the repository from Nexus.
	Delete(ctx context.Context, api *nexusAPI, name, repoType string) error
	// SupportedTypes lists the repository types the format supports.
	SupportedTypes() []string
}

// dispatcher routes an operation to the typeOps registered for the requested
// repository type. It is the only implementation of formatHandler; a format is
// fully described by the bindings it registers.
type dispatcher struct {
	// format is the Nexus repository format name.
	format string
	// types maps a repository type to the operations implementing it.
	types map[string]typeOps
}

// Observe reports whether the repository exists and matches the spec.
func (d *dispatcher) Observe(ctx context.Context, api *nexusAPI, name, repoType string, desired desiredRepo) (repoState, error) {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return repoState{}, err
	}

	return ops.Observe(ctx, api, name, desired)
}

// Create creates the repository in Nexus.
func (d *dispatcher) Create(ctx context.Context, api *nexusAPI, repoType string, desired desiredRepo) error {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return ops.Create(ctx, api, desired)
}

// Update reconciles the repository in Nexus with the spec.
func (d *dispatcher) Update(ctx context.Context, api *nexusAPI, name, repoType string, desired desiredRepo) error {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return ops.Update(ctx, api, name, desired)
}

// Delete removes the repository from Nexus. Nexus deletes every format and
// repository type through the same endpoint, so the type is only validated.
func (d *dispatcher) Delete(ctx context.Context, api *nexusAPI, name, repoType string) error {
	_, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return nexus.APIError(api.DeleteRepositories(ctx, name).Execute())
}

// SupportedTypes lists the repository types the format supports, sorted so
// that error messages and tests are deterministic.
func (d *dispatcher) SupportedTypes() []string {
	types := make([]string, 0, len(d.types))
	for repoType := range d.types {
		types = append(types, repoType)
	}

	slices.Sort(types)

	return types
}

// opsFor returns the operations registered for repoType.
func (d *dispatcher) opsFor(repoType string) (typeOps, error) {
	ops, supported := d.types[repoType]
	if !supported {
		return nil, errors.Errorf("format %q does not support %q repositories, supported types are %v", d.format, repoType, d.SupportedTypes())
	}

	return ops, nil
}

// typeBinding pairs a repository type with the operations implementing it.
type typeBinding struct {
	// repoType is the Nexus repository type: hosted, proxy or group.
	repoType string
	// ops implements the type for one specific format.
	ops typeOps
}

// hosted binds the hosted repositories of a format to their Nexus endpoints
// and payload builder.
func hosted[Req, Resp any](get getFunc[Resp], create, update writeFunc[Req], build buildFunc[Req]) typeBinding {
	return typeBinding{repoType: repoTypeHosted, ops: typedOps[Req, Resp]{get: get, create: create, update: update, build: build}}
}

// proxy binds the proxy repositories of a format to their Nexus endpoints and
// payload builder.
func proxy[Req, Resp any](get getFunc[Resp], create, update writeFunc[Req], build buildFunc[Req]) typeBinding {
	return typeBinding{repoType: repoTypeProxy, ops: typedOps[Req, Resp]{get: get, create: create, update: update, build: build}}
}

// group binds the group repositories of a format to their Nexus endpoints and
// payload builder.
func group[Req, Resp any](get getFunc[Resp], create, update writeFunc[Req], build buildFunc[Req]) typeBinding {
	return typeBinding{repoType: repoTypeGroup, ops: typedOps[Req, Resp]{get: get, create: create, update: update, build: build}}
}

// handlers holds every registered repository format, keyed by format name. It
// is populated by the registerFormat calls in the format_*.go files.
var handlers = map[string]formatHandler{}

// registerFormat registers the handler for a Nexus repository format. It
// panics on a duplicate registration, which can only be a programming error in
// the format_*.go files.
func registerFormat(format string, bindings ...typeBinding) {
	_, duplicate := handlers[format]
	if duplicate {
		panic("repository format registered twice: " + format)
	}

	types := make(map[string]typeOps, len(bindings))
	for _, binding := range bindings {
		types[binding.repoType] = binding.ops
	}

	handlers[format] = &dispatcher{format: format, types: types}
}

// getHandler returns the handler registered for a Nexus repository format, or
// nil when the format is unknown.
func getHandler(format string) formatHandler {
	handler, registered := handlers[format]
	if !registered {
		return nil
	}

	return handler
}

// supportedFormats lists every registered repository format, sorted.
func supportedFormats() []string {
	formats := make([]string, 0, len(handlers))
	for format := range handlers {
		formats = append(formats, format)
	}

	slices.Sort(formats)

	return formats
}
