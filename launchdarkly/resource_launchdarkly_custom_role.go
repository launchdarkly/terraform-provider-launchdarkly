package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// Retry window for role DELETEs rejected with a 409 because the role is
// still assigned to teams or members. The default is sized so that
// unassignment operations scheduled in the same apply (which Terraform does
// not order relative to this destroy) have time to land; users with larger
// applies can raise it via a `timeouts { delete = ... }` block. Individual
// waits grow exponentially from the initial value up to the cap.
const (
	customRoleDeleteTimeoutDefault      = time.Minute
	customRoleDeleteConflictInitialWait = 500 * time.Millisecond
	customRoleDeleteConflictMaxWait     = 15 * time.Second
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

This resource allows you to create and manage custom roles within your LaunchDarkly organization.

-> **Note:** You cannot delete a custom role while it is still assigned to any team, member, or access token. By default, Terraform destroys a removed resource before it updates the resources that referenced it, so a single apply that deletes this role and removes its assignments, for example from a ` + "`launchdarkly_team`'s `custom_role_keys` or a `launchdarkly_team_member`'s `custom_roles`" + `, attempts the deletion while the assignments still exist and fails with a conflict. To delete a role and remove its assignments in one apply, set ` + "`lifecycle { create_before_destroy = true }`" + ` on this resource so that Terraform updates the referencing resources first. Alternatively, manage the assignment with a [` + "`launchdarkly_team_role_mapping`" + `](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/resources/team_role_mapping) resource, which Terraform destroys before the role and which does not require the lifecycle setting. With ` + "`create_before_destroy`" + `, a change that forces replacement, such as a new ` + "`key`" + `, creates the replacement role before it destroys the original, and role names must be unique: change ` + "`name`" + ` in the same apply to avoid a conflict. The provider retries a conflicting deletion for one minute by default, configurable with a ` + "`timeouts { delete = ... }`" + ` block, to absorb propagation delays. If the role is still assigned when the retry window closes, the deletion fails with a conflict error.`,

		Importer: &schema.ResourceImporter{
			State: resourceCustomRoleImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(customRoleDeleteTimeoutDefault),
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
	customRoleKey := effectiveCustomRoleKey(d)
	if customRoleKey == "" {
		return diag.Errorf(
			"%s is required for custom role creation. If the embedded schema omits it, set the Terraform resource id (Crossplane external-name) to the LaunchDarkly role key before create.", KEY)
	}
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := optionalStringAttr(d, DESCRIPTION)
	customRoleBasePermissions := optionalStringAttr(d, BASE_PERMISSIONS)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(getOptionalInterfaceSlice(d, POLICY_STATEMENTS))
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

	var created *ldapi.CustomRole
	err = client.withConcurrency(client.ctx, func() error {
		created, _, err = client.ld.CustomRolesApi.PostCustomRole(client.ctx).CustomRolePost(customRoleBody).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create custom role with name %q: %s", customRoleName, handleLdapiErr(err))
	}

	id := customRoleKey
	if created != nil && created.Key != "" {
		id = created.Key
	}
	d.SetId(id)
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

	if customRole.Key != "" {
		d.SetId(customRole.Key)
	}

	_ = resourceDataSetSkipMissingKey(d, KEY, customRole.Key)
	_ = resourceDataSetSkipMissingKey(d, NAME, customRole.Name)
	desc := ""
	if customRole.Description != nil {
		desc = *customRole.Description
	}
	_ = resourceDataSetSkipMissingKey(d, DESCRIPTION, desc)
	basePerms := ""
	if customRole.BasePermissions != nil {
		basePerms = *customRole.BasePermissions
	}
	_ = resourceDataSetSkipMissingKey(d, BASE_PERMISSIONS, basePerms)

	// Because "policy" is now deprecated in favor of "policy_statements", only set "policy" if it has
	// already been set by the user.
	// TODO: Somehow this seems to also add an empty policystatement of
	// 	policy {
	// 		+ actions   = []
	// 		+ resources = []
	// 	  }
	if _, ok := d.GetOk(POLICY); ok {
		policies := policiesToResourceData(customRole.Policy)
		err = resourceDataSetSkipMissingKey(d, POLICY, policies)
	} else {
		err = resourceDataSetSkipMissingKey(d, POLICY_STATEMENTS, policyStatementsToResourceData(statementsToStatementReps(customRole.Policy)))
	}

	if err != nil {
		return diag.Errorf("could not set policy on custom role with id %q: %v", customRoleID, err)
	}
	return diags
}

func resourceCustomRoleUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	customRoleKey := effectiveCustomRoleKey(d)
	if customRoleKey == "" {
		return diag.Errorf("cannot update custom role: %s is empty and resource id is empty", KEY)
	}
	customRoleName := d.Get(NAME).(string)
	customRoleDescription := optionalStringAttr(d, DESCRIPTION)
	customRoleBasePermissions := optionalStringAttr(d, BASE_PERMISSIONS)
	customRolePolicies := policiesFromResourceData(d)
	policyStatements, err := policyStatementsFromResourceData(getOptionalInterfaceSlice(d, POLICY_STATEMENTS))
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

	deleteResp, err := deleteCustomRole(client, customRoleKey)
	if err == nil {
		return diags
	}
	// 409: the role is still assigned to teams and/or members. Terraform
	// only builds dependency edges from the new configuration, so an apply
	// that deletes this role and also unassigns it (a team's
	// custom_role_keys update, a member's custom_roles update, a
	// team_role_mapping destroy) runs both operations unordered and the
	// DELETE can fire before the unassignment lands (REL-12313). Retry
	// with backoff, releasing the concurrency slot between attempts so
	// those other operations can proceed in the meantime.
	deadline := time.Now().Add(d.Timeout(schema.TimeoutDelete))
	// The SDK enforces the delete timeout as a context deadline on ctx.
	// Stop retrying just inside it so the detailed still-assigned
	// diagnostic below is returned instead of a bare "context deadline
	// exceeded" from the ctx.Done branch.
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline.Add(-2 * time.Second)
	}
	wait := customRoleDeleteConflictInitialWait
	for isStatusConflict(deleteResp) && time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			if isStatusConflict(deleteResp) {
				return customRoleStillAssignedDiag(customRoleKey, err)
			}
			return diag.Errorf("failed to delete custom role with key %q: %s", customRoleKey, ctx.Err())
		case <-time.After(wait):
		}
		if wait *= 2; wait > customRoleDeleteConflictMaxWait {
			wait = customRoleDeleteConflictMaxWait
		}
		deleteResp, err = deleteCustomRole(client, customRoleKey)
		if err == nil {
			return diags
		}
	}
	if isStatusConflict(deleteResp) {
		return customRoleStillAssignedDiag(customRoleKey, err)
	}
	return diag.Errorf("failed to delete custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
}

func customRoleStillAssignedDiag(customRoleKey string, err error) diag.Diagnostics {
	return diag.Errorf("failed to delete custom role with key %q: still assigned to teams or members: %s\n\nLaunchDarkly rejects deleting a custom role while it is assigned. Remove the role from every launchdarkly_team (custom_role_keys), launchdarkly_team_member (custom_roles), launchdarkly_team_role_mapping, and access token that references it, then re-apply. To remove the assignments and delete the role in the same apply: Terraform updates referencing resources only after this destroy, so set `lifecycle { create_before_destroy = true }` on this launchdarkly_custom_role to order the unassignments first, or manage the assignment with launchdarkly_team_role_mapping, then re-apply.", customRoleKey, handleLdapiErr(err))
}

func deleteCustomRole(client *Client, customRoleKey string) (*http.Response, error) {
	var deleteResp *http.Response
	err := client.withConcurrency(client.ctx, func() error {
		var e error
		deleteResp, e = client.ld.CustomRolesApi.DeleteCustomRole(client.ctx, customRoleKey).Execute()
		return e
	})
	return deleteResp, err
}

func resourceCustomRoleExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return customRoleExists(d.Id(), metaRaw.(*Client))
}

func customRoleExists(customRoleKey string, client *Client) (bool, error) {
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.CustomRolesApi.GetCustomRole(client.ctx, customRoleKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceCustomRoleImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	key := strings.TrimSpace(d.Id())
	_ = d.Set(KEY, key)

	return []*schema.ResourceData{d}, nil
}
