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

// Package e2e provides reusable helpers for end-to-end tests that exercise
// the provider against a live Nexus instance and real Kubernetes API server.
//
// All files except this one carry the `e2e` build tag, so they are excluded
// from regular `go test ./...` runs. To execute the suite use:
//
//	go test -tags=e2e -timeout=30m ./internal/test/e2e/...
//
// or, equivalently, `make e2e.test`. Both invocations expect the following
// environment variables to be set:
//
//	KUBECONFIG           - points to a cluster running the provider
//	NEXUS_URL            - base URL of the Nexus API (e.g. http://localhost:8081)
//	NEXUS_USER           - admin user for direct API verification
//	NEXUS_PASS           - admin password for direct API verification
//
// The ProviderConfig used by managed resources is named "default" by default
// and can be overridden via the NEXUS_PROVIDERCONFIG environment variable.
package e2e
