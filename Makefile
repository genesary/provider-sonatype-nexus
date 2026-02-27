# Project settings
PROJECT_NAME := provider-sonatype-nexus
PROJECT_REPO := github.com/genesary/$(PROJECT_NAME)

# Image settings
REGISTRY ?= ghcr.io
IMAGE_NAME ?= genesary/$(PROJECT_NAME)
IMAGE_TAG ?= latest

# Crossplane package settings
XPKG_FILE ?= provider-sonatype-nexus.xpkg

# Go settings
GO_VERSION := 1.22
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

# Tools
CONTROLLER_GEN := $(GOBIN)/controller-gen
GOLANGCI_LINT := $(GOBIN)/golangci-lint

# Kubernetes manifests
CRD_DIR := package/crds
RBAC_DIR := package/rbac

.PHONY: all
all: generate build test

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate
generate: ## Generate code (CRDs, DeepCopy, etc.)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	$(CONTROLLER_GEN) crd:crdVersions=v1 paths="./apis/..." output:crd:artifacts:config=$(CRD_DIR)

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

##@ Build

.PHONY: build
build: generate fmt vet ## Build the provider binary
	go build -o bin/provider ./cmd/provider

.PHONY: run
run: generate fmt vet ## Run the provider locally
	go run ./cmd/provider

##@ Testing

.PHONY: test
test: generate fmt vet ## Run unit tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires running Nexus)
	go test -v -tags=integration ./...

.PHONY: coverage
coverage: test ## Generate coverage report
	go tool cover -html=coverage.out -o coverage.html

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) .

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

##@ Crossplane Package

.PHONY: xpkg-build
xpkg-build: docker-build ## Build Crossplane package (xpkg) with embedded runtime
	crossplane xpkg build \
		--package-root=package \
		--embed-runtime-image=$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		--package-file=$(XPKG_FILE)

.PHONY: xpkg-push
xpkg-push: ## Push Crossplane package to registry
	crossplane xpkg push \
		$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(XPKG_FILE)

##@ Install Tools

.PHONY: install-tools
install-tools: $(CONTROLLER_GEN) $(GOLANGCI_LINT) ## Install required tools

$(CONTROLLER_GEN):
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

$(GOLANGCI_LINT):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

##@ Clean

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(XPKG_FILE)

.PHONY: clean-generated
clean-generated: ## Clean generated files
	rm -f apis/v1alpha1/zz_generated.deepcopy.go
	rm -rf $(CRD_DIR)/*

##@ E2E Testing

KIND_CLUSTER_NAME ?= nexus-e2e
E2E_IMAGE_TAG ?= e2e

E2E_REGISTRY ?= localhost:5001

.PHONY: e2e-setup
e2e-setup: xpkg-build ## Setup e2e test environment (Kind + Crossplane + Nexus + Provider)
	@echo "Starting local registry..."
	docker rm -f kind-registry 2>/dev/null || true
	docker run -d --restart=always -p 5001:5000 --network bridge --name kind-registry registry:2
	@echo "Creating Kind cluster..."
	kind create cluster --config e2e/kind-config.yaml --wait 60s || true
	docker network connect kind kind-registry 2>/dev/null || true
	@echo "Installing Crossplane..."
	helm repo add crossplane-stable https://charts.crossplane.io/stable 2>/dev/null || true
	helm repo update
	helm upgrade --install crossplane crossplane-stable/crossplane \
		--namespace crossplane-system --create-namespace --wait --timeout 120s
	@echo "Pushing xpkg to local registry..."
	crossplane xpkg push $(E2E_REGISTRY)/provider-sonatype-nexus:$(E2E_IMAGE_TAG) -f $(XPKG_FILE)
	@echo "Deploying Nexus..."
	kubectl apply -f e2e/manifests/nexus.yaml
	@echo "Installing Provider via Crossplane..."
	kubectl apply -f e2e/manifests/provider.yaml
	@echo "Waiting for Provider to be healthy..."
	kubectl wait --for=condition=Healthy providers.pkg.crossplane.io/provider-sonatype-nexus --timeout=180s
	@echo "Applying ProviderConfig..."
	kubectl apply -f e2e/manifests/provider-config.yaml
	@echo "Waiting for Nexus..."
	kubectl wait --for=condition=available deployment/nexus -n nexus --timeout=300s || echo "Nexus still starting..."
	@echo "E2E environment setup complete!"
	@echo "Nexus will be available at http://localhost:8081 (default: admin/admin123)"

.PHONY: e2e-wait
e2e-wait: ## Wait for all e2e components to be ready
	chmod +x e2e/tests/*.sh e2e/run-e2e.sh
	NEXUS_URL=http://localhost:8081 ./e2e/tests/00-wait-ready.sh

.PHONY: e2e-run
e2e-run: ## Run e2e tests
	chmod +x e2e/tests/*.sh e2e/run-e2e.sh
	NEXUS_URL=http://localhost:8081 ./e2e/run-e2e.sh run

.PHONY: e2e-cleanup
e2e-cleanup: ## Cleanup e2e test environment
	kind delete cluster --name $(KIND_CLUSTER_NAME) || true
	docker rm -f kind-registry 2>/dev/null || true

.PHONY: e2e
e2e: e2e-setup e2e-wait e2e-run ## Run full e2e test cycle (setup + tests + keeps cluster)

.PHONY: e2e-full
e2e-full: e2e e2e-cleanup ## Run full e2e test cycle with cleanup
