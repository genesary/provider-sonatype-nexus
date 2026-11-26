package v1alpha1_test

import (
	"testing"

	"github.com/genesary/provider-sonatype-nexus/apis/internal/crdtest"
)

// TestSecuritySSLTruststorePemValidation checks that exactly one source of
// certificate is accepted. The sources of pemFrom are those of SAML's
// idpMetadataFrom, which TestSAMLIdPMetadataValidation checks.
func TestSecuritySSLTruststorePemValidation(t *testing.T) {
	t.Parallel()

	from := map[string]any{"secretRef": map[string]any{"name": "ca", "key": "ca.crt"}}

	crdtest.RunForProvider(t, "instance.nexus.crossplane.io_securityssltruststores.yaml", map[string]crdtest.Case{
		"Inline": {ForProvider: map[string]any{"pem": "-----BEGIN CERTIFICATE-----"}},
		"From":   {ForProvider: map[string]any{"pemFrom": from}},
		"None": {
			ForProvider: map[string]any{},
			WantErr:     "exactly one of pem and pemFrom is required",
		},
		"Both": {
			ForProvider: map[string]any{"pem": "-----BEGIN CERTIFICATE-----", "pemFrom": from},
			WantErr:     "exactly one of pem and pemFrom is required",
		},
		"EmptyInline": {
			ForProvider: map[string]any{"pem": ""},
			WantErr:     "pem in body should be at least 1 chars long",
		},
		"EmptyFrom": {
			ForProvider: map[string]any{"pemFrom": map[string]any{}},
			WantErr:     "exactly one of secretRef and endpointUrl is required",
		},
	})
}
