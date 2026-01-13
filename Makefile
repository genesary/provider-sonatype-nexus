# Project settings
PROJECT_NAME := provider-sonatype-nexus
PROJECT_REPO := github.com/crossplane-contrib/$(PROJECT_NAME)

# Image settings
REGISTRY ?= docker.io
IMAGE_NAME ?= crossplane/$(PROJECT_NAME)
IMAGE_TAG ?= latest

# Go settings
GO_VERSION := 1.21
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

.PHONY: clean-generated
clean-generated: ## Clean generated files
	rm -f apis/v1alpha1/zz_generated.deepcopy.go
	rm -rf $(CRD_DIR)/*
