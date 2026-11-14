package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/xeipuuv/gojsonschema"
)

// jsonStringValidator rejects strings that are not valid JSON.
type jsonStringValidator struct{}

func (jsonStringValidator) Description(context.Context) string         { return "must be valid JSON" }
func (jsonStringValidator) MarkdownDescription(context.Context) string { return "must be valid JSON" }
func (jsonStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if s == "" {
		return
	}
	var js interface{}
	if err := json.Unmarshal([]byte(s), &js); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("%q: invalid JSON: %s", req.Path, err))
	}
}

// jsonSchemaStringValidator rejects strings that are not valid JSON Schema.
type jsonSchemaStringValidator struct{}

func (jsonSchemaStringValidator) Description(context.Context) string {
	return "must be valid JSON Schema"
}
func (jsonSchemaStringValidator) MarkdownDescription(context.Context) string {
	return "must be valid JSON Schema"
}
func (jsonSchemaStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	(jsonStringValidator{}).ValidateString(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if s == "" {
		return
	}
	loader := gojsonschema.NewStringLoader(s)
	if _, err := gojsonschema.NewSchema(loader); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON Schema", fmt.Sprintf("%q: invalid JSON Schema: %s", req.Path, err))
	}
}

// jsonNormalizePlanModifier suppresses diffs caused by semantically
// equivalent JSON. When the state already has a semantically-equal
// value but a different textual form, re-use the state value so
// terraform doesn't show spurious diffs. terraform-core enforces
// plan-equals-config for Required attributes, so we can only swap in
// the state value, not rewrite the plan to a canonical form.
type jsonNormalizePlanModifier struct{}

func (jsonNormalizePlanModifier) Description(context.Context) string {
	return "Suppress diffs caused by semantically equivalent JSON"
}
func (jsonNormalizePlanModifier) MarkdownDescription(context.Context) string {
	return "Suppress diffs caused by semantically equivalent JSON"
}
func (jsonNormalizePlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	old := req.StateValue.ValueString()
	newV := req.PlanValue.ValueString()
	if old == "" && newV == "" {
		return
	}
	if old == "" || newV == "" {
		return
	}
	var oldJSON, newJSON interface{}
	if err := json.Unmarshal([]byte(old), &oldJSON); err != nil {
		return
	}
	if err := json.Unmarshal([]byte(newV), &newJSON); err != nil {
		return
	}
	if reflect.DeepEqual(oldJSON, newJSON) {
		resp.PlanValue = req.StateValue
	}
}
