package launchdarkly

import (
	"fmt"
	"log"
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
			KEY: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A unique key that will be used to reference the custom role in your code",
				ForceNew:     true,
				ValidateFunc: validateKey(),
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A name for the custom role",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the custom role",
			},
			POLICY:            policyArraySchema(),
			POLICY_STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{}),
		},
	}
}

func resourceCustomRoleCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(d.Get(POLICY_STATEMENTS).([]interface{}))
	if err != nil {
		return err
	}
	if len(policyStatements) > 0 {
		customRolePolicies = statementsToPolicies(policyStatements)
	}

	customRoleBody := ldapi.CustomRoleBody{
		Key:         customRoleKey,
		Name:        customRoleName,
		Description: customRoleDescription,
		Policy:      customRolePolicies,
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.CustomRolesApi.PostCustomRole(client.ctx, customRoleBody)
	})
	if err != nil {
		return fmt.Errorf("failed to create custom role with name %q: %s", customRoleName, handleLdapiErr(err))
	}

	d.SetId(customRoleKey)
	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleID := d.Id()

	customRoleRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID)
	})
	customRole := customRoleRaw.(ldapi.CustomRole)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find custom role with id %q, removing from state", customRoleID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get custom role with id %q: %s", customRoleID, handleLdapiErr(err))
	}

	_ = d.Set(KEY, customRole.Key)
	_ = d.Set(NAME, customRole.Name)
	_ = d.Set(DESCRIPTION, customRole.Description)

	// Because "policy" is now deprecated in favor of "policy_statements", only set "policy" if it has
	// already been set by the user.
	if _, ok := d.GetOk(POLICY); ok {
		err = d.Set(POLICY, policiesToResourceData(customRole.Policy))
	} else {
		err = d.Set(POLICY_STATEMENTS, policyStatementsToResourceData(policiesToStatements(customRole.Policy)))
	}

	if err != nil {
		return fmt.Errorf("could not set policy on custom role with id %q: %v", customRoleID, err)
	}
	return nil
}

func resourceCustomRoleUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(d.Get(POLICY_STATEMENTS).([]interface{}))
	if err != nil {
		return err
	}
	if len(policyStatements) > 0 {
		customRolePolicies = statementsToPolicies(policyStatements)
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &customRoleName),
		patchReplace("/description", &customRoleDescription),
		patchReplace("/policy", &customRolePolicies),
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.CustomRolesApi.PatchCustomRole(client.ctx, customRoleKey, patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to update custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return resourceCustomRoleRead(d, metaRaw)
}

func resourceCustomRoleDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.CustomRolesApi.DeleteCustomRole(client.ctx, customRoleKey)
		return nil, res, err
	})

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
	_ = d.Set(KEY, d.Id())

	return []*schema.ResourceData{d}, nil
}
