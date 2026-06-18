// Package iam provides clients for the Nexus IAM API group.
package iam

import (
	"context"
	"reflect"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SecurityRealmClient defines the interface for realm management.
type SecurityRealmClient interface {
	ListActiveRealms(ctx context.Context) ([]string, error)
	ListAvailableRealms(ctx context.Context) ([]security.Realm, error)
	ActivateRealms(ctx context.Context, ids []string) error
}

// NewSecurityRealmClient creates a SecurityRealmClient from credentials.
func NewSecurityRealmClient(creds nexus.Credentials) (SecurityRealmClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nexusClient.Security(), nil
}

// GenerateSecurityRealmObservation returns the observed realm state.
func GenerateSecurityRealmObservation(availableRealms []security.Realm, activeRealms []string) iamv1alpha1.SecurityRealmObservation {
	obs := iamv1alpha1.SecurityRealmObservation{
		ActiveRealms: activeRealms,
	}

	if availableRealms == nil {
		return obs
	}

	realmInfos := make([]iamv1alpha1.RealmInfo, len(availableRealms))
	for idx, realmItem := range availableRealms {
		realmInfos[idx] = iamv1alpha1.RealmInfo{
			ID:   realmItem.ID,
			Name: realmItem.Name,
		}
	}

	obs.AvailableRealms = realmInfos

	return obs
}

// IsSecurityRealmUpToDate reports whether the CR spec matches observed.
func IsSecurityRealmUpToDate(cr *iamv1alpha1.SecurityRealm) bool {
	return reflect.DeepEqual(cr.Spec.ForProvider.ActiveRealms, cr.Status.AtProvider.ActiveRealms)
}
