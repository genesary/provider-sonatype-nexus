package content

import (
	"context"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// ScriptClient is the interface for managing Nexus Groovy scripts.
type ScriptClient interface {
	GetScript(ctx context.Context, name string) (*schema.Script, error)
	CreateScript(ctx context.Context, script *schema.Script) error
	UpdateScript(ctx context.Context, script *schema.Script) error
	DeleteScript(ctx context.Context, name string) error
}

// scriptClientImpl wraps a ScriptService, adding a list-fallback for Get.
type scriptClientImpl struct {
	nexus.ScriptService
}

// GetScript fetches a script by name, falling back to list-scan when the
// direct GET returns 404 (some Nexus versions don't support GET by name).
func (c *scriptClientImpl) GetScript(ctx context.Context, name string) (*schema.Script, error) {
	s, err := c.ScriptService.GetScript(ctx, name)
	if err == nil {
		return s, nil
	}

	if !helpers.IsNotFound(err) {
		return nil, err
	}

	all, listErr := c.ListScripts(ctx)
	if listErr != nil {
		return nil, err
	}

	for i := range all {
		if all[i].Name == name {
			return &all[i], nil
		}
	}

	return nil, err
}

// NewScriptClient creates a ScriptClient backed by a live Nexus connection.
func NewScriptClient(creds nexus.Credentials) (ScriptClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &scriptClientImpl{nexusClient.Script()}, nil
}

// GenerateScript builds the nexus3 Schema Script from a Script CR.
func GenerateScript(cr *contentv1alpha1.Script) *schema.Script {
	return &schema.Script{
		Name:    cr.Spec.ForProvider.Name,
		Type:    cr.Spec.ForProvider.Type,
		Content: cr.Spec.ForProvider.Content,
	}
}

// IsScriptUpToDate returns true when the live script matches the desired spec.
func IsScriptUpToDate(cr *contentv1alpha1.Script, observed *schema.Script) bool {
	p := cr.Spec.ForProvider

	return p.Type == observed.Type && p.Content == observed.Content
}

// GenerateScriptObservation creates an observation from a live script.
func GenerateScriptObservation(observed *schema.Script) contentv1alpha1.ScriptObservation {
	if observed == nil {
		return contentv1alpha1.ScriptObservation{}
	}

	return contentv1alpha1.ScriptObservation{
		Name:    observed.Name,
		Type:    observed.Type,
		Content: observed.Content,
	}
}

// IsForbidden reports whether the error indicates that the Nexus Scripting API
// is disabled (HTTP 403). This happens when nexus.scripts.allowCreation=false.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "403") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "access denied")
}
