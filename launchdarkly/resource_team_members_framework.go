package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// teamMembersMaxBatchSize is the maximum number of members POST
	// /api/v2/members accepts in one request.
	teamMembersMaxBatchSize = 50
	// teamMembersMaxTeamsPerMember is the maximum number of teams the API
	// accepts for a single member.
	teamMembersMaxTeamsPerMember = 50
)

var (
	_ resource.Resource                     = &TeamMembersResource{}
	_ resource.ResourceWithImportState      = &TeamMembersResource{}
	_ resource.ResourceWithModifyPlan       = &TeamMembersResource{}
	_ resource.ResourceWithConfigValidators = &TeamMembersResource{}
)

type TeamMembersResource struct {
	client *Client
}

func NewTeamMembersResource() resource.Resource {
	return &TeamMembersResource{}
}

// teamMembersEntryModel is one member in the batch. It lives in a
// MapNestedAttribute keyed by lowercase email, so the computed `id` is safe
// here: map element identity comes from the key, not from a hash of the
// object (which is why a SetNestedAttribute would be unusable — see the
// provider authoring guide on set-hash instability with Computed fields).
type teamMembersEntryModel struct {
	ID             types.String `tfsdk:"id"`
	Email          types.String `tfsdk:"email"`
	FirstName      types.String `tfsdk:"first_name"`
	LastName       types.String `tfsdk:"last_name"`
	Role           types.String `tfsdk:"role"`
	CustomRoles    types.Set    `tfsdk:"custom_roles"`
	TeamKeys       types.Set    `tfsdk:"team_keys"`
	RoleAttributes types.Map    `tfsdk:"role_attributes"`
}

type teamMembersResourceModel struct {
	ID                 types.String                     `tfsdk:"id"`
	AdoptExisting      types.Bool                       `tfsdk:"adopt_existing"`
	DeletionProtection types.Bool                       `tfsdk:"deletion_protection"`
	Members            map[string]teamMembersEntryModel `tfsdk:"members"`
}

// teamMembersEntryAttrTypes mirrors teamMembersEntryModel for map pinning.
var teamMembersEntryAttrTypes = map[string]attr.Type{
	ID:              types.StringType,
	EMAIL:           types.StringType,
	FIRST_NAME:      types.StringType,
	LAST_NAME:       types.StringType,
	ROLE:            types.StringType,
	CUSTOM_ROLES:    types.SetType{ElemType: types.StringType},
	TEAM_KEYS:       types.SetType{ElemType: types.StringType},
	ROLE_ATTRIBUTES: types.MapType{ElemType: types.ListType{ElemType: types.StringType}},
}

func teamMembersEntryObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: teamMembersEntryAttrTypes}
}

