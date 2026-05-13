package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &FeatureFlagEnvironmentDataSource{}

type FeatureFlagEnvironmentDataSource struct {
	client *Client
}

type FeatureFlagEnvironmentDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	FlagID         types.String `tfsdk:"flag_id"`
	EnvKey         types.String `tfsdk:"env_key"`
	On             types.Bool   `tfsdk:"on"`
	Targets        types.Set    `tfsdk:"targets"`
	ContextTargets types.Set    `tfsdk:"context_targets"`
	Rules          types.List   `tfsdk:"rules"`
	Prerequisites  types.List   `tfsdk:"prerequisites"`
	Fallthrough    types.List   `tfsdk:"fallthrough"`
	TrackEvents    types.Bool   `tfsdk:"track_events"`
	OffVariation   types.Int64  `tfsdk:"off_variation"`
}

var ffeTargetAttrTypes = map[string]attr.Type{
	VALUES:    types.ListType{ElemType: types.StringType},
	VARIATION: types.Int64Type,
}

var ffeContextTargetAttrTypes = map[string]attr.Type{
	VALUES:       types.ListType{ElemType: types.StringType},
	VARIATION:    types.Int64Type,
	CONTEXT_KIND: types.StringType,
}

var ffePrerequisiteAttrTypes = map[string]attr.Type{
	FLAG_KEY:  types.StringType,
	VARIATION: types.Int64Type,
}

var ffeRuleAttrTypes = map[string]attr.Type{
	DESCRIPTION:     types.StringType,
	CLAUSES:         types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkClauseAttrTypes}},
	VARIATION:       types.Int64Type,
	ROLLOUT_WEIGHTS: types.ListType{ElemType: types.Int64Type},
	BUCKET_BY:       types.StringType,
	CONTEXT_KIND:    types.StringType,
}

var ffeFallthroughAttrTypes = map[string]attr.Type{
	VARIATION:       types.Int64Type,
	ROLLOUT_WEIGHTS: types.ListType{ElemType: types.Int64Type},
	BUCKET_BY:       types.StringType,
	CONTEXT_KIND:    types.StringType,
}

func NewFeatureFlagEnvironmentDataSource() datasource.DataSource {
	return &FeatureFlagEnvironmentDataSource{}
}

func (d *FeatureFlagEnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_environment"
}

func (d *FeatureFlagEnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment-specific feature flag data source.\n\nThis data source allows you to retrieve environment-specific feature flag information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, Description: "Composite ID `project_key/env_key/flag_key`."},
			FLAG_ID:      schema.StringAttribute{Required: true, Description: "Flag ID in the format `project_key/flag_key`."},
			ENV_KEY:      schema.StringAttribute{Required: true, Description: "The environment key."},
			ON:           schema.BoolAttribute{Computed: true, Description: "Whether targeting is enabled."},
			TRACK_EVENTS: schema.BoolAttribute{Computed: true, Description: "Whether to send event data back to LaunchDarkly."},
			OFF_VARIATION: schema.Int64Attribute{
				Computed:    true,
				Description: "Variation index to serve when targeting is disabled.",
			},
		},
		Blocks: map[string]schema.Block{
			TARGETS: schema.SetNestedBlock{
				Description: "Individual user targets per variation.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES: schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						VARIATION: schema.Int64Attribute{Computed: true},
					},
				},
			},
			CONTEXT_TARGETS: schema.SetNestedBlock{
				Description: "Individual context-kind targets per variation.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES: schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						VARIATION:    schema.Int64Attribute{Computed: true},
						CONTEXT_KIND: schema.StringAttribute{Computed: true},
					},
				},
			},
			PREREQUISITES: schema.ListNestedBlock{
				Description: "Prerequisite flag rules.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FLAG_KEY:  schema.StringAttribute{Computed: true},
						VARIATION: schema.Int64Attribute{Computed: true},
					},
				},
			},
			RULES: schema.ListNestedBlock{
				Description: "Logical targeting rules.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						DESCRIPTION:  schema.StringAttribute{Computed: true},
						VARIATION:    schema.Int64Attribute{Computed: true},
						BUCKET_BY:    schema.StringAttribute{Computed: true},
						CONTEXT_KIND: schema.StringAttribute{Computed: true},
						ROLLOUT_WEIGHTS: schema.ListAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
					},
					Blocks: map[string]schema.Block{
						CLAUSES: frameworkClausesDataSourceBlock(),
					},
				},
			},
			FALLTHROUGH: schema.ListNestedBlock{
				Description: "Default variation served when no other targeting applies (single element).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VARIATION:    schema.Int64Attribute{Computed: true},
						BUCKET_BY:    schema.StringAttribute{Computed: true},
						CONTEXT_KIND: schema.StringAttribute{Computed: true},
						ROLLOUT_WEIGHTS: schema.ListAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
	}
}

