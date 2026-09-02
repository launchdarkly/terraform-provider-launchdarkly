package launchdarkly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v24"
)

// conflictEmailExistsInAccount is the error code POST /api/v2/members returns
// when one or more of the posted emails already belong to a member of this
// account. It is the only member-create conflict this resource can recover
// from.
const conflictEmailExistsInAccount = "email_already_exists_in_account"

// teamMembersCreateMaxAttempts bounds the create/conflict loop. Each attempt
// can only shrink the pending set, so this is a safety net against a server
// that keeps reporting conflicts for emails we have already removed.
const teamMembersCreateMaxAttempts = 3

// membersConflictRep is the error body POST /api/v2/members returns for a
// member-create conflict.
type membersConflictRep struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	InvalidEmails []string `json:"invalid_emails"`
}

// parseMembersConflict reports the lowercased emails that already exist in the
// account when the given response body is that conflict. Any other error code,
// a malformed body, or a conflict naming no emails returns ok=false so the
// caller surfaces the original error instead of trying to recover.
func parseMembersConflict(body []byte) (emails []string, ok bool) {
	var rep membersConflictRep
	if err := json.Unmarshal(body, &rep); err != nil {
		return nil, false
	}
	if rep.Code != conflictEmailExistsInAccount || len(rep.InvalidEmails) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(rep.InvalidEmails))
	for _, e := range rep.InvalidEmails {
		out = append(out, strings.ToLower(strings.TrimSpace(e)))
	}
	return out, true
}

// membersConflictFromErr inspects the HTTP status before the body, so only a
// genuine 400 from the members endpoint can be interpreted as a conflict.
func membersConflictFromErr(err error, res *http.Response) (emails []string, ok bool) {
	if res == nil || res.StatusCode != http.StatusBadRequest {
		return nil, false
	}
	var apiErr *ldapi.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}
	return parseMembersConflict(apiErr.Body())
}

// resolvedMember is a batch email resolved to a real LaunchDarkly member.
type resolvedMember struct {
	ID      string
	Adopted bool
}

// newMemberFormFromEntry builds the create payload for one batch entry,
// mirroring launchdarkly_team_member's form construction.
//
// Team keys are deliberately NOT sent on the form. Inline teamKeys makes the
// API assign teams one member at a time inside the create request, and a large
// batch exceeds the service's request deadline (~24 seconds), killing the
// request after members were already persisted. Teams are assigned afterwards
// with one bounded PATCH per team instead (assignDeclaredTeams).
func newMemberFormFromEntry(ctx context.Context, email string, m teamMembersEntryModel) (ldapi.NewMemberForm, diag.Diagnostics) {
	var diags diag.Diagnostics
	form := ldapi.NewMemberForm{Email: email}

	if !m.FirstName.IsNull() && !m.FirstName.IsUnknown() {
		v := m.FirstName.ValueString()
		form.FirstName = &v
	}
	if !m.LastName.IsNull() && !m.LastName.IsUnknown() {
		v := m.LastName.ValueString()
		form.LastName = &v
	}
	if !m.Role.IsNull() && !m.Role.IsUnknown() && m.Role.ValueString() != "" {
		v := m.Role.ValueString()
		form.Role = &v
	}

	customRoles, d := stringSliceFromSet(ctx, m.CustomRoles)
	diags.Append(d...)
	form.CustomRoles = customRoles

	roleAttrs, d := frameworkRoleAttributesFromMap(ctx, m.RoleAttributes)
	diags.Append(d...)
	form.RoleAttributes = roleAttrs

	return form, diags
}

// removeFormsByEmail drops the named emails from a pending create payload.
func removeFormsByEmail(forms []ldapi.NewMemberForm, emails []string) []ldapi.NewMemberForm {
	if len(emails) == 0 {
		return forms
	}
	drop := make(map[string]struct{}, len(emails))
	for _, e := range emails {
		drop[strings.ToLower(e)] = struct{}{}
	}
	out := make([]ldapi.NewMemberForm, 0, len(forms))
	for _, f := range forms {
		if _, found := drop[strings.ToLower(f.Email)]; found {
			continue
		}
		out = append(out, f)
	}
	return out
}

