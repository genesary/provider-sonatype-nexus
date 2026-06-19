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

package iam_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestSecuritySSLTruststoreCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	pemCert := generateSelfSignedCert(t)

	cert := &iamv1alpha1.SecuritySSLTruststore{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-ssl-cert"},
		Spec: iamv1alpha1.SecuritySSLTruststoreSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.SecuritySSLTruststoreParameters{
				Pem: pemCert,
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
