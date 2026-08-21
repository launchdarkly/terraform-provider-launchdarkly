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

// customRoleDeleteConflictBackoff is the wait schedule between DELETE retries
// when the API returns a 409 because the role is still assigned to teams or
// members. The total window (~60s) is sized so that unassignment operations
// scheduled in the same apply (which Terraform does not order relative to
// this destroy) have time to land. Package-level so tests can shorten it.
var customRoleDeleteConflictBackoff = []time.Duration{
	500 * time.Millisecond,
	time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
	15 * time.Second,
	15 * time.Second,
}

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

-> **Note:** A custom role cannot be deleted while it is still assigned to any team, member, or access token. Terraform only orders operations by references in the *current* configuration, so an apply that both deletes this role and removes its assignments (for example, from a ` + "`launchdarkly_team`'s `custom_role_keys` or a `launchdarkly_team_member`'s `custom_roles`" + `) does not guarantee the assignments are removed first. To handle this, the provider retries the deletion for up to a minute while conflicting assignments are removed by the same apply. If the role is still assigned after that, the deletion fails with a conflict error — remove the remaining assignments and re-apply.`,

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
	for _, wait := range customRoleDeleteConflictBackoff {
		if !isStatusConflict(deleteResp) {
			break
		}
		select {
		case <-ctx.Done():
			return diag.Errorf("failed to delete custom role with key %q: %s", customRoleKey, ctx.Err())
		case <-time.After(wait):
		}
		deleteResp, err = deleteCustomRole(client, customRoleKey)
		if err == nil {
			return diags
		}
	}
	if isStatusConflict(deleteResp) {
		return diag.Errorf("failed to delete custom role with key %q: still assigned to teams or members: %s\n\nLaunchDarkly rejects deleting a custom role while it is assigned. Remove the role from every launchdarkly_team (custom_role_keys), launchdarkly_team_member (custom_roles), launchdarkly_team_role_mapping, and access token that references it, then re-apply. If this apply was already removing those assignments, they did not complete within the retry window — re-running the apply should succeed.", customRoleKey, handleLdapiErr(err))
	}
	return diag.Errorf("failed to delete custom role with key %q: %s", customRoleKey, handleLdapiErr(err))
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
