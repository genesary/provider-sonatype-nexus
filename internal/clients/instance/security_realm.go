// Package instance provides clients for the Nexus instance API group.
package instance

import (
	"context"
	"reflect"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SecurityRealmClient defines the interface for realm management.
type SecurityRealmClient interface {
	ListActive(ctx context.Context) ([]string, error)
	ListAvailable(ctx context.Context) ([]v395.RealmApiXO, error)
	Activate(ctx context.Context, ids []string) error
}

// securityRealmClient implements SecurityRealmClient over the Nexus API.
type securityRealmClient struct {
	api *v395.SecurityManagementRealmsAPIService
}

// NewSecurityRealmClient creates a SecurityRealmClient from credentials.
func NewSecurityRealmClient(creds nexus.Credentials) (SecurityRealmClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &securityRealmClient{api: nc.SecurityManagementRealmsAPI}, nil
}

// ListActive returns the IDs of the active realms, in priority order.
func (c *securityRealmClient) ListActive(ctx context.Context) ([]string, error) {
	realms, resp, err := c.api.ListSecurityRealmsActive(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return realms, nil
}

// ListAvailable returns every realm Nexus offers.
func (c *securityRealmClient) ListAvailable(ctx context.Context) ([]v395.RealmApiXO, error) {
	realms, resp, err := c.api.ListSecurityRealmsAvailable(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return realms, nil
}

// Activate sets the active realms, in priority order.
func (c *securityRealmClient) Activate(ctx context.Context, ids []string) error {
	return nexus.APIError(c.api.UpdateSecurityRealmsActive(ctx).RequestBody(ids).Execute())
}

// GenerateSecurityRealmObservation returns the observed realm state.
func GenerateSecurityRealmObservation(availableRealms []v395.RealmApiXO, activeRealms []string) instancev1alpha1.SecurityRealmObservation {
	obs := instancev1alpha1.SecurityRealmObservation{
		ActiveRealms: activeRealms,
	}

	if availableRealms == nil {
		return obs
	}

	realmInfos := make([]instancev1alpha1.RealmInfo, len(availableRealms))
	for idx, realmItem := range availableRealms {
		realmInfos[idx] = instancev1alpha1.RealmInfo{
			ID:   realmItem.GetId(),
			Name: realmItem.GetName(),
		}
	}

	obs.AvailableRealms = realmInfos

	return obs
}

// IsSecurityRealmUpToDate reports whether the CR spec matches observed.
func IsSecurityRealmUpToDate(cr *instancev1alpha1.SecurityRealm) bool {
	return reflect.DeepEqual(cr.Spec.ForProvider.ActiveRealms, cr.Status.AtProvider.ActiveRealms)
}
