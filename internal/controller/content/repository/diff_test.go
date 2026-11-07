package repository

import (
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// TestIsUpToDateIgnoresServerSideFields checks that the fields Nexus adds to
// its responses but the provider never submits are not reported as drift.
func TestIsUpToDateIgnoresServerSideFields(t *testing.T) {
	t.Parallel()

	desired := v395.AptHostedApiRepository{
		Name:    "apt-hosted",
		Online:  true,
		Storage: v395.HostedStorageAttributes{BlobStoreName: "default", StrictContentTypeValidation: true, WritePolicy: "ALLOW"},
		Apt:     v395.AptHostedRepositoriesAttributes{Distribution: "bookworm"},
	}

	observed := desired
	// Nexus decorates the response with a component block the provider never
	// sends.
	observed.Component = &v395.ComponentAttributes{ProprietaryComponents: new(false)}

	upToDate, _, err := isUpToDate(desired, observed)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: server-side fields must not count as drift")
	}
}

// TestIsUpToDateDetectsFormatSpecificDrift checks the fields the previous
// name-and-online comparison silently ignored.
func TestIsUpToDateDetectsFormatSpecificDrift(t *testing.T) {
	t.Parallel()

	base := v395.AptHostedApiRepository{
		Name:    "apt-hosted",
		Online:  true,
		Storage: v395.HostedStorageAttributes{BlobStoreName: "default", StrictContentTypeValidation: true, WritePolicy: "ALLOW"},
		Apt:     v395.AptHostedRepositoriesAttributes{Distribution: "bookworm"},
		Cleanup: &v395.CleanupPolicyAttributes{PolicyNames: []string{"weekly"}},
	}

	tests := []struct {
		name  string
		drift func(repo *v395.AptHostedApiRepository)
	}{
		{
			name:  "Distribution",
			drift: func(repo *v395.AptHostedApiRepository) { repo.Apt.Distribution = "trixie" },
		},
		{
			name:  "BlobStoreName",
			drift: func(repo *v395.AptHostedApiRepository) { repo.Storage.BlobStoreName = "other" },
		},
		{
			name: "StrictContentTypeValidation",
			drift: func(repo *v395.AptHostedApiRepository) {
				repo.Storage.StrictContentTypeValidation = false
			},
		},
		{
			name: "WritePolicy",
			drift: func(repo *v395.AptHostedApiRepository) {
				repo.Storage.WritePolicy = "ALLOW_ONCE"
			},
		},
		{
			name:  "Online",
			drift: func(repo *v395.AptHostedApiRepository) { repo.Online = false },
		},
		{
			name: "CleanupPolicies",
			drift: func(repo *v395.AptHostedApiRepository) {
				repo.Cleanup = &v395.CleanupPolicyAttributes{PolicyNames: []string{"daily"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			observed := base
			tt.drift(&observed)

			upToDate, _, err := isUpToDate(base, observed)
			if err != nil {
				t.Fatalf("isUpToDate() error = %v", err)
			}

			if upToDate {
				t.Errorf("isUpToDate() = true, want false: drift on %s was not detected", tt.name)
			}
		})
	}
}

// TestIsUpToDateIgnoresCredentials checks that credentials, which Nexus never
// returns, do not make every reconcile look like drift.
func TestIsUpToDateIgnoresCredentials(t *testing.T) {
	t.Parallel()

	desired := v395.AptProxyApiRepository{
		Name:   "apt-proxy",
		Online: true,
		Proxy:  v395.ProxyAttributes{RemoteUrl: "https://deb.debian.org", ContentMaxAge: 1440, MetadataMaxAge: 1440},
		HttpClient: v395.HttpClientAttributes{
			AutoBlock: new(true),
			Authentication: &v395.HttpClientConnectionAuthenticationAttributes{
				Type:     "username",
				Username: new("robot"),
				Password: new("s3cret"),
			},
		},
	}

	observed := desired
	observed.HttpClient.Authentication = nil

	upToDate, _, err := isUpToDate(desired, observed)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: credentials are never returned by Nexus")
	}
}

// TestIsUpToDateComparesRoutingRule checks that the routing rule assigned to a
// proxy repository is compared against the one Nexus reports.
func TestIsUpToDateComparesRoutingRule(t *testing.T) {
	t.Parallel()

	desired := v395.AptProxyRepositoryApiRequest{Name: "apt-proxy", RoutingRuleName: new("block-npm")}
	matching := v395.AptProxyApiRepository{Name: "apt-proxy", RoutingRuleName: new("block-npm")}

	upToDate, _, err := isUpToDate(desired, matching)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: the routing rule matches")
	}

	diverging := matching
	diverging.RoutingRuleName = new("something-else")

	upToDate, _, err = isUpToDate(desired, diverging)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if upToDate {
		t.Error("isUpToDate() = true, want false: a different routing rule is drift")
	}
}

