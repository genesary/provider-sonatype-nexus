// Package utils provides utility functions for the provider.
package utils

import "strings"

// IsNotFound checks if an error indicates a resource was not found.
// This handles various "not found" error formats from the Nexus API.
//
// Examples of detected errors:
//   - HTTP 404 responses
//   - Messages containing "not found"
//   - Messages containing "does not exist"
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

// IsConflict checks if an error indicates a resource conflict.
// This typically happens when trying to create a resource that already exists.
func IsConflict(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "409") ||
		strings.Contains(msg, "conflict") ||
		strings.Contains(msg, "already exists")
}
