//go:build e2e

/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestSecuritySSLTruststoreCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	pemCert := generateSelfSignedCert(t)

	cert := &iamv1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-ssl-cert", Namespace: "default"},
		Spec: iamv1alpha1.SecuritySSLTruststoreSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.SecuritySSLTruststoreParameters{
				Pem: &pemCert,
			},
		},
	}

	f.CreateAndWaitForReady(t, cert, 2*time.Minute)
	e2e.AssertReady(t, cert)
	e2e.AssertSynced(t, cert)

	certs, err := f.ListSSLCertificates()
	if err != nil {
		t.Fatalf("listing SSL certificates from Nexus: %v", err)
	}
	if len(certs) == 0 {
		t.Fatal("no SSL certificates found in Nexus truststore after creating one")
	}
}

// TestSecuritySSLTruststoreFromSecret checks that a certificate is read from a
// Secret, and follows it.
func TestSecuritySSLTruststoreFromSecret(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-ssl-source", Namespace: "default"}}
	f.Apply(t, secret)
	t.Cleanup(func() { f.Delete(t, secret) })

	publish := func(data string) error {
		secret.Data = map[string][]byte{"ca.crt": []byte(data)}

		return f.Kube.Update(ctx, secret)
	}

	testTruststoreRotation(t, f, "e2e-test-ssl-secret", publish, nexusv1alpha1.DocumentSource{
		SecretRef: secretKey(secret.Name, "ca.crt"),
	})
}

// TestSecuritySSLTruststoreFromURL checks that a certificate is fetched from a
// URL that requires authentication, and follows it.
func TestSecuritySSLTruststoreFromURL(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	server := f.NewFileServer(t, "e2e-test-ssl-url")

	testTruststoreRotation(t, f, "e2e-test-ssl-url", server.Serve, nexusv1alpha1.DocumentSource{
		EndpointURL: new(server.URL()),
		EndpointCredentials: &nexusv1alpha1.EndpointCredentials{
			UsernameSecretRef: secretKey(server.SecretName(), "username"),
			PasswordSecretRef: secretKey(server.SecretName(), "password"),
		},
	})
}

// TestSecuritySSLTruststoreURLUnavailable checks that the certificate last
// fetched from a URL is used while the URL is unavailable, and that it is not
// for another URL.
func TestSecuritySSLTruststoreURLUnavailable(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()
	server := f.NewFileServer(t, "e2e-test-ssl-down")
	pemCert := generateSelfSignedCert(t)

	err := server.Serve(pemCert)
	if err != nil {
		t.Fatalf("serving the certificate: %v", err)
	}

	cr := newTruststoreFrom(f, "e2e-test-ssl-down", nexusv1alpha1.DocumentSource{
		EndpointURL: new(server.URL()),
		EndpointCredentials: &nexusv1alpha1.EndpointCredentials{
			UsernameSecretRef: secretKey(server.SecretName(), "username"),
			PasswordSecretRef: secretKey(server.SecretName(), "password"),
		},
	})
	f.CreateAndWaitForReady(t, cr, 2*time.Minute)

	// update changes the spec and waits until the change is reconciled.
	update := func(mutate func()) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := f.Kube.Get(ctx, client.ObjectKeyFromObject(cr), cr)
			if err != nil {
				return err
			}

			mutate()

			return f.Kube.Update(ctx, cr)
		})
		if err != nil {
			t.Fatalf("updating the SecuritySSLTruststore: %v", err)
		}

		err = f.WaitForReconcile(ctx, cr, 2*time.Minute)
		if err != nil {
			t.Fatalf("waiting for the update to be reconciled: %v", err)
		}
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("stopping the file server: %v", err)
	}

	// Any change of the spec tells when the resource was reconciled.
	update(func() {
		cr.Spec.ManagementPolicies = xpv2.ManagementPolicies{
			xpv2.ManagementActionObserve, xpv2.ManagementActionCreate, xpv2.ManagementActionUpdate, xpv2.ManagementActionDelete,
		}
	})

	e2e.AssertSynced(t, cr)

	cert, err := f.FindSSLCertificate(pemCert)
	if err != nil || cert == nil {
		t.Errorf("certificate in Nexus = %v, %v, want it kept", cert, err)
	}

	update(func() { cr.Spec.ForProvider.PemFrom.EndpointURL = new(server.URL() + ".other") })
	e2e.AssertCondition(t, cr, xpv2.TypeSynced, corev1.ConditionFalse)
}

