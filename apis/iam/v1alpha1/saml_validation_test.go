package v1alpha1_test

import (
	"maps"
	"testing"

	"github.com/genesary/provider-sonatype-nexus/apis/internal/crdtest"
)

// TestSAMLIdPMetadataValidation checks that exactly one source of identity
// provider metadata is accepted.
func TestSAMLIdPMetadataValidation(t *testing.T) {
	t.Parallel()

	forProvider := func(source map[string]any) map[string]any {
		out := map[string]any{"entityId": "nexus", "usernameAttribute": "username"}
		maps.Copy(out, source)

		return out
	}

	from := func(source map[string]any) map[string]any {
		return forProvider(map[string]any{"idpMetadataFrom": source})
	}

	secretRef := map[string]any{"name": "idp", "key": "metadata.xml"}
	credentials := map[string]any{"passwordSecretRef": secretRef}

	crdtest.RunForProvider(t, "iam.nexus.crossplane.io_samls.yaml", map[string]crdtest.Case{
		"Inline": {ForProvider: forProvider(map[string]any{"idpMetadata": "<xml/>"})},
		"Secret": {ForProvider: from(map[string]any{"secretRef": secretRef})},
		"URL":    {ForProvider: from(map[string]any{"endpointUrl": "https://idp.example.com/metadata"})},
		"URLWithCredentials": {
			ForProvider: from(map[string]any{"endpointUrl": "http://idp.example.com/metadata", "endpointCredentials": credentials}),
		},
		"None": {
			ForProvider: forProvider(nil),
			WantErr:     "exactly one of idpMetadata and idpMetadataFrom is required",
		},
		"InlineAndFrom": {
			ForProvider: forProvider(map[string]any{"idpMetadata": "<xml/>", "idpMetadataFrom": map[string]any{"secretRef": secretRef}}),
			WantErr:     "exactly one of idpMetadata and idpMetadataFrom is required",
		},
		"EmptyInline": {
			ForProvider: forProvider(map[string]any{"idpMetadata": ""}),
			WantErr:     "idpMetadata in body should be at least 1 chars long",
		},
		"EmptyFrom": {
			ForProvider: from(map[string]any{}),
			WantErr:     "exactly one of secretRef and endpointUrl is required",
		},
		"SecretAndURL": {
			ForProvider: from(map[string]any{"secretRef": secretRef, "endpointUrl": "https://idp.example.com/metadata"}),
			WantErr:     "exactly one of secretRef and endpointUrl is required",
		},
		"SecretWithCredentials": {
			ForProvider: from(map[string]any{"secretRef": secretRef, "endpointCredentials": credentials}),
			WantErr:     "endpointCredentials requires endpointUrl",
		},
		"NotHTTP": {
			ForProvider: from(map[string]any{"endpointUrl": "file:///etc/passwd"}),
			WantErr:     "endpointUrl in body should match",
		},
	})
}
