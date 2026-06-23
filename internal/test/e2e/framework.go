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

package e2e

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	nexus3 "github.com/datadrivers/go-nexus-client/nexus3"

	"github.com/genesary/provider-sonatype-nexus/apis"
	nexusclient "github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// Environment variable names recognised by the framework.
const (
	EnvNexusURL        = "NEXUS_URL"
	EnvNexusUser       = "NEXUS_USER"
	EnvNexusPass       = "NEXUS_PASS"
	EnvProviderConfig  = "NEXUS_PROVIDERCONFIG"
)

// DefaultProviderConfigName is used when EnvProviderConfig is unset.
const DefaultProviderConfigName = "default"

// Framework groups the clients every e2e test needs: a controller-runtime
// client wired to the project's schemes, and a Nexus API client
// authenticated with admin credentials. Construct one per test with New.
type Framework struct {
	// Kube is a controller-runtime client with the provider's APIs registered.
	Kube client.Client
	// Nexus talks to the Nexus REST API directly, used to verify the
	// state the provider reconciled into Nexus.
	Nexus *nexus3.NexusClient
	// ProviderConfigName is the ProviderConfig managed resources should reference.
	ProviderConfigName string
}

// New constructs a Framework, failing the test immediately on missing config
// or unreachable cluster — there is no value in deferring those errors.
func New(t *testing.T) *Framework {
	t.Helper()

	url := mustEnv(t, EnvNexusURL)
	user := mustEnv(t, EnvNexusUser)
	pass := mustEnv(t, EnvNexusPass)

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("loading kubeconfig: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("registering core scheme: %v", err)
	}
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("registering provider schemes: %v", err)
	}

	kc, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("building kube client: %v", err)
	}

	nc, err := nexusclient.NewClient(nexusclient.Credentials{
		URL:      url,
		Username: user,
		Password: pass,
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("building nexus client: %v", err)
	}

	pc := os.Getenv(EnvProviderConfig)
	if pc == "" {
		pc = DefaultProviderConfigName
	}

	return &Framework{
		Kube:               kc,
		Nexus:              nc,
		ProviderConfigName: pc,
	}
}

func mustEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
