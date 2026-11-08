package instance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// DefaultScriptType is the script language submitted when the spec sets none.
const DefaultScriptType = "groovy"

// ErrScriptingDisabled is wrapped by the error Create and Update return when
// Nexus refuses to store a script because script creation is disabled.
var ErrScriptingDisabled = errors.New("the Nexus scripting API is disabled: set nexus.scripts.allowCreation=true " +
	"in <nexus-data>/etc/nexus.properties and restart Nexus to manage scripts")

// ErrScriptTypeChanged is wrapped by the error returned when a script must
// change type: Nexus ignores the type of a script update and keeps the one
// the script was created with.
var ErrScriptTypeChanged = errors.New("nexus cannot change the type of a stored script; either recreate the resource " +
	"with the stored type, or delete the script in Nexus so that it is recreated with the spec type")

// ScriptClient provides methods for managing Nexus scripts.
type ScriptClient interface {
	Create(ctx context.Context, script v395.ScriptXO) error
	Delete(ctx context.Context, name string) error
	Get(ctx context.Context, name string) (*v395.ScriptXO, error)
	Update(ctx context.Context, name string, script v395.ScriptXO) error
}

// scriptClient implements ScriptClient over the Nexus API.
type scriptClient struct {
	api *v395.ScriptAPIService
}

// NewScriptClient returns a ScriptClient backed by a live Nexus connection.
func NewScriptClient(creds nexus.Credentials) (ScriptClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &scriptClient{api: nc.ScriptAPI}, nil
}

// Create stores a new script. When script creation is disabled in Nexus, the
// error wraps ErrScriptingDisabled.
func (c *scriptClient) Create(ctx context.Context, script v395.ScriptXO) error {
	return scriptWriteError(c.api.CreateScript(ctx).ScriptXO(script).Execute())
}

// Delete deletes the script with the given name.
func (c *scriptClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteScript(ctx, name).Execute())
}

// Get returns the script with the given name, or an error wrapping
// nexus.ErrNotFound when there is none.
func (c *scriptClient) Get(ctx context.Context, name string) (*v395.ScriptXO, error) {
	script, resp, err := c.api.GetScript(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return script, nil
}

// Update replaces the script with the given name. When script creation is
// disabled in Nexus, the error wraps ErrScriptingDisabled.
func (c *scriptClient) Update(ctx context.Context, name string, script v395.ScriptXO) error {
	return scriptWriteError(c.api.UpdateScript(ctx, name).ScriptXO(script).Execute())
}

// scriptWriteError turns the outcome of a script write into a provider error.
// Nexus answers 410 Gone when script creation is disabled, which is wrapped
// in ErrScriptingDisabled so that users are told how to enable it.
func scriptWriteError(resp *http.Response, err error) error {
	apiErr := nexus.APIError(resp, err)
	if apiErr != nil && resp != nil && resp.StatusCode == http.StatusGone {
		return errors.Wrap(ErrScriptingDisabled, apiErr.Error())
	}

	return apiErr
}

// GenerateScript builds the Nexus script from the CR spec.
func GenerateScript(params *instancev1alpha1.ScriptParameters) v395.ScriptXO {
	return v395.ScriptXO{
		Name:    params.Name,
		Type:    ScriptType(params),
		Content: params.Content,
	}
}

// ScriptType returns the script language the spec asks for, the default when
// unset.
func ScriptType(params *instancev1alpha1.ScriptParameters) string {
	if params.Type == "" {
		return DefaultScriptType
	}

	return params.Type
}

// IsScriptTypeUpToDate returns true when the observed script language is the
// one the spec asks for. The Groovy engine answers to both "groovy" and
// "Groovy", so a script adopted with either spelling matches.
func IsScriptTypeUpToDate(params *instancev1alpha1.ScriptParameters, observedType string) bool {
	return strings.EqualFold(observedType, ScriptType(params))
}

// IsScriptUpToDate returns true when the Nexus script matches the CR spec.
func IsScriptUpToDate(params *instancev1alpha1.ScriptParameters, observed *v395.ScriptXO) bool {
	return IsScriptTypeUpToDate(params, observed.Type) && observed.Content == params.Content
}

// GenerateScriptObservation converts a Nexus script to an observation. The
// source is recorded as its digest only.
func GenerateScriptObservation(observed *v395.ScriptXO) instancev1alpha1.ScriptObservation {
	return instancev1alpha1.ScriptObservation{
		Type:          observed.Type,
		ContentSHA256: ScriptContentSHA256(observed.Content),
	}
}

// ScriptContentSHA256 returns the hex encoded SHA-256 digest of a script
// source.
func ScriptContentSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])
}
