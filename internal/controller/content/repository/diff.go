package repository

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// unmanagedPaths lists dotted JSON paths of a repository payload that are
// excluded from drift detection because Nexus never echoes back what the
// provider sent, which would otherwise report drift on every single reconcile:
//
//   - the signing keys and the HTTP client credentials are redacted or dropped
//     by the API, as any credential should be;
//   - the npm block is accepted on write but left out of the read response
//     from Nexus 3.66 onwards, where the setting moved to the firewall
//     configuration. It is still submitted so it keeps working where Nexus
//     honours it, but it cannot be read back and so cannot be diffed.
var unmanagedPaths = map[string]struct{}{
	"aptSigning":                {},
	"yumSigning":                {},
	"httpClient.authentication": {},
	"npm":                       {},
}

// observedFieldAliases maps a field the provider writes to the differently
// named field Nexus reports it back under. Routing rules are submitted as
// "routingRule" but read back as "routingRuleName" (NEXUS-30973).
var observedFieldAliases = map[string]string{
	"routingRule": "routingRuleName",
}

// isUpToDate reports whether the repository Nexus returned already holds every
// value the provider intends to set. It also returns that repository rendered
// as a generic JSON object, so the caller can record the observed state
// without knowing the concrete format type.
//
// The comparison is deliberately one-way. Nexus decorates its responses with
// fields the provider never submits - format, type, url, component, firewall,
// server side defaults - and a symmetric comparison would report those as
// permanent drift. Only the fields the desired payload actually carries are
// asserted, which makes drift detection complete for every managed field and
// stable for everything else.
func isUpToDate[T any](desired, observed T) (upToDate bool, observedFields map[string]any, err error) {
	desiredFields, err := repositoryFields(desired)
	if err != nil {
		return false, nil, errors.Wrap(err, "cannot encode the desired repository")
	}

	observedFields, err = repositoryFields(observed)
	if err != nil {
		return false, nil, errors.Wrap(err, "cannot encode the observed repository")
	}

	return fieldsMatch(desiredFields, observedFields, ""), observedFields, nil
}

// repositoryFields renders a repository payload as a generic JSON object, so
// that drift detection does not have to know the concrete format type.
func repositoryFields(repo any) (map[string]any, error) {
	encoded, err := json.Marshal(repo)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}

	err = json.Unmarshal(encoded, &fields)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

// fieldsMatch reports whether every managed field of desired is matched by
// observed. Path is the dotted path of the object being compared, empty at the
// root of the payload.
func fieldsMatch(desired, observed map[string]any, path string) bool {
	for field, want := range desired {
		fieldPath := joinPath(path, field)

		_, unmanaged := unmanagedPaths[fieldPath]
		if unmanaged {
			continue
		}

		if !valueMatches(want, observed[observedField(field)], fieldPath) {
			return false
		}
	}

	return true
}

// joinPath appends a field name to the dotted path of its parent object.
func joinPath(path, field string) string {
	if path == "" {
		return field
	}

	return path + "." + field
}

// observedField returns the name Nexus reports the given payload field under.
func observedField(field string) string {
	alias, aliased := observedFieldAliases[field]
	if aliased {
		return alias
	}

	return field
}

// valueMatches reports whether an observed value satisfies the desired one.
// Desired values that carry no intent - null and the empty string - always
// match, because the provider only omits a value when the spec left it unset.
// Empty arrays and false and zero do carry intent and are compared.
func valueMatches(want, got any, path string) bool {
	switch wanted := want.(type) {
	case nil:
		return true
	case string:
		return wanted == "" || got == want
	case map[string]any:
		// A block Nexus does not report is compared against nothing rather
		// than rejected outright: a spec may declare a block whose fields are
		// all unset, and that asks for nothing.
		observed, _ := got.(map[string]any)

		return fieldsMatch(wanted, observed, path)
	case []any:
		observed, isArray := got.([]any)
		if !isArray {
			return len(wanted) == 0
		}

		return arrayMatches(wanted, observed, path)
	default:
		return got == want
	}
}

// arrayMatches reports whether two JSON arrays hold the same values. Arrays of
// strings - group member names, cleanup policy names - are compared as
// unordered collections, because Nexus does not preserve submission order.
func arrayMatches(want, got []any, path string) bool {
	if len(want) != len(got) {
		return false
	}

	wantStrings, wantIsStrings := jsonStrings(want)

	gotStrings, gotIsStrings := jsonStrings(got)
	if wantIsStrings && gotIsStrings {
		return helpers.AreStringSlicesEqual(wantStrings, gotStrings)
	}

	for index := range want {
		if !valueMatches(want[index], got[index], path) {
			return false
		}
	}

	return true
}

// jsonStrings converts a JSON array to a string slice, reporting false when
// any element is not a string.
func jsonStrings(values []any) ([]string, bool) {
	strings := make([]string, 0, len(values))

	for _, value := range values {
		str, isString := value.(string)
		if !isString {
			return nil, false
		}

		strings = append(strings, str)
	}

	return strings, true
}
