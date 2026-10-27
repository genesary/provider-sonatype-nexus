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

		return getCredentialsFromClusterSpec(ctx, kube, cpc.Spec)
	}

	providerConfig := &v1alpha1.ProviderConfig{}

	err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: managed.GetNamespace()}, providerConfig)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get ProviderConfig")
	}

	return GetCredentialsFromSpec(ctx, kube, providerConfig.GetNamespace(), providerConfig.Spec)
}

// GetCredentialsFromSpec extracts Nexus credentials from a ProviderConfigSpec.
// namespace is the namespace of the ProviderConfig that holds spec; its
// Secrets are always read from there.
func GetCredentialsFromSpec(ctx context.Context, kube kubeclient.Client, namespace string, spec v1alpha1.ProviderConfigSpec) (Credentials, error) {
	return resolveCredentials(ctx, kube, spec.URL, spec.InsecureSkipVerify,
		credentialSource{source: spec.Username.Source, selector: toSelector(spec.Username.SecretRef, namespace)},
		credentialSource{source: spec.Password.Source, selector: toSelector(spec.Password.SecretRef, namespace)},
	)
}

// getCredentialsFromClusterSpec extracts Nexus credentials from a
// ClusterProviderConfigSpec. It is cluster-scoped and authored by cluster
// administrators, so its selectors name the namespace of their Secret.
func getCredentialsFromClusterSpec(ctx context.Context, kube kubeclient.Client, spec v1alpha1.ClusterProviderConfigSpec) (Credentials, error) {
	return resolveCredentials(ctx, kube, spec.URL, spec.InsecureSkipVerify,
		credentialSource{source: spec.Username.Source, selector: spec.Username.SecretRef},
		credentialSource{source: spec.Password.Source, selector: spec.Password.SecretRef},
	)
}

// credentialSource is one credential of a provider config, with its Secret
// selector already bound to a namespace.
type credentialSource struct {
	source   xpv2.CredentialsSource
	selector *xpv2.SecretKeySelector
}

// resolveCredentials reads the Secrets selected by usernameSrc and passwordSrc.
func resolveCredentials(ctx context.Context, kube kubeclient.Client, url string, insecure *bool, usernameSrc, passwordSrc credentialSource) (Credentials, error) {
	for _, cred := range []struct {
		name string
		src  credentialSource
	}{{"username", usernameSrc}, {"password", passwordSrc}} {
		if cred.src.source != xpv2.CredentialsSourceSecret {
			return Credentials{}, errors.Errorf("credentials source %q for %s is not supported", cred.src.source, cred.name)
		}

		if cred.src.selector == nil {
			return Credentials{}, errors.Errorf("secretRef is required for %s", cred.name)
		}
	}

	username, err := readSecretValue(ctx, kube, usernameSrc.selector)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get username from secret")
	}

	password, err := readSecretValue(ctx, kube, passwordSrc.selector)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get password from secret")
	}

	return Credentials{
		URL:      url,
		Username: username,
		Password: password,
		Insecure: ptr.Deref(insecure, false),
	}, nil
}

// GetSecretValue retrieves a value from a Secret in namespace, the namespace
// of the object that declares selector.
func GetSecretValue(ctx context.Context, kube kubeclient.Client, namespace string, selector *xpv2.LocalSecretKeySelector) (string, error) {
	if selector == nil {
		return "", errors.New("secretKeySelector is nil")
	}

	return readSecretValue(ctx, kube, selector.ToSecretKeySelector(secretNamespace(namespace)))
}

// readSecretValue retrieves the value selected by selector.
func readSecretValue(ctx context.Context, kube kubeclient.Client, selector *xpv2.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{Name: selector.Name, Namespace: selector.Namespace}, secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	data, ok := secret.Data[selector.Key]
	if !ok {
		return "", errors.Errorf("secret does not contain key %q", selector.Key)
	}

	return string(data), nil
}

// toSelector binds a local selector to namespace; a nil selector stays nil.
func toSelector(local *xpv2.LocalSecretKeySelector, namespace string) *xpv2.SecretKeySelector {
	if local == nil {
		return nil
	}

	return local.ToSecretKeySelector(namespace)
}
