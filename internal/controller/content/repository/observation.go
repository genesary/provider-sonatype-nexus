package repository

import (
	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// generateRepositoryObservation renders the repository Nexus reported as the
// observed state of the managed resource.
//
// It reads the generic JSON object the observation produced rather than a
// format specific type, so every format and repository type reports the same
// fields, and a field simply stays empty where the format does not have it -
// a hosted repository has no remote URL, a proxy repository has no members.
//
// The URL is passed in rather than read from the repository: go-nexus-client
// drops the url field Nexus returns, and Nexus always serves a repository at
// <base>/repository/<name>.
func generateRepositoryObservation(fields map[string]any, url string) repositoryv1alpha1.RepositoryObservation {
	storage := objectField(fields, "storage")

	observation := repositoryv1alpha1.RepositoryObservation{
		Name:                        stringField(fields, "name"),
		Online:                      boolField(fields, "online"),
		BlobStoreName:               stringField(storage, "blobStoreName"),
		StrictContentTypeValidation: boolField(storage, "strictContentTypeValidation"),
		WritePolicy:                 stringField(storage, "writePolicy"),
		CleanupPolicyNames:          stringSliceField(objectField(fields, "cleanup"), "policyNames"),
		RemoteURL:                   stringField(objectField(fields, "proxy"), "remoteUrl"),
		MemberNames:                 stringSliceField(objectField(fields, "group"), "memberNames"),
		RoutingRuleName:             stringField(fields, "routingRuleName"),
	}

	if url != "" {
		observation.URL = &url
	}

	return observation
}

// objectField returns the nested object at field, or nil when the field is
// absent or is not an object. Indexing the nil result is safe, so callers can
// chain lookups without checking.
func objectField(fields map[string]any, field string) map[string]any {
	nested, isObject := fields[field].(map[string]any)
	if !isObject {
		return nil
	}

	return nested
}

// stringField returns the string at field, empty when absent or not a string.
func stringField(fields map[string]any, field string) string {
	value, isString := fields[field].(string)
	if !isString {
		return ""
	}

	return value
}

// boolField returns a pointer to the boolean at field, nil when absent or not
// a boolean, so that "false" and "not reported" stay distinguishable.
func boolField(fields map[string]any, field string) *bool {
	value, isBool := fields[field].(bool)
	if !isBool {
		return nil
	}

	return &value
}

// stringSliceField returns the array of strings at field, nil when absent, not
// an array, or holding anything that is not a string.
func stringSliceField(fields map[string]any, field string) []string {
	values, isArray := fields[field].([]any)
	if !isArray {
		return nil
	}

	strings, allStrings := jsonStrings(values)
	if !allStrings || len(strings) == 0 {
		return nil
	}

	return strings
}
