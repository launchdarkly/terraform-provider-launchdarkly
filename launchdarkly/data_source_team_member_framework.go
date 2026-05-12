package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TeamMemberDataSource{}

type TeamMemberDataSource struct {
	client *Client
}

type TeamMemberDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Email          types.String `tfsdk:"email"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Role           types.String `tfsdk:"role"`
	CustomRoles    types.Set    `tfsdk:"custom_roles"`
	RoleAttributes types.Set    `tfsdk:"role_attributes"`
}

func NewTeamMemberDataSource() datasource.DataSource {
	return &TeamMemberDataSource{}
}

func (d *TeamMemberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (d *TeamMemberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly team member data source.\n\nThis data source allows you to retrieve team member information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true, Description: "The 24-character member ID."},
			EMAIL:      schema.StringAttribute{Required: true, Description: "The unique email address associated with the team member."},
			FIRST_NAME: schema.StringAttribute{Computed: true, Description: "First name."},
			LAST_NAME:  schema.StringAttribute{Computed: true, Description: "Last name."},
			ROLE:       schema.StringAttribute{Computed: true, Description: "The member's role (owner, reader, writer, admin)."},
			CUSTOM_ROLES: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Custom role keys associated with the team member.",
			},
		},
		Blocks: map[string]schema.Block{
			ROLE_ATTRIBUTES: frameworkRoleAttributesDataSourceBlock(),
		},
	}
}

func (d *TeamMemberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *TeamMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data TeamMemberDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberEmail := data.Email.ValueString()
	member, err := getTeamMemberByEmail(d.client, memberEmail)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find team member", err.Error())
		return
	}

	data.ID = types.StringValue(member.Id)
	data.Email = types.StringValue(member.Email)
	data.FirstName = stringValueFromPointer(member.FirstName)
	data.LastName = stringValueFromPointer(member.LastName)
	data.Role = types.StringValue(member.Role)

	customRolesSet, diags := setFromStringSlice(ctx, member.CustomRoles)
	resp.Diagnostics.Append(diags...)
	data.CustomRoles = customRolesSet

	roleAttrs, diags := frameworkRoleAttributesValue(ctx, member.RoleAttributes)
	resp.Diagnostics.Append(diags...)
	data.RoleAttributes = roleAttrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
