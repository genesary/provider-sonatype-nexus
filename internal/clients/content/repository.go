package content

import (
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// NewRepositoryClient creates a RepositoryClient from credentials.
func NewRepositoryClient(creds nexus.Credentials) (*repository.RepositoryService, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Repository, nil
}
