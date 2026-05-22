package launchdarkly

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &FeatureFlagDataSource{}

type FeatureFlagDataSource struct {
	client *Client
}

type FeatureFlagDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ProjectKey             types.String `tfsdk:"project_key"`
	Key                    types.String `tfsdk:"key"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	MaintainerID           types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey      types.String `tfsdk:"maintainer_team_key"`
	Tags                   types.Set    `tfsdk:"tags"`
	VariationType          types.String `tfsdk:"variation_type"`
	Variations             types.List   `tfsdk:"variations"`
	Temporary              types.Bool   `tfsdk:"temporary"`
	IncludeInSnippet       types.Bool   `tfsdk:"include_in_snippet"`
	ClientSideAvailability types.List   `tfsdk:"client_side_availability"`
	CustomProperties       types.Set    `tfsdk:"custom_properties"`
	Defaults               types.List   `tfsdk:"defaults"`
	Archived               types.Bool   `tfsdk:"archived"`
	Deprecated             types.Bool   `tfsdk:"deprecated"`
	ViewKeys               types.Set    `tfsdk:"view_keys"`
	Views                  types.List   `tfsdk:"views"`
}

var featureFlagVariationAttrTypes = map[string]attr.Type{
	NAME:        types.StringType,
	DESCRIPTION: types.StringType,
	VALUE:       types.StringType,
}

var featureFlagCustomPropertyAttrTypes = map[string]attr.Type{
	KEY:   types.StringType,
	NAME:  types.StringType,
	VALUE: types.ListType{ElemType: types.StringType},
}

var featureFlagDefaultsAttrTypes = map[string]attr.Type{
	ON_VARIATION:  types.Int64Type,
	OFF_VARIATION: types.Int64Type,
}

func NewFeatureFlagDataSource() datasource.DataSource {
	return &FeatureFlagDataSource{}
}

func (d *FeatureFlagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag"
}

func (d *FeatureFlagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly feature flag data source.\n\nThis data source allows you to retrieve feature flag information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "Composite ID `project_key/key`."},
			PROJECT_KEY: schema.StringAttribute{Required: true, Description: "The feature flag's project key."},
			KEY:         schema.StringAttribute{Required: true, Description: "The unique feature flag key."},
			NAME:        schema.StringAttribute{Computed: true, Description: "Human-readable name."},
			DESCRIPTION: schema.StringAttribute{Computed: true, Description: "Feature flag description."},
			MAINTAINER_ID: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The feature flag maintainer's 24 character alphanumeric team member ID. `maintainer_team_key` cannot be set if `maintainer_id` is set. If neither is set, it will automatically be or stay set to the member ID associated with the API key used by your LaunchDarkly Terraform provider or the most recently-set maintainer.",
			},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The key of the associated team that maintains this feature flag. `maintainer_id` cannot be set if `maintainer_team_key` is set",
			},
			TAGS:           schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags."},
			VARIATION_TYPE: schema.StringAttribute{Computed: true, Description: fmt.Sprintf("Variation type: %q, %q, %q, or %q.", BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION)},
			TEMPORARY:      schema.BoolAttribute{Computed: true, Description: "Whether the flag is temporary."},
			INCLUDE_IN_SNIPPET: schema.BoolAttribute{
				Computed:           true,
				Description:        "Deprecated: use client_side_availability.using_environment_id.",
				DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
			},
			ARCHIVED:   schema.BoolAttribute{Computed: true, Description: "Whether the flag is archived."},
			DEPRECATED: schema.BoolAttribute{Computed: true, Description: "Whether the flag is deprecated."},
			VIEW_KEYS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "View keys linked to the flag.",
			},
			VIEWS: schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Legacy view keys list.",
			},
			VARIATIONS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Possible variations for the flag.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						NAME:        schema.StringAttribute{Computed: true, Description: "Variation name."},
						DESCRIPTION: schema.StringAttribute{Computed: true, Description: "Variation description."},
						VALUE:       schema.StringAttribute{Computed: true, Description: "Variation value (stringified per variation_type)."},
					},
				},
			},
			CLIENT_SIDE_AVAILABILITY: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Client-side availability settings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						USING_ENVIRONMENT_ID: schema.BoolAttribute{Computed: true},
						USING_MOBILE_KEY:     schema.BoolAttribute{Computed: true},
					},
				},
			},
			CUSTOM_PROPERTIES: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Custom properties.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY:  schema.StringAttribute{Computed: true},
						NAME: schema.StringAttribute{Computed: true},
						VALUE: schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			DEFAULTS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Default variation indices for new environments.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						ON_VARIATION:  schema.Int64Attribute{Computed: true},
						OFF_VARIATION: schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *FeatureFlagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *FeatureFlagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data FeatureFlagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var flag *ldapi.FeatureFlag
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		flag, _, err = d.client.ld.FeatureFlagsApi.GetFeatureFlag(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(flag.Key)
	data.Name = types.StringValue(flag.Name)
	if flag.Description != nil {
		data.Description = types.StringValue(*flag.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if flag.MaintainerId != nil {
		data.MaintainerID = types.StringValue(*flag.MaintainerId)
	} else {
		data.MaintainerID = types.StringValue("")
	}
	if flag.MaintainerTeamKey != nil {
		data.MaintainerTeamKey = types.StringValue(*flag.MaintainerTeamKey)
	} else {
		data.MaintainerTeamKey = types.StringValue("")
	}
	data.Temporary = types.BoolValue(flag.Temporary)
	data.Archived = types.BoolValue(flag.Archived)
	data.Deprecated = types.BoolValue(flag.GetDeprecated())

	tagsSet, diags := setFromStringSlice(ctx, flag.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	// client_side_availability + include_in_snippet
	csaType := types.ObjectType{AttrTypes: map[string]attr.Type{
		USING_ENVIRONMENT_ID: types.BoolType,
		USING_MOBILE_KEY:     types.BoolType,
	}}
	usingEnvID := false
	usingMobile := false
	if flag.ClientSideAvailability != nil {
		csa := *flag.ClientSideAvailability
		if csa.UsingEnvironmentId != nil {
			usingEnvID = *csa.UsingEnvironmentId
		}
		if csa.UsingMobileKey != nil {
			usingMobile = *csa.UsingMobileKey
		}
	}
	csaObj, diags := types.ObjectValue(map[string]attr.Type{
		USING_ENVIRONMENT_ID: types.BoolType,
		USING_MOBILE_KEY:     types.BoolType,
	}, map[string]attr.Value{
		USING_ENVIRONMENT_ID: types.BoolValue(usingEnvID),
		USING_MOBILE_KEY:     types.BoolValue(usingMobile),
	})
	resp.Diagnostics.Append(diags...)
	csaList, diags := types.ListValue(csaType, []attr.Value{csaObj})
	resp.Diagnostics.Append(diags...)
	data.ClientSideAvailability = csaList
	data.IncludeInSnippet = types.BoolValue(usingEnvID)

	// variations
	variationType, err := variationsToVariationType(flag.Variations)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine variation type", err.Error())
		return
	}
	data.VariationType = types.StringValue(variationType)

	variationObjectType := types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}
	variationElements := make([]attr.Value, 0, len(flag.Variations))
	for _, v := range flag.Variations {
		valueString, err := variationValueToString(&v.Value, variationType)
		if err != nil {
			resp.Diagnostics.AddError("Failed to serialise variation value", err.Error())
			return
		}
		nameStr := ""
		if v.Name != nil {
			nameStr = *v.Name
		}
		descStr := ""
		if v.Description != nil {
			descStr = *v.Description
		}
		obj, d := types.ObjectValue(featureFlagVariationAttrTypes, map[string]attr.Value{
			NAME:        types.StringValue(nameStr),
			DESCRIPTION: types.StringValue(descStr),
			VALUE:       types.StringValue(valueString),
		})
		resp.Diagnostics.Append(d...)
		variationElements = append(variationElements, obj)
	}
	variationsList, diags := types.ListValue(variationObjectType, variationElements)
	resp.Diagnostics.Append(diags...)
	data.Variations = variationsList

	// custom_properties — sort each property's values for stable plan output.
	cpObjectType := types.ObjectType{AttrTypes: featureFlagCustomPropertyAttrTypes}
	cpElements := make([]attr.Value, 0, len(flag.CustomProperties))
	for k, cp := range flag.CustomProperties {
		sortedValues := make([]string, len(cp.Value))
		copy(sortedValues, cp.Value)
		sort.Strings(sortedValues)
		valuesList, d := listFromStringSlice(ctx, sortedValues)
		resp.Diagnostics.Append(d...)
		obj, d := types.ObjectValue(featureFlagCustomPropertyAttrTypes, map[string]attr.Value{
			KEY:   types.StringValue(k),
			NAME:  types.StringValue(cp.Name),
			VALUE: valuesList,
		})
		resp.Diagnostics.Append(d...)
		cpElements = append(cpElements, obj)
	}
	cpSet, diags := types.SetValue(cpObjectType, cpElements)
	resp.Diagnostics.Append(diags...)
	data.CustomProperties = cpSet

	// defaults
	defaultsObjectType := types.ObjectType{AttrTypes: featureFlagDefaultsAttrTypes}
	var on, off int64
	if flag.Defaults != nil {
		on = int64(flag.Defaults.OnVariation)
		off = int64(flag.Defaults.OffVariation)
	} else {
		on = 0
		off = int64(len(flag.Variations) - 1)
	}
	defaultsObj, diags := types.ObjectValue(featureFlagDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:  types.Int64Value(on),
		OFF_VARIATION: types.Int64Value(off),
	})
	resp.Diagnostics.Append(diags...)
	defaultsList, diags := types.ListValue(defaultsObjectType, []attr.Value{defaultsObj})
	resp.Diagnostics.Append(diags...)
	data.Defaults = defaultsList

	// view associations (best-effort)
	viewKeys := []string{}
	if betaClient, bcErr := newBetaClient(d.client.apiKey, d.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY); bcErr == nil {
		if vk, vErr := getViewsContainingFlag(betaClient, projectKey, key); vErr == nil {
			viewKeys = vk
		}
	}
	viewKeysSet, diags := setFromStringSlice(ctx, viewKeys)
	resp.Diagnostics.Append(diags...)
	data.ViewKeys = viewKeysSet
	viewsList, diags := listFromStringSlice(ctx, viewKeys)
	resp.Diagnostics.Append(diags...)
	data.Views = viewsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
