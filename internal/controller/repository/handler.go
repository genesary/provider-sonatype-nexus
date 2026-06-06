package repository

import (
	"context"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// FormatHandler defines the operations for a specific repository format.
// Each format (maven, docker, npm, etc.) implements this interface.
// This pattern simplifies the main controller and makes it easy to add new
// formats.
type FormatHandler interface {
	// Observe checks if the repository exists and if it matches the desired state.
	// Returns (exists, upToDate) where:
	//   - exists: true if the repository exists in Nexus
	//   - upToDate: true if the existing repository matches the desired spec
	Observe(ctx context.Context, client nexus.Client, name string, repoType string, cr *repositoryv1alpha1.Repository) (exists bool, upToDate bool)

	// Create creates a new repository in Nexus.
	Create(ctx context.Context, client nexus.Client, cr *repositoryv1alpha1.Repository, repoType string) error

	// Update updates an existing repository in Nexus.
	Update(ctx context.Context, client nexus.Client, name string, cr *repositoryv1alpha1.Repository, repoType string) error

	// Delete deletes a repository from Nexus.
	Delete(ctx context.Context, client nexus.Client, name string, repoType string) error

	// SupportedTypes returns the repository types supported by this format.
	// Common types are: "hosted", "proxy", "group"
	SupportedTypes() []string
}

// handlers maps format names to their handlers.
var handlers = make(map[string]FormatHandler)

// RegisterHandler registers a format handler.
func RegisterHandler(format string, handler FormatHandler) {
	handlers[format] = handler
}

// GetHandler returns the handler for a given format.
func GetHandler(format string) FormatHandler {
	return handlers[format]
}

// init registers all format handlers.
func init() {
	// Core formats with full support (hosted, proxy, group)
	RegisterHandler("maven2", &MavenHandler{})
	RegisterHandler("docker", &DockerHandler{})
	RegisterHandler("npm", &NpmHandler{})
	RegisterHandler("raw", &RawHandler{})
	RegisterHandler("nuget", &NugetHandler{})
	RegisterHandler("pypi", &PypiHandler{})
	RegisterHandler("rubygems", &RubygemsHandler{})
	RegisterHandler("yum", &YumHandler{})
	RegisterHandler("r", &RHandler{})
	RegisterHandler("cargo", &CargoHandler{})
	RegisterHandler("bower", &BowerHandler{})

	// Formats with partial type support
	RegisterHandler("apt", &AptHandler{})       // hosted, proxy only
	RegisterHandler("helm", &HelmHandler{})     // hosted, proxy only
	RegisterHandler("go", &GoHandler{})         // proxy, group only
	RegisterHandler("gitlfs", &GitLfsHandler{}) // hosted only

	// Proxy-only formats
	RegisterHandler("cocoapods", &CocoapodsHandler{})
	RegisterHandler("conan", &ConanHandler{})
	RegisterHandler("conda", &CondaHandler{})
}
