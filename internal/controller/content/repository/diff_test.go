package repository

import (
	"testing"

	schema "github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
)

// TestIsUpToDateIgnoresServerSideFields checks that the fields Nexus adds to
// its responses but the provider never submits are not reported as drift.
func TestIsUpToDateIgnoresServerSideFields(t *testing.T) {
	t.Parallel()

	desired := schema.AptHostedRepository{
		Name:    "apt-hosted",
		Online:  true,
		Storage: schema.HostedStorage{BlobStoreName: "default", StrictContentTypeValidation: true, WritePolicy: new(schema.StorageWritePolicyAllow)},
		Apt:     schema.AptHosted{Distribution: "bookworm"},
	}

	observed := desired
	// Nexus decorates the response with a component block the provider never
	// sends.
	observed.Component = &schema.Component{ProprietaryComponents: false}

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

	base := schema.AptHostedRepository{
		Name:    "apt-hosted",
		Online:  true,
		Storage: schema.HostedStorage{BlobStoreName: "default", StrictContentTypeValidation: true, WritePolicy: new(schema.StorageWritePolicyAllow)},
		Apt:     schema.AptHosted{Distribution: "bookworm"},
		Cleanup: &schema.Cleanup{PolicyNames: []string{"weekly"}},
	}

	tests := []struct {
		name  string
		drift func(repo *schema.AptHostedRepository)
	}{
		{
			name:  "Distribution",
			drift: func(repo *schema.AptHostedRepository) { repo.Apt.Distribution = "trixie" },
		},
		{
			name:  "BlobStoreName",
			drift: func(repo *schema.AptHostedRepository) { repo.Storage.BlobStoreName = "other" },
		},
		{
			name: "StrictContentTypeValidation",
			drift: func(repo *schema.AptHostedRepository) {
				repo.Storage.StrictContentTypeValidation = false
			},
		},
		{
			name: "WritePolicy",
			drift: func(repo *schema.AptHostedRepository) {
				repo.Storage.WritePolicy = new(schema.StorageWritePolicyAllowDeny)
			},
		},
		{
			name:  "Online",
			drift: func(repo *schema.AptHostedRepository) { repo.Online = false },
		},
		{
			name: "CleanupPolicies",
			drift: func(repo *schema.AptHostedRepository) {
				repo.Cleanup = &schema.Cleanup{PolicyNames: []string{"daily"}}
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

	desired := schema.AptProxyRepository{
		Name:   "apt-proxy",
		Online: true,
		Proxy:  schema.Proxy{RemoteURL: "https://deb.debian.org", ContentMaxAge: 1440, MetadataMaxAge: 1440},
		HTTPClient: schema.HTTPClient{
			AutoBlock: true,
			Authentication: &schema.HTTPClientAuthentication{
				Type:     schema.HTTPClientAuthenticationTypeUsername,
				Username: "robot",
				Password: "s3cret",
			},
		},
	}

	observed := desired
	observed.Authentication = nil

	upToDate, _, err := isUpToDate(desired, observed)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: credentials are never returned by Nexus")
	}
}

// TestIsUpToDateReadsRoutingRuleAlias checks that a routing rule submitted as
// routingRule is compared against the routingRuleName Nexus reports.
func TestIsUpToDateReadsRoutingRuleAlias(t *testing.T) {
	t.Parallel()

	desired := schema.AptProxyRepository{Name: "apt-proxy", RoutingRule: new("block-npm")}

	matching := desired
	matching.RoutingRule = nil
	matching.RoutingRuleName = new("block-npm")

	upToDate, _, err := isUpToDate(desired, matching)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: routingRule is reported back as routingRuleName")
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

	desired := schema.RawGroupRepository{
		Name:  "raw-group",
		Group: schema.Group{MemberNames: []string{"raw-hosted", "raw-proxy"}},
	}

	reordered := schema.RawGroupRepository{
		Name:  "raw-group",
		Group: schema.Group{MemberNames: []string{"raw-proxy", "raw-hosted"}},
	}

	upToDate, _, err := isUpToDate(desired, reordered)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: member order is not significant")
	}

	different := schema.RawGroupRepository{
		Name:  "raw-group",
		Group: schema.Group{MemberNames: []string{"raw-hosted"}},
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

	withEmptyCleanup := schema.RawHostedRepository{
		Name:    "raw-hosted",
		Online:  true,
		Cleanup: &schema.Cleanup{},
	}

	reportedWithoutCleanup := schema.RawHostedRepository{Name: "raw-hosted", Online: true}

	upToDate, _, err := isUpToDate(withEmptyCleanup, reportedWithoutCleanup)
	if err != nil {
		t.Fatalf("isUpToDate() error = %v", err)
	}

	if !upToDate {
		t.Error("isUpToDate() = false, want true: an empty cleanup block asks for nothing")
	}

	withPolicies := schema.RawHostedRepository{
		Name:    "raw-hosted",
		Online:  true,
		Cleanup: &schema.Cleanup{PolicyNames: []string{"weekly"}},
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
