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

// IsSecurityRealmUpToDate reports whether the CR matches observed realms.
func IsSecurityRealmUpToDate(cr *iamv1alpha1.SecurityRealm, activeRealms []string) bool {
	return reflect.DeepEqual(cr.Spec.ForProvider.ActiveRealms, activeRealms)
}

// GenerateSecurityRealmObservation returns the observed realm state.
func GenerateSecurityRealmObservation(availableRealms []security.Realm) iamv1alpha1.SecurityRealmObservation {
	if availableRealms == nil {
		return iamv1alpha1.SecurityRealmObservation{}
	}

	realmInfos := make([]iamv1alpha1.RealmInfo, len(availableRealms))
	for idx, realmItem := range availableRealms {
		realmInfos[idx] = iamv1alpha1.RealmInfo{
			ID:   realmItem.ID,
			Name: realmItem.Name,
		}
	}

	return iamv1alpha1.SecurityRealmObservation{AvailableRealms: realmInfos}
}
