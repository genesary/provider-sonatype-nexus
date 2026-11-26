package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// EndpointCredentials holds optional HTTP credentials for an endpoint a
// managed resource fetches a document from. The secrets must be in the same
// namespace as the managed resource.
type EndpointCredentials struct {
	// UsernameSecretRef references a secret key with the HTTP username.
	// +kubebuilder:validation:Optional
	UsernameSecretRef *xpv2.LocalSecretKeySelector `json:"usernameSecretRef,omitempty"`

	// PasswordSecretRef references a secret key with the HTTP password
	// or bearer token.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.LocalSecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// DocumentSource is where the provider reads a document from.
type DocumentSource struct {
	// SecretRef references a secret key holding the document. The secret must
	// be in the same namespace as the managed resource.
	// +kubebuilder:validation:Optional
	SecretRef *xpv2.LocalSecretKeySelector `json:"secretRef,omitempty"`

	// EndpointURL is the URL the document is fetched from, on every
	// reconcile. When that fails, the document last fetched from it is used
	// instead, provided writeConnectionSecretToRef is set.
	// +kubebuilder:validation:Optional
	EndpointURL *string `json:"endpointUrl,omitempty"`

	// EndpointCredentials are optional HTTP credentials for EndpointURL: a
	// username and a password for basic authentication, or a password alone
	// for a bearer token.
	// +kubebuilder:validation:Optional
	EndpointCredentials *EndpointCredentials `json:"endpointCredentials,omitempty"`
}
