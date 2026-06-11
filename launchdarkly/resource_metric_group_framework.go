package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &MetricGroupResource{}
	_ resource.ResourceWithImportState = &MetricGroupResource{}
	_ resource.ResourceWithModifyPlan  = &MetricGroupResource{}
)

type MetricGroupResource struct {
	client *Client
	beta   *Client
}

type MetricGroupResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectKey   types.String `tfsdk:"project_key"`
	Key          types.String `tfsdk:"key"`
	Name         types.String `tfsdk:"name"`
	Kind         types.String `tfsdk:"kind"`
	Description  types.String `tfsdk:"description"`
	MaintainerID types.String `tfsdk:"maintainer_id"`
	Tags         types.Set    `tfsdk:"tags"`
	Metrics      types.List   `tfsdk:"metrics"`
	Version      types.Int64  `tfsdk:"version"`
}

func NewMetricGroupResource() resource.Resource { return &MetricGroupResource{} }

func (r *MetricGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_group"
}

func (r *MetricGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly metric group resource.

~> **Beta:** This resource uses a beta LaunchDarkly API. Beta resources may change or be removed in future versions.

This resource allows you to create and manage metric groups within your LaunchDarkly project. A metric group is an ordered ` + "`funnel`" + ` or an unordered ` + "`standard`" + ` collection of metrics that you can reference from experiments. To learn more, read [Experimentation Documentation](https://docs.launchdarkly.com/home/experimentation).`,
		Attributes: metricGroupSchemaAttributes(),
	}
}

func metricGroupSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The ID of this resource in the format `project_key/key`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The metric group's project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The unique key that references the metric group.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The human-friendly name for the metric group.",
		},
		KIND: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The type of the metric group. Available choices are `funnel` and `standard`. A `funnel` metric group is an ordered list of metrics; a `standard` metric group is an unordered collection.", true),
			Validators:    []validator.String{oneOfValidator{allowed: []string{METRIC_GROUP_KIND_FUNNEL, METRIC_GROUP_KIND_STANDARD}}},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "A description of the metric group's purpose.",
		},
		MAINTAINER_ID: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The LaunchDarkly member ID of the member who maintains the metric group. If not set when the metric group is created, the provider assigns the member associated with the access token. Service tokens have no associated member, so configurations using one must set this explicitly.",
			Validators:  []validator.String{idValidator()},
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Tags associated with the metric group.",
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(tagValidator()),
			},
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version of the metric group.",
		},
		METRICS: schema.ListNestedAttribute{
			Required:    true,
			Description: "An ordered list of the metrics in this metric group. Must contain at least two metrics. For `funnel` metric groups the order is significant and each metric requires a `name_in_group`.",
			Validators: []validator.List{
				listvalidator.SizeAtLeast(2),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					KEY: schema.StringAttribute{
						Required:    true,
						Description: "The key of the metric to include in the group.",
						Validators:  []validator.String{keyValidator()},
					},
					NAME_IN_GROUP: schema.StringAttribute{
						Optional:    true,
						Description: "The name of the metric when used within this metric group. Can differ from the metric's own name. Required for `funnel` metric groups and not permitted for `standard` metric groups.",
					},
				},
			},
		},
	}
}

