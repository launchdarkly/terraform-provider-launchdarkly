package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v10"
)

func resourceTeam() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTeamCreate,
		ReadContext:   resourceTeamRead,
		UpdateContext: resourceTeamUpdate,
		DeleteContext: resourceTeamDelete,
		Exists:        resourceTeamExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The team's unique key",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The team's description",
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The team's human-readable name",
			},
			MEMBER_IDS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A list of team member IDs as strings",
			},
			MAINTAINERS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A list of team maintainer IDs as strings",
			},
			CUSTOM_ROLE_KEYS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A list of custom role keys for the team",
			},
		},
	}
}

func interfaceToArr(old interface{}) []string {
	interfaceArr := old.(*schema.Set).List()

	stringArr := make([]string, len(interfaceArr))
	for i, str := range interfaceArr {
		stringArr[i] = str.(string)
	}

	return stringArr
}

func makeAddAndRemoveArrays(old, updated []string) (remove, add []string) {
	m := make(map[string]bool)
	intersectionMap := make(map[string]bool)

	// creates the intersection
	for _, item := range old {
		m[item] = true
	}

	for _, item := range updated {
		if _, ok := m[item]; ok {
			intersectionMap[item] = true
		}
	}

	for _, item := range old {
		// if item in old isn't in intersecion append it
		_, ok := intersectionMap[item]
		if !ok {
			remove = append(remove, item)
		}
	}

	for _, item := range updated {
		// if item in new isn't in intersecion append it
		_, ok := intersectionMap[item]
		if !ok {
			add = append(add, item)
		}
	}

	return remove, add
}

func resourceTeamCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	description := d.Get(DESCRIPTION).(string)
	memberIDs := d.Get(MEMBER_IDS).(*schema.Set).List()
	maintainers := d.Get(MAINTAINERS).(*schema.Set).List()
	customRoleKeys := d.Get(CUSTOM_ROLE_KEYS).(*schema.Set).List()

	stringMemberIDs := make([]string, len(memberIDs))
	for i := range memberIDs {
		stringMemberIDs[i] = memberIDs[i].(string)
	}

	stringMaintainers := make([]string, len(maintainers))
	for i := range maintainers {
		stringMaintainers[i] = maintainers[i].(string)
	}

	stringCustomRoleKeys := make([]string, len(customRoleKeys))
	for i := range customRoleKeys {
		stringCustomRoleKeys[i] = customRoleKeys[i].(string)
	}

	maintainTeam := "maintainTeam"
	permissionGrantArray := make([]ldapi.PermissionGrantInput, 0)

	if len(maintainers) > 0 {
		permissionGrantArray = append(permissionGrantArray, ldapi.PermissionGrantInput{
			ActionSet: &maintainTeam,
			MemberIDs: stringMaintainers,
		})
	}

	teamBody := ldapi.TeamPostInput{
		CustomRoleKeys:   stringCustomRoleKeys,
		Description:      &description,
		Key:              key,
		MemberIDs:        stringMemberIDs,
		Name:             name,
		PermissionGrants: permissionGrantArray,
	}

	_, _, err := client.ld.TeamsApi.PostTeam(client.ctx).TeamPostInput(teamBody).Execute()

	if err != nil {
		return diag.Errorf("Error when calling `TeamsApi.PostTeam`: %v\n\n request: %v", err, teamBody)
	}
	d.SetId(key)

	return resourceTeamRead(ctx, d, metaRaw)
}

func resourceTeamRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	teamKey := d.Id()

	team, res, err := client.ld.TeamsApi.GetTeam(client.ctx, teamKey).Expand("roles,projects,maintainers").Execute()
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find team %q, removing from state", teamKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find team %q, removing from state", teamKey),
		})
		d.SetId("")
		return diags
	}

	if err != nil {
		return diag.Errorf("failed to get team %q: %v", teamKey, err)
	}

	members := make([]ldapi.Member, 0)
	i := int64(0)
	empty := ""
	next := &empty
	filter := fmt.Sprintf("team:%s", teamKey)

	for next != nil {
		memberResponse, res, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Limit(50).Offset(i * 50).Filter(filter).Execute()
		if isStatusNotFound(res) {
			log.Printf("[WARN] failed to find members for team %q, removing from state", teamKey)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("[WARN] failed to find members for team %q, removing team from state", teamKey),
			})
			d.SetId("")
			return diags
		}

		if err != nil {
			return diag.Errorf("failed to get members for team %q: %v", teamKey, err)
		}

		members = append(members, memberResponse.Items...)
		next = memberResponse.Links["next"].Href
		i++
	}

	customRoleKeys := make([]string, len(team.Roles.Items))
	for i, v := range team.Roles.Items {
		customRoleKeys[i] = *v.Key
	}

	maintainers := make([]string, len(team.Maintainers.Items))
	for i, m := range team.Maintainers.Items {
		maintainers[i] = m.Id
	}

	member_ids := make([]string, len(members))
	for i, m := range members {
		member_ids[i] = m.Id
	}

	if err != nil {
		return diag.Errorf("failed to get team %q: %s", teamKey, handleLdapiErr(err))
	}

	d.SetId(teamKey)
	_ = d.Set(KEY, team.Key)
	_ = d.Set(NAME, team.Name)
	_ = d.Set(DESCRIPTION, team.Description)
	_ = d.Set(MEMBER_IDS, member_ids)
	_ = d.Set(MAINTAINERS, maintainers)
	_ = d.Set(CUSTOM_ROLE_KEYS, customRoleKeys)

	return diags
}

func resourceTeamUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	instructions := make([]map[string]interface{}, 0)
	maintainTeam := "maintainTeam"

	if d.HasChange(NAME) {
		name := d.Get(NAME)
		instruction := make(map[string]interface{})
		instruction["kind"] = "updateName"
		instruction["value"] = name.(string)
		instructions = append(instructions, instruction)
	}
	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION)
		instruction := make(map[string]interface{})
		instruction["kind"] = "updateDescription"
		instruction["value"] = description.(string)
		instructions = append(instructions, instruction)
	}
	if d.HasChange(MEMBER_IDS) {
		old, update := d.GetChange(MEMBER_IDS)

		oldArr := interfaceToArr(old)
		updateArr := interfaceToArr(update)

		remove, add := makeAddAndRemoveArrays(oldArr, updateArr)

		fmt.Printf("Old members array: %v, New members array: %v", oldArr, updateArr)

		if len(remove) > 0 {
			instruction := make(map[string]interface{})
			instruction["kind"] = "removeMembers"
			instruction["values"] = remove
			instructions = append(instructions, instruction)
		}

		if len(add) > 0 {
			instruction := make(map[string]interface{})
			instruction["kind"] = "addMembers"
			instruction["values"] = add
			instructions = append(instructions, instruction)
		}
	}
	if d.HasChange(MAINTAINERS) {
		old, update := d.GetChange(MAINTAINERS)

		oldArr := interfaceToArr(old)
		updateArr := interfaceToArr(update)
		remove, add := makeAddAndRemoveArrays(oldArr, updateArr)

		if len(remove) > 0 {
			removeInstruction := make(map[string]interface{})
			removeInstruction["kind"] = "removePermissionGrants"
			removeInstruction["actionSet"] = maintainTeam
			removeInstruction["memberIDs"] = remove
			instructions = append(instructions, removeInstruction)
		}

		if len(add) > 0 {
			addInstruction := make(map[string]interface{})
			addInstruction["kind"] = "addPermissionGrants"
			addInstruction["actionSet"] = maintainTeam
			addInstruction["memberIDs"] = add
			instructions = append(instructions, addInstruction)
		}
	}

	if d.HasChange(CUSTOM_ROLE_KEYS) {
		old, update := d.GetChange(CUSTOM_ROLE_KEYS)
		oldArr := interfaceToArr(old)
		updateArr := interfaceToArr(update)

		remove, add := makeAddAndRemoveArrays(oldArr, updateArr)

		if len(remove) > 0 {
			instruction := make(map[string]interface{})
			instruction["kind"] = "removeCustomRoles"
			instruction["values"] = remove
			instructions = append(instructions, instruction)
		}

		if len(add) > 0 {
			instruction := make(map[string]interface{})
			instruction["kind"] = "addCustomRoles"
			instruction["values"] = add
			instructions = append(instructions, instruction)
		}
	}

	if len(instructions) > 0 {
		patch := ldapi.TeamPatchInput{
			Comment:      nil,
			Instructions: instructions,
		}

		teamKey := d.Get(KEY).(string)
		_, _, err := client.ld.TeamsApi.PatchTeam(client.ctx, teamKey).TeamPatchInput(patch).Execute()

		if err != nil {
			return diag.Errorf("failed to update team member with id %q: %s", teamKey, handleLdapiErr(err))
		}
	}

	return resourceTeamRead(ctx, d, metaRaw)
}

func resourceTeamDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)

	_, err := client.ld.TeamsApi.DeleteTeam(client.ctx, d.Id()).Execute()
	if err != nil {
		return diag.Errorf("failed to delete team with key: %q: %s", d.Id(), handleLdapiErr(err))
	}

	return diags
}

func resourceTeamExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return teamExists(d.Id(), metaRaw.(*Client))
}

func teamExists(teamKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.TeamsApi.GetTeam(meta.ctx, teamKey).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get team with key: %q: %v", teamKey, handleLdapiErr(err))
	}

	return true, nil
}
