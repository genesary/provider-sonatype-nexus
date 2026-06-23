package instance

import (
	"maps"
	"reflect"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// CapabilityClient provides methods for managing Nexus capabilities.
type CapabilityClient interface {
	Create(create nexussdk.CapabilityCreate) (*nexussdk.Capability, error)
	Delete(id string) error
	Get(id string) (*nexussdk.Capability, error)
	List() ([]nexussdk.Capability, error)
	ListTypes() ([]nexussdk.TypeDescriptor, error)
	Update(id string, update nexussdk.CapabilityUpdate) error
}

// NewCapabilityClient returns a CapabilityClient backed by a live Nexus
// connection.
func NewCapabilityClient(creds nexus.Credentials) (CapabilityClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Capability, nil
}

// GenerateCapabilityCreate builds a CapabilityCreate from the CR's spec.
func GenerateCapabilityCreate(params *instancev1alpha1.CapabilityParameters) nexussdk.CapabilityCreate {
	return nexussdk.CapabilityCreate{
		Type:       params.TypeId,
		Notes:      params.Notes,
		Enabled:    params.Enabled,
		Properties: cloneProperties(params.Properties),
	}
}

// GenerateCapabilityUpdate builds a CapabilityUpdate from the CR's spec and
// the server-assigned ID.
func GenerateCapabilityUpdate(params *instancev1alpha1.CapabilityParameters, id string) nexussdk.CapabilityUpdate {
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
