// Package nexus provides a client interface for Sonatype Nexus
// Repository Manager.
package nexus

import (
	"context"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/datadrivers/go-nexus-client/nexus3"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// Credentials contains the credentials for connecting to Nexus.
type Credentials struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// NewClient creates a new Nexus client from the provided credentials.
func NewClient(creds Credentials) (*nexus3.NexusClient, error) {
	cfg := client.Config{
		URL:      creds.URL,
		Username: creds.Username,
		Password: creds.Password,
		Insecure: creds.Insecure,
	}

	nc := nexus3.NewClient(cfg)
	if nc == nil {
		return nil, errors.New("failed to create Nexus client")
	}

	return nc, nil
}

// clusterProviderConfigKind is the Kind name for ClusterProviderConfig.
const clusterProviderConfigKind = "ClusterProviderConfig"

// GetCredentials resolves the ProviderConfig or ClusterProviderConfig
// referenced by mg and extracts Nexus credentials from the referenced secret.
// It dispatches on providerConfigRef.Kind: ClusterProviderConfig is looked up
// cluster-wide; ProviderConfig (or empty) is looked up in mg's namespace.
func GetCredentials(ctx context.Context, kube kubeclient.Client, managed interface {
	GetNamespace() string
	GetProviderConfigReference() *xpv2.ProviderConfigReference
}) (Credentials, error) {
	ref := managed.GetProviderConfigReference()
	if ref == nil {
		return Credentials{}, errors.New("providerConfigRef is not set")
	}

	if ref.Kind == clusterProviderConfigKind {
		cpc := &v1alpha1.ClusterProviderConfig{}

		err := kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc)
		if err != nil {
			return Credentials{}, errors.Wrap(err, "cannot get ClusterProviderConfig")
		}

		return GetCredentialsFromSpec(ctx, kube, cpc.Spec)
	}

	providerConfig := &v1alpha1.ProviderConfig{}

	err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: managed.GetNamespace()}, providerConfig)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get ProviderConfig")
	}

	return GetCredentialsFromSpec(ctx, kube, providerConfig.Spec)
}

// GetCredentialsFromSpec extracts Nexus credentials from a ProviderConfigSpec.
func GetCredentialsFromSpec(ctx context.Context, kube kubeclient.Client, spec v1alpha1.ProviderConfigSpec) (Credentials, error) {
	if spec.Username.Source != xpv2.CredentialsSourceSecret {
		return Credentials{}, errors.Errorf("credentials source %q for username is not supported", spec.Username.Source)
	}

	if spec.Password.Source != xpv2.CredentialsSourceSecret {
		return Credentials{}, errors.Errorf("credentials source %q for password is not supported", spec.Password.Source)
	}

	if spec.Username.SecretRef == nil {
		return Credentials{}, errors.New("secretRef is required for username")
	}

	if spec.Password.SecretRef == nil {
		return Credentials{}, errors.New("secretRef is required for password")
	}

	username, err := GetSecretValue(ctx, kube, spec.Username.SecretRef)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get username from secret")
	}

	password, err := GetSecretValue(ctx, kube, spec.Password.SecretRef)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get password from secret")
	}

	return Credentials{
		URL:      spec.URL,
		Username: username,
		Password: password,
		Insecure: ptr.Deref(spec.InsecureSkipVerify, false),
	}, nil
}

// GetCredentialsFromSecret extracts Nexus credentials from a
// Kubernetes secret.
//
// Deprecated: Use GetCredentials which correctly handles both
// ProviderConfig (namespace-scoped) and ClusterProviderConfig (cluster-scoped).
func GetCredentialsFromSecret(ctx context.Context, kube kubeclient.Client, providerConfig *v1alpha1.ProviderConfig) (Credentials, error) {
	return GetCredentialsFromSpec(ctx, kube, providerConfig.Spec)
}

// GetSecretValue retrieves a value from a Kubernetes secret using a
// SecretKeySelector.
func GetSecretValue(ctx context.Context, kube kubeclient.Client, selector *xpv2.SecretKeySelector) (string, error) {
	if selector == nil {
		return "", errors.New("secretKeySelector is nil")
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: selector.Namespace,
	}, secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	data, ok := secret.Data[selector.Key]
	if !ok {
		return "", errors.Errorf("secret does not contain key %q", selector.Key)
	}

	return string(data), nil
}
