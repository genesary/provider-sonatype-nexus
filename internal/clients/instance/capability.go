package instance

import (
	"context"
	"maps"
	"reflect"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// Capability is the Nexus representation of a capability.
type Capability = v395.CapabilityDTO

// CapabilityClient provides methods for managing Nexus capabilities.
type CapabilityClient interface {
	Create(ctx context.Context, create Capability) (*Capability, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*Capability, error)
	List(ctx context.Context) ([]Capability, error)
	ListTypes(ctx context.Context) ([]v395.CapabilityTypeDTO, error)
	Update(ctx context.Context, id string, update Capability) error
}

// capabilityClient implements CapabilityClient over the Nexus API.
type capabilityClient struct {
	api *v395.CapabilitiesAPIService
}

// NewCapabilityClient returns a CapabilityClient backed by a live Nexus
// connection.
func NewCapabilityClient(creds nexus.Credentials) (CapabilityClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &capabilityClient{api: nc.CapabilitiesAPI}, nil
}

// Create creates a capability and returns it with its server-assigned ID.
func (c *capabilityClient) Create(ctx context.Context, create Capability) (*Capability, error) {
	created, resp, err := c.api.CreateCapabilities(ctx).CapabilityDTO(create).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return created, nil
}

// Delete deletes the capability with the given ID.
func (c *capabilityClient) Delete(ctx context.Context, id string) error {
	return nexus.APIError(c.api.DeleteCapabilities(ctx, id).Execute())
}

// Get returns the capability with the given ID, or an error wrapping
// nexus.ErrNotFound when there is none. Nexus has no endpoint reading a single
// capability, so it is looked up in the full list.
func (c *capabilityClient) Get(ctx context.Context, capabilityID string) (*Capability, error) {
	capabilities, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range capabilities {
		if capabilities[i].GetId() == capabilityID {
			return &capabilities[i], nil
		}
	}

	return nil, errors.Wrapf(nexus.ErrNotFound, "capability %q", capabilityID)
}

// List returns every capability.
func (c *capabilityClient) List(ctx context.Context) ([]Capability, error) {
	capabilities, resp, err := c.api.ListCapabilities(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return capabilities, nil
}

// ListTypes returns every capability type Nexus offers.
func (c *capabilityClient) ListTypes(ctx context.Context) ([]v395.CapabilityTypeDTO, error) {
	types, resp, err := c.api.ListCapabilitiesTypes(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return types, nil
}

// Update updates the capability with the given ID.
func (c *capabilityClient) Update(ctx context.Context, id string, update Capability) error {
	return nexus.APIError(c.api.UpdateCapabilities(ctx, id).CapabilityDTO(update).Execute())
}

// GenerateCapabilityCreate builds a capability creation request from the CR's
// spec.
func GenerateCapabilityCreate(params *instancev1alpha1.CapabilityParameters) Capability {
	return Capability{
		Type:       new(params.TypeId),
		Notes:      new(params.Notes),
		Enabled:    new(params.Enabled),
		Properties: new(cloneProperties(params.Properties)),
	}
}

// GenerateCapabilityUpdate builds a capability update request from the CR's
// spec and the server-assigned ID.
func GenerateCapabilityUpdate(params *instancev1alpha1.CapabilityParameters, id string) Capability {
	update := GenerateCapabilityCreate(params)
	update.Id = new(id)

	return update
}

// IsCapabilityUpToDate returns true when the Nexus capability matches the CR
// spec.
func IsCapabilityUpToDate(cr *instancev1alpha1.Capability, observed *Capability) bool {
	params := cr.Spec.ForProvider

	if params.TypeId != observed.GetType() {
		return false
	}

	if params.Enabled != observed.GetEnabled() {
		return false
	}

	if params.Notes != observed.GetNotes() {
		return false
	}

	desired := params.Properties
	if desired == nil {
		desired = map[string]string{}
	}

	actual := observed.GetProperties()
	if actual == nil {
		actual = map[string]string{}
	}

	return reflect.DeepEqual(desired, actual)
}

// GenerateCapabilityObservation converts a Nexus capability to an observation.
func GenerateCapabilityObservation(observed *Capability) instancev1alpha1.CapabilityObservation {
	if observed == nil {
		return instancev1alpha1.CapabilityObservation{}
	}

	return instancev1alpha1.CapabilityObservation{
		ID:         observed.GetId(),
		TypeId:     observed.GetType(),
		Enabled:    observed.GetEnabled(),
		Notes:      observed.GetNotes(),
		Properties: maps.Clone(observed.GetProperties()),
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
