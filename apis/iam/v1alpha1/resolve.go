package v1alpha1

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
)

// resolveNames resolves the names in values, the field at path, from the
// target managed resources that refs or selector designate in namespace, by
// their external name.
func resolveNames(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, path string,
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
		Extract:       reference.ExternalName(),
		Namespace:     namespace,
	})
	if err != nil {
		return errors.Wrap(err, path)
	}

	*values = rsp.ResolvedValues
	*refs = rsp.ResolvedReferences

	return nil
}

// resolveName resolves the name in value, the field at path, from the
// target managed resource that ref or selector designates in namespace, by
// its external name.
func resolveName(
	ctx context.Context,
	resolver *reference.APIResolver,
	namespace, path string,
	target reference.To,
	value **string,
	ref **xpv2.Reference,
	selector *xpv2.Selector,
) error {
	rsp, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(*value),
		Reference:    *ref,
		Selector:     selector,
		To:           target,
		Extract:      reference.ExternalName(),
		Namespace:    namespace,
	})
	if err != nil {
		return errors.Wrap(err, path)
	}

	*value = reference.ToPtrValue(rsp.ResolvedValue)
	*ref = rsp.ResolvedReference

	return nil
}
