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
	ID:           types.StringType,
	EMAIL:        types.StringType,
	FIRST_NAME:   types.StringType,
	LAST_NAME:    types.StringType,
	ROLE:         types.StringType,
	CUSTOM_ROLES: types.SetType{ElemType: types.StringType},
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
		},
		Blocks: map[string]schema.Block{
			TEAM_MEMBERS: schema.ListNestedBlock{
				Description: "The members that were found.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						ID:         schema.StringAttribute{Computed: true, Description: "The 24-character member ID."},
						EMAIL:      schema.StringAttribute{Computed: true},
						FIRST_NAME: schema.StringAttribute{Computed: true},
						LAST_NAME:  schema.StringAttribute{Computed: true},
						ROLE:       schema.StringAttribute{Computed: true},
						CUSTOM_ROLES: schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
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
		allMembers, err := getAllTeamMembers(d.client)
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
				resp.Diagnostics.AddError("Team member not found", fmt.Sprintf("No team member found for email: %s", memberEmail))
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
		obj, d := types.ObjectValue(teamMembersMemberAttrTypes, map[string]attr.Value{
			ID:           types.StringValue(m.Id),
			EMAIL:        types.StringValue(m.Email),
			FIRST_NAME:   stringValueFromPointer(m.FirstName),
			LAST_NAME:    stringValueFromPointer(m.LastName),
			ROLE:         types.StringValue(m.Role),
			CUSTOM_ROLES: customRolesSet,
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
