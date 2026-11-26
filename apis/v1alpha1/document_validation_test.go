package v1alpha1_test

import (
	"testing"

	"github.com/genesary/provider-sonatype-nexus/apis/internal/crdtest"
)

// TestEndpointCredentialsValidation checks that a username is accepted only
// together with a password, which it would otherwise be silently ignored
// without. It checks the rules through the SAML, one of the managed resources
// embedding a DocumentSource.
func TestEndpointCredentialsValidation(t *testing.T) {
	t.Parallel()

	secretRef := map[string]any{"name": "idp", "key": "key"}

	withCredentials := func(credentials map[string]any) map[string]any {
		return map[string]any{
			"entityId":          "nexus",
			"usernameAttribute": "username",
			"idpMetadataFrom": map[string]any{
				"endpointUrl":         "https://idp.example.com/metadata",
				"endpointCredentials": credentials,
			},
		}
	}

	crdtest.RunForProvider(t, "iam.nexus.crossplane.io_samls.yaml", map[string]crdtest.Case{
		"None":     {ForProvider: withCredentials(map[string]any{})},
		"Password": {ForProvider: withCredentials(map[string]any{"passwordSecretRef": secretRef})},
		"UsernameAndPassword": {
			ForProvider: withCredentials(map[string]any{"usernameSecretRef": secretRef, "passwordSecretRef": secretRef}),
		},
		"UsernameOnly": {
			ForProvider: withCredentials(map[string]any{"usernameSecretRef": secretRef}),
			WantErr:     "usernameSecretRef requires passwordSecretRef",
		},
	})
}