// TestIsUpToDateComparesMembersUnordered checks that group members are matched
// as a set, because Nexus does not preserve the submitted order.
func TestIsUpToDateComparesMembersUnordered(t *testing.T) {
	t.Parallel()

	desired := v395.RawGroupApiRepository{
		Name:  "raw-group",
		Group: v395.GroupAttributes{MemberNames: []string{"raw-hosted", "raw-proxy"}},
	}

	reordered := v395.RawGroupApiRepository{
		Name:  "raw-group",
		Group: v395.GroupAttributes{MemberNames: []string{"raw-proxy", "raw-hosted"}},
	}

	upToDate, _, err := isUpToDate(desired, reordered)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: member order is not significant")
	}

	different := v395.RawGroupApiRepository{
		Name:  "raw-group",
		Group: v395.GroupAttributes{MemberNames: []string{"raw-hosted"}},
	}

	upToDate, _, err = isUpToDate(desired, different)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if upToDate {
		t.Error("isUpToDate() = true, want false: a missing member is drift")
	}
}

// TestIsUpToDateHandlesEmptyBlocks checks a spec that declares a block without
// setting anything in it. Nexus reports such a block as null, and demanding it
// back would leave the repository permanently drifted and rewritten on every
// reconcile.
func TestIsUpToDateHandlesEmptyBlocks(t *testing.T) {
	t.Parallel()

	withEmptyCleanup := v395.RawHostedApiRepository{
		Name:    "raw-hosted",
		Online:  true,
		Cleanup: &v395.CleanupPolicyAttributes{},
	}

	reportedWithoutCleanup := v395.RawHostedApiRepository{Name: "raw-hosted", Online: true}

	upToDate, _, err := isUpToDate(withEmptyCleanup, reportedWithoutCleanup)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: an empty cleanup block asks for nothing")
	}

	withPolicies := v395.RawHostedApiRepository{
		Name:    "raw-hosted",
		Online:  true,
		Cleanup: &v395.CleanupPolicyAttributes{PolicyNames: []string{"weekly"}},
	}

	upToDate, _, err = isUpToDate(withPolicies, reportedWithoutCleanup)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if upToDate {
		t.Error("isUpToDate() = true, want false: a requested cleanup policy is missing")
	}
}

// TestValueMatches covers the leaf comparison rules directly.
func TestValueMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want any
		got  any
		res  bool
	}{
		{name: "NullDesiredAlwaysMatches", want: nil, got: "anything", res: true},
		{name: "EmptyStringDesiredAlwaysMatches", want: "", got: "anything", res: true},
		{name: "EqualStrings", want: "a", got: "a", res: true},
		{name: "DifferentStrings", want: "a", got: "b", res: false},
		{name: "FalseIsCompared", want: false, got: true, res: false},
		{name: "ZeroIsCompared", want: float64(0), got: float64(1), res: false},
		{name: "MissingObservedValue", want: "a", got: nil, res: false},
		{name: "EmptyArrayAgainstMissing", want: []any{}, got: nil, res: true},
		{name: "EmptyArrayAgainstPopulated", want: []any{}, got: []any{"a"}, res: false},
		{name: "EmptyObjectAgainstMissing", want: map[string]any{}, got: nil, res: true},
		{name: "ObjectOfUnsetFieldsAgainstMissing", want: map[string]any{"policyNames": nil}, got: nil, res: true},
		{name: "ObjectOfSetFieldsAgainstMissing", want: map[string]any{"policyNames": []any{"weekly"}}, got: nil, res: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := valueMatches(tt.want, tt.got, "field"); got != tt.res {
				t.Errorf("valueMatches(%v, %v) = %v, want %v", tt.want, tt.got, got, tt.res)
			}
		})
	}
}
