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

	// ErrSecretNamespaceMismatch is returned when a SecretKeySelector on a
	// namespaced object points at a Secret in another namespace.
	ErrSecretNamespaceMismatch = "secret reference must be in the same namespace as the referencing object"
)

// secretObjectKey returns the object key of the Secret selected by selector,
// bound to namespace, the namespace of the object that declares selector.
//
// The provider reads Secrets with its own service account, so an unbound
// selector would let anyone who can create a namespaced object read Secrets
// from every namespace the provider can see. An empty selector namespace
// defaults to namespace and any other namespace is rejected. An empty
// namespace means the declaring object is cluster-scoped, and only then is
// the selector namespace used as written.
func secretObjectKey(namespace string, selector *xpv2.SecretKeySelector) (types.NamespacedName, error) {
	if selector == nil {
		return types.NamespacedName{}, errors.New("secretKeySelector is nil")
	}

	if namespace == "" {
		return types.NamespacedName{Name: selector.Name, Namespace: selector.Namespace}, nil
	}

	if selector.Namespace != "" && selector.Namespace != namespace {
		return types.NamespacedName{}, errors.Errorf("%s: secret %q is in namespace %q, expected %q",
			ErrSecretNamespaceMismatch, selector.Name, selector.Namespace, namespace)
	}

	return types.NamespacedName{Name: selector.Name, Namespace: namespace}, nil
}

// localSecretNamespace returns the namespace for a connection secret referenced
// via LocalSecretReference. Every managed resource in this provider is
// namespace-scoped, so GetNamespace is normally set; the
// metav1.NamespaceDefault fallback only guards against an unset namespace.
func localSecretNamespace(managed resource.Managed) string {
	if namespace := managed.GetNamespace(); namespace != "" {
		return namespace
	}

	return metav1.NamespaceDefault
}

// GetSecretBytes retrieves raw bytes from the specified key of a Kubernetes
// secret identified by a SecretKeySelector. namespace is the namespace of the
// object that declares selector (empty for cluster-scoped objects); see
// secretObjectKey for how the two are reconciled.
func GetSecretBytes(
	ctx context.Context,
	kube kubeclient.Client,
	namespace string,
	selector *xpv2.SecretKeySelector,
) ([]byte, error) {
	key, err := secretObjectKey(namespace, selector)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}

	err = kube.Get(ctx, key, secret)
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