// formEmails lists the emails still pending in a create payload.
func formEmails(forms []ldapi.NewMemberForm) []string {
	out := make([]string, 0, len(forms))
	for _, f := range forms {
		out = append(out, strings.ToLower(f.Email))
	}
	sort.Strings(out)
	return out
}

// memberIDPattern matches a LaunchDarkly member ID (a 24-character hex
// object ID). Validating import IDs up front turns a pasted email or team key
// into an immediate, named error instead of a confusing count-mismatch from
// the filtered lookup.
var memberIDPattern = regexp.MustCompile(`^[a-f0-9]{24}$`)

// parseImportMemberIDs splits and validates a comma-separated import ID.
func parseImportMemberIDs(raw string) ([]string, error) {
	ids := make([]string, 0)
	for _, id := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if !memberIDPattern.MatchString(trimmed) {
			return nil, fmt.Errorf(
				"%q is not a member ID (expected a 24-character hex ID such as 5f0cd446a77cba0b4c5644a7); "+
					"the import ID is a comma-separated list of member IDs", trimmed)
		}
		ids = append(ids, trimmed)
	}
	if len(ids) == 0 || len(ids) > teamMembersMaxBatchSize {
		return nil, fmt.Errorf(
			"expected between 1 and %d comma-separated member IDs, for example "+
				"'5f0cd446a77cba0b4c5644a7,5f0cd446a77cba0b4c5644a8'", teamMembersMaxBatchSize)
	}
	return ids, nil
}

