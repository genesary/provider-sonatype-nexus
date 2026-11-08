//go:build e2e

package instance_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// Environment variables holding the Azure storage account an Azure blob
// store is created in on Nexus Repository Pro. The container must exist.
const (
	envAzureAccountName   = "E2E_AZURE_ACCOUNT_NAME"
	envAzureAccountKey    = "E2E_AZURE_ACCOUNT_KEY"
	envAzureContainerName = "E2E_AZURE_CONTAINER_NAME"
)

// TestBlobStoreAzure checks Azure blob stores. On Nexus Community Edition,
// which refuses them, it checks that the resource reports that Nexus
// Repository Pro is required, and that it can still be deleted. On Nexus
// Repository Pro it creates the blob store in the storage account named by
// the E2E_AZURE_* environment variables, and skips without them.
func TestBlobStoreAzure(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	pro, err := f.ProEdition()
	if err != nil {
		t.Fatalf("cannot probe the Nexus edition: %v", err)
	}

	accountName, accountKey, containerName := "e2eaccount", "e2e-placeholder-key", "e2e-container"

	if pro {
		accountName, accountKey, containerName = os.Getenv(envAzureAccountName), os.Getenv(envAzureAccountKey), os.Getenv(envAzureContainerName)
		if accountName == "" || accountKey == "" || containerName == "" {
			t.Skipf("Azure blob stores need a storage account on Nexus Repository Pro: set %s, %s and %s",
				envAzureAccountName, envAzureAccountKey, envAzureContainerName)
		}
	}

	const (
		bsName     = "e2e-test-azure"
		secretName = "e2e-azure-account-key"
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"accountKey": []byte(accountKey)},
	}
	if err := f.Kube.Create(context.Background(), secret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating account key secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), secret)
	})

	bs := &instancev1alpha1.BlobStore{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-azure-blobstore", Namespace: "default"},
		Spec: instancev1alpha1.BlobStoreSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.BlobStoreParameters{
				Name: bsName,
				Type: "Azure",
				AzureConfig: &instancev1alpha1.AzureConfig{
					AccountName:          accountName,
					ContainerName:        containerName,
					AuthenticationMethod: "ACCOUNTKEY",
					AccountKeySecretRef: &xpv2.LocalSecretKeySelector{
						LocalSecretReference: xpv2.LocalSecretReference{Name: secretName},
						Key:                  "accountKey",
					},
				},
			},
		},
	}

	if pro {
		f.CreateAndWaitForReady(t, bs, 2*time.Minute)

		got, err := f.FetchBlobStoreAzure(bsName)
		if err != nil {
			t.Fatalf("fetching Azure blob store from Nexus: %v", err)
		}
		if got == nil {
			t.Fatalf("Azure blob store %q not found in Nexus", bsName)
		}
		if got.BucketConfiguration.AccountName != accountName {
			t.Errorf("Azure blob store account = %q, want %q", got.BucketConfiguration.AccountName, accountName)
		}

		return
	}

	// Community Edition: the create fails, and the reason must be explicit.
	f.Apply(t, bs)
	t.Cleanup(func() {
		f.Delete(t, bs)
	})

	ctx := context.Background()
	key := client.ObjectKeyFromObject(bs)
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := f.Kube.Get(ctx, key, bs); err != nil {
			return false, err
		}

		synced := bs.GetCondition(xpv2.TypeSynced)

		return synced.Status == corev1.ConditionFalse && strings.Contains(synced.Message, instanceclient.ErrProEditionRequired.Error()), nil
	})
	if err != nil {
		t.Fatalf("waiting for the BlobStore to report that Nexus Repository Pro is required: %v; conditions: %s",
			err, e2e.SummariseConditions(bs))
	}

	if ready := bs.GetCondition(xpv2.TypeReady); ready.Status == corev1.ConditionTrue {
		t.Errorf("BlobStore reports Ready=True although Nexus refused it")
	}

	// Nothing exists in Nexus, so the deletion must not be blocked.
	f.Delete(t, bs)
	if err := f.WaitForDeletion(ctx, bs, 2*time.Minute); err != nil {
		t.Fatalf("waiting for the BlobStore deletion: %v", err)
	}
}
