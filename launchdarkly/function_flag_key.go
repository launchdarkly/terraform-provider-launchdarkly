package launchdarkly

// Phase 6.3 scaffold: provider::launchdarkly::flag_key — an additive
// helper function exposed via the framework `function.Function`
// interface. Lets HCL authors compose project/flag IDs without manual
// string concatenation:
//
//   composite_id = provider::launchdarkly::flag_key("my-project", "my-flag")
//   # => "my-project/my-flag"
//
// Per MIGRATION_PLAN_NON_BREAKING.md §Phase 6.3, provider functions
// are framework-only (TF 1.8+, framework 1.8+). Cannot ship until
// Phase 5 cutover lands.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &FlagKeyFunction{}

type FlagKeyFunction struct{}

func NewFlagKeyFunction() function.Function {
	return &FlagKeyFunction{}
}

func (f *FlagKeyFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "flag_key"
}

func (f *FlagKeyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Compose a LaunchDarkly flag composite ID",
		Description: "Returns a string in the form `<project_key>/<flag_key>` suitable for use as a flag composite ID.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "project_key", Description: "Project key."},
			function.StringParameter{Name: "flag_key", Description: "Flag key."},
		},
		Return: function.StringReturn{},
	}
}

func (f *FlagKeyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var projectKey, flagKey string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &projectKey, &flagKey))
	if resp.Error != nil {
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, projectKey+"/"+flagKey))
}
