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

var _ datasource.DataSource = &SegmentDataSource{}

type SegmentDataSource struct {
	client *Client
}

type SegmentDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectKey           types.String `tfsdk:"project_key"`
	EnvKey               types.String `tfsdk:"env_key"`
	Key                  types.String `tfsdk:"key"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Tags                 types.Set    `tfsdk:"tags"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	Included             types.List   `tfsdk:"included"`
	Excluded             types.List   `tfsdk:"excluded"`
	IncludedContexts     types.List   `tfsdk:"included_contexts"`
	ExcludedContexts     types.List   `tfsdk:"excluded_contexts"`
	Rules                types.List   `tfsdk:"rules"`
	Unbounded            types.Bool   `tfsdk:"unbounded"`
	UnboundedContextKind types.String `tfsdk:"unbounded_context_kind"`
	ViewKeys             types.Set    `tfsdk:"view_keys"`
	Views                types.List   `tfsdk:"views"`
}

var segmentTargetAttrTypes = map[string]attr.Type{
	VALUES:       types.ListType{ElemType: types.StringType},
	CONTEXT_KIND: types.StringType,
}

var segmentRuleAttrTypes = map[string]attr.Type{
	CLAUSES:              types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkClauseAttrTypes}},
	WEIGHT:               types.Int64Type,
	BUCKET_BY:            types.StringType,
	ROLLOUT_CONTEXT_KIND: types.StringType,
}

func NewSegmentDataSource() datasource.DataSource {
	return &SegmentDataSource{}
}

func (d *SegmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (d *SegmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly segment data source.\n\nThis data source allows you to retrieve segment information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "Composite ID `project_key/env_key/key`."},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Description: "The segment's project key."},
			ENV_KEY:       schema.StringAttribute{Required: true, Description: "The segment's environment key."},
			KEY:           schema.StringAttribute{Required: true, Description: "The unique key that references the segment."},
			NAME:          schema.StringAttribute{Computed: true, Description: "Human-friendly name for the segment."},
			DESCRIPTION:   schema.StringAttribute{Computed: true, Description: "Segment description."},
			CREATION_DATE: schema.Int64Attribute{Computed: true, Description: "UNIX epoch ms timestamp."},
			TAGS:          schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags."},
			INCLUDED: schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "User keys included in the segment.",
			},
			EXCLUDED: schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "User keys excluded from the segment.",
			},
			UNBOUNDED:              schema.BoolAttribute{Computed: true, Description: "Whether this is a Big Segment."},
			UNBOUNDED_CONTEXT_KIND: schema.StringAttribute{Computed: true, Description: "Context kind for the big segment."},
			VIEW_KEYS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "View keys linked to this segment.",
			},
			VIEWS: schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Legacy view keys list (backwards-compat).",
			},
			INCLUDED_CONTEXTS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Non-user target objects included in the segment.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						VALUES: schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						CONTEXT_KIND: schema.StringAttribute{Computed: true},
					},
				},
			},
			EXCLUDED_CONTEXTS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Non-user target objects excluded from the segment.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						VALUES: schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						CONTEXT_KIND: schema.StringAttribute{Computed: true},
					},
				},
			},
			RULES: schema.ListNestedAttribute{
				Computed:    true,
				Description: "Custom rules applied to the segment.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						WEIGHT:               schema.Int64Attribute{Computed: true, Description: "Rule weight (1-100000)."},
						BUCKET_BY:            schema.StringAttribute{Computed: true, Description: "Attribute for bucketing contexts."},
						ROLLOUT_CONTEXT_KIND: schema.StringAttribute{Computed: true, Description: "Context kind for the rollout."},
						CLAUSES:              frameworkClausesDataSourceAttribute(),
					},
				},
			},
		},
	}
}

