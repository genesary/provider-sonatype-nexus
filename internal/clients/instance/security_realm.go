// Package instance provides clients for the Nexus instance API group.
package instance

import (
	"reflect"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SecurityRealmClient defines the interface for realm management.
type SecurityRealmClient interface {
	ListActive() ([]string, error)
	ListAvailable() ([]security.Realm, error)
	Activate(ids []string) error
}

// NewSecurityRealmClient creates a SecurityRealmClient from credentials.
func NewSecurityRealmClient(creds nexus.Credentials) (SecurityRealmClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Security.Realm, nil
}

// GenerateSecurityRealmObservation returns the observed realm state.
func GenerateSecurityRealmObservation(availableRealms []security.Realm, activeRealms []string) instancev1alpha1.SecurityRealmObservation {
	obs := instancev1alpha1.SecurityRealmObservation{
		ActiveRealms: activeRealms,
	}

	if availableRealms == nil {
		return obs
	}

	realmInfos := make([]instancev1alpha1.RealmInfo, len(availableRealms))
	for idx, realmItem := range availableRealms {
		realmInfos[idx] = instancev1alpha1.RealmInfo{
			ID:   realmItem.ID,
			Name: realmItem.Name,
		}
	}

	obs.AvailableRealms = realmInfos

	return obs
}

// IsSecurityRealmUpToDate reports whether the CR spec matches observed.
func IsSecurityRealmUpToDate(cr *instancev1alpha1.SecurityRealm) bool {
	return reflect.DeepEqual(cr.Spec.ForProvider.ActiveRealms, cr.Status.AtProvider.ActiveRealms)
}
