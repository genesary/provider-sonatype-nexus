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
	// and the connection Secret caches none fetched from it.
	errUnavailable = "endpoint does not serve the document and none is cached"
	// errOtherEndpoint is returned when the connection Secret caches a
	// document that was not fetched from the endpoint.
	errOtherEndpoint = "cached document was not fetched from this endpoint"
)

// endpointKeySuffix is appended to Source.Key to name the connection Secret
// key that records which endpoint the cached document was fetched from.
const endpointKeySuffix = "Endpoint"

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
	// managed resource, for Read to fall back on.
	ConnectionDetails managed.ConnectionDetails
}

// Read returns the document that owner reads from the source. The Secrets the
// source references are read from the namespace of owner.
//
// While the endpoint is unavailable, Read returns the document cached in the
// connection Secret of owner, if it was fetched from that same endpoint. A
// controller keeps that cache current by publishing the ConnectionDetails of
// the documents it reads.
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

	data, err := s.readCache(ctx, kube, owner)
	if err != nil {
		// The endpoint error is the one to act on: having no cache is normal.
		return Document{}, errors.Wrap(endpointErr, errUnavailable)
	}

	return s.document(data), nil
}

// document returns the Document holding data read from the source.
func (s Source) document(data []byte) Document {
	return Document{
		Data: data,
		ConnectionDetails: managed.ConnectionDetails{
			s.Key:                     data,
			s.Key + endpointKeySuffix: []byte(s.endpointID()),
		},
	}
}

// readCache returns the document cached in the connection Secret of owner. It
// fails when that document was fetched from another endpoint: falling back on
// it would hide that the endpoint owner now names never served anything.
func (s Source) readCache(ctx context.Context, kube client.Reader, owner resource.ModernManaged) ([]byte, error) {
	ref := owner.GetWriteConnectionSecretToReference()
	secrets := nexus.MemoizeSecrets(kube)

	// Connection Secrets written by earlier versions do not record the
	// endpoint. Their document is trusted until the next successful fetch.
	endpoint, err := nexus.GetLocalConnectionSecretBytes(ctx, secrets, owner, ref, s.Key+endpointKeySuffix)
	if err == nil && string(endpoint) != s.endpointID() {
		return nil, errors.New(errOtherEndpoint)
	}

	return nexus.GetLocalConnectionSecretBytes(ctx, secrets, owner, ref, s.Key)
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