// lookupMemberIDsByEmail resolves emails to member IDs with a single filtered
// request. Matching is exact and case-insensitive: the email filter is used
// rather than a free-text query, which would also match display names.
func (r *TeamMembersResource) lookupMemberIDsByEmail(emails []string) (map[string]string, error) {
	if len(emails) == 0 {
		return map[string]string{}, nil
	}
	members, err := getTeamMembersByEmail(r.client, emails)
	if err != nil {
		return nil, err
	}
	byEmail := make(map[string]string, len(members))
	for _, m := range members {
		byEmail[strings.ToLower(m.Email)] = m.Id
	}
	missing := make([]string, 0)
	for _, e := range emails {
		if _, found := byEmail[strings.ToLower(e)]; !found {
			missing = append(missing, e)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("could not find existing members for: %s", strings.Join(missing, ", "))
	}
	return byEmail, nil
}

// createMemberBatch invites the given entries with one request and resolves
// every email to a member ID.
//
// When the account already contains one of the emails, the behavior depends on
// adoptExisting: by default the apply fails and names the conflicting emails,
// because silently taking ownership of an existing person means this resource
// would later delete them. With adoptExisting the existing members are looked
// up, recorded as adopted, and the create is retried without them.
//
// It returns an error unless every entry ends up resolved, so callers never
// persist a partially-resolved batch.
func (r *TeamMembersResource) createMemberBatch(
	ctx context.Context,
	entries map[string]teamMembersEntryModel,
	adoptExisting bool,
) (resolved map[string]resolvedMember, adopted []string, diags diag.Diagnostics) {
	resolved = make(map[string]resolvedMember, len(entries))
	if len(entries) == 0 {
		return resolved, nil, diags
	}

	pending := make([]ldapi.NewMemberForm, 0, len(entries))
	for email, m := range entries {
		form, d := newMemberFormFromEntry(ctx, email, m)
		diags.Append(d...)
		if diags.HasError() {
			return nil, nil, diags
		}
		pending = append(pending, form)
	}
	// Deterministic request ordering keeps failures reproducible.
	sort.Slice(pending, func(i, j int) bool { return pending[i].Email < pending[j].Email })

	for attempt := 0; attempt < teamMembersCreateMaxAttempts && len(pending) > 0; attempt++ {
		var created *ldapi.Members
		var res *http.Response
		err := r.client.withConcurrency(r.client.ctx, func() error {
			var e error
			created, res, e = r.client.ld.AccountMembersApi.PostMembers(r.client.ctx).NewMemberForm(pending).Execute()
			return e
		})
		if err == nil {
			for _, item := range created.Items {
				resolved[strings.ToLower(item.Email)] = resolvedMember{ID: item.Id}
			}
			pending = nil
			break
		}

		conflicts, isConflict := membersConflictFromErr(err, res)
		if !isConflict {
			addLdapiError(&diags, fmt.Sprintf("Failed to invite %d team member(s)", len(pending)), err)
			return nil, nil, diags
		}
		if !adoptExisting {
			sort.Strings(conflicts)
			diags.AddError(
				"Members already exist in this account",
				fmt.Sprintf(
					"These emails already belong to members of this account: %s\n\n"+
						"Either remove them from the members map, or set adopt_existing = true to manage them "+
						"with this resource. Adopted members have their roles and team assignments reconciled to "+
						"this configuration, and are deleted when removed from the batch or when this resource is "+
						"destroyed.",
					strings.Join(conflicts, ", "),
				),
			)
			return nil, nil, diags
		}
		found, lookupErr := r.lookupMemberIDsByEmail(conflicts)
		if lookupErr != nil {
			diags.AddError("Failed to adopt existing members", lookupErr.Error())
			return nil, nil, diags
		}
		for email, id := range found {
			resolved[email] = resolvedMember{ID: id, Adopted: true}
			adopted = append(adopted, email)
		}
		pending = removeFormsByEmail(pending, conflicts)
	}

	if len(pending) > 0 {
		diags.AddError(
			"Could not invite every team member",
			fmt.Sprintf(
				"Gave up after %d attempts with these members unresolved: %s. No partial state was saved; "+
					"re-run the apply to try again.",
				teamMembersCreateMaxAttempts, strings.Join(formEmails(pending), ", "),
			),
		)
		return nil, nil, diags
	}

	// Every requested entry must be accounted for before the caller writes state.
	unresolved := make([]string, 0)
	for email := range entries {
		if _, found := resolved[email]; !found {
			unresolved = append(unresolved, email)
		}
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		diags.AddError(
			"LaunchDarkly did not return every invited member",
			fmt.Sprintf("No member was returned for: %s", strings.Join(unresolved, ", ")),
		)
		return nil, nil, diags
	}

	sort.Strings(adopted)
	return resolved, adopted, diags
}

// applyResolvedIDs copies resolved member IDs onto the matching map entries.
func applyResolvedIDs(members map[string]teamMembersEntryModel, resolved map[string]resolvedMember) {
	for email, rm := range resolved {
		entry, found := members[email]
		if !found {
			continue
		}
		entry.ID = types.StringValue(rm.ID)
		members[email] = entry
	}
}

// memberIDsFromModel lists the member IDs currently recorded in the model.
func memberIDsFromModel(members map[string]teamMembersEntryModel) []string {
	ids := make([]string, 0, len(members))
	for _, m := range members {
		if !m.ID.IsNull() && !m.ID.IsUnknown() && m.ID.ValueString() != "" {
			ids = append(ids, m.ID.ValueString())
		}
	}
	sort.Strings(ids)
	return ids
}

// fetchMembersByID resolves member IDs to members with one filtered, paginated
// request rather than one request per member: a 50-member batch refreshing 50
// times would recreate the very rate-limit problem this resource exists to
// avoid. Members that no longer exist are simply absent from the result.
//
// Note the returned map is keyed by LOWERCASE EMAIL, not by the IDs queried:
// email is this resource's natural key, and every caller joins the result
// back onto the email-keyed members map.
func (r *TeamMembersResource) fetchMembersByID(ids []string) (map[string]ldapi.Member, error) {
	if len(ids) == 0 {
		return map[string]ldapi.Member{}, nil
	}
	idFilter := fmt.Sprintf("id:%s", strings.Join(ids, "|"))
	expand := "roleAttributes"
	members, err := getMembersPaginated(r.client, &idFilter, &expand, nil, teamMemberLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members by ID: %v", handleLdapiErr(err))
	}
	byEmail := make(map[string]ldapi.Member, len(members))
	for _, m := range members {
		byEmail[strings.ToLower(m.Email)] = m
	}
	return byEmail, nil
}

// customRoleResolver caches custom-role ID<->key lookups for the lifetime of
// one CRUD call. The API reports role IDs while configuration uses keys, and a
// batch usually reuses the same handful of roles, so each distinct role is
// resolved at most once per operation in either direction. Uncached values are
// collected and resolved in one helper call per direction, so a future bulk
// endpoint adoption is a single-site change.
type customRoleResolver struct {
	client  *Client
	idToKey map[string]string
	keyToID map[string]string
}

func newCustomRoleResolver(client *Client) *customRoleResolver {
	return &customRoleResolver{client: client, idToKey: map[string]string{}, keyToID: map[string]string{}}
}

// keysFor resolves role IDs to keys, preserving input order.
func (c *customRoleResolver) keysFor(ids []string) ([]string, error) {
	uncached := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, found := c.idToKey[id]; !found {
			uncached = append(uncached, id)
		}
	}
	if len(uncached) > 0 {
		resolved, err := customRoleIDsToKeys(c.client, uncached)
		if err != nil {
			return nil, err
		}
		if len(resolved) != len(uncached) {
			return nil, fmt.Errorf("expected %d custom role keys, got %d", len(uncached), len(resolved))
		}
		for i, id := range uncached {
			c.idToKey[id] = resolved[i]
			c.keyToID[resolved[i]] = id
		}
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, c.idToKey[id])
	}
	return keys, nil
}

