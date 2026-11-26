package v1alpha1_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"sigs.k8s.io/yaml"
)

// repositoryCRDPath is the generated Repository CustomResourceDefinition.
const repositoryCRDPath = "../../../package/crds/content.nexus.crossplane.io_repositories.yaml"

// repositoryForProviderValidator returns a validator of the CEL rules of the
// spec.forProvider schema of the generated Repository CRD.
func repositoryForProviderValidator(t *testing.T) (*cel.Validator, *schema.Structural) {
	t.Helper()

	raw, err := os.ReadFile(repositoryCRDPath)
	if err != nil {
		t.Fatalf("reading the CRD: %v", err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}

	err = yaml.Unmarshal(raw, crd)
	if err != nil {
		t.Fatalf("decoding the CRD: %v", err)
	}

	v1Props := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"]

	props := &apiextensions.JSONSchemaProps{}

	err = apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&v1Props, props, nil)
	if err != nil {
		t.Fatalf("converting the schema: %v", err)
	}

	structural, err := schema.NewStructural(props)
	if err != nil {
		t.Fatalf("building the structural schema: %v", err)
	}

	return cel.NewValidator(structural, false, celconfig.PerCallLimit), structural
}

// TestRepositorySettingsValidation checks the CEL rules restricting the
// repository settings to the formats, types and policies Nexus honors them
// on.
func TestRepositorySettingsValidation(t *testing.T) {
	t.Parallel()

	validator, structural := repositoryForProviderValidator(t)

	preemptive := map[string]any{"authentication": map[string]any{"type": "username", "preemptive": true}}
	conan := func(version string) map[string]any {
		return map[string]any{"name": "conan", "format": "conan", "type": "proxy", "conanProxy": map[string]any{"conanVersion": version}}
	}

	tests := map[string]struct {
		forProvider map[string]any
		old         map[string]any
		wantErr     string
	}{
		"PreemptiveOnMavenProxy": {
			forProvider: map[string]any{"name": "m", "format": "maven2", "type": "proxy", "httpClient": preemptive},
		},
		"PreemptiveOnTerraformProxy": {
			forProvider: map[string]any{"name": "t", "format": "terraform", "type": "proxy", "httpClient": preemptive},
		},
		"PreemptiveOnNpmProxy": {
			forProvider: map[string]any{"name": "n", "format": "npm", "type": "proxy", "httpClient": preemptive},
			wantErr:     "preemptive is only supported on maven2, pypi and terraform proxy repositories",
		},
		"PreemptiveOnPypiGroup": {
			forProvider: map[string]any{"name": "p", "format": "pypi", "type": "group", "httpClient": preemptive},
			wantErr:     "preemptive is only supported on maven2, pypi and terraform proxy repositories",
		},
		"AuthenticationWithoutPreemptive": {
			forProvider: map[string]any{"name": "n", "format": "npm", "type": "proxy", "httpClient": map[string]any{
				"authentication": map[string]any{"type": "bearerToken", "bearerTokenSecretRef": map[string]any{"name": "s", "key": "k"}},
			}},
		},
		"EcrAuthOnDockerProxy": {
			forProvider: map[string]any{"name": "d", "format": "docker", "type": "proxy", "dockerProxy": map[string]any{
				"ecrAuth": map[string]any{"enabled": true, "registryId": "123456789012"},
			}},
		},
		"EcrAuthOnOciProxy": {
			forProvider: map[string]any{"name": "o", "format": "oci", "type": "proxy", "ociProxy": map[string]any{
				"ecrAuth": map[string]any{"enabled": true},
			}},
			wantErr: "ociProxy.ecrAuth is not supported",
		},
		"LatestPolicyWithAllowOnce": {
			forProvider: map[string]any{"name": "d", "format": "docker", "type": "hosted", "storage": map[string]any{
				"blobStoreName": "default", "writePolicy": "ALLOW_ONCE", "latestPolicy": true,
			}},
		},
		"LatestPolicyWithAllow": {
			forProvider: map[string]any{"name": "d", "format": "docker", "type": "hosted", "storage": map[string]any{
				"blobStoreName": "default", "writePolicy": "ALLOW", "latestPolicy": true,
			}},
			wantErr: "latestPolicy requires the ALLOW_ONCE writePolicy",
		},
		"LatestPolicyWithoutWritePolicy": {
			forProvider: map[string]any{"name": "d", "format": "oci", "type": "hosted", "storage": map[string]any{
				"blobStoreName": "default", "latestPolicy": true,
			}},
			wantErr: "latestPolicy requires the ALLOW_ONCE writePolicy",
		},
		"LatestPolicyOff": {
			forProvider: map[string]any{"name": "d", "format": "docker", "type": "hosted", "storage": map[string]any{
				"blobStoreName": "default", "latestPolicy": false,
			}},
		},
		"ConanVersionUnchanged": {
			forProvider: conan("V1"),
			old:         conan("V1"),
		},
		"ConanVersionChanged": {
			forProvider: conan("V2"),
			old:         conan("V1"),
			wantErr:     "conanVersion is immutable",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var old any
			if tc.old != nil {
				old = tc.old
			}

			errs, _ := validator.Validate(context.Background(), field.NewPath("forProvider"), structural,
				tc.forProvider, old, celconfig.RuntimeCELCostBudget)

			if tc.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("Validate() errors = %v, want none", errs)
				}

				return
			}

			if len(errs) == 0 || !strings.Contains(errs.ToAggregate().Error(), tc.wantErr) {
				t.Fatalf("Validate() errors = %v, want %q", errs, tc.wantErr)
			}
		})
	}
}
