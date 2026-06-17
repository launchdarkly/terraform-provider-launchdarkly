package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceOAuthClient = `
data "launchdarkly_oauth_client" "testing" {
	client_id = "%s"
}
`
)

func TestAccDataSourceOAuthClient_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceOAuthClient, "this-client-does-not-exist"),
				ExpectError: regexp.MustCompile("Error: 404 Not Found"),
			},
		},
	})
}

func TestAccDataSourceOAuthClient_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	post := ldapi.OauthClientPost{
		Name:        ldapi.PtrString("OAuth Client Data Source Test"),
		RedirectUri: ldapi.PtrString("https://app.example.com/oauth/callback"),
		Description: ldapi.PtrString("an OAuth client to test the terraform data source"),
	}
	created, _, err := client.ld.OAuth2ClientsApi.CreateOAuth2Client(client.ctx).OauthClientPost(post).Execute()
	require.NoError(t, err)

	defer func() {
		_, err := client.ld.OAuth2ClientsApi.DeleteOAuthClient(client.ctx, created.ClientId).Execute()
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_oauth_client.testing"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOAuthClient, created.ClientId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, CLIENT_ID, created.ClientId),
					resource.TestCheckResourceAttr(resourceName, ID, created.ClientId),
					resource.TestCheckResourceAttr(resourceName, NAME, created.Name),
					resource.TestCheckResourceAttr(resourceName, REDIRECT_URI, created.RedirectUri),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, *created.Description),
					resource.TestCheckResourceAttrSet(resourceName, ACCOUNT_ID),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
		},
	})
}
