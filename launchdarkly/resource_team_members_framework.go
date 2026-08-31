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

// Destroy-guard text, shared by ModifyPlan and Delete so the plan-time and
// apply-time messages cannot drift apart.
const (
	teamMembersDestroyBlockedSummary = "Cannot destroy: deletion protection is enabled"
	teamMembersDestroyBlockedDetail  = "Destroying this resource deletes every member in the batch from your " +
		"LaunchDarkly account.\n\nTo proceed:\n\n" +
		"  1. Set deletion_protection = false on this resource and run terraform apply\n" +
		"  2. Run the destroy again\n\n" +
		"If you have already removed the resource block from your configuration, restore it, apply with " +
		"deletion_protection = false, then remove it again."
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

-> **Note:** A large batch, especially one that also assigns teams, can take longer than the provider's default 20 second ` + "`http_timeout`" + `. Raise it on the provider when inviting tens of members at once: a 50 member batch with team assignments has been observed to need well over a minute. If the request does time out, LaunchDarkly may still have created the members even though Terraform recorded nothing; re-apply with ` + "`adopt_existing = true`" + ` to take ownership of them and finish the work.

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
	var plan teamMembersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateMemberBatch(plan.Members); err != nil {
		resp.Diagnostics.AddError("Invalid members batch", err.Error())
		return
	}

	resolved, adopted, diags := r.createMemberBatch(ctx, plan.Members, plan.AdoptExisting.ValueBool())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	applyResolvedIDs(plan.Members, resolved)

	if len(adopted) > 0 {
		resp.Diagnostics.AddWarning(
			"Adopted existing team members",
			fmt.Sprintf(
				"These members already existed and are now managed by this resource, which means they will be "+
					"deleted if you remove them from the batch or destroy the resource: %s",
				strings.Join(adopted, ", "),
			),
		)
		// Bring adopted members in line with the configuration in this same
		// apply, so adoption does not leave their roles or teams stale.
		resp.Diagnostics.Append(r.reconcileAdoptedMembers(ctx, plan.Members, adopted)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(r.hydrateMembers(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(newTeamMembersBatchID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamMembersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := r.fetchMembersByID(memberIDsFromModel(state.Members))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team members", err.Error())
		return
	}
	roles := newCustomRoleKeyResolver(r.client)
	for email, entry := range state.Members {
		member, found := live[email]
		if !found {
			// Deleted outside Terraform: drop it so the next plan recreates it.
			delete(state.Members, email)
			continue
		}
		refreshEntryFromMember(ctx, &entry, &member, roles, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Members[email] = entry
	}
	if len(state.Members) == 0 {
		// Every managed member is gone; drop the resource so it is recreated.
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state teamMembersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateMemberBatch(plan.Members); err != nil {
		resp.Diagnostics.AddError("Invalid members batch", err.Error())
		return
	}

	diff := diffMemberBatches(state.Members, plan.Members)

	// Protection is read from prior state, not the plan, so that disabling it
	// and performing the destructive change cannot happen in a single apply.
	if state.DeletionProtection.ValueBool() && isFullReplacement(state.Members, diff) {
		resp.Diagnostics.AddError(
			"Refusing to replace every member in the batch",
			"This update removes every member this resource manages, which deletes those members from your "+
				"LaunchDarkly account.\n\nIf that is intentional, set deletion_protection = false and apply that "+
				"change on its own, then apply this replacement.",
		)
		return
	}

	// Members are created before removed ones are deleted: a failure part-way
	// through then leaves people with access rather than without it. The
	// tradeoff is that swapping a full batch needs a spare seat.
	if len(diff.toCreate) > 0 {
		resolved, adopted, diags := r.createMemberBatch(ctx, diff.toCreate, plan.AdoptExisting.ValueBool())
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			if seatLimitDiagnostics(diags) {
				resp.Diagnostics.AddWarning(
					"Seat limit reached while adding members",
					"New members are invited before removed ones are deleted, so replacing members at exactly "+
						"your seat limit fails here. Remove the departing members in one apply and add the new "+
						"ones in the next, or add seats.",
				)
			}
			return
		}
		applyResolvedIDs(plan.Members, resolved)
		if len(adopted) > 0 {
			resp.Diagnostics.AddWarning(
				"Adopted existing team members",
				fmt.Sprintf(
					"These members already existed and are now managed by this resource, which means they will "+
						"be deleted if you remove them from the batch or destroy the resource: %s",
					strings.Join(adopted, ", "),
				),
			)
			resp.Diagnostics.Append(r.reconcileAdoptedMembers(ctx, plan.Members, adopted)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Unchanged and changed entries keep the IDs they already had.
	for email, id := range diff.retained {
		if entry, found := plan.Members[email]; found {
			entry.ID = types.StringValue(id)
			plan.Members[email] = entry
		}
	}
	for email, patched := range diff.toPatch {
		if entry, found := plan.Members[email]; found {
			entry.ID = patched.ID
			plan.Members[email] = entry
		}
	}

	resp.Diagnostics.Append(r.patchChangedMembers(ctx, diff.toPatch, state.Members)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.deleteMembersByID(diff.toDeleteIDs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.hydrateMembers(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamMembersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(teamMembersDestroyBlockedSummary, teamMembersDestroyBlockedDetail)
		return
	}
	resp.Diagnostics.Append(r.deleteMembersByID(memberIDsFromModel(state.Members))...)
}

func (r *TeamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rawIDs := strings.Split(req.ID, ",")
	ids := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	if len(ids) == 0 || len(ids) > teamMembersMaxBatchSize {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf(
				"Expected between 1 and %d comma-separated member IDs, for example "+
					"'5f0cd446a77cba0b4c5644a7,5f0cd446a77cba0b4c5644a8'.",
				teamMembersMaxBatchSize,
			),
		)
		return
	}

	live, err := r.fetchMembersByID(ids)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import team members", err.Error())
		return
	}
	if len(live) != len(ids) {
		resp.Diagnostics.AddError(
			"Failed to import team members",
			fmt.Sprintf("Asked for %d member IDs but LaunchDarkly returned %d. Check that every ID exists.", len(ids), len(live)),
		)
		return
	}

	roles := newCustomRoleKeyResolver(r.client)
	members := make(map[string]teamMembersEntryModel, len(live))
	for email, member := range live {
		// Import records only what this resource reconciles: identity, role,
		// custom roles, and role attributes.
		//
		// first_name and last_name are left null because they are used only
		// when a member is created and are never read back, so importing them
		// would produce a diff against any configuration that omits them.
		//
		// team_keys is left null because this resource manages only the team
		// associations a configuration declares. Recording a member's full
		// current membership would mean the first apply after an import
		// silently removed them from every team the configuration did not
		// happen to mention.
		entry := teamMembersEntryModel{
			ID:             types.StringValue(member.Id),
			Email:          types.StringValue(email),
			FirstName:      types.StringNull(),
			LastName:       types.StringNull(),
			CustomRoles:    types.SetNull(types.StringType),
			TeamKeys:       types.SetNull(types.StringType),
			RoleAttributes: types.MapNull(types.ListType{ElemType: types.StringType}),
		}
		refreshEntryFromMember(ctx, &entry, &member, roles, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		members[email] = entry
	}

	state := teamMembersResourceModel{
		ID:                 types.StringValue(newTeamMembersBatchID()),
		AdoptExisting:      types.BoolValue(false),
		DeletionProtection: types.BoolValue(true),
		Members:            members,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ModifyPlan pins each entry's email to its map key and surfaces the deletion
// guards while planning, so a destructive change is reported before anyone
// approves an apply that could not succeed.
func (r *TeamMembersResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy: the plan is null and only prior state is available.
	if req.Plan.Raw.IsNull() {
		if req.State.Raw.IsNull() {
			return
		}
		var state teamMembersResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if state.DeletionProtection.ValueBool() {
			resp.Diagnostics.AddError(teamMembersDestroyBlockedSummary, teamMembersDestroyBlockedDetail)
		}
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
	if resp.Diagnostics.HasError() {
		return
	}

	// On an update, entries the prior state does not know about need their
	// computed id planned as unknown rather than null.
	var priorMembers map[string]teamMembersEntryModel
	if !req.State.Raw.IsNull() {
		var state teamMembersResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		priorMembers = state.Members
		pinned, d = markNewEntriesIDUnknown(teamMembersEntryObjectType(), pinned, priorMembers)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !pinned.Equal(planned) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(MEMBERS), pinned)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update: refuse a whole-batch replacement while protection is on.
	if priorMembers == nil {
		return
	}
	var state teamMembersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.DeletionProtection.ValueBool() || plan.Members == nil {
		return
	}
	if isFullReplacement(state.Members, diffMemberBatches(state.Members, plan.Members)) {
		resp.Diagnostics.AddError(
			"Refusing to replace every member in the batch",
			"This change removes every member this resource manages, which deletes those members from your "+
				"LaunchDarkly account.\n\nIf that is intentional, set deletion_protection = false and apply that "+
				"change on its own, then apply this replacement.",
		)
	}
}
