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
// via LocalSecretReference. For cluster-scoped resources whose GetNamespace
// returns "" it falls back to metav1.NamespaceDefault.
func localSecretNamespace(managed resource.Managed) string {
	if namespace := managed.GetNamespace(); namespace != "" {
		return namespace
	}

	return metav1.NamespaceDefault
}

// GetSecretBytes retrieves raw bytes from the specified key of a Kubernetes
// secret identified by a full SecretKeySelector (name, namespace, key).
func GetSecretBytes(
	ctx context.Context,
	kube kubeclient.Client,
	selector *xpv2.SecretKeySelector,
) ([]byte, error) {
	if selector == nil {
		return nil, errors.New("secretKeySelector is nil")
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: selector.Namespace,
	}, secret)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get secret")
	}

	data, ok := secret.Data[selector.Key]
	if !ok {
		return nil, errors.Errorf("secret %s/%s has no key %q", selector.Namespace, selector.Name, selector.Key)
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
