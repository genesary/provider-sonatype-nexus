package document

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// maxBytes caps the document fetched from an endpoint. It is about what a
// Secret can hold, and far above the size of the documents Nexus takes.
const maxBytes = 1 << 20

// endpointClient fetches documents from their endpoint. It bounds each fetch
// with a timeout and follows redirects like the default client.
var endpointClient = &http.Client{
	Transport: nexus.Transport(false),
	Timeout:   nexus.RequestTimeout,
}

// fetch performs the HTTP GET for the document at url. namespace is the one
// the credential secret selectors are bound to.
func fetch(ctx context.Context, kube client.Reader, namespace, url string, creds *nexusv1alpha1.EndpointCredentials) ([]byte, error) {
	req, err := nexus.NewRequest(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build endpoint request")
	}

	err = applyCredentials(ctx, kube, namespace, req, creds)
	if err != nil {
		return nil, err
	}

	resp, err := nexus.Do(endpointClient, req)
	if err != nil {
		return nil, errors.Wrap(err, "endpoint request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	data, err := nexus.ReadLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read endpoint response")
	}

	return data, nil
}

// applyCredentials sets HTTP auth headers on the request.
func applyCredentials(ctx context.Context, kube client.Reader, namespace string, req *http.Request, creds *nexusv1alpha1.EndpointCredentials) error {
	if creds == nil || creds.PasswordSecretRef == nil {
		return nil
	}

	if creds.UsernameSecretRef == nil {
		token, err := nexus.GetSecretValue(ctx, kube, namespace, creds.PasswordSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get endpoint token")
		}

		req.Header.Set("Authorization", "Bearer "+token)

		return nil
	}

	// Username and password are usually two keys of one Secret.
	secrets := nexus.MemoizeSecrets(kube)

	username, err := nexus.GetSecretValue(ctx, secrets, namespace, creds.UsernameSecretRef)
	if err != nil {
		return errors.Wrap(err, "cannot get endpoint username")
	}

	password, err := nexus.GetSecretValue(ctx, secrets, namespace, creds.PasswordSecretRef)
	if err != nil {
		return errors.Wrap(err, "cannot get endpoint password")
	}

	req.SetBasicAuth(username, password)

	return nil
}
