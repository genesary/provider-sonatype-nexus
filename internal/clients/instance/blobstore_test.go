package instance_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// proOnlyBody is the body Nexus 3.96 Community answers a write of an Azure
// blob store with, along with 402 Payment Required.
const proOnlyBody = "Azure Blob Storage is a Pro only feature."

// newTestAzureClient creates an Azure blob store client pointing at the test
// server.
func newTestAzureClient(t *testing.T, serverURL string) instanceclient.BlobStoreAzureClient {
	t.Helper()

	clients, err := instanceclient.NewBlobStoreClients(nexus.Credentials{
		URL:      serverURL,
		Username: "admin",
		Password: "password",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewBlobStoreClients() unexpected error: %v", err)
	}

	return clients.Azure
}

// newAzureModel returns a minimal Azure blob store.
func newAzureModel() v395.AzureBlobStoreApiModel {
	return v395.AzureBlobStoreApiModel{
		Name: "azure",
		BucketConfiguration: v395.AzureBlobStoreApiBucketConfiguration{
			AccountName:    "account",
			ContainerName:  "container",
			Authentication: v395.AzureBlobStoreApiAuthentication{AuthenticationMethod: "MANAGEDIDENTITY"},
		},
	}
}

// TestAzureBlobStoreRequiresPro checks that the 402 Nexus Community
// Edition answers with is reported as ErrAzureBlobStoreRequiresPro, keeping the
// Nexus explanation, on every write.
func TestAzureBlobStoreRequiresPro(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(proOnlyBody))
	}))
	t.Cleanup(server.Close)

	client := newTestAzureClient(t, server.URL)

	for name, call := range map[string]func() error{
		"Create": func() error { return client.Create(context.Background(), newAzureModel()) },
		"Update": func() error { return client.Update(context.Background(), "azure", newAzureModel()) },
		"Get": func() error {
			_, err := client.Get(context.Background(), "azure")

			return err
		},
	} {
		err := call()
		if !errors.Is(err, instanceclient.ErrAzureBlobStoreRequiresPro) {
			t.Errorf("%s() error = %v, want it to wrap ErrAzureBlobStoreRequiresPro", name, err)
		}

		if err != nil && !strings.Contains(err.Error(), proOnlyBody) {
			t.Errorf("%s() error = %v, want it to quote the Nexus explanation", name, err)
		}
	}
}

// TestAzureBlobStoreErrors checks that answers other than 402 keep the usual
// error mapping.
func TestAzureBlobStoreErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status       int
		wantNotFound bool
	}{
		"NotFound":    {status: http.StatusNotFound, wantNotFound: true},
		"ServerError": {status: http.StatusInternalServerError},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))
			t.Cleanup(server.Close)

			_, err := newTestAzureClient(t, server.URL).Get(context.Background(), "azure")
			if err == nil {
				t.Fatal("Get() expected an error")
			}

			if nexus.IsNotFound(err) != tc.wantNotFound {
				t.Errorf("Get() error = %v, IsNotFound = %v, want %v", err, nexus.IsNotFound(err), tc.wantNotFound)
			}

			if errors.Is(err, instanceclient.ErrAzureBlobStoreRequiresPro) {
				t.Errorf("Get() error = %v, unexpectedly wraps ErrAzureBlobStoreRequiresPro", err)
			}
		})
	}
}

// TestAzureBlobStoreRequests checks the paths and payloads of the Azure blob
// store client.
func TestAzureBlobStoreRequests(t *testing.T) {
	t.Parallel()

	var requests []string

	var created v395.AzureBlobStoreApiModel

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(newAzureModel())
		default:
			_ = json.NewDecoder(r.Body).Decode(&created)

			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestAzureClient(t, server.URL)

	err := client.Create(context.Background(), newAzureModel())
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if created.BucketConfiguration.AccountName != "account" {
		t.Errorf("Create() sent %+v", created)
	}

	err = client.Update(context.Background(), "azure", newAzureModel())
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	got, err := client.Get(context.Background(), "azure")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if got.BucketConfiguration.ContainerName != "container" {
		t.Errorf("Get() = %+v", got)
	}

	want := []string{
		"POST /service/rest/v1/blobstores/azure",
		"PUT /service/rest/v1/blobstores/azure/azure",
		"GET /service/rest/v1/blobstores/azure/azure",
	}
	if strings.Join(requests, ",") != strings.Join(want, ",") {
		t.Errorf("requests = %v, want %v", requests, want)
	}
}