func (d *FeatureFlagEnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *FeatureFlagEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data FeatureFlagEnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flagID := data.FlagID.ValueString()
	projectKey, flagKey, err := flagIdToKeys(flagID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid flag_id", err.Error())
		return
	}
	envKey := data.EnvKey.ValueString()
	if envKey == "" {
		resp.Diagnostics.AddError("env_key is required", "env_key must be set on the data source.")
		return
	}

	envExists, err := environmentExists(projectKey, envKey, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check environment existence", err.Error())
		return
	}
	if !envExists {
		resp.Diagnostics.AddError("Environment not found", fmt.Sprintf("Environment %q in project %q does not exist.", envKey, projectKey))
		return
	}

	flag, res, err := getFeatureFlagEnvironment(d.client, projectKey, flagKey, envKey)
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Flag not found", fmt.Sprintf("Flag %q in project %q not found.", flagKey, projectKey))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get flag", err)
		return
	}
	if flag.Environments == nil {
		resp.Diagnostics.AddError("Flag environments missing", fmt.Sprintf("Flag %q returned no environments map.", flagKey))
		return
	}
	environment, ok := (*flag.Environments)[envKey]
	if !ok {
		resp.Diagnostics.AddError("Environment not found on flag", fmt.Sprintf("Environment %q not present on flag %q.", envKey, flagKey))
		return
	}

	data.ID = types.StringValue(projectKey + "/" + envKey + "/" + flagKey)
	data.FlagID = types.StringValue(projectKey + "/" + flag.Key)
	data.On = types.BoolValue(environment.On)
	data.TrackEvents = types.BoolValue(environment.TrackEvents)
	if environment.OffVariation != nil {
		data.OffVariation = types.Int64Value(int64(*environment.OffVariation))
	} else {
		data.OffVariation = types.Int64Value(0)
	}

	// targets / context_targets
	data.Targets = ffeTargetsValue(ctx, environment.Targets, false, &resp.Diagnostics)
	data.ContextTargets = ffeTargetsValue(ctx, environment.ContextTargets, true, &resp.Diagnostics)

	// prerequisites
	prereqObjectType := types.ObjectType{AttrTypes: ffePrerequisiteAttrTypes}
	prereqElements := make([]attr.Value, 0, len(environment.Prerequisites))
	for _, p := range environment.Prerequisites {
		obj, d := types.ObjectValue(ffePrerequisiteAttrTypes, map[string]attr.Value{
			FLAG_KEY:  types.StringValue(p.Key),
			VARIATION: types.Int64Value(int64(p.Variation)),
		})
		resp.Diagnostics.Append(d...)
		prereqElements = append(prereqElements, obj)
	}
	prereqList, diags := types.ListValue(prereqObjectType, prereqElements)
	resp.Diagnostics.Append(diags...)
	data.Prerequisites = prereqList

	// rules
	data.Rules = ffeRulesValue(ctx, environment.Rules, &resp.Diagnostics)

	// fallthrough
	data.Fallthrough = ffeFallthroughValue(ctx, environment.Fallthrough, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func ffeTargetsValue(ctx context.Context, targets []ldapi.Target, isContextTarget bool, diags interface {
	AddError(string, string)
},
) types.Set {
	var objectType types.ObjectType
	if isContextTarget {
		objectType = types.ObjectType{AttrTypes: ffeContextTargetAttrTypes}
	} else {
		objectType = types.ObjectType{AttrTypes: ffeTargetAttrTypes}
	}
	elements := make([]attr.Value, 0, len(targets))
	for _, t := range targets {
		// SDKv2 filters out user-context "phantom" targets from context_targets.
		if isContextTarget && t.ContextKind != nil && *t.ContextKind == "user" {
			continue
		}
		valuesList, _ := listFromStringSlice(ctx, t.Values)
		if isContextTarget {
			contextKind := ""
			if t.ContextKind != nil {
				contextKind = *t.ContextKind
			}
			obj, _ := types.ObjectValue(ffeContextTargetAttrTypes, map[string]attr.Value{
				VALUES:       valuesList,
				VARIATION:    types.Int64Value(int64(t.Variation)),
				CONTEXT_KIND: types.StringValue(contextKind),
			})
			elements = append(elements, obj)
		} else {
			obj, _ := types.ObjectValue(ffeTargetAttrTypes, map[string]attr.Value{
				VALUES:    valuesList,
				VARIATION: types.Int64Value(int64(t.Variation)),
			})
			elements = append(elements, obj)
		}
	}
	set, _ := types.SetValue(objectType, elements)
	return set
}

func ffeRulesValue(ctx context.Context, rules []ldapi.Rule, diags interface {
	AddError(string, string)
},
) types.List {
	objectType := types.ObjectType{AttrTypes: ffeRuleAttrTypes}
	elements := make([]attr.Value, 0, len(rules))
	for _, r := range rules {
		clauses, _ := frameworkClausesValue(ctx, r.Clauses)
		variation := int64(0)
		if r.Variation != nil {
			variation = int64(*r.Variation)
		}
		bucketBy := ""
		contextKind := ""
		rolloutWeights := []attr.Value{}
		if r.Rollout != nil {
			for _, w := range r.Rollout.Variations {
				rolloutWeights = append(rolloutWeights, types.Int64Value(int64(w.Weight)))
			}
			if r.Rollout.BucketBy != nil {
				bucketBy = *r.Rollout.BucketBy
			}
			if r.Rollout.ContextKind != nil {
				contextKind = *r.Rollout.ContextKind
			}
		}
		weightsList, _ := types.ListValue(types.Int64Type, rolloutWeights)
		description := ""
		if r.Description != nil {
			description = *r.Description
		}
		obj, _ := types.ObjectValue(ffeRuleAttrTypes, map[string]attr.Value{
			DESCRIPTION:     types.StringValue(description),
			CLAUSES:         clauses,
			VARIATION:       types.Int64Value(variation),
			ROLLOUT_WEIGHTS: weightsList,
			BUCKET_BY:       types.StringValue(bucketBy),
			CONTEXT_KIND:    types.StringValue(contextKind),
		})
		elements = append(elements, obj)
	}
	list, _ := types.ListValue(objectType, elements)
	return list
}

func ffeFallthroughValue(ctx context.Context, fallthroughRep *ldapi.VariationOrRolloutRep, diags interface {
	AddError(string, string)
},
) types.List {
	objectType := types.ObjectType{AttrTypes: ffeFallthroughAttrTypes}
	if fallthroughRep == nil {
		list, _ := types.ListValue(objectType, []attr.Value{})
		return list
	}
	variation := int64(0)
	if fallthroughRep.Variation != nil {
		variation = int64(*fallthroughRep.Variation)
	}
	bucketBy := ""
	contextKind := ""
	rolloutWeights := []attr.Value{}
	if fallthroughRep.Rollout != nil {
		for _, w := range fallthroughRep.Rollout.Variations {
			rolloutWeights = append(rolloutWeights, types.Int64Value(int64(w.Weight)))
		}
		if fallthroughRep.Rollout.BucketBy != nil {
			bucketBy = *fallthroughRep.Rollout.BucketBy
		}
		if fallthroughRep.Rollout.ContextKind != nil {
			contextKind = *fallthroughRep.Rollout.ContextKind
		}
	}
	weightsList, _ := types.ListValue(types.Int64Type, rolloutWeights)
	obj, _ := types.ObjectValue(ffeFallthroughAttrTypes, map[string]attr.Value{
		VARIATION:       types.Int64Value(variation),
		ROLLOUT_WEIGHTS: weightsList,
		BUCKET_BY:       types.StringValue(bucketBy),
		CONTEXT_KIND:    types.StringValue(contextKind),
	})
	list, _ := types.ListValue(objectType, []attr.Value{obj})
	return list
}