func (r *TeamMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (r *TeamMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *TeamMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly team members resource for inviting up to 50 members, with their team assignments, in a single API call.

This resource batches member creation into one ` + "`POST /api/v2/members`" + ` request instead of one request per member, which avoids the member-write rate limits that per-member resources hit on large onboardings. The ` + "`members`" + ` map is keyed by lowercase email.

-> **Note:** Manage any given member with **either** this resource or ` + "[`launchdarkly_team_member`](/docs/providers/launchdarkly/r/team_member.html)" + `, never both. Likewise, if you assign teams here with ` + "`team_keys`" + `, do not also manage those teams' membership with ` + "`launchdarkly_team.member_ids`" + ` — the two fight over the same association.

-> **Note:** ` + "`first_name`" + ` and ` + "`last_name`" + ` are only used when a member is created. LaunchDarkly does not allow the provider to change a member's name afterwards; the member does that themselves.

-> **Note:** Updates create new members before deleting removed ones, so swapping a full batch at exactly your seat limit fails on the create. Remove members in one apply and add in the next, or add seats.

~> **Warning:** Removing an entry from ` + "`members`" + ` deletes that member from your LaunchDarkly account, and destroying this resource deletes every member in the batch. ` + "`deletion_protection`" + ` (enabled by default) blocks destroys and whole-batch replacements until you disable it in a separate apply.`,
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The unique identifier for this batch of team members.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			ADOPT_EXISTING: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to take over members who already exist in your account. When `false`, the default, applying a batch that contains an existing member's email fails and tells you which emails conflict. When `true`, those members are brought under this resource's management: their roles and team assignments are reconciled to this configuration, and they are deleted when you remove them from the batch or destroy the resource.",
			},
			DELETION_PROTECTION: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to block operations that would delete every member in the batch. When `true`, the default, destroying this resource fails, and so does any single update that removes all of the members it manages. Set it to `false` and apply that change first, then perform the destroy or replacement.",
			},
			MEMBERS: schema.MapNestedAttribute{
				Required:    true,
				Description: "The members to invite, keyed by lowercase email address. At most 50 entries; for larger teams, split them across multiple `launchdarkly_team_members` resources. This is the authoritative set: removing an entry deletes that member.",
				Validators: []validator.Map{
					mapvalidator.SizeBetween(1, teamMembersMaxBatchSize),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						ID: schema.StringAttribute{
							Computed:      true,
							Description:   "The 24 character alphanumeric ID of the team member.",
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						EMAIL: schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							Description:   "The member's email address. Must equal the map key; it defaults to the map key when omitted.",
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						FIRST_NAME: schema.StringAttribute{
							Optional:    true,
							Description: "The member's given name. Used only when the member is created; afterwards only the member can change it.",
						},
						LAST_NAME: schema.StringAttribute{
							Optional:    true,
							Description: "The member's family name. Used only when the member is created; afterwards only the member can change it.",
						},
						ROLE: schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The role associated with the member. Supported roles are `reader`, `writer`, `no_access`, or `admin`. If you don't specify a role, `reader` is assigned by default.",
							Validators: []validator.String{
								oneOfValidator{allowed: []string{"reader", "writer", "admin", "no_access"}},
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						CUSTOM_ROLES: schema.SetAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "The list of custom role keys associated with the member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\n-> **Note:** each member must have either a `role` or `custom_roles` argument.",
						},
						TEAM_KEYS: schema.SetAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "The keys of the teams to add the member to. The teams must already exist. This resource manages only the associations you declare here: teams the member belongs to for other reasons are left alone, but removing a key you previously declared removes the member from that team.",
							Validators: []validator.Set{
								setvalidator.SizeAtMost(teamMembersMaxTeamsPerMember),
								setvalidator.ValueStringsAre(keyValidator()),
							},
						},
						ROLE_ATTRIBUTES: frameworkRoleAttributesResourceAttribute(),
					},
				},
			},
		},
	}
}

func (r *TeamMembersResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{teamMembersBatchValidator{}}
}

// teamMembersBatchValidator enforces the batch-level rules that schema
// validators cannot express: lowercase email map keys, inner email agreeing
// with its map key, and the per-entry "at least one of role or custom_roles"
// requirement that launchdarkly_team_member also enforces.
type teamMembersBatchValidator struct{}

func (teamMembersBatchValidator) Description(context.Context) string {
	return "members must be keyed by lowercase email and each entry must set at least one of role or custom_roles"
}

func (teamMembersBatchValidator) MarkdownDescription(ctx context.Context) string {
	return teamMembersBatchValidator{}.Description(ctx)
}

func (teamMembersBatchValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg teamMembersResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An unknown members map (for example built from another resource's
	// output) cannot be checked until apply.
	if cfg.Members == nil {
		return
	}
	if err := validateMemberBatch(cfg.Members); err != nil {
		resp.Diagnostics.AddError("Invalid members batch", err.Error())
	}
}

// validateMemberBatch checks batch-level invariants. Unknown values are
// skipped rather than coerced, so a config whose values come from another
// resource still plans cleanly and is checked at apply time instead.
func validateMemberBatch(members map[string]teamMembersEntryModel) error {
	if len(members) == 0 {
		return fmt.Errorf("members must contain at least 1 entry")
	}
	if len(members) > teamMembersMaxBatchSize {
		return fmt.Errorf(
			"members must contain at most %d entries, got %d; split larger groups across multiple launchdarkly_team_members resources",
			teamMembersMaxBatchSize, len(members),
		)
	}
	for email, m := range members {
		if email != strings.ToLower(email) {
			return fmt.Errorf("members key %q must be lowercase", email)
		}
		if !isPlausibleEmail(email) {
			return fmt.Errorf("members key %q must be an email address", email)
		}
		if !m.Email.IsNull() && !m.Email.IsUnknown() {
			if got := m.Email.ValueString(); strings.ToLower(got) != email {
				return fmt.Errorf("member %q sets email %q: email must equal its map key or be omitted", email, got)
			}
		}
		roleKnown := !m.Role.IsUnknown()
		customKnown := !m.CustomRoles.IsUnknown()
		if !roleKnown || !customKnown {
			continue // defer to apply
		}
		roleSet := !m.Role.IsNull() && m.Role.ValueString() != ""
		customSet := !m.CustomRoles.IsNull() && len(m.CustomRoles.Elements()) > 0
		if !roleSet && !customSet {
			return fmt.Errorf("member %q must set at least one of role or custom_roles", email)
		}
	}
	return nil
}

// isPlausibleEmail is a deliberately loose check: LaunchDarkly is the
// authority on address validity, so this only catches obvious mistakes
// such as using a name or team key as the map key.
func isPlausibleEmail(s string) bool {
	at := strings.Index(s, "@")
	if at <= 0 || at != strings.LastIndex(s, "@") || at == len(s)-1 {
		return false
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	return strings.Contains(s[at+1:], ".")
}

func (r *TeamMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("not implemented", "Create is implemented in a later change")
}

func (r *TeamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("not implemented", "Read is implemented in a later change")
}

func (r *TeamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("not implemented", "Update is implemented in a later change")
}

func (r *TeamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("not implemented", "Delete is implemented in a later change")
}

func (r *TeamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("not implemented", "ImportState is implemented in a later change")
}

// ModifyPlan pins each entry's Optional+Computed email to its map key. A new
// map entry that omits email plans it as null, and Read then fills it from
// the key, which trips Terraform's plan-vs-apply consistency check. Pinning
// at plan time makes the planned and applied values identical.
func (r *TeamMembersResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan teamMembersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var planned types.Map
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(MEMBERS), &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pinned, d := pinMapKeysToAttr(teamMembersEntryObjectType(), planned, EMAIL)
	resp.Diagnostics.Append(d...)
	if !resp.Diagnostics.HasError() && !pinned.Equal(planned) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(MEMBERS), pinned)...)
	}
}
