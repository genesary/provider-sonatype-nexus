package helpers

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// errUpdateExternalName is returned when persisting the external name
	// fails.
	errUpdateExternalName = "cannot update managed resource external name"
	// errUnexpectedKind is returned when the initializer is handed a managed
	// resource of another kind.
	errUnexpectedKind = "unexpected managed resource kind"
)

// WithSpecExternalName replaces the reconciler's default initializer, which
// sets the external name from metadata.name, with one that sets it from the
// Nexus identifier the spec names. metadata.name need not match that
// identifier, so the default could make the controller observe, adopt and
// later delete an unrelated Nexus object that happens to carry the CR's name.
//
// An external name that is already set is kept, so an explicit
// crossplane.io/external-name annotation still wins.
func WithSpecExternalName[T resource.Managed](kube client.Client, name func(T) string) managed.ReconcilerOption {
	return managed.WithInitializers(SpecExternalName(kube, name))
}

// WithServerAssignedExternalName removes the reconciler's default initializer
// for kinds whose external name is an identifier Nexus assigns on creation.
// The external name then stays empty until Create records that identifier,
// instead of holding metadata.name, which names no Nexus object.
//
// It also declares the external name non-deterministic, overriding the default
// set by ReconcilerOptions: a create interrupted before its identifier was
// recorded cannot be found again, and retrying it could create a duplicate, so
// the reconciler must stop and ask for intervention instead.
func WithServerAssignedExternalName() managed.ReconcilerOption {
	return func(reconciler *managed.Reconciler) {
		managed.WithInitializers()(reconciler)
		managed.WithDeterministicExternalName(false)(reconciler)
	}
}

// SpecExternalName returns an initializer that sets the external name of a
// managed resource without one to name(cr), and persists it.
func SpecExternalName[T resource.Managed](kube client.Client, name func(T) string) managed.Initializer {
	return managed.InitializerFn(func(ctx context.Context, managedResource resource.Managed) error {
		if meta.GetExternalName(managedResource) != "" {
			return nil
		}

		cr, ok := managedResource.(T)
		if !ok {
			return errors.Errorf("%s %T", errUnexpectedKind, managedResource)
		}

		externalName := name(cr)
		if externalName == "" {
			return nil
		}

		meta.SetExternalName(managedResource, externalName)

		return errors.Wrap(kube.Update(ctx, managedResource), errUpdateExternalName)
	})
}
