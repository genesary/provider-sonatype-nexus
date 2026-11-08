// Package crdgate opens the safe-start gate of the managed resource
// controllers once their CustomResourceDefinitions are established.
//
// It replaces crossplane-runtime's customresourcesgate, which caches and
// reconciles every CustomResourceDefinition of the cluster. This one only
// watches the CustomResourceDefinitions of the given kinds.
package crdgate

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/customresourcesgate"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// controllerName is the name of the gate controller.
const controllerName = "crd-gate"

// CRDName returns the name of the CustomResourceDefinition of gvk.
//
// It guesses the plural of the kind the way Kubernetes does, which matches
// the plural controller-gen generates for every kind of this provider. A
// test checks it against the generated CustomResourceDefinitions.
func CRDName(gvk schema.GroupVersionKind) string {
	plural, _ := meta.UnsafeGuessKindToResource(gvk)

	return plural.GroupResource().String()
}

// Setup adds a controller that marks each served version of the
// CustomResourceDefinitions of gvks ready in opts.Gate while the definition
// is established.
//
// A field selector can only match one name, so each definition is watched
// through its own informer holding that single object, with its schemas
// stripped. The informers run as manager caches, which start and sync before
// the controllers.
func Setup(mgr ctrl.Manager, opts controller.Options, gvks []schema.GroupVersionKind) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named(controllerName)
	readers := make(map[string]client.Reader, len(gvks))

	for _, gvk := range gvks {
		name := CRDName(gvk)

		crdCache, err := cache.New(mgr.GetConfig(), cache.Options{
			HTTPClient: mgr.GetHTTPClient(),
			Scheme:     mgr.GetScheme(),
			Mapper:     mgr.GetRESTMapper(),
			ByObject: map[client.Object]cache.ByObject{
				&apiextensionsv1.CustomResourceDefinition{}: {
					Field:     fields.OneTermEqualSelector("metadata.name", name),
					Transform: customresourcesgate.TransformStripCRDSchema,
				},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "cannot create the cache of CustomResourceDefinition %s", name)
		}

		err = mgr.Add(managerCache{Cache: crdCache})
		if err != nil {
			return errors.Wrapf(err, "cannot add the cache of CustomResourceDefinition %s", name)
		}

		readers[name] = crdCache
		builder = builder.WatchesRawSource(source.Kind(crdCache, &apiextensionsv1.CustomResourceDefinition{},
			&handler.TypedEnqueueRequestForObject[*apiextensionsv1.CustomResourceDefinition]{}))
	}

	return builder.Complete(NewReconciler(opts.Logger.WithValues("controller", controllerName), opts.Gate, readers))
}

// managerCache is a cache that the manager starts and waits for along with
// its own, before the controllers, rather than as one of them.
type managerCache struct {
	cache.Cache
}

// GetCache returns the cache.
func (c managerCache) GetCache() cache.Cache {
	return c.Cache
}

// Reconciler sets the gate conditions of CustomResourceDefinitions.
type Reconciler struct {
	log  logging.Logger
	gate controller.Gate
	// readers reads each watched CustomResourceDefinition, by name.
	readers map[string]client.Reader
}

// NewReconciler returns a reconciler that sets gate conditions from the
// CustomResourceDefinitions read through readers, by name.
func NewReconciler(log logging.Logger, gate controller.Gate, readers map[string]client.Reader) *Reconciler {
	return &Reconciler{log: log, gate: gate, readers: readers}
}

// Reconcile marks the served versions of a CustomResourceDefinition ready
// while it is established, and every version not ready otherwise, like
// crossplane-runtime's customresourcesgate.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	reader, watched := r.readers[req.Name]
	if !watched {
		return reconcile.Result{}, nil
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}

	err := reader.Get(ctx, req.NamespacedName, crd)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(client.IgnoreNotFound(err), "cannot get CustomResourceDefinition %s", req.Name)
	}

	ready := isEstablished(crd) && crd.GetDeletionTimestamp().IsZero()

	for _, version := range crd.Spec.Versions {
		gvk := schema.GroupVersionKind{Group: crd.Spec.Group, Version: version.Name, Kind: crd.Spec.Names.Kind}

		switch {
		case !ready:
			r.gate.Set(gvk, false)
		case version.Served:
			r.log.Debug("Kind is ready", "gvk", gvk)
			r.gate.Set(gvk, true)
		}
	}

	return reconcile.Result{}, nil
}

// isEstablished reports whether crd has an Established condition that is
// true.
func isEstablished(crd *apiextensionsv1.CustomResourceDefinition) bool {
	for _, condition := range crd.Status.Conditions {
		if condition.Type == apiextensionsv1.Established {
			return condition.Status == apiextensionsv1.ConditionTrue
		}
	}

	return false
}
