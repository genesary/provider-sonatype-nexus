# ====================================================================================
# Setup Project
PROJECT_NAME := provider-sonatype-nexus
PROJECT_REPO := github.com/genesary/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 1.64
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

IMAGES = $(PROJECT_NAME)
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= ghcr.io/genesary
XPKG_REG_ORGS_NO_PROMOTE ?= ghcr.io/genesary
XPKGS = $(PROJECT_NAME)
-include build/makelib/xpkg.mk

# NOTE: we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.provider-sonatype-nexus: do.build.images

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE: the build submodule currently overrides XDG_CACHE_HOME in order to
# force the Helm 3 to use the .work/helm directory. This causes Go on Linux
# machines to use that directory as the build cache as well. We should adjust
# this behavior in the build submodule because it is also causing Linux users
# to duplicate their build cache, but for now we just make it easier to identify
# its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

go.mod.cachedir:
	@go env GOMODCACHE

# NOTE: we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(CROSSPLANE_CLI)

# Generate CRDs and DeepCopy methods using controller-gen.
generate.run:
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.3 object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.3 crd:crdVersions=v1 paths="./apis/..." output:crd:artifacts:config=package/crds

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

# ====================================================================================
# End to End Testing

CROSSPLANE_VERSION = 1.19.0
-include build/makelib/local.xpkg.mk
-include build/makelib/controlplane.mk

local-deploy: build controlplane.up local.xpkg.deploy.provider.$(PROJECT_NAME)
	@$(INFO) running locally built provider
	@$(KUBECTL) wait crd providers.pkg.crossplane.io --for=create --timeout 5m
	@$(KUBECTL) wait provider.pkg $(PROJECT_NAME) --for condition=Healthy --for condition=Installed --for=create --timeout 5m
	@$(OK) running locally built provider

.PHONY: submodules fallthrough run

# ====================================================================================
# Special Targets

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special
