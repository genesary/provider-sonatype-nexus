package nexus

import (
	"context"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
)

const (
	// ErrConnectionSecretRefNil is returned when writeConnectionSecretToRef is
	// not set.
	ErrConnectionSecretRefNil = "writeConnectionSecretToRef is not set"

	// ErrConnectionSecretNotFound is returned when the connection secret cannot
	// be found.
	ErrConnectionSecretNotFound = "cannot get connection secret"

	// ErrConnectionSecretKeyNotFound is returned when the key is absent from
	// the connection secret.
	ErrConnectionSecretKeyNotFound = "connection secret has no key"
)

// localSecretNamespace returns the namespace for a connection secret referenced
// via LocalSecretReference. Every managed resource in this provider is
// namespace-scoped, so GetNamespace is normally set; the
// metav1.NamespaceDefault fallback only guards against an unset namespace.
func localSecretNamespace(managed resource.Managed) string {
	return secretNamespace(managed.GetNamespace())
}

// secretNamespace returns namespace, or metav1.NamespaceDefault when it is
// unset.
func secretNamespace(namespace string) string {
	if namespace != "" {
		return namespace
	}

	return metav1.NamespaceDefault
}

// WithDefaultKey returns a copy of selector whose empty Key is set to key. A
// nil selector stays nil.
func WithDefaultKey(selector *xpv2.LocalSecretKeySelector, key string) *xpv2.LocalSecretKeySelector {
	if selector == nil {
		return nil
	}

	out := *selector
	if out.Key == "" {
		out.Key = key
	}

	return &out
}

// GetSecretBytes retrieves raw bytes from the specified key of a Kubernetes
// secret in namespace, the namespace of the object that declares selector.
func GetSecretBytes(
	ctx context.Context,
	kube kubeclient.Client,
	namespace string,
	selector *xpv2.LocalSecretKeySelector,
) ([]byte, error) {
	if selector == nil {
		return nil, errors.New("secretKeySelector is nil")
	}

	key := types.NamespacedName{Name: selector.Name, Namespace: secretNamespace(namespace)}
	secret := &corev1.Secret{}

	err := kube.Get(ctx, key, secret)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get secret")
	}

	data, ok := secret.Data[selector.Key]
	if !ok {
		return nil, errors.Errorf("secret %s/%s has no key %q", key.Namespace, key.Name, selector.Key)
	}

	return data, nil
}

// GetLocalConnectionSecretBytes retrieves raw bytes from the key of the
// connection secret referenced by ref. The namespace is derived from managed;
// for cluster-scoped resources it falls back to "default".
func GetLocalConnectionSecretBytes(
	ctx context.Context,
	kube kubeclient.Client,
	managed resource.Managed,
	ref *xpv2.LocalSecretReference,
	key string,
) ([]byte, error) {
	if ref == nil {
		return nil, errors.New(ErrConnectionSecretRefNil)
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: localSecretNamespace(managed),
	}, secret)
	if err != nil {
		return nil, errors.Wrap(err, ErrConnectionSecretNotFound)
	}

	data, ok := secret.Data[key]
	if !ok {
		return nil, errors.Errorf("%s %q", ErrConnectionSecretKeyNotFound, key)
	}

	return data, nil
}
