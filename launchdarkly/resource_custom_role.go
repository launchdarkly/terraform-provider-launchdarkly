package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
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
			key: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			name: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			description: &schema.Schema{
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

	_, _, err := client.LaunchDarkly.CustomRolesApi.PostCustomRole(client.Ctx, customRoleBody)
	if err != nil {
		return fmt.Errorf("failed to create custom role with name %q: %s", customRoleName, handleLdapiErr(err))
	}

	d.SetId(customRoleKey)
	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleId := d.Id()

	customRole, _, err := client.LaunchDarkly.CustomRolesApi.GetCustomRole(client.Ctx, customRoleId)
	if err != nil {
		return fmt.Errorf("failed to get custom role with id %q: %s", customRoleId, handleLdapiErr(err))
	}

	d.Set(key, customRole.Key)
	d.Set(name, customRole.Name)
	d.Set(description, customRole.Description)
	err = d.Set(policy, policiesToResourceData(customRole.Policy))
	if err != nil {
		return fmt.Errorf("could not set policy on custom role with id %q: %v", customRoleId, err)
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

	_, _, err := client.LaunchDarkly.CustomRolesApi.PatchCustomRole(client.Ctx, customRoleKey, patch)
	if err != nil {
		return fmt.Errorf("failed to update custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	_, err := client.LaunchDarkly.CustomRolesApi.DeleteCustomRole(client.Ctx, customRoleKey)
	if err != nil {
		return fmt.Errorf("failed to delete custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return nil
}

func resourceCustomRoleExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return customRoleExists(d.Id(), metaRaw.(*Client))
}

func customRoleExists(customRoleKey string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.CustomRolesApi.GetCustomRole(meta.Ctx, customRoleKey)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
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
