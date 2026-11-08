package controller

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/genesary/provider-sonatype-nexus/internal/controller/crdgate"
)

// TestManagedKinds checks that every managed resource controller is gated on
// the name of a CustomResourceDefinition the package ships, and that every
// managed resource definition the package ships has a controller.
func TestManagedKinds(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join("..", "..", "package", "crds", "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("cannot list the packaged CustomResourceDefinitions: %v", err)
	}

	// The files are named <group>_<plural>.yaml. The ProviderConfig
	// definitions, in the top level group, are not gated.
	shipped := make(map[string]bool, len(files))

	for _, file := range files {
		group, plural, found := strings.Cut(strings.TrimSuffix(filepath.Base(file), ".yaml"), "_")
		if found && group != "nexus.crossplane.io" {
			shipped[plural+"."+group] = true
		}
	}

	for _, gvk := range managedKinds() {
		name := crdgate.CRDName(gvk)
		if !shipped[name] {
			t.Errorf("CRDName(%s) = %s, which the package does not ship", gvk, name)
		}

		delete(shipped, name)
	}

	for name := range shipped {
		t.Errorf("CustomResourceDefinition %s has no managed resource controller", name)
	}
}
