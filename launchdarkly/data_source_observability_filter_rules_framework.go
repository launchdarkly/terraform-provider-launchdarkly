package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ObservabilityFilterRulesDataSource{}

type ObservabilityFilterRulesDataSource struct {
	client *Client
}

type ObservabilityFilterRulesDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectKey  types.String `tfsdk:"project_key"`
	ProductType types.String `tfsdk:"product_type"`
	Rules       types.List   `tfsdk:"rules"`
}

var observabilityFilterRuleDataAttrTypes = map[string]attr.Type{
	KEY:          types.StringType,
	NAME:         types.StringType,
	PRODUCT_TYPE: types.StringType,
	ENABLED:      types.BoolType,
	QUERY:        types.StringType,
	OPERATION:    types.StringType,
	RATE:         types.Float64Type,
	ATTRIBUTES:   types.ListType{ElemType: types.StringType},
	VERSION:      types.Int64Type,
}

func NewObservabilityFilterRulesDataSource() datasource.DataSource {
	return &ObservabilityFilterRulesDataSource{}
}

func (d *ObservabilityFilterRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_filter_rules"
}

func (d *ObservabilityFilterRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly observability filter rules data source.

~> **Beta:** This data source uses a beta API. Beta data sources may change or be removed in future versions.

This data source allows you to list the observability ingestion filter rules of a LaunchDarkly project, optionally for one product.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The data source ID: the project key, or `project_key/product_type` when `product_type` is set.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
				Validators:  []validator.String{keyValidator()},
			},
			PRODUCT_TYPE: schema.StringAttribute{
				Optional:    true,
				Description: "Only list rules for this product. Must be one of `sessions`, `errors`, `logs`, `traces`, or `metrics`. Omit to list the rules of every product.",
				Validators:  []validator.String{stringvalidator.OneOf(observabilityProductTypes...)},
			},
			RULES: schema.ListNestedAttribute{
				Computed:    true,
				Description: "The filter rules, in the order LaunchDarkly returns them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						KEY:          schema.StringAttribute{Computed: true, Description: "The rule key."},
						NAME:         schema.StringAttribute{Computed: true, Description: "The rule name."},
						PRODUCT_TYPE: schema.StringAttribute{Computed: true, Description: "The product the rule filters."},
						ENABLED:      schema.BoolAttribute{Computed: true, Description: "Whether the rule is enabled."},
						QUERY:        schema.StringAttribute{Computed: true, Description: "The search query that selects the data the rule applies to."},
						OPERATION:    schema.StringAttribute{Computed: true, Description: "`include` or `exclude`."},
						RATE:         schema.Float64Attribute{Computed: true, Description: "The fraction of matching data the operation applies to."},
						ATTRIBUTES:   schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Attribute keys the rule filters instead of the whole item."},
						VERSION:      schema.Int64Attribute{Computed: true, Description: "The rule's version."},
					},
				},
			},
		},
	}
}

func (d *ObservabilityFilterRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ObservabilityFilterRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ObservabilityFilterRulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	productType := ""
	if !data.ProductType.IsNull() {
		productType = data.ProductType.ValueString()
	}
	rules, err := listObservabilityFilterRules(ctx, d.client, projectKey, productType)
	if err != nil {
		addObservabilityError(&resp.Diagnostics, fmt.Sprintf("Failed to list observability filter rules for project %q", projectKey), err, "")
		return
	}

	elems := make([]attr.Value, 0, len(rules))
	for _, rule := range rules {
		attrs := rule.Attributes
		if attrs == nil {
			attrs = []string{}
		}
		attrList, diags := types.ListValueFrom(ctx, types.StringType, attrs)
		resp.Diagnostics.Append(diags...)
		obj, diags := types.ObjectValue(observabilityFilterRuleDataAttrTypes, map[string]attr.Value{
			KEY:          types.StringValue(rule.Key),
			NAME:         types.StringValue(rule.Name),
			PRODUCT_TYPE: types.StringValue(rule.ProductType),
			ENABLED:      types.BoolValue(rule.Enabled),
			QUERY:        types.StringValue(rule.Query),
			OPERATION:    types.StringValue(rule.Operation),
			RATE:         types.Float64Value(rule.Rate),
			ATTRIBUTES:   attrList,
			VERSION:      types.Int64Value(rule.Version),
		})
		resp.Diagnostics.Append(diags...)
		elems = append(elems, obj)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: observabilityFilterRuleDataAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Rules = list
	data.ID = types.StringValue(projectKey)
	if productType != "" {
		data.ID = types.StringValue(projectKey + "/" + productType)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
