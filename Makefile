# Project settings
PROJECT_NAME := provider-sonatype-nexus
PROJECT_REPO := github.com/genesary/$(PROJECT_NAME)

# Image settings
REGISTRY ?= ghcr.io
IMAGE_NAME ?= genesary/$(PROJECT_NAME)-controller
IMAGE_TAG ?= latest

# Crossplane package settings
XPKG_NAME ?= genesary/$(PROJECT_NAME)
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
xpkg-build: generate ## Build Crossplane package (xpkg)
	@echo "Updating controller image in crossplane.yaml..."
	sed -i.bak 's|image:.*|image: $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)|' package/crossplane.yaml && rm -f package/crossplane.yaml.bak
	crossplane xpkg build \
		--package-root=package \
		--package-file=$(XPKG_FILE)

.PHONY: xpkg-push
xpkg-push: ## Push Crossplane package to registry
	crossplane xpkg push \
		$(REGISTRY)/$(XPKG_NAME):$(IMAGE_TAG) \
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

.PHONY: e2e-setup
e2e-setup: docker-build ## Setup e2e test environment (Kind + Nexus + Provider)
	@echo "Creating Kind cluster..."
	kind create cluster --config e2e/kind-config.yaml --wait 60s || true
	@echo "Loading provider image into Kind..."
	kind load docker-image $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) --name $(KIND_CLUSTER_NAME)
	docker tag $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) provider-sonatype-nexus-controller:$(E2E_IMAGE_TAG)
	kind load docker-image provider-sonatype-nexus-controller:$(E2E_IMAGE_TAG) --name $(KIND_CLUSTER_NAME)
	@echo "Installing CRDs..."
	kubectl apply -f $(CRD_DIR)/
	@echo "Deploying Nexus..."
	kubectl apply -f e2e/manifests/nexus.yaml
	@echo "Deploying Provider..."
	kubectl apply -f e2e/manifests/provider.yaml
	@echo "Waiting for deployments..."
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

.PHONY: e2e
e2e: e2e-setup e2e-wait e2e-run ## Run full e2e test cycle (setup + tests + keeps cluster)

.PHONY: e2e-full
e2e-full: e2e e2e-cleanup ## Run full e2e test cycle with cleanup
