package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
	"github.com/pkg/errors"
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
		return errors.Wrapf(err, "failed to create custom role with name %q", customRoleName)
	}

	d.SetId(customRoleKey)

	// LaunchDarkly's api does not allow tags to be passed in during webhook creation so we do an update
	//err = resourceWebhookUpdate(d, metaRaw)
	//if err != nil {
	//	return errors.Wrapf(err, "During webhook creation. Webhook name: %q", webhookName)
	//}

	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleId := d.Id()

	customRole, _, err := client.LaunchDarkly.CustomRolesApi.GetCustomRole(client.Ctx, customRoleId)
	if err != nil {
		return errors.Wrapf(err, "failed to get custom role with id %q", customRoleId)
	}

	d.Set(key, customRole.Key)
	d.Set(name, customRole.Name)
	d.Set(description, customRole.Description)
	d.Set(policy, policiesToResourceData(customRole.Policy))
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
		return errors.Wrapf(err, "failed to update custom role with key %q", customRoleKey)
	}

	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	_, err := client.LaunchDarkly.CustomRolesApi.DeleteCustomRole(client.Ctx, customRoleKey)
	if err != nil {
		return errors.Wrapf(err, "failed to delete custom role with key %q", customRoleKey)
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
		return false, errors.Wrapf(err, "failed to get custom role with key %q", customRoleKey)
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