// idsFor resolves role keys to IDs, preserving input order.
func (c *customRoleResolver) idsFor(keys []string) ([]string, error) {
	uncached := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, found := c.keyToID[key]; !found {
			uncached = append(uncached, key)
		}
	}
	if len(uncached) > 0 {
		resolved, err := customRoleKeysToIDs(c.client, uncached)
		if err != nil {
			return nil, err
		}
		if len(resolved) != len(uncached) {
			return nil, fmt.Errorf("expected %d custom role IDs, got %d", len(uncached), len(resolved))
		}
		for i, key := range uncached {
			c.keyToID[key] = resolved[i]
			c.idToKey[resolved[i]] = key
		}
	}
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, c.keyToID[key])
	}
	return ids, nil
}

// refreshEntryFromMember maps live API state onto one batch entry.
//
// Two deliberate choices: first and last name are left as configured, because
// LaunchDarkly does not let the provider change them after creation and
// overwriting them here would produce permanent diffs; and team_keys reports
// only the teams this entry declares, so teams a member joined elsewhere are
// not pulled in as drift.
func refreshEntryFromMember(
	ctx context.Context,
	entry *teamMembersEntryModel,
	member *ldapi.Member,
	roles *customRoleResolver,
	diags *diag.Diagnostics,
) {
	entry.ID = types.StringValue(member.Id)
	entry.Email = types.StringValue(strings.ToLower(member.Email))
	entry.Role = types.StringValue(member.Role)

	customRoleKeys, err := roles.keysFor(member.CustomRoles)
	if err != nil {
		diags.AddError("Failed to resolve custom role keys", err.Error())
		return
	}
	rolesSet, d := setFromStringSlicePreservingPlan(ctx, customRoleKeys, entry.CustomRoles)
	diags.Append(d...)
	entry.CustomRoles = rolesSet

	declared, d := stringSliceFromSet(ctx, entry.TeamKeys)
	diags.Append(d...)
	if !diags.HasError() {
		live := make(map[string]struct{}, len(member.Teams))
		for _, t := range member.Teams {
			live[t.Key] = struct{}{}
		}
		stillAssigned := make([]string, 0, len(declared))
		for _, key := range declared {
			if _, found := live[key]; found {
				stillAssigned = append(stillAssigned, key)
			}
		}
		teamsSet, d := setFromStringSlicePreservingPlan(ctx, stillAssigned, entry.TeamKeys)
		diags.Append(d...)
		entry.TeamKeys = teamsSet
	}

	attrs, d := frameworkRoleAttributesValue(ctx, member.RoleAttributes)
	diags.Append(d...)
	entry.RoleAttributes = attrs
}

