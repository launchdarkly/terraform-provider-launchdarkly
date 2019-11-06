package launchdarkly

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceCustomRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceCustomRoleCreate,
		Read:   resourceCustomRoleRead,
		Update: resourceCustomRoleUpdate,
		Delete: resourceCustomRoleDelete,
		Exists: resourceCustomRoleExists,

		Importer: &schema.ResourceImporter{
			State: resourceCustomRoleImport,
		},

		Schema: map[string]*schema.Schema{
			key: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateKey(),
			},
			name: {
				Type:     schema.TypeString,
				Required: true,
			},
			description: {
				Type:     schema.TypeString,
				Optional: true,
			},
			policy: policyArraySchema(),
		},
	}
}

func resourceCustomRoleCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(key).(string)
	customRoleName := d.Get(name).(string)
	customRoleDescription := d.Get(description).(string)
	customRolePolicies := policiesFromResourceData(d)

	customRoleBody := ldapi.CustomRoleBody{
		Key:         customRoleKey,
		Name:        customRoleName,
		Description: customRoleDescription,
		Policy:      customRolePolicies,
	}

	_, _, err := client.ld.CustomRolesApi.PostCustomRole(client.ctx, customRoleBody)
	if err != nil {
		return fmt.Errorf("failed to create custom role with name %q: %s", customRoleName, handleLdapiErr(err))
	}

	d.SetId(customRoleKey)
	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleID := d.Id()

	customRole, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID)
	if isStatusNotFound(res) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get custom role with id %q: %s", customRoleID, handleLdapiErr(err))
	}

	_ = d.Set(key, customRole.Key)
	_ = d.Set(name, customRole.Name)
	_ = d.Set(description, customRole.Description)
	err = d.Set(policy, policiesToResourceData(customRole.Policy))
	if err != nil {
		return fmt.Errorf("could not set policy on custom role with id %q: %v", customRoleID, err)
	}
	return nil
}

func resourceCustomRoleUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(key).(string)
	customRoleName := d.Get(name).(string)
	customRoleDescription := d.Get(description).(string)
	customRolePolicies := policiesFromResourceData(d)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &customRoleName),
		patchReplace("/description", &customRoleDescription),
		patchReplace("/policy", &customRolePolicies),
	}

	_, _, err := repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.CustomRolesApi.PatchCustomRole(client.ctx, customRoleKey, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	_, err := client.ld.CustomRolesApi.DeleteCustomRole(client.ctx, customRoleKey)
	if err != nil {
		return fmt.Errorf("failed to delete custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return nil
}

func resourceCustomRoleExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return customRoleExists(d.Id(), metaRaw.(*Client))
}

func customRoleExists(customRoleKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.CustomRolesApi.GetCustomRole(meta.ctx, customRoleKey)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceCustomRoleImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())

	if err := resourceCustomRoleRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
