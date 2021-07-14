package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceAccessToken() *schema.Resource {
	tokenPolicySchema := policyStatementsSchema()
	tokenPolicySchema.Description = "A list of policy statements defining the permissions for the token. May be used in place of a built-in or custom role."
	return &schema.Resource{
		Create: resourceAccessTokenCreate,
		Read:   resourceAccessTokenRead,
		Update: resourceAccessTokenUpdate,
		Delete: resourceAccessTokenDelete,
		Exists: resourceAccessTokenExists,

		Schema: map[string]*schema.Schema{
			NAME: {
				Type:        schema.TypeString,
				Description: "The human-readable name of the access token",
				Required:    true,
			},
			ROLE: {
				Type:         schema.TypeString,
				Description:  "The name of a built-in role for the token",
				Optional:     true,
				ValidateFunc: validateTeamMemberRole,
			},
			CUSTOM_ROLES: {
				Type:        schema.TypeSet,
				Description: "A set of custom role keys to use as access limits for the access token",
				Set:         schema.HashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			POLICY_STATEMENTS: tokenPolicySchema,
			SERVICE_TOKEN: {
				Type:        schema.TypeBool,
				Description: "Whether the token will be a service token https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens",
				Optional:    true,
				ForceNew:    true,
				Default:     false,
			},
			DEFAULT_API_VERSION: {
				Type:        schema.TypeInt,
				Description: "The default API version for this token. Defaults to the latest API version.",
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			TOKEN: {
				Type:        schema.TypeString,
				Description: "The secret key used to authorize usage of the LaunchDarkly API",
				Computed:    true,
				Sensitive:   true,
			},
			EXPIRE: {
				Type:        schema.TypeInt,
				Description: "Replace the computed token secret with a new value. The expired secret will no longer be able to authorize usage of the LaunchDarkly API. Should be an expiration time for the current token secret, expressed as a Unix epoch time in milliseconds. Setting this to a negative value will expire the existing token immediately. To reset the token value again, change 'expire' to a new value. Setting this field at resource creation time WILL NOT set an expiration time for the token.",
				Optional:    true,
			},
		},
	}
}

func resourceAccessTokenCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	accessTokenName := d.Get(NAME).(string)
	accessTokenRole := d.Get(ROLE).(string)
	serviceToken := d.Get(SERVICE_TOKEN).(bool)
	defaultApiVersion := d.Get(DEFAULT_API_VERSION).(int)
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoles := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoles[i] = cr.(string)
	}
	policyStatements, err := policyStatementsFromResourceData(d)
	if err != nil {
		return err
	}

	accessTokenBody := ldapi.TokenBody{
		Name:              accessTokenName,
		Role:              accessTokenRole,
		CustomRoleIds:     customRoles,
		InlineRole:        policyStatements,
		ServiceToken:      serviceToken,
		DefaultApiVersion: int32(defaultApiVersion),
	}

	tokenRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.AccessTokensApi.PostToken(client.ctx, accessTokenBody)
	})
	token := tokenRaw.(ldapi.Token)
	if err != nil {
		return fmt.Errorf("failed to create access token with name %q: %s", accessTokenName, handleLdapiErr(err))
	}

	_ = d.Set(TOKEN, token.Token)
	d.SetId(token.Id)
	return resourceAccessTokenRead(d, metaRaw)
}

func resourceAccessTokenRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	accessTokenID := d.Id()

	accessTokenRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.AccessTokensApi.GetToken(client.ctx, accessTokenID)
	})
	accessToken := accessTokenRaw.(ldapi.Token)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find access token with id %q, removing from state", accessTokenID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get access token with id %q: %s", accessTokenID, handleLdapiErr(err))
	}

	_ = d.Set(NAME, accessToken.Name)
	if accessToken.Role != "" {
		_ = d.Set(ROLE, accessToken.Role)
	}
	if len(accessToken.CustomRoleIds) > 0 {
		customRoleKeys, err := customRoleIDsToKeys(client, accessToken.CustomRoleIds)
		if err != nil {
			return err
		}
		_ = d.Set(CUSTOM_ROLES, customRoleKeys)
	}
	_ = d.Set(SERVICE_TOKEN, accessToken.ServiceToken)
	_ = d.Set(DEFAULT_API_VERSION, accessToken.DefaultApiVersion)

	policies := accessToken.InlineRole
	if len(policies) > 0 {
		err = d.Set(POLICY_STATEMENTS, policyStatementsToResourceData(policies))
		if err != nil {
			return fmt.Errorf("could not set policy on access token with id %q: %v", accessTokenID, err)
		}
	}

	return nil
}

func resourceAccessTokenUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	accessTokenID := d.Id()
	accessTokenName := d.Get(NAME).(string)
	accessTokenRole := d.Get(ROLE).(string)
	customRolesRaw := d.Get(CUSTOM_ROLES).(*schema.Set).List()

	customRoleKeys := make([]string, len(customRolesRaw))
	for i, cr := range customRolesRaw {
		customRoleKeys[i] = cr.(string)
	}
	customRoleIds, err := customRoleKeysToIDs(client, customRoleKeys)
	if err != nil {
		return err
	}

	policyStatements, err := policyStatementsFromResourceData(d)
	if err != nil {
		return err
	}
	p := statementsToPolicies(policyStatements)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &accessTokenName),
	}

	if d.HasChange(ROLE) {
		var op ldapi.PatchOperation
		if accessTokenRole == "" {
			op = patchRemove("/role")
		} else {
			op = patchReplace("/role", &accessTokenRole)
		}
		patch = append(patch, op)
	}
	if d.HasChange(CUSTOM_ROLES) {
		var op ldapi.PatchOperation
		if len(customRoleIds) == 0 {
			op = patchRemove("/customRoleIds")
		} else {
			op = patchReplace("/customRoleIds", &customRoleIds)
		}
		patch = append(patch, op)
	}
	if d.HasChange(POLICY_STATEMENTS) {
		var op ldapi.PatchOperation
		if len(p) == 0 {
			op = patchRemove("/inlineRole")
		} else {
			op = patchReplace("/inlineRole", &p)
		}
		patch = append(patch, op)
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.AccessTokensApi.PatchToken(client.ctx, accessTokenID, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update access token with id %q: %s", accessTokenID, handleLdapiErr(err))
	}

	// Reset the access token if the expire field has been updated
	if d.HasChange(EXPIRE) {
		oldExpireRaw, newExpireRaw := d.GetChange(EXPIRE)
		oldExpire := oldExpireRaw.(int)
		newExpire := newExpireRaw.(int)
		opts := ldapi.AccessTokensApiResetTokenOpts{}
		if oldExpire != newExpire && newExpire != 0 {
			if newExpire > 0 {
				opts.Expiry = optional.NewInt64(int64(newExpire))
			}
			token, _, err := client.ld.AccessTokensApi.ResetToken(client.ctx, accessTokenID, &opts)
			if err != nil {
				return fmt.Errorf("failed to reset access token with id %q: %s", accessTokenID, handleLdapiErr(err))
			}
			_ = d.Set(EXPIRE, newExpire)
			_ = d.Set(TOKEN, token.Token)
		}
	}

	return resourceAccessTokenRead(d, metaRaw)
}

func resourceAccessTokenDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	accessTokenID := d.Id()

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.AccessTokensApi.DeleteToken(client.ctx, accessTokenID)
		return nil, res, err
	})
	if err != nil {
		return fmt.Errorf("failed to delete access token with id %q: %s", accessTokenID, handleLdapiErr(err))
	}

	return nil
}

func resourceAccessTokenExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return accessTokenExists(d.Id(), metaRaw.(*Client))
}

func accessTokenExists(accessTokenID string, meta *Client) (bool, error) {
	_, res, err := meta.ld.AccessTokensApi.GetToken(meta.ctx, accessTokenID)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get access token with id %q: %s", accessTokenID, handleLdapiErr(err))
	}

	return true, nil
}
