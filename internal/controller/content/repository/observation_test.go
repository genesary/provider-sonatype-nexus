package repository

import (
	"slices"
	"testing"

	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"k8s.io/utils/ptr"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// observationOf renders a repository payload the way the controller does:
// through the generic JSON object the observation produced.
func observationOf(t *testing.T, repo any, url string) repositoryv1alpha1.RepositoryObservation {
	t.Helper()

	fields, err := repositoryFields(repo)
	if err != nil {
		t.Fatalf("repositoryFields() error = %v", err)
	}

	return generateRepositoryObservation(fields, url)
}

// TestGenerateRepositoryObservationHosted checks the fields a hosted
// repository reports.
func TestGenerateRepositoryObservationHosted(t *testing.T) {
	t.Parallel()

	repo := schema.AptHostedRepository{
		Name:   "apt-hosted",
		Online: true,
		Storage: schema.HostedStorage{
			BlobStoreName:               "default",
			StrictContentTypeValidation: true,
			WritePolicy:                 new(schema.StorageWritePolicyAllow),
		},
		Cleanup: &schema.Cleanup{PolicyNames: []string{"weekly"}},
	}

	got := observationOf(t, repo, "https://nexus.example.com/repository/apt-hosted")

	if got.Name != "apt-hosted" || got.BlobStoreName != "default" || got.WritePolicy != "ALLOW" {
		t.Errorf("observation = (%q, %q, %q), want (apt-hosted, default, ALLOW)", got.Name, got.BlobStoreName, got.WritePolicy)
	}

	if !ptr.Deref(got.Online, false) || !ptr.Deref(got.StrictContentTypeValidation, false) {
		t.Errorf("online = %v, strictContentTypeValidation = %v, want both true", got.Online, got.StrictContentTypeValidation)
	}

	if !slices.Equal(got.CleanupPolicyNames, []string{"weekly"}) {
		t.Errorf("cleanupPolicyNames = %v, want [weekly]", got.CleanupPolicyNames)
	}

	if got.RemoteURL != "" || got.MemberNames != nil {
		t.Errorf("hosted repository reported remoteUrl %q and members %v, want neither", got.RemoteURL, got.MemberNames)
	}
}

// TestGenerateRepositoryObservationProxy checks the fields a proxy repository
// reports, including the routing rule Nexus renames on the way out.
func TestGenerateRepositoryObservationProxy(t *testing.T) {
	t.Parallel()

	repo := schema.AptProxyRepository{
		Name:            "apt-proxy",
		Online:          false,
		Storage:         schema.Storage{BlobStoreName: "blobs", StrictContentTypeValidation: false},
		Proxy:           schema.Proxy{RemoteURL: "https://deb.debian.org"},
		RoutingRuleName: new("block-all"),
	}

	got := observationOf(t, repo, "https://nexus.example.com/repository/apt-proxy")

	if got.Name != "apt-proxy" || got.BlobStoreName != "blobs" || got.RemoteURL != "https://deb.debian.org" {
		t.Errorf("observation = (%q, %q, %q), want (apt-proxy, blobs, https://deb.debian.org)", got.Name, got.BlobStoreName, got.RemoteURL)
	}

	if got.RoutingRuleName != "block-all" {
		t.Errorf("routingRuleName = %q, want %q", got.RoutingRuleName, "block-all")
	}

	if ptr.Deref(got.Online, true) || ptr.Deref(got.StrictContentTypeValidation, true) {
		t.Errorf("online = %v, strictContentTypeValidation = %v, want both false and reported", got.Online, got.StrictContentTypeValidation)
	}

	if got.WritePolicy != "" || got.MemberNames != nil {
		t.Errorf("proxy repository reported writePolicy %q and members %v, want neither", got.WritePolicy, got.MemberNames)
	}
}

// TestGenerateRepositoryObservationGroup checks that a group repository
// reports its members.
func TestGenerateRepositoryObservationGroup(t *testing.T) {
	t.Parallel()

	repo := schema.RawGroupRepository{
		Name:    "raw-group",
		Online:  true,
		Storage: schema.Storage{BlobStoreName: "default"},
		Group:   schema.Group{MemberNames: []string{"raw-hosted", "raw-proxy"}},
	}

	got := observationOf(t, repo, "https://nexus.example.com/repository/raw-group")

	if !slices.Equal(got.MemberNames, []string{"raw-hosted", "raw-proxy"}) {
		t.Errorf("memberNames = %v, want [raw-hosted raw-proxy]", got.MemberNames)
	}

	if got.RemoteURL != "" {
		t.Errorf("group repository reported remoteUrl %q, want none", got.RemoteURL)
	}
}

// TestGenerateRepositoryObservationToleratesMissingFields checks that a sparse
// or absent payload does not panic and reports nothing rather than zero values
// that would read as observed facts.
func TestGenerateRepositoryObservationToleratesMissingFields(t *testing.T) {
	t.Parallel()

	got := generateRepositoryObservation(nil, "")
	if got.URL != nil || got.Name != "" || got.Online != nil || got.CleanupPolicyNames != nil {
		t.Errorf("observation of nothing = %+v, want an empty observation", got)
	}

	// A payload whose blocks are present but null must be read the same way.
	sparse := map[string]any{"name": "repo", "storage": nil, "cleanup": nil, "group": nil, "proxy": nil}

	got = generateRepositoryObservation(sparse, "https://nexus.example.com/repository/repo")
	if got.Name != "repo" || got.BlobStoreName != "" || got.MemberNames != nil {
		t.Errorf("observation of a sparse payload = %+v, want only the name", got)
	}

	if got.URL == nil {
		t.Error("URL was not reported")
	}
}
