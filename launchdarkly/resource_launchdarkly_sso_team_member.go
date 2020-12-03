package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceSSOTeamMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceSSOMemberCreate,
		Read:   resourceSSOMemberRead,
		Update: resourceSSOMemberUpdate,
		Delete: resourceSSOMemberDelete,
		Exists: resourceSSOMemberExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			EMAIL: {
				Type:     schema.TypeString,
				Required: true,
			},
			ROLE: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateTeamMemberRole,
			},
			CUSTOM_ROLES: {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceSSOMemberCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberEmail := d.Get(EMAIL).(string)

	member, err := getTeamMemberByEmail(client, memberEmail)
	if err != nil {
		return err
	}
	memberID := member.Id
	memberRole := ldapi.Role(d.Get(ROLE).(string))
	if len(memberRole) == 0 {
		memberRole = *member.Role
	}
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoleKeys := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoleKeys[i] = cr.(string)
	}
	customRoleIds, err := customRoleKeysToIDs(client, customRoleKeys)
	if err != nil {
		return err
	}

	d.SetId(memberID)
	patch := []ldapi.PatchOperation{
		// these are the only fields we are allowed to update:
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRoleIds),
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.TeamMembersApi.PatchMember(client.ctx, d.Id(), patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to assign role to team member with id %q: %s", memberID, handleLdapiErr(err))
	}
	return resourceSSOMemberRead(d, metaRaw)
}

func resourceSSOMemberRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberID := d.Id()

	memberRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.TeamMembersApi.GetMember(client.ctx, memberID)
	})
	member := memberRaw.(ldapi.Member)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find member with id %q, removing from state", memberID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get member with id %q: %v", memberID, err)
	}

	d.SetId(member.Id)
	_ = d.Set(EMAIL, member.Email)
	_ = d.Set(ROLE, member.Role)

	customRoleKeys, err := customRoleIDsToKeys(client, member.CustomRoles)
	if err != nil {
		return err
	}
	err = d.Set(CUSTOM_ROLES, customRoleKeys)
	if err != nil {
		return fmt.Errorf("failed to set custom roles on team member with id %q: %v", member.Id, err)
	}
	return nil
}

func resourceSSOMemberDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.TeamMembersApi.DeleteMember(client.ctx, d.Id())
		return nil, res, err
	})
	if err != nil {
		return fmt.Errorf("failed to delete team member with id %q: %s", d.Id(), handleLdapiErr(err))
	}

	return nil
}

func resourceSSOMemberExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return teamMemberExists(d.Id(), metaRaw.(*Client))
}

func resourceSSOMemberUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	memberRole := ldapi.Role(d.Get(ROLE).(string))
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoleKeys := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoleKeys[i] = cr.(string)
	}
	customRoleIds, err := customRoleKeysToIDs(client, customRoleKeys)
	if err != nil {
		return err
	}

	patch := []ldapi.PatchOperation{
		// these are the only fields we are allowed to update:
		patchReplace("/role", &memberRole),
		patchReplace("/customRoles", &customRoleIds),
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.TeamMembersApi.PatchMember(client.ctx, d.Id(), patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to assign role to team member with id %q: %s", d.Id(), handleLdapiErr(err))
	}

	return resourceSSOMemberRead(d, metaRaw)
}
