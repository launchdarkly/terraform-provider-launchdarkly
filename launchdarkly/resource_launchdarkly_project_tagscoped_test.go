package launchdarkly

// Regression coverage for project creation under a tag-scoped custom role.
//
// Custom-role policies commonly scope project permissions by tag, e.g.
//
//	{"resources": ["proj/team-*;team-tag"], "actions": ["*"], "effect": "allow"}
//
// LaunchDarkly authorizes POST /api/v2/projects against the key and tags in
// the request body, and PATCH /api/v2/projects/{key} against the project's
// current tags. The provider used to POST the project bare (name, key,
// environments) and PATCH the tags on afterwards, so under such a policy the
// POST was denied as an untagged create and, had it succeeded, the follow-up
// PATCH would have been denied against a still-untagged project. Terraform
// could only create projects with a token holding un-scoped createProject.
//
// The CI token is an admin token, so this file mints a service token whose
// inline role grants access only to `proj/<key>;<tag>` and its environments,
// and runs the provider with that token. The project must create, import,
// update and destroy end to end; before the fix the first apply fails with
// the 403 customers reported.

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v24"
)

// tokenProvider wraps the real provider and overrides Configure so the client
// authenticates with the given token rather than LAUNCHDARKLY_ACCESS_TOKEN.
type tokenProvider struct {
	provider.Provider
	token string
}

func (p *tokenProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	host := os.Getenv(LAUNCHDARKLY_API_HOST)
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	client, err := newClient(p.token, host, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

func tokenProviderFactories(token string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"launchdarkly": providerserver.NewProtocol6WithError(&tokenProvider{
			Provider: NewPluginProvider("test")(),
			token:    token,
		}),
	}
}

// newTagScopedServiceToken creates a service token whose only grants are on
// the project <projectKey> when it carries <tag>, and on that project's
// environments. It returns the token secret and deletes the token when the
// test ends.
func newTagScopedServiceToken(t *testing.T, projectKey, tag string) string {
	t.Helper()
	client := mustTestAccClient()

	name := fmt.Sprintf("tf-acc-tag-scoped-%s", projectKey)
	serviceToken := true
	project := fmt.Sprintf("proj/%s;%s", projectKey, tag)
	post := ldapi.AccessTokenPost{
		Name:         &name,
		ServiceToken: &serviceToken,
		InlineRole: []ldapi.StatementPost{
			{Effect: "allow", Actions: []string{"*"}, Resources: []string{project}},
			{Effect: "allow", Actions: []string{"*"}, Resources: []string{project + ":env/*"}},
		},
	}

	var token *ldapi.Token
	err := client.withConcurrency(client.ctx, func() error {
		var e error
		token, _, e = client.ld.AccessTokensApi.PostToken(client.ctx).AccessTokenPost(post).Execute()
		return e
	})
	if err != nil {
		t.Fatalf("creating tag-scoped service token: %s", handleLdapiErr(err))
	}
	if token.Token == nil || *token.Token == "" {
		t.Fatal("tag-scoped service token was created but its secret is missing from the response")
	}
	t.Cleanup(func() {
		if _, err := client.ld.AccessTokensApi.DeleteToken(client.ctx, token.Id).Execute(); err != nil {
			t.Logf("cleanup of tag-scoped service token %q failed: %v", token.Id, err)
		}
	})
	return *token.Token
}

func testAccProjectTagScopedConfig(projectKey, tag, projectName, envName string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "tag_scoped" {
	key  = "%s"
	name = "%s"
	tags = ["%s"]
	default_client_side_availability = {
	  using_environment_id = true
	  using_mobile_key     = false
	}
	environments = {
	  "production" = {
	    name                 = "%s"
	    color                = "ff0000"
	    tags                 = ["prod"]
	    default_ttl          = 15
	    secure_mode          = true
	    default_track_events = true
	    require_comments     = true
	    confirm_changes      = true
	    critical             = true
	  }
	}
}
`, projectKey, projectName, tag, envName)
}

// TestAccProject_TagScopedRole_Lifecycle creates, imports, updates and
// destroys a project with a token that is only allowed to touch
// `proj/<key>;<tag>`. Every create-time attribute is set so the test also
// proves the POST body carries them (the create path no longer PATCHes the
// project or its environments afterwards, except for approval_settings).
func TestAccProject_TagScopedRole_Lifecycle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	tag := "tf-acc-scope-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.tag_scoped"

	// The scoped token is minted before the test case runs, so mirror the
	// TF_ACC gate resource.ParallelTest applies to its own steps.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("acceptance test skipped unless env %q is set", resource.EnvTfAcc)
	}
	testAccPreCheck(t)
	scopedToken := newTagScopedServiceToken(t, projectKey, tag)
	t.Cleanup(func() { failOnOrphanedProject(t, projectKey) })

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: tokenProviderFactories(scopedToken),
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectTagScopedConfig(projectKey, tag, "Tag Scoped Project", "Production"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "Tag Scoped Project"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", tag),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "default_client_side_availability.using_mobile_key", "false"),
					resource.TestCheckResourceAttr(resourceName, "environments.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.name", "Production"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.color", "ff0000"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.tags.0", "prod"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.default_ttl", "15"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.secure_mode", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.default_track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.require_comments", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.confirm_changes", "true"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.critical", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// default_client_side_availability is Optional-only: Read keeps
				// it null when there is no prior state to anchor it (import),
				// see projectCSAValueFromAPI.
				ImportStateVerifyIgnore: []string{DEFAULT_CLIENT_SIDE_AVAILABILITY},
			},
			{
				// Update path under the same scoped token: the project now
				// carries the tag, so PATCHing it is within policy.
				Config: testAccProjectTagScopedConfig(projectKey, tag, "Tag Scoped Project Renamed", "Production Renamed"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Tag Scoped Project Renamed"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", tag),
					resource.TestCheckResourceAttr(resourceName, "environments.production.name", "Production Renamed"),
					resource.TestCheckResourceAttr(resourceName, "environments.production.critical", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// default_client_side_availability is Optional-only: Read keeps
				// it null when there is no prior state to anchor it (import),
				// see projectCSAValueFromAPI.
				ImportStateVerifyIgnore: []string{DEFAULT_CLIENT_SIDE_AVAILABILITY},
			},
		},
	})
}
