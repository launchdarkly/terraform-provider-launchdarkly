package launchdarkly

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &TeamMembersDataSource{}

type TeamMembersDataSource struct {
	client *Client
}

type TeamMembersDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Emails        types.List   `tfsdk:"emails"`
	IgnoreMissing types.Bool   `tfsdk:"ignore_missing"`
	TeamMembers   types.List   `tfsdk:"team_members"`
}

var teamMembersMemberAttrTypes = map[string]attr.Type{
	ID:              types.StringType,
	EMAIL:           types.StringType,
	FIRST_NAME:      types.StringType,
	LAST_NAME:       types.StringType,
	ROLE:            types.StringType,
	CUSTOM_ROLES:    types.SetType{ElemType: types.StringType},
	ROLE_ATTRIBUTES: types.MapType{ElemType: types.ListType{ElemType: types.StringType}},
}

func NewTeamMembersDataSource() datasource.DataSource {
	return &TeamMembersDataSource{}
}

func (d *TeamMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (d *TeamMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly team members data source.\n\nThis data source allows you to retrieve team member information from your LaunchDarkly organization on multiple team members.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "A hash of the returned member IDs."},
			EMAILS: schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "An array of unique email addresses associated with the team members.",
			},
			IGNORE_MISSING: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A boolean to determine whether to ignore members that weren't found.",
			},
			TEAM_MEMBERS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "The members that were found. The following attributes are available for each member:\n\n- `id` - The 24 character alphanumeric ID of the team member.\n\n- `first_name` - The team member's given name.\n\n- `last_name` - The team member's family name.\n\n- `role` - The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`.\n\n- `custom_roles` - (Optional) The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						ID:         schema.StringAttribute{Computed: true, Description: "The 24 character alphanumeric ID of the team member."},
						EMAIL:      schema.StringAttribute{Computed: true, Description: "The unique email address associated with the team member."},
						FIRST_NAME: schema.StringAttribute{Computed: true, Description: "The team member's given name."},
						LAST_NAME:  schema.StringAttribute{Computed: true, Description: "The team member's family name."},
						ROLE:       schema.StringAttribute{Computed: true, Description: "The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`."},
						CUSTOM_ROLES: schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).",
						},
						ROLE_ATTRIBUTES: frameworkRoleAttributesDataSourceAttribute(),
					},
				},
			},
		},
	}
}

func (d *TeamMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *TeamMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data TeamMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emails, diags := stringSliceFromList(ctx, data.Emails)
	resp.Diagnostics.Append(diags...)
	ignoreMissing := false
	if !data.IgnoreMissing.IsNull() && !data.IgnoreMissing.IsUnknown() {
		ignoreMissing = data.IgnoreMissing.ValueBool()
	}

	var members []ldapi.Member
	expectedCount := len(emails)

	if expectedCount > 0 {
		allMembers, err := getTeamMembersByEmail(d.client, emails)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list team members", err.Error())
			return
		}
		for _, memberEmail := range emails {
			var foundMember ldapi.Member
			memberFound := false
			for _, m := range allMembers {
				if m.Email == memberEmail {
					foundMember = m
					memberFound = true
					break
				}
			}
			if !memberFound {
				if ignoreMissing {
					continue
				}
				resp.Diagnostics.AddError(fmt.Sprintf("No team member found for email: %s", memberEmail), "")
				return
			}
			members = append(members, foundMember)
		}
	}

	if !ignoreMissing && len(members) != expectedCount {
		resp.Diagnostics.AddError("Member count mismatch", fmt.Sprintf("unexpected number of users returned (%d != %d)", len(members), expectedCount))
		return
	}

	objectType := types.ObjectType{AttrTypes: teamMembersMemberAttrTypes}
	elements := make([]attr.Value, 0, len(members))
	ids := make([]string, 0, len(members))
	for _, m := range members {
		customRolesSet, d := setFromStringSlice(ctx, m.CustomRoles)
		resp.Diagnostics.Append(d...)
		roleAttrsSet, d := frameworkRoleAttributesValue(ctx, m.RoleAttributes)
		resp.Diagnostics.Append(d...)
		obj, d := types.ObjectValue(teamMembersMemberAttrTypes, map[string]attr.Value{
			ID:              types.StringValue(m.Id),
			EMAIL:           types.StringValue(m.Email),
			FIRST_NAME:      stringValueFromPointer(m.FirstName),
			LAST_NAME:       stringValueFromPointer(m.LastName),
			ROLE:            types.StringValue(m.Role),
			CUSTOM_ROLES:    customRolesSet,
			ROLE_ATTRIBUTES: roleAttrsSet,
		})
		resp.Diagnostics.Append(d...)
		elements = append(elements, obj)
		ids = append(ids, m.Id)
	}
	memberList, diags := types.ListValue(objectType, elements)
	resp.Diagnostics.Append(diags...)
	data.TeamMembers = memberList
	data.IgnoreMissing = types.BoolValue(ignoreMissing)

	h := sha1.New()
	if _, err := h.Write([]byte(strings.Join(ids, "-"))); err != nil {
		resp.Diagnostics.AddError("Failed to compute member ID hash", err.Error())
		return
	}
	data.ID = types.StringValue("team_members#" + base64.URLEncoding.EncodeToString(h.Sum(nil)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