// hydrateMembers refreshes every entry from the API in one request so that the
// state written at the end of an apply reflects what LaunchDarkly actually
// holds. It fails if any managed member is missing, rather than writing state
// that claims a member exists when it does not.
func (r *TeamMembersResource) hydrateMembers(ctx context.Context, model *teamMembersResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	ids := memberIDsFromModel(model.Members)
	if len(ids) == 0 {
		return diags
	}
	live, err := r.fetchMembersByID(ids)
	if err != nil {
		diags.AddError("Failed to read back team members", err.Error())
		return diags
	}
	roles := newCustomRoleResolver(r.client)
	missing := make([]string, 0)
	for email, entry := range model.Members {
		member, found := live[email]
		if !found {
			missing = append(missing, email)
			continue
		}
		refreshEntryFromMember(ctx, &entry, &member, roles, &diags)
		if diags.HasError() {
			return diags
		}
		model.Members[email] = entry
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		diags.AddError(
			"Team members missing after write",
			fmt.Sprintf("LaunchDarkly did not return these members immediately after they were written: %s", strings.Join(missing, ", ")),
		)
	}
	return diags
}

// newTeamMembersBatchID mints the resource's own identifier. The batch has no
// natural key of its own — its members are keyed by email — so a generated ID
// is used and then held stable by UseStateForUnknown.
func newTeamMembersBatchID() string {
	return uuid.New().String()
}

// patchableAttrsDiffer reports whether the attributes a per-member PATCH can
// change differ between two entries. Team keys are deliberately excluded:
// team membership is reconciled with grouped per-team requests, so a team-only
// difference must never trigger a member PATCH — at batch scale that turns
// into one wasted request per member.
func patchableAttrsDiffer(a, b teamMembersEntryModel) bool {
	return !a.Role.Equal(b.Role) ||
		!a.CustomRoles.Equal(b.CustomRoles) ||
		!a.RoleAttributes.Equal(b.RoleAttributes)
}

// memberAttrsDiffer reports whether the attributes this resource can actually
// change differ between two entries, including team membership. First and last
// name are excluded because LaunchDarkly does not allow the provider to update
// them after creation. Used to classify entries during diffing; the narrower
// patchableAttrsDiffer decides whether a member PATCH is actually sent.
func memberAttrsDiffer(a, b teamMembersEntryModel) bool {
	return patchableAttrsDiffer(a, b) ||
		!a.TeamKeys.Equal(b.TeamKeys)
}

// patchMemberAttributes brings one member's role, custom roles, and role
// attributes in line with the configuration, mirroring how
// launchdarkly_team_member builds its patch. Custom-role key->ID lookups go
// through the shared per-operation resolver so a batch of members sharing a
// role costs one lookup, not one per member.
func (r *TeamMembersResource) patchMemberAttributes(
	ctx context.Context,
	memberID string,
	desired teamMembersEntryModel,
	prior teamMembersEntryModel,
	roles *customRoleResolver,
) diag.Diagnostics {
	var diags diag.Diagnostics

	role := desired.Role.ValueString()
	customRoleKeys, d := stringSliceFromSet(ctx, desired.CustomRoles)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	customRoleIDs, err := roles.idsFor(customRoleKeys)
	if err != nil {
		diags.AddError("Failed to look up custom role IDs", err.Error())
		return diags
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/role", &role),
		patchReplace("/customRoles", &customRoleIDs),
	}
	patch = append(patch, frameworkRoleAttributePatches(ctx, desired.RoleAttributes, prior.RoleAttributes)...)

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AccountMembersApi.PatchMember(r.client.ctx, memberID).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&diags, fmt.Sprintf("Failed to update team member %q", desired.Email.ValueString()), err)
	}
	return diags
}

// teamMembershipDelta is the set of member IDs to add to or remove from a team.
type teamMembershipDelta struct {
	add    []string
	remove []string
}

