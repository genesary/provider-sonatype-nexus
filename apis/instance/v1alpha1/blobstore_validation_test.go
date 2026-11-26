package v1alpha1_test

import (
	"testing"

	"github.com/genesary/provider-sonatype-nexus/apis/internal/crdtest"
)

// TestBlobStoreAzureValidation checks the CEL rules tying the Azure and S3
// configurations to the blob store type and the account key to the
// authentication method.
func TestBlobStoreAzureValidation(t *testing.T) {
	t.Parallel()

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

	crdtest.RunForProvider(t, "instance.nexus.crossplane.io_blobstores.yaml", map[string]crdtest.Case{
		"AzureWithAccountKey": {
			ForProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": accountKeyConfig},
		},
		"AzureWithManagedIdentity": {
			ForProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": managedIdentityConfig},
		},
		"FileWithoutAzureConfig": {
			ForProvider: map[string]any{"name": "file", "type": "File"},
		},
		"AzureWithoutAzureConfig": {
			ForProvider: map[string]any{"name": "azure", "type": "Azure"},
			WantErr:     "azureConfig is required when type is Azure",
		},
		"FileWithAzureConfig": {
			ForProvider: map[string]any{"name": "file", "type": "File", "azureConfig": managedIdentityConfig},
			WantErr:     "azureConfig is only allowed when type is Azure",
		},
		"AccountKeyWithoutSecret": {
			ForProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": map[string]any{
				"accountName":          "account",
				"containerName":        "nexus-blobs",
				"authenticationMethod": "ACCOUNTKEY",
			}},
			WantErr: "accountKeySecretRef is required when authenticationMethod is ACCOUNTKEY",
		},
		"S3WithS3Config": {
			ForProvider: map[string]any{"name": "s3", "type": "S3", "s3Config": map[string]any{"bucket": "nexus"}},
		},
		"AzureWithS3Config": {
			ForProvider: map[string]any{
				"name": "azure", "type": "Azure", "azureConfig": managedIdentityConfig,
				"s3Config": map[string]any{"bucket": "nexus"},
			},
			WantErr: "s3Config is only allowed when type is S3",
		},
		"FileWithS3Config": {
			ForProvider: map[string]any{"name": "file", "type": "File", "s3Config": map[string]any{"bucket": "nexus"}},
			WantErr:     "s3Config is only allowed when type is S3",
		},
		"ManagedIdentityWithSecret": {
			ForProvider: map[string]any{"name": "azure", "type": "Azure", "azureConfig": map[string]any{
				"accountName":          "account",
				"containerName":        "nexus-blobs",
				"authenticationMethod": "MANAGEDIDENTITY",
				"accountKeySecretRef":  map[string]any{"name": "azure", "key": "accountKey"},
			}},
			WantErr: "accountKeySecretRef is only allowed when authenticationMethod is ACCOUNTKEY",
		},
	})
}
