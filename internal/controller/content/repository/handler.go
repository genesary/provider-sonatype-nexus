package repository

import (
	"slices"

	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// nexusAPI is the root of the generated Nexus repository API. Format bindings
// pick the service they need out of it.
type nexusAPI = pkgrepository.RepositoryService

// repoAPI is the slice of a generated Nexus repository service the handlers
// need. Every *common.RepositoryService[T] of go-nexus-client satisfies it, so
// a format only has to point at the service it wants.
type repoAPI[T any] interface {
	Get(name string) (*T, error)
	Create(repo T) error
	Update(name string, repo T) error
	Delete(name string) error
}

// apiFunc selects the typed Nexus service serving one format and repository
// type.
type apiFunc[T any] func(svc *pkgrepository.RepositoryService) repoAPI[T]

// buildFunc renders the desired state of a Repository as the payload Nexus
// expects for one format and repository type.
type buildFunc[T any] func(desired desiredRepo) T

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
	Observe(svc *pkgrepository.RepositoryService, name string, desired desiredRepo) (repoState, error)
	// Create creates the repository in Nexus.
	Create(svc *pkgrepository.RepositoryService, desired desiredRepo) error
	// Update reconciles the repository in Nexus with the spec.
	Update(svc *pkgrepository.RepositoryService, name string, desired desiredRepo) error
	// Delete removes the repository from Nexus.
	Delete(svc *pkgrepository.RepositoryService, name string) error
}

// typedOps implements typeOps for a single Nexus repository schema type. It
// holds the only two things that differ between formats: which service serves
// the type, and how the spec turns into that type's payload. Everything else -
// error handling, not-found detection, drift detection - is shared.
type typedOps[T any] struct {
	// api selects the Nexus service serving this format and repository type.
	api apiFunc[T]
	// build renders the spec as this format and repository type's payload.
	build buildFunc[T]
}

// Observe reads the repository from Nexus and compares it with the spec. A
// missing repository is reported as not existing; every other failure is
// returned, so that a transient outage is never mistaken for a deleted
// repository.
func (o typedOps[T]) Observe(svc *pkgrepository.RepositoryService, name string, desired desiredRepo) (repoState, error) {
	observed, err := o.api(svc).Get(name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return repoState{}, nil
		}

		return repoState{}, err
	}

	if observed == nil {
		return repoState{}, nil
	}

	upToDate, fields, err := isUpToDate(o.build(desired), *observed)
	if err != nil {
		return repoState{exists: true}, err
	}

	return repoState{exists: true, upToDate: upToDate, fields: fields}, nil
}

// Create creates the repository in Nexus.
func (o typedOps[T]) Create(svc *pkgrepository.RepositoryService, desired desiredRepo) error {
	return o.api(svc).Create(o.build(desired))
}

// Update reconciles the repository in Nexus with the spec.
func (o typedOps[T]) Update(svc *pkgrepository.RepositoryService, name string, desired desiredRepo) error {
	return o.api(svc).Update(name, o.build(desired))
}

// Delete removes the repository from Nexus.
func (o typedOps[T]) Delete(svc *pkgrepository.RepositoryService, name string) error {
	return o.api(svc).Delete(name)
}

// formatHandler performs the external operations for every repository type a
// Nexus repository format supports.
type formatHandler interface {
	// Observe reports whether the repository exists and matches the spec.
	Observe(svc *pkgrepository.RepositoryService, name, repoType string, desired desiredRepo) (repoState, error)
	// Create creates the repository in Nexus.
	Create(svc *pkgrepository.RepositoryService, repoType string, desired desiredRepo) error
	// Update reconciles the repository in Nexus with the spec.
	Update(svc *pkgrepository.RepositoryService, name, repoType string, desired desiredRepo) error
	// Delete removes the repository from Nexus.
	Delete(svc *pkgrepository.RepositoryService, name, repoType string) error
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
func (d *dispatcher) Observe(svc *pkgrepository.RepositoryService, name, repoType string, desired desiredRepo) (repoState, error) {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return repoState{}, err
	}

	return ops.Observe(svc, name, desired)
}

// Create creates the repository in Nexus.
func (d *dispatcher) Create(svc *pkgrepository.RepositoryService, repoType string, desired desiredRepo) error {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return ops.Create(svc, desired)
}

// Update reconciles the repository in Nexus with the spec.
func (d *dispatcher) Update(svc *pkgrepository.RepositoryService, name, repoType string, desired desiredRepo) error {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return ops.Update(svc, name, desired)
}

// Delete removes the repository from Nexus.
func (d *dispatcher) Delete(svc *pkgrepository.RepositoryService, name, repoType string) error {
	ops, err := d.opsFor(repoType)
	if err != nil {
		return err
	}

	return ops.Delete(svc, name)
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

// hosted binds the hosted repositories of a format to their Nexus service and
// payload builder.
func hosted[T any](api apiFunc[T], build buildFunc[T]) typeBinding {
	return typeBinding{repoType: repoTypeHosted, ops: typedOps[T]{api: api, build: build}}
}

// proxy binds the proxy repositories of a format to their Nexus service and
// payload builder.
func proxy[T any](api apiFunc[T], build buildFunc[T]) typeBinding {
	return typeBinding{repoType: repoTypeProxy, ops: typedOps[T]{api: api, build: build}}
}

// group binds the group repositories of a format to their Nexus service and
// payload builder.
func group[T any](api apiFunc[T], build buildFunc[T]) typeBinding {
	return typeBinding{repoType: repoTypeGroup, ops: typedOps[T]{api: api, build: build}}
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