// applyTeamMembershipDeltas groups membership changes by team so that a batch
// of members joining the same team costs one request per team rather than one
// per member.
func (r *TeamMembersResource) applyTeamMembershipDeltas(deltas map[string]*teamMembershipDelta) diag.Diagnostics {
	var diags diag.Diagnostics
	teamKeys := make([]string, 0, len(deltas))
	for key := range deltas {
		teamKeys = append(teamKeys, key)
	}
	sort.Strings(teamKeys)

	for _, teamKey := range teamKeys {
		delta := deltas[teamKey]
		instructions := make([]map[string]interface{}, 0, 2)
		if len(delta.remove) > 0 {
			sort.Strings(delta.remove)
			instructions = append(instructions, map[string]interface{}{"kind": "removeMembers", "values": delta.remove})
		}
		if len(delta.add) > 0 {
			sort.Strings(delta.add)
			instructions = append(instructions, map[string]interface{}{"kind": "addMembers", "values": delta.add})
		}
		if len(instructions) == 0 {
			continue
		}
		patch := ldapi.TeamPatchInput{Instructions: instructions}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.TeamsApi.PatchTeam(r.client.ctx, teamKey).TeamPatchInput(patch).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&diags, fmt.Sprintf("Failed to update membership of team %q", teamKey), err)
			return diags
		}
	}
	return diags
}

// createdEmails lists the emails that were created (not adopted) in a resolved
// batch, sorted for deterministic request ordering.
func createdEmails(resolved map[string]resolvedMember) []string {
	out := make([]string, 0, len(resolved))
	for email, rm := range resolved {
		if !rm.Adopted {
			out = append(out, email)
		}
	}
	sort.Strings(out)
	return out
}

// assignDeclaredTeams adds newly created members to their declared teams with
// one grouped PATCH per team. This is the second half of Create: the POST
// carries no teamKeys (see newMemberFormFromEntry), so every request stays
// small enough to finish well under the service's ~24s request deadline, and a
// failure is scoped to one team rather than part-way through a per-member loop
// inside the create request. Adopted members are handled separately by
// reconcileAdoptedMembers, which diffs against their live team membership.
func (r *TeamMembersResource) assignDeclaredTeams(
	ctx context.Context,
	members map[string]teamMembersEntryModel,
	emails []string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	deltas := map[string]*teamMembershipDelta{}
	for _, email := range emails {
		entry, found := members[email]
		if !found || entry.ID.IsNull() || entry.ID.IsUnknown() || entry.ID.ValueString() == "" {
			continue
		}
		keys, d := stringSliceFromSet(ctx, entry.TeamKeys)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		for _, key := range keys {
			if deltas[key] == nil {
				deltas[key] = &teamMembershipDelta{}
			}
			deltas[key].add = append(deltas[key].add, entry.ID.ValueString())
		}
	}
	diags.Append(r.applyTeamMembershipDeltas(deltas)...)
	return diags
}

// collectTeamDeltas records the team membership changes needed to move one
// member from its prior team set to the desired one.
func collectTeamDeltas(
	ctx context.Context,
	deltas map[string]*teamMembershipDelta,
	memberID string,
	desired, prior teamMembersEntryModel,
	diags *diag.Diagnostics,
) {
	desiredKeys, d := stringSliceFromSet(ctx, desired.TeamKeys)
	diags.Append(d...)
	priorKeys, d := stringSliceFromSet(ctx, prior.TeamKeys)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	remove, add := stringAddRemove(priorKeys, desiredKeys)
	for _, key := range add {
		if deltas[key] == nil {
			deltas[key] = &teamMembershipDelta{}
		}
		deltas[key].add = append(deltas[key].add, memberID)
	}
	for _, key := range remove {
		if deltas[key] == nil {
			deltas[key] = &teamMembershipDelta{}
		}
		deltas[key].remove = append(deltas[key].remove, memberID)
	}
}

