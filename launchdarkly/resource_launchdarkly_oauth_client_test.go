package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccOAuthClientCreate = `
resource "launchdarkly_oauth_client" "test" {
	name         = "Terraform OAuth client - %s"
	redirect_uri = "https://app.example.com/oauth/callback"
	description  = "Created by the acceptance test suite."
}
`
	testAccOAuthClientUpdate = `
resource "launchdarkly_oauth_client" "test" {
	name         = "Updated Terraform OAuth client - %s"
	redirect_uri = "https://app.example.com/oauth/callback/v2"
	description  = "Updated by the acceptance test suite."
}
`
)

func TestAccOAuthClient_CreateUpdate(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_oauth_client.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckOAuthClientDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccOAuthClientCreate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOAuthClientExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Terraform OAuth client - "+name),
					resource.TestCheckResourceAttr(resourceName, REDIRECT_URI, "https://app.example.com/oauth/callback"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Created by the acceptance test suite."),
					resource.TestCheckResourceAttrSet(resourceName, CLIENT_ID),
					resource.TestCheckResourceAttrSet(resourceName, CLIENT_SECRET),
					resource.TestCheckResourceAttrSet(resourceName, ACCOUNT_ID),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The client secret is only returned on create, so it can never
				// be reconciled on import.
				ImportStateVerifyIgnore: []string{CLIENT_SECRET},
			},
			{
				Config: fmt.Sprintf(testAccOAuthClientUpdate, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOAuthClientExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Updated Terraform OAuth client - "+name),
					resource.TestCheckResourceAttr(resourceName, REDIRECT_URI, "https://app.example.com/oauth/callback/v2"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "Updated by the acceptance test suite."),
				),
			},
		},
	})
}

func testAccCheckOAuthClientExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("OAuth client ID is not set")
		}
		client := mustTestAccClient()
		_, _, err := client.ld.OAuth2ClientsApi.GetOAuthClientById(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting OAuth client. %s", err)
		}
		return nil
	}
}

// testAccCheckOAuthClientDestroy verifies the OAuth client has been destroyed.
func testAccCheckOAuthClientDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_oauth_client" {
			continue
		}

		_, res, err := client.ld.OAuth2ClientsApi.GetOAuthClientById(client.ctx, rs.Primary.ID).Execute()

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("OAuth client %s still exists", rs.Primary.ID)
	}
	return nil
}