// newTruststoreFrom returns a SecuritySSLTruststore reading its certificate
// from source, and caching it.
func newTruststoreFrom(f *e2e.Framework, name string, source nexusv1alpha1.DocumentSource) *iamv1alpha1.SecuritySSLTruststore {
	return &iamv1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: iamv1alpha1.SecuritySSLTruststoreSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference:          &xpv2.ProviderConfigReference{Kind: "ProviderConfig", Name: f.ProviderConfigName},
				WriteConnectionSecretToReference: &xpv2.LocalSecretReference{Name: name + "-cache"},
			},
			ForProvider: iamv1alpha1.SecuritySSLTruststoreParameters{PemFrom: &source},
		},
	}
}

// testTruststoreRotation publishes a certificate bundle to source, checks that
// a SecuritySSLTruststore reading it adds its first certificate, and that
// publishing another certificate replaces it.
func testTruststoreRotation(t *testing.T, f *e2e.Framework, name string, publish func(string) error, source nexusv1alpha1.DocumentSource) {
	t.Helper()

	ctx := context.Background()
	first, other, rotated := generateSelfSignedCert(t), generateSelfSignedCert(t), generateSelfSignedCert(t)

	err := publish(first + other)
	if err != nil {
		t.Fatalf("publishing the certificate: %v", err)
	}

	cr := newTruststoreFrom(f, name, source)

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)

	// holds reports whether the truststore holds the certificate.
	holds := func(pemText string) bool {
		cert, err := f.FindSSLCertificate(pemText)
		if err != nil {
			t.Fatalf("listing SSL certificates from Nexus: %v", err)
		}

		return cert != nil
	}

	if !holds(first) || holds(other) {
		t.Errorf("truststore holds first=%t, other=%t of the bundle, want only the first", holds(first), holds(other))
	}

	err = publish(rotated)
	if err != nil {
		t.Fatalf("publishing the rotated certificate: %v", err)
	}

	err = f.Nudge(ctx, cr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.WaitForNexus(ctx, 2*time.Minute, func() (bool, error) { return holds(rotated) && !holds(first), nil })
	if err != nil {
		t.Fatalf("waiting for the rotated certificate: %v", err)
	}

	cert, err := f.FindSSLCertificate(rotated)
	if err != nil || cert == nil {
		t.Fatalf("finding the rotated certificate: %v, %v", cert, err)
	}

	// Without this, deleting the resource would leave the certificate behind.
	err = f.WaitForNexus(ctx, 10*time.Second, func() (bool, error) {
		err := f.Kube.Get(ctx, client.ObjectKeyFromObject(cr), cr)

		return meta.GetExternalName(cr) == cert.GetId(), err
	})
	if err != nil {
		t.Errorf("external name = %q, want %q", meta.GetExternalName(cr), cert.GetId())
	}

	f.Delete(t, cr)

	err = f.WaitForDeletion(ctx, cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for deletion: %v", err)
	}

	if holds(rotated) {
		t.Error("truststore still holds the certificate after deletion")
	}
}

// secretKey returns a selector of the given Secret key.
func secretKey(name, key string) *xpv2.LocalSecretKeySelector {
	return &xpv2.LocalSecretKeySelector{LocalSecretReference: xpv2.LocalSecretReference{Name: name}, Key: key}
}

func generateSelfSignedCert(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "e2e-test",
			Organization: []string{"TestOrg"},
			Country:      []string{"US"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encoding certificate: %v", err)
	}
	return buf.String()
}