// reconcileAdoptedMembers aligns members that were adopted rather than created
// with the configuration: adoption should not leave a member's role or team
// assignments at whatever they happened to be beforehand.
func (r *TeamMembersResource) reconcileAdoptedMembers(
	ctx context.Context,
	members map[string]teamMembersEntryModel,
	adopted []string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(adopted) == 0 {
		return diags
	}

	ids := make([]string, 0, len(adopted))
	for _, email := range adopted {
		if entry, found := members[email]; found && !entry.ID.IsNull() {
			ids = append(ids, entry.ID.ValueString())
		}
	}
	live, err := r.fetchMembersByID(ids)
	if err != nil {
		diags.AddError("Failed to read adopted members", err.Error())
		return diags
	}

	roles := newCustomRoleResolver(r.client)
	deltas := map[string]*teamMembershipDelta{}
	for _, email := range adopted {
		desired, found := members[email]
		if !found {
			continue
		}
		member, found := live[email]
		if !found {
			diags.AddError("Failed to read adopted member", fmt.Sprintf("no member returned for %s", email))
			return diags
		}
		// Build the member's current state so the patch only sends real changes.
		current := desired
		refreshEntryFromMember(ctx, &current, &member, roles, &diags)
		if diags.HasError() {
			return diags
		}
		// current.TeamKeys is scoped to declared teams, so compare against the
		// member's full live membership when computing team deltas.
		liveTeams := make([]string, 0, len(member.Teams))
		for _, t := range member.Teams {
			liveTeams = append(liveTeams, t.Key)
		}
		liveTeamSet, d := setFromStringSlicePreservingPlan(ctx, liveTeams, desired.TeamKeys)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		priorForTeams := current
		priorForTeams.TeamKeys = liveTeamSet

		if patchableAttrsDiffer(current, desired) {
			diags.Append(r.patchMemberAttributes(ctx, member.Id, desired, current, roles)...)
			if diags.HasError() {
				return diags
			}
		}
		collectTeamDeltas(ctx, deltas, member.Id, desired, priorForTeams, &diags)
		if diags.HasError() {
			return diags
		}
	}

	// Only add adopted members to declared teams; never remove them from teams
	// this configuration does not mention.
	for _, delta := range deltas {
		delta.remove = nil
	}
	diags.Append(r.applyTeamMembershipDeltas(deltas)...)
	return diags
}

// memberBatchDiff is the work needed to move the batch from state to plan.
type memberBatchDiff struct {
	// toCreate holds entries present in the plan but not in state.
	toCreate map[string]teamMembersEntryModel
	// toPatch holds entries whose updatable attributes changed, each carrying
	// its existing member ID.
	toPatch map[string]teamMembersEntryModel
	// toDeleteIDs holds the member IDs of entries dropped from the plan.
	toDeleteIDs []string
	// retained maps unchanged entries to their existing member IDs.
	retained map[string]string
}

// diffMemberBatches compares state and plan. Both are keyed by email, so
// membership changes are a direct key comparison rather than an inference from
// attribute values.
func diffMemberBatches(state, plan map[string]teamMembersEntryModel) memberBatchDiff {
	d := memberBatchDiff{
		toCreate: map[string]teamMembersEntryModel{},
		toPatch:  map[string]teamMembersEntryModel{},
		retained: map[string]string{},
	}
	for email, planned := range plan {
		prior, exists := state[email]
		if !exists {
			d.toCreate[email] = planned
			continue
		}
		if memberAttrsDiffer(prior, planned) {
			planned.ID = prior.ID
			d.toPatch[email] = planned
			continue
		}
		if !prior.ID.IsNull() && prior.ID.ValueString() != "" {
			d.retained[email] = prior.ID.ValueString()
		}
	}
	for email, prior := range state {
		if _, stillPresent := plan[email]; stillPresent {
			continue
		}
		if !prior.ID.IsNull() && prior.ID.ValueString() != "" {
			d.toDeleteIDs = append(d.toDeleteIDs, prior.ID.ValueString())
		}
	}
	sort.Strings(d.toDeleteIDs)
	return d
}

// isFullReplacement reports whether applying the diff would delete every
// member this resource currently manages. Entries without an ID were never
// created, so they do not count toward the total.
func isFullReplacement(state map[string]teamMembersEntryModel, d memberBatchDiff) bool {
	managed := 0
	for _, m := range state {
		if !m.ID.IsNull() && m.ID.ValueString() != "" {
			managed++
		}
	}
	return managed > 0 && len(d.toDeleteIDs) == managed
}

