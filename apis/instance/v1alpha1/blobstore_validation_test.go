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

// blobStoreCRDPath is the generated BlobStore CustomResourceDefinition.
const blobStoreCRDPath = "../../../package/crds/instance.nexus.crossplane.io_blobstores.yaml"

// forProviderValidator returns a validator of the CEL rules of the
// spec.forProvider schema of the generated BlobStore CRD.
func forProviderValidator(t *testing.T) (*cel.Validator, *schema.Structural) {
	t.Helper()

	raw, err := os.ReadFile(blobStoreCRDPath)
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

// TestBlobStoreAzureValidation checks the CEL rules tying the Azure
// configuration to the blob store type and the account key to the
// authentication method.
func TestBlobStoreAzureValidation(t *testing.T) {
	t.Parallel()

	validator, structural := forProviderValidator(t)

	accountKeyConfig := map[string]any{
		"accountName":          "account",
		"containerName":        "nexus-blobs",
		"authenticationMethod": "ACCOUNTKEY",
		"accountKeySecretRef":  map[string]any{"name": "azure", "key": "accountKey"},
	}
	managedIdentityConfig := map[string]any{
		"accountName":          "account",
		"containerName":        "nexus-blobs",
		"authenticationMethod": "MANAGEDIDENTITY",
	}

	tests := map[string]struct {
		forProvider map[string]any
		wantErr     string
	}{
		"AzureWithAccountKey": {
			forProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": accountKeyConfig},
		},
		"AzureWithManagedIdentity": {
			forProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": managedIdentityConfig},
		},
		"FileWithoutAzureConfig": {
			forProvider: map[string]any{"name": "file", "type": "File"},
		},
		"AzureWithoutAzureConfig": {
			forProvider: map[string]any{"name": "azure", "type": "Azure"},
			wantErr:     "azureConfig is required when type is Azure",
		},
		"FileWithAzureConfig": {
			forProvider: map[string]any{"name": "file", "type": "File", "azureConfig": managedIdentityConfig},
			wantErr:     "azureConfig is only allowed when type is Azure",
		},
		"AccountKeyWithoutSecret": {
			forProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": map[string]any{
				"accountName":          "account",
				"containerName":        "nexus-blobs",
				"authenticationMethod": "ACCOUNTKEY",
			}},
			wantErr: "accountKeySecretRef is required when authenticationMethod is ACCOUNTKEY",
		},
		"ManagedIdentityWithSecret": {
			forProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": map[string]any{
				"accountName":          "account",
				"containerName":        "nexus-blobs",
				"authenticationMethod": "MANAGEDIDENTITY",
				"accountKeySecretRef":  map[string]any{"name": "azure", "key": "accountKey"},
			}},
			wantErr: "accountKeySecretRef is only allowed when authenticationMethod is ACCOUNTKEY",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			errs, _ := validator.Validate(context.Background(), field.NewPath("forProvider"), structural,
				tc.forProvider, nil, celconfig.RuntimeCELCostBudget)

			if tc.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("Validate() errors = %v, want none", errs)
				}

				return
			}

			if !strings.Contains(errs.ToAggregate().Error(), tc.wantErr) {
				t.Fatalf("Validate() errors = %v, want %q", errs, tc.wantErr)
			}
		})
	}
}
