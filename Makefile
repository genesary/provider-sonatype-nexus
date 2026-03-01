# ====================================================================================
# Setup Project

PROJECT_NAME ?= provider-sonatype-nexus
PROJECT_REPO ?= github.com/genesary/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64

# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

# Set a sane default so that the nprocs calculation below is less noisy on the initial
# loading of this file
NPROCS ?= 1

GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))

GO_REQUIRED_VERSION ?= 1.24
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_SUBDIRS += cmd internal apis
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

KIND_VERSION = v0.27.0
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

REGISTRY_ORGS ?= ghcr.io/genesary
IMAGES = $(PROJECT_NAME)
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= ghcr.io/genesary
XPKG_REG_ORGS_NO_PROMOTE ?= ghcr.io/genesary
XPKGS = $(PROJECT_NAME)
-include build/makelib/xpkg.mk

# ====================================================================================
# Fallthrough

# run `make help` to see the targets and options

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# NOTE: we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.provider-sonatype-nexus: do.build.images

# ====================================================================================
# Targets

go.cachedir:
	@go env GOCACHE

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# Generate CRDs and DeepCopy methods using controller-gen.
generate.run:
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@$(GOBIN)/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	@$(GOBIN)/controller-gen crd:crdVersions=v1 paths="./apis/..." output:crd:artifacts:config=package/crds

# ====================================================================================
# End to End Testing

CROSSPLANE_VERSION = 1.19.0
CROSSPLANE_CLI_VERSION = v1.19.0
CROSSPLANE_NAMESPACE = crossplane-system
-include build/makelib/local.xpkg.mk
-include build/makelib/controlplane.mk

local-deploy: build controlplane.up local.xpkg.deploy.provider.$(PROJECT_NAME)
	@$(INFO) running locally built provider
	@$(KUBECTL) wait crd providers.pkg.crossplane.io --for=create --timeout 5m
	@$(KUBECTL) wait provider.pkg $(PROJECT_NAME) --for condition=Healthy --for condition=Installed --for=create --timeout 5m
	@$(OK) running locally built provider

.PHONY: submodules fallthrough

# ====================================================================================
# Special Targets

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    local-deploy          Deploy the provider locally using Kind + Crossplane.

endef
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special

vendor: modules.download
vendor.check: modules.check
