package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v7"
)

func resourceCustomRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCustomRoleCreate,
		ReadContext:   resourceCustomRoleRead,
		UpdateContext: resourceCustomRoleUpdate,
		DeleteContext: resourceCustomRoleDelete,
		Exists:        resourceCustomRoleExists,

		Importer: &schema.ResourceImporter{
			State: resourceCustomRoleImport,
		},

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "A unique key that will be used to reference the custom role in your code",
				ForceNew:         true,
				ValidateDiagFunc: validateKey(),
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

func resourceCustomRoleCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(d.Get(POLICY_STATEMENTS).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	if len(policyStatements) > 0 {
		customRolePolicies = policyStatements
	}

	customRoleBody := ldapi.CustomRolePost{
		Key:         customRoleKey,
		Name:        customRoleName,
		Description: ldapi.PtrString(customRoleDescription),
		Policy:      customRolePolicies,
	}

	_, _, err = client.ld.CustomRolesApi.PostCustomRole(client.ctx).CustomRolePost(customRoleBody).Execute()

	if err != nil {
		return diag.Errorf("failed to create custom role with name %q: %s", customRoleName, handleLdapiErr(err))
	}

	d.SetId(customRoleKey)
	return resourceCustomRoleRead(ctx, d, metaRaw)
}

func resourceCustomRoleRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	customRoleID := d.Id()

	customRole, res, err := client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID).Execute()

	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find custom role with id %q, removing from state", customRoleID)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find custom role with id %q, removing from state", customRoleID),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get custom role with id %q: %s", customRoleID, handleLdapiErr(err))
	}

	_ = d.Set(KEY, customRole.Key)
	_ = d.Set(NAME, customRole.Name)
	_ = d.Set(DESCRIPTION, customRole.Description)

	// Because "policy" is now deprecated in favor of "policy_statements", only set "policy" if it has
	// already been set by the user.
	if _, ok := d.GetOk(POLICY); ok {
		err = d.Set(POLICY, policiesToResourceData(customRole.Policy))
	} else {
		err = d.Set(POLICY_STATEMENTS, policyStatementsToResourceData(statementsToStatementReps(customRole.Policy)))
	}

	if err != nil {
		return diag.Errorf("could not set policy on custom role with id %q: %v", customRoleID, err)
	}
	return nil
}

func resourceCustomRoleUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(d.Get(POLICY_STATEMENTS).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	if len(policyStatements) > 0 {
		customRolePolicies = policyStatements
	}

	patch := ldapi.PatchWithComment{
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", &customRoleName),
			patchReplace("/description", &customRoleDescription),
			patchReplace("/policy", &customRolePolicies),
		}}

	_, _, err = client.ld.CustomRolesApi.PatchCustomRole(client.ctx, customRoleKey).PatchWithComment(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return resourceCustomRoleRead(ctx, d, metaRaw)
}

func resourceCustomRoleDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	_, err := client.ld.CustomRolesApi.DeleteCustomRole(client.ctx, customRoleKey).Execute()

	if err != nil {
		return diag.Errorf("failed to delete custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return diags
}

func resourceCustomRoleExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return customRoleExists(d.Id(), metaRaw.(*Client))
}

func customRoleExists(customRoleKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.CustomRolesApi.GetCustomRole(meta.ctx, customRoleKey).Execute()
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
