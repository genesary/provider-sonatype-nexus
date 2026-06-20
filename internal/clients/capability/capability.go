// Package capability provides helpers for managing Nexus Capability resources.
package capability

import (
	"context"
	"maps"
	"reflect"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// CapabilityClient provides methods for managing Nexus capabilities.
type CapabilityClient interface {
	GetCapability(ctx context.Context, id string) (*nexussdk.Capability, error)
	CreateCapability(ctx context.Context, create nexussdk.CapabilityCreate) (*nexussdk.Capability, error)
	UpdateCapability(ctx context.Context, id string, update nexussdk.CapabilityUpdate) error
	DeleteCapability(ctx context.Context, id string) error
}

// NewCapabilityClient returns a CapabilityClient backed by a live Nexus
// connection.
func NewCapabilityClient(creds nexus.Credentials) (CapabilityClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Capability(), nil
}

// GenerateCapabilityCreate builds a CapabilityCreate from the CR's spec.
func GenerateCapabilityCreate(cr *instancev1alpha1.Capability) nexussdk.CapabilityCreate {
	params := cr.Spec.ForProvider

	return nexussdk.CapabilityCreate{
		Type:       params.TypeId,
		Notes:      params.Notes,
		Enabled:    params.Enabled,
		Properties: cloneProperties(params.Properties),
	}
}

// GenerateCapabilityUpdate builds a CapabilityUpdate from the CR's spec and
// the server-assigned ID.
func GenerateCapabilityUpdate(cr *instancev1alpha1.Capability, id string) nexussdk.CapabilityUpdate {
	params := cr.Spec.ForProvider

	return nexussdk.CapabilityUpdate{
		ID:         id,
		Type:       params.TypeId,
		Notes:      params.Notes,
		Enabled:    params.Enabled,
		Properties: cloneProperties(params.Properties),
	}
}

// IsCapabilityUpToDate returns true when the Nexus capability matches the CR
// spec.
func IsCapabilityUpToDate(cr *instancev1alpha1.Capability, observed *nexussdk.Capability) bool {
	params := cr.Spec.ForProvider

	if params.TypeId != observed.Type {
		return false
	}

	if params.Enabled != observed.Enabled {
		return false
	}

	if params.Notes != observed.Notes {
		return false
	}

	desired := params.Properties
	if desired == nil {
		desired = map[string]string{}
	}

	actual := observed.Properties
	if actual == nil {
		actual = map[string]string{}
	}

	return reflect.DeepEqual(desired, actual)
}

// GenerateCapabilityObservation converts a Nexus capability to an observation.
func GenerateCapabilityObservation(observed *nexussdk.Capability) instancev1alpha1.CapabilityObservation {
	if observed == nil {
		return instancev1alpha1.CapabilityObservation{}
	}

	return instancev1alpha1.CapabilityObservation{
		ID: observed.ID,
	}
}

// cloneProperties returns a copy of src, or an empty map if src is nil.
func cloneProperties(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}

	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)

	return dst
}
