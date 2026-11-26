// Package document reads the documents that managed resources apply to Nexus,
// such as a license file, from the Secret or the HTTP endpoint their spec
// names.
package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// ErrNoSource is returned when a Source names no place to read from.
	ErrNoSource = "neither a secret reference nor an endpoint URL is configured"

	// errUnavailable is returned when the endpoint does not serve the document
	// and the connection Secret caches none fetched from it, either because it
	// caches no document or because it caches one from another endpoint.
	errUnavailable = "endpoint does not serve the document and the connection Secret caches none fetched from it"
	// errOtherEndpoint is returned when the connection Secret caches a
	// document that was not fetched from the endpoint.
	errOtherEndpoint = "cached document was not fetched from this endpoint"
)

// endpointKey is the connection Secret key that records which endpoint the
// cached document was fetched from. A managed resource caches one document
// only, so the key does not depend on Source.Key.
const endpointKey = "endpointUrl"

// Source is where a managed resource reads a document from.
type Source struct {
	// From names the Secret or the endpoint that holds the document. SecretRef
	// takes precedence over EndpointURL.
	From *nexusv1alpha1.DocumentSource

	// Key is the connection Secret key the document is cached under.
	Key string
}

// Document is a document read from its Source.
type Document struct {
	// Data is the content of the document.
	Data []byte

	// ConnectionDetails cache the document in the connection Secret of the
	// managed resource, for Read to fall back on. Publish them only once Nexus
	// has accepted Data: a document Nexus refuses would otherwise replace the
	// last one it accepted as the fallback.
	ConnectionDetails managed.ConnectionDetails
}

// Read returns the document that owner reads from the source. The Secrets the
// source references are read from the namespace of owner.
//
// While the endpoint is unavailable, Read returns the document cached in the
// connection Secret of owner, if it was fetched from that same endpoint. A
// controller keeps that cache current by publishing the ConnectionDetails of
// the documents Nexus accepts, and only of those.
func (s Source) Read(ctx context.Context, kube client.Reader, owner resource.ModernManaged) (Document, error) {
	if s.From == nil || (s.From.SecretRef == nil && s.From.EndpointURL == nil) {
		return Document{}, errors.New(ErrNoSource)
	}

	if s.From.SecretRef != nil {
		data, err := nexus.GetSecretBytes(ctx, kube, owner.GetNamespace(), s.From.SecretRef)
		if err != nil {
			return Document{}, err
		}

		return s.document(data), nil
	}

	data, endpointErr := fetch(ctx, kube, owner.GetNamespace(), *s.From.EndpointURL, s.From.EndpointCredentials)
	if endpointErr == nil {
		return s.document(data), nil
	}

	data, recorded, err := s.readCache(ctx, kube, owner)
	if err != nil {
		// The endpoint error is the one to act on: having no cache is normal.
		return Document{}, errors.Wrap(endpointErr, errUnavailable)
	}

	cached := s.document(data)
	if !recorded {
		// Do not record an endpoint that never served this document: it
		// would keep the cache trusted past the next successful fetch.
		delete(cached.ConnectionDetails, endpointKey)
	}

	return cached, nil
}

// document returns the Document holding data read from the source.
func (s Source) document(data []byte) Document {
	return Document{
		Data: data,
		ConnectionDetails: managed.ConnectionDetails{
			s.Key:       data,
			endpointKey: []byte(s.endpointID()),
		},
	}
}

// readCache returns the document cached in the connection Secret of owner, and
// whether that Secret records the endpoint it was fetched from. It fails when
// that document was fetched from another endpoint: falling back on it would
// hide that the endpoint owner now names never served anything.
func (s Source) readCache(ctx context.Context, kube client.Reader, owner resource.ModernManaged) (data []byte, recorded bool, err error) {
	ref := owner.GetWriteConnectionSecretToReference()
	secrets := nexus.MemoizeSecrets(kube)

	// Connection Secrets written by earlier versions do not record the
	// endpoint. Their document is trusted until the next successful fetch.
	endpoint, err := nexus.GetLocalConnectionSecretBytes(ctx, secrets, owner, ref, endpointKey)
	recorded = err == nil

	if recorded && string(endpoint) != s.endpointID() {
		return nil, false, errors.New(errOtherEndpoint)
	}

	data, err = nexus.GetLocalConnectionSecretBytes(ctx, secrets, owner, ref, s.Key)

	return data, recorded, err
}

// endpointID identifies the endpoint the document is fetched from, or is empty
// when it is not fetched from one. It is a digest, because a URL can carry
// secrets.
func (s Source) endpointID() string {
	if s.From.SecretRef != nil || s.From.EndpointURL == nil {
		return ""
	}

	sum := sha256.Sum256([]byte(*s.From.EndpointURL))

	return hex.EncodeToString(sum[:])
}