func (r *MetricGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newMetricGroupBetaClient(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *MetricGroupResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newMetricGroupBetaClient(r.client)
}

// ModifyPlan enforces the funnel/standard rules around name_in_group and marks
// the computed `version` as unknown whenever any user-controlled attribute
// changes, so the post-apply refresh does not trip "inconsistent result".
func (r *MetricGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan MetricGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kind := plan.Kind.ValueString()
	if !plan.Metrics.IsNull() && !plan.Metrics.IsUnknown() {
		var metrics []metricGroupMetricModel
		resp.Diagnostics.Append(plan.Metrics.ElementsAs(ctx, &metrics, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, m := range metrics {
			nameInGroupSet := !m.NameInGroup.IsNull() && !m.NameInGroup.IsUnknown() && m.NameInGroup.ValueString() != ""
			switch kind {
			case METRIC_GROUP_KIND_FUNNEL:
				if !nameInGroupSet {
					resp.Diagnostics.AddError("funnel metric groups require 'name_in_group' on every metric", fmt.Sprintf("metric %q is missing name_in_group", m.Key.ValueString()))
				}
			case METRIC_GROUP_KIND_STANDARD:
				if nameInGroupSet {
					resp.Diagnostics.AddError("standard metric groups do not accept 'name_in_group'", fmt.Sprintf("metric %q must not set name_in_group", m.Key.ValueString()))
				}
			}
		}
	}

	if !req.State.Raw.IsNull() {
		var state MetricGroupResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		planVersion := plan.Version
		plan.Version = state.Version
		if !reflect.DeepEqual(plan, state) {
			plan.Version = types.Int64Unknown()
		} else {
			plan.Version = planVersion
		}
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *MetricGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MetricGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError("Failed to check project", err.Error())
			return
		}
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("cannot find project with key %q", projectKey))
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if tags == nil {
		tags = []string{}
	}

	var metricsModels []metricGroupMetricModel
	resp.Diagnostics.Append(plan.Metrics.ElementsAs(ctx, &metricsModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	kind := plan.Kind.ValueString()
	// The POST body requires maintainerId (the API rejects an empty value with
	// 400 "maintainer ID is required"), so resolve the token's own member when
	// the practitioner doesn't set one. Service tokens have no member — those
	// must set maintainer_id explicitly.
	maintainerID := plan.MaintainerID.ValueString()
	if maintainerID == "" {
		var me *ldapi.Member
		err := r.client.withConcurrency(r.client.ctx, func() error {
			var e error
			me, _, e = r.client.ld.AccountMembersApi.GetMember(r.client.ctx, "me").Execute()
			return e
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to resolve a default maintainer for the metric group",
				fmt.Sprintf("the metric group API requires a maintainer ID and the access token does not map to a member (service tokens have none) — set maintainer_id explicitly: %s", handleLdapiErr(err)),
			)
			return
		}
		maintainerID = me.Id
	}

	post := ldapi.MetricGroupPost{
		Key:          &key,
		Name:         name,
		Kind:         kind,
		MaintainerId: maintainerID,
		Tags:         tags,
		Metrics:      metricGroupInputsFromModels(metricsModels),
	}
	if description := plan.Description.ValueString(); description != "" {
		post.Description = &description
	}

	err = beta.withConcurrency(beta.ctx, func() error {
		_, _, e := beta.ld.MetricsBetaApi.CreateMetricGroup(beta.ctx, projectKey).MetricGroupPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating metric group resource: %q", key), err)
		return
	}

	plan.ID = types.StringValue(projectKey + "/" + key)
	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MetricGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MetricGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MetricGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MetricGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()

	var patch []ldapi.PatchOperation
	if !plan.Name.Equal(state.Name) {
		patch = append(patch, patchReplace("/name", plan.Name.ValueString()))
	}
	if !plan.Description.Equal(state.Description) {
		patch = append(patch, patchReplace("/description", plan.Description.ValueString()))
	}
	if !plan.MaintainerID.Equal(state.MaintainerID) && plan.MaintainerID.ValueString() != "" {
		patch = append(patch, patchReplace("/maintainerId", plan.MaintainerID.ValueString()))
	}
	if !plan.Tags.Equal(state.Tags) {
		tags, d := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(d...)
		if tags == nil {
			tags = []string{}
		}
		patch = append(patch, patchReplace("/tags", tags))
	}
	// Ordered comparison is safe for both kinds: the API stores and echoes
	// metrics in insertion order even for `standard` groups (verified against
	// the live API 2026-06-11), so a config reorder converges in one apply
	// rather than producing a perpetual diff.
	if !plan.Metrics.Equal(state.Metrics) {
		var metricsModels []metricGroupMetricModel
		resp.Diagnostics.Append(plan.Metrics.ElementsAs(ctx, &metricsModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		patch = append(patch, patchReplace("/metrics", metricGroupInputsFromModels(metricsModels)))
	}

	if len(patch) > 0 {
		err = beta.withConcurrency(beta.ctx, func() error {
			_, _, e := beta.ld.MetricsBetaApi.PatchMetricGroup(beta.ctx, projectKey, key).PatchOperation(patch).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating metric group resource %q in project %q", key, projectKey), err)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MetricGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MetricGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		res, e = beta.ld.MetricsBetaApi.DeleteMetricGroup(beta.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting metric group resource %q", data.Key.ValueString()), err)
	}
}

func (r *MetricGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, key, err := metricGroupIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *MetricGroupResource) readIntoModel(
	ctx context.Context,
	projectKey, key string,
	data *MetricGroupResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}

	var group *ldapi.MetricGroupRep
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		group, res, err = beta.ld.MetricsBetaApi.GetMetricGroup(beta.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get metric group %q", key), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(group.Key)
	data.Name = types.StringValue(group.Name)
	data.Kind = types.StringValue(group.Kind)
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(group.Description)
	data.Version = types.Int64Value(int64(group.Version))

	if maintainerID := metricGroupMaintainerID(group.Maintainer); maintainerID != "" {
		data.MaintainerID = types.StringValue(maintainerID)
	} else if data.MaintainerID.IsNull() || data.MaintainerID.IsUnknown() {
		data.MaintainerID = types.StringValue("")
	}

	tagsSet, d := setFromStringSlice(ctx, group.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	metricsList, err := metricGroupMetricsToList(group.Metrics)
	if err != nil {
		diags.AddError("Failed to read metric group metrics", err.Error())
		return
	}
	data.Metrics = metricsList
}