// patchChangedMembers applies attribute and team-membership changes for every
// entry whose configuration moved.
func (r *TeamMembersResource) patchChangedMembers(
	ctx context.Context,
	changed map[string]teamMembersEntryModel,
	state map[string]teamMembersEntryModel,
) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(changed) == 0 {
		return diags
	}

	emails := sortedMemberEmails(changed)
	deltas := map[string]*teamMembershipDelta{}
	roles := newCustomRoleResolver(r.client)
	for _, email := range emails {
		desired := changed[email]
		prior := state[email]
		memberID := desired.ID.ValueString()
		if memberID == "" {
			diags.AddError(
				"Cannot update team member",
				fmt.Sprintf("no member ID recorded for %s", email),
			)
			return diags
		}
		if patchableAttrsDiffer(desired, prior) {
			diags.Append(r.patchMemberAttributes(ctx, memberID, desired, prior, roles)...)
			if diags.HasError() {
				return diags
			}
		}
		if !desired.TeamKeys.Equal(prior.TeamKeys) {
			collectTeamDeltas(ctx, deltas, memberID, desired, prior, &diags)
			if diags.HasError() {
				return diags
			}
		}
	}
	diags.Append(r.applyTeamMembershipDeltas(deltas)...)
	return diags
}

// sortedMemberEmails gives a stable iteration order over a members map.
func sortedMemberEmails(members map[string]teamMembersEntryModel) []string {
	out := make([]string, 0, len(members))
	for email := range members {
		out = append(out, email)
	}
	sort.Strings(out)
	return out
}

// deleteMembersByID removes members one at a time, since the API has no bulk
// delete. Members already gone are treated as success.
func (r *TeamMembersResource) deleteMembersByID(ids []string) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, id := range ids {
		var res *http.Response
		err := r.client.withConcurrency(r.client.ctx, func() error {
			var e error
			res, e = r.client.ld.AccountMembersApi.DeleteMember(r.client.ctx, id).Execute()
			return e
		})
		if err != nil && !isStatusNotFound(res) {
			addLdapiError(&diags, fmt.Sprintf("Failed to delete team member %q", id), err)
			return diags
		}
	}
	return diags
}

// seatLimitDiagnostics reports whether a failure looks like a seat or invite
// limit, which is worth explaining because this resource adds members before
// removing them.
func seatLimitDiagnostics(diags diag.Diagnostics) bool {
	for _, d := range diags.Errors() {
		text := strings.ToLower(d.Summary() + " " + d.Detail())
		if strings.Contains(text, "seat_limit_reached") ||
			strings.Contains(text, "invite_limit_reached") ||
			strings.Contains(text, "seat limit") {
			return true
		}
	}
	return false
}

// markNewEntriesIDUnknown sets the computed id of planned map entries that have
// no prior state to Unknown.
//
// The framework plans a brand-new map element's Computed attributes as null,
// because UseStateForUnknown has no prior state to carry forward, and apply
// then filling in a real member ID trips Terraform's plan-versus-apply
// consistency check. Only the update path reaches this: on create the whole
// resource is new, so its attributes are already unknown.
func markNewEntriesIDUnknown(
	objType types.ObjectType,
	planned types.Map,
	priorState map[string]teamMembersEntryModel,
) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if planned.IsNull() || planned.IsUnknown() {
		return planned, diags
	}
	els := planned.Elements()
	if len(els) == 0 {
		return planned, diags
	}
	out := make(map[string]attr.Value, len(els))
	changed := false
	for key, el := range els {
		obj, ok := el.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			out[key] = el
			continue
		}
		prior, existed := priorState[key]
		hadID := existed && !prior.ID.IsNull() && prior.ID.ValueString() != ""
		attrs := obj.Attributes()
		idVal, hasID := attrs[ID]
		if !hadID && hasID {
			if sv, isStr := idVal.(types.String); isStr && sv.IsNull() {
				attrs[ID] = types.StringUnknown()
				newObj, d := types.ObjectValue(objType.AttrTypes, attrs)
				diags.Append(d...)
				out[key] = newObj
				changed = true
				continue
			}
		}
		out[key] = el
	}
	if !changed || diags.HasError() {
		return planned, diags
	}
	newMap, d := types.MapValue(objType, out)
	diags.Append(d...)
	return newMap, diags
}
