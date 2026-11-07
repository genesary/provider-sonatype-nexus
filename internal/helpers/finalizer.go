package helpers

import (
	"context"
	"reflect"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// statusField is the name of the status field of every managed resource type.
const statusField = "Status"

// statusPreservingFinalizer is a resource.Finalizer that keeps the in-memory
// status of the object it adds its finalizer to.
//
// The managed reconciler adds its finalizer after Observe, with an Update
// that replaces the object with the API server's copy. On the first reconcile
// of a resource that already exists in Nexus, that copy has none of the
// status Observe just recorded: the Ready condition and status.atProvider. The
// reconciler then persists a status without them, and the resource only
// becomes Ready on the next poll, a full poll interval later. This affects
// observe-only resources, resources adopting an existing Nexus object, and
// the singleton settings kinds, whose Nexus object always exists.
type statusPreservingFinalizer struct {
	resource.Finalizer
}

// NewStatusPreservingFinalizer returns the finalizer of the managed
// reconciler, on client, which keeps the status recorded before it is added.
func NewStatusPreservingFinalizer(kube client.Client) resource.Finalizer {
	return statusPreservingFinalizer{Finalizer: resource.NewAPIFinalizer(kube, managed.FinalizerName)}
}

// AddFinalizer adds the finalizer to obj, then restores the status obj had
// before the update replaced it with the API server's copy.
func (f statusPreservingFinalizer) AddFinalizer(ctx context.Context, obj resource.Object) error {
	before := obj.DeepCopyObject()

	err := f.Finalizer.AddFinalizer(ctx, obj)
	if err != nil {
		return errors.Wrap(err, "cannot add the finalizer")
	}

	restoreStatus(obj, before)

	return nil
}

// restoreStatus copies the Status field of before into obj, when both are
// pointers to the same struct type with such a field.
func restoreStatus(obj resource.Object, before any) {
	target := reflect.ValueOf(obj)
	source := reflect.ValueOf(before)

	if !source.IsValid() || target.Kind() != reflect.Pointer || source.Type() != target.Type() || target.Elem().Kind() != reflect.Struct {
		return
	}

	status := target.Elem().FieldByName(statusField)
	if !status.IsValid() || !status.CanSet() {
		return
	}

	status.Set(source.Elem().FieldByName(statusField))
}
