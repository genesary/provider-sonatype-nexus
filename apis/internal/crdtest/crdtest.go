// Package crdtest checks objects against the generated
// CustomResourceDefinitions in tests.
package crdtest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"sigs.k8s.io/yaml"
)

// crdsPath is the directory of the generated CustomResourceDefinitions,
// relative to the API version packages.
const crdsPath = "../../../package/crds"

// Case is a spec.forProvider to check, and the error it must be rejected with.
type Case struct {
	// ForProvider is the spec.forProvider to check.
	ForProvider map[string]any
	// WantErr is part of the error expected, or empty when none is.
	WantErr string
}

// RunForProvider checks each case against the schema, CEL rules included, of
// the spec.forProvider of the CustomResourceDefinition in the given file. It
// must be called from an API version package.
func RunForProvider(t *testing.T, crdFile string, cases map[string]Case) {
	t.Helper()

	props := forProviderSchema(t, crdFile)

	structural, err := schema.NewStructural(props)
	if err != nil {
		t.Fatalf("building the structural schema: %v", err)
	}

	schemaValidator, _, err := validation.NewSchemaValidator(props)
	if err != nil {
		t.Fatalf("building the schema validator: %v", err)
	}

	celValidator := cel.NewValidator(structural, false, celconfig.PerCallLimit)

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := field.NewPath("forProvider")
			errs := validation.ValidateCustomResource(path, testCase.ForProvider, schemaValidator)

			// As the API server does, only evaluate the rules of objects that
			// match the schema the rules are written for.
			if len(errs) == 0 {
				errs, _ = celValidator.Validate(context.Background(), path, structural, testCase.ForProvider, nil, celconfig.RuntimeCELCostBudget)
			}

			switch {
			case testCase.WantErr == "" && len(errs) != 0:
				t.Fatalf("errors = %v, want none", errs)
			case testCase.WantErr != "" && (len(errs) == 0 || !strings.Contains(errs.ToAggregate().Error(), testCase.WantErr)):
				t.Fatalf("errors = %v, want %q", errs, testCase.WantErr)
			}
		})
	}
}

// forProviderSchema returns the spec.forProvider schema of the
// CustomResourceDefinition in the given file.
func forProviderSchema(t *testing.T, crdFile string) *apiextensions.JSONSchemaProps {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(crdsPath, crdFile)) //nolint:gosec // G304: tests name the file
	if err != nil {
		t.Fatalf("reading the CRD: %v", err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}

	err = yaml.Unmarshal(raw, crd)
	if err != nil {
		t.Fatalf("decoding the CRD: %v", err)
	}

	v1Props := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"]
	props := &apiextensions.JSONSchemaProps{}

	err = apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&v1Props, props, nil)
	if err != nil {
		t.Fatalf("converting the schema: %v", err)
	}

	return props
}
