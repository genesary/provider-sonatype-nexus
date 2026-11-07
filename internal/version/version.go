// Package version holds the version of the provider, set at build time.
package version

// Version is the version of the provider. The Makefile sets it through
// -ldflags "-X .../internal/version.Version=$(VERSION)".
var Version = "dev"
