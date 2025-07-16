package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceCustomRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCustomRoleCreate,
		ReadContext:   resourceCustomRoleRead,
		UpdateContext: resourceCustomRoleUpdate,
		DeleteContext: resourceCustomRoleDelete,
		Exists:        resourceCustomRoleExists,

		Description: `Provides a LaunchDarkly custom role resource.

-> **Note:** Custom roles are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage custom roles within your LaunchDarkly organization.`,

		Importer: &schema.ResourceImporter{
			State: resourceCustomRoleImport,
		},

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("A unique key that will be used to reference the custom role in your code.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKey(),
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A name for the custom role. This must be unique within your organization.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the custom role.",
			},
			BASE_PERMISSIONS: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The base permission level - either `reader` or `no_access`. While newer API versions default to `no_access`, this field defaults to `reader` in keeping with previous API versions.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"reader", "no_access"}, false)),
				Default:          "reader",
			},
			POLICY: policyArraySchema(),
			POLICY_STATEMENTS: policyStatementsSchema(
				policyStatementSchemaOptions{
					optional:      true,
					conflictsWith: []string{POLICY},
					description:   "An array of the policy statements that define the permissions for the custom role. This field accepts [role attributes](https://docs.launchdarkly.com/home/getting-started/vocabulary#role-attribute). To use role attributes, use the syntax `$${roleAttribute/<YOUR_ROLE_ATTRIBUTE>}` in lieu of your usual resource keys.",
				}),
		},
	}
}

func resourceCustomRoleCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRoleBasePermissions := d.Get(BASE_PERMISSIONS).(string)
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
	if customRoleBasePermissions != "" {
		customRoleBody.BasePermissions = ldapi.PtrString(customRoleBasePermissions)
	}

	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.CustomRolesApi.PostCustomRole(client.ctx).CustomRolePost(customRoleBody).Execute()
		return err
	})
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

	var customRole *ldapi.CustomRole
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		customRole, res, err = client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleID).Execute()
		return err
	})

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
	_ = d.Set(BASE_PERMISSIONS, customRole.BasePermissions)

	// Because "policy" is now deprecated in favor of "policy_statements", only set "policy" if it has
	// already been set by the user.
	// TODO: Somehow this seems to also add an empty policystatement of
	// 	policy {
	// 		+ actions   = []
	// 		+ resources = []
	// 	  }
	if _, ok := d.GetOk(POLICY); ok {
		policies := policiesToResourceData(customRole.Policy)
		err = d.Set(POLICY, policies)
	} else {
		err = d.Set(POLICY_STATEMENTS, policyStatementsToResourceData(statementsToStatementReps(customRole.Policy)))
	}

	if err != nil {
		return diag.Errorf("could not set policy on custom role with id %q: %v", customRoleID, err)
	}
	return diags
}

func resourceCustomRoleUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	customRoleKey := d.Get(KEY).(string)
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := d.Get(DESCRIPTION).(string)
	customRoleBasePermissions := d.Get(BASE_PERMISSIONS).(string)
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
	if customRoleBasePermissions != "" {
		patch.Patch = append(patch.Patch, patchReplace("/basePermissions", &customRoleBasePermissions))
	}

	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.CustomRolesApi.PatchCustomRole(client.ctx, customRoleKey).PatchWithComment(patch).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return resourceCustomRoleRead(ctx, d, metaRaw)
}

func resourceCustomRoleDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	customRoleKey := d.Id()

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.CustomRolesApi.DeleteCustomRole(client.ctx, customRoleKey).Execute()
		return err
	})

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
