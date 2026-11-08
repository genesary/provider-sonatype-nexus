// Package resolve resolves the fields of managed resources that name other
// Nexus objects from the managed resources that manage those objects.
package resolve

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// selfValue is what the referencing resource extracts to when a selector or a
// reference designates it. It is never a Nexus name, so it cannot collide with
// a target's, and it is dropped from the resolved values.
const selfValue = "\x00self"

// readyExternalName extracts the external name of a target managed resource
// once it is Ready, meaning its controller has observed it in Nexus, and
// nothing before. The resolver treats an empty value as a target that is not
// ready yet: a required reference fails, so the referencing resource is
// retried rather than created against an object Nexus does not have yet.
//
// The managed resource named self, if any, extracts to selfValue instead, so
// that a selector matching the referencing resource does not wait on a
// resource that cannot become Ready before it resolves.
func readyExternalName(self string) reference.ExtractValueFn {
	return func(target resource.Managed) string {
		if self != "" && target.GetName() == self {
			return selfValue
		}

		if target.GetCondition(xpv2.TypeReady).Status != corev1.ConditionTrue {
			return ""
		}

		return meta.GetExternalName(target)
	}
}

// Names resolves the names in values, the field at path, from the target
// managed resources that refs or selector designate in namespace, by their
// external name once they are Ready. A managed resource named self is left
// out, so that a selector matching the referencing resource does not make it
// contain itself; pass an empty self when the target is of another kind.
// Targets an Optional policy skipped because they are not Ready are left out.
func Names(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, self, path string,
	target reference.To,
	values *[]string,
	refs *[]xpv2.Reference,
	selector *xpv2.Selector,
) error {
	rsp, err := resolver.ResolveMultiple(ctx, reference.MultiResolutionRequest{
		CurrentValues: *values,
		References:    *refs,
		Selector:      selector,
		To:            target,
		Extract:       readyExternalName(self),
		Namespace:     namespace,
	})
	if err != nil {
		return errors.Wrap(err, path)
	}

	*values, *refs = withoutUnresolved(rsp.ResolvedValues, rsp.ResolvedReferences)

	return nil
}

// withoutUnresolved returns values and refs, which the resolver pairs up by
// index, without the entries that resolved to the referencing resource itself
// or to nothing.
func withoutUnresolved(values []string, refs []xpv2.Reference) ([]string, []xpv2.Reference) {
	if len(values) != len(refs) {
		return values, refs
	}

	keptValues := make([]string, 0, len(values))
	keptRefs := make([]xpv2.Reference, 0, len(refs))

	for idx, value := range values {
		if value == "" || value == selfValue {
			continue
		}

		keptValues = append(keptValues, value)
		keptRefs = append(keptRefs, refs[idx])
	}

	return keptValues, keptRefs
}

// Name resolves the name in value, the field at path, from the target managed
// resource that ref or selector designates in namespace, by its external name
// once it is Ready. A target an Optional policy skipped because it is not
// Ready leaves value empty.
func Name(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, path string,
	target reference.To,
	value *string,
	ref **xpv2.Reference,
	selector *xpv2.Selector,
) error {
	rsp, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: *value,
		Reference:    *ref,
		Selector:     selector,
		To:           target,
		Extract:      readyExternalName(""),
		Namespace:    namespace,
	})
	if err != nil {
		return errors.Wrap(err, path)
	}

	*value = rsp.ResolvedValue
	*ref = rsp.ResolvedReference

	return nil
}

// OptionalName is Name for a field that is a pointer, and is nil when empty.
func OptionalName(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, path string,
	target reference.To,
	value **string,
	ref **xpv2.Reference,
	selector *xpv2.Selector,
) error {
	resolved := reference.FromPtrValue(*value)

	err := Name(ctx, resolver, namespace, path, target, &resolved, ref, selector)
	if err != nil {
		return err
	}

	*value = reference.ToPtrValue(resolved)

	return nil
}