func (d *SegmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *SegmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data SegmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()
	segmentKey := data.Key.ValueString()

	var segment *ldapi.UserSegment
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		segment, _, err = d.client.ld.SegmentsApi.GetSegment(d.client.ctx, projectKey, envKey, segmentKey).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get segment %q of project %q: %s", segmentKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(projectKey + "/" + envKey + "/" + segmentKey)
	data.Name = types.StringValue(segment.Name)
	if segment.Description != nil {
		data.Description = types.StringValue(*segment.Description)
	} else {
		data.Description = types.StringValue("")
	}
	data.CreationDate = types.Int64Value(segment.CreationDate)

	tagsSet, diags := setFromStringSlice(ctx, segment.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	if segment.Unbounded != nil {
		data.Unbounded = types.BoolValue(*segment.Unbounded)
	} else {
		data.Unbounded = types.BoolValue(false)
	}
	if segment.UnboundedContextKind != nil {
		data.UnboundedContextKind = types.StringValue(*segment.UnboundedContextKind)
	} else {
		data.UnboundedContextKind = types.StringValue("")
	}

	includedList, diags := listFromStringSlice(ctx, segment.Included)
	resp.Diagnostics.Append(diags...)
	data.Included = includedList

	excludedList, diags := listFromStringSlice(ctx, segment.Excluded)
	resp.Diagnostics.Append(diags...)
	data.Excluded = excludedList

	data.IncludedContexts = segmentTargetsToFrameworkListImpl(ctx, segment.IncludedContexts)
	data.ExcludedContexts = segmentTargetsToFrameworkListImpl(ctx, segment.ExcludedContexts)

	data.Rules = segmentRulesToFrameworkList(ctx, segment.Rules)

	// View association: best-effort. Failure logs in SDKv2; here we
	// surface empty.
	viewKeys := []string{}
	betaClient, bcErr := newBetaClient(d.client.apiKey, d.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if bcErr == nil {
		var env *ldapi.Environment
		err = d.client.withConcurrency(d.client.ctx, func() error {
			env, _, err = d.client.ld.EnvironmentsApi.GetEnvironment(d.client.ctx, projectKey, envKey).Execute()
			return err
		})
		if err == nil {
			if vk, vErr := getViewsContainingSegment(betaClient, projectKey, env.Id, segmentKey); vErr == nil {
				viewKeys = vk
			}
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

func segmentTargetsToFrameworkListImpl(ctx context.Context, targets []ldapi.SegmentTarget) types.List {
	objectType := types.ObjectType{AttrTypes: segmentTargetAttrTypes}
	elements := make([]attr.Value, 0, len(targets))
	for _, t := range targets {
		values := []string{}
		if t.Values != nil {
			values = t.Values
		}
		valuesList, _ := listFromStringSlice(ctx, values)
		contextKind := ""
		if t.ContextKind != nil {
			contextKind = *t.ContextKind
		}
		obj, _ := types.ObjectValue(segmentTargetAttrTypes, map[string]attr.Value{
			VALUES:       valuesList,
			CONTEXT_KIND: types.StringValue(contextKind),
		})
		elements = append(elements, obj)
	}
	list, _ := types.ListValue(objectType, elements)
	return list
}

// segmentRulesToFrameworkList converts LD-API UserSegmentRule slices
// to a framework List<Object> matching segmentRuleAttrTypes, with
// nested clauses converted via frameworkClausesValue.
func segmentRulesToFrameworkList(ctx context.Context, rules []ldapi.UserSegmentRule) types.List {
	objectType := types.ObjectType{AttrTypes: segmentRuleAttrTypes}
	elements := make([]attr.Value, 0, len(rules))
	for _, r := range rules {
		clauses, _ := frameworkClausesValue(ctx, r.Clauses)
		var weight int64
		if r.Weight != nil {
			weight = int64(*r.Weight)
		}
		bucketBy := ""
		if r.BucketBy != nil {
			bucketBy = *r.BucketBy
		}
		rolloutContextKind := ""
		if r.RolloutContextKind != nil {
			rolloutContextKind = *r.RolloutContextKind
		}
		obj, _ := types.ObjectValue(segmentRuleAttrTypes, map[string]attr.Value{
			CLAUSES:              clauses,
			WEIGHT:               types.Int64Value(weight),
			BUCKET_BY:            types.StringValue(bucketBy),
			ROLLOUT_CONTEXT_KIND: types.StringValue(rolloutContextKind),
		})
		elements = append(elements, obj)
	}
	list, _ := types.ListValue(objectType, elements)
	return list
}
