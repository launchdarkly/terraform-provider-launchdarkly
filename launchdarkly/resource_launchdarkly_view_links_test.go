package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testAccViewLinksCreate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test2.key
	]
	
	comment = "Test linking flags to view"
}
`

	testAccViewLinksUpdate = `
resource "launchdarkly_project" "test" {
	name = "%s"
	key  = "%s"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "test-view"
	name        = "Test View"
	description = "Test view for link testing"
	maintainer_id = "%s"
}

resource "launchdarkly_feature_flag" "test1" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-1"
	name        = "Test Flag 1"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test2" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-2"
	name        = "Test Flag 2"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "test3" {
	project_key = launchdarkly_project.test.key
	key         = "test-flag-3"
	name        = "Test Flag 3"
	variation_type = "boolean"
}

resource "launchdarkly_view_links" "test" {
	project_key = launchdarkly_project.test.key
	view_key    = launchdarkly_view.test.key
	
	flags = [
		launchdarkly_feature_flag.test1.key,
		launchdarkly_feature_flag.test3.key
	]
	
	comment = "Updated test linking flags to view"
}
`
)

func TestAccViewLinks_Create(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerId := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksCreate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "project_key", projectKey),
					resource.TestCheckResourceAttr(resourceName, "view_key", "test-view"),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-2"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Test linking flags to view"),
				),
			},
		},
	})
}

func TestAccViewLinks_Update(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerId := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksCreate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-2"),
				),
			},
			{
				Config: fmt.Sprintf(testAccViewLinksUpdate, projectName, projectKey, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewLinksExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "flags.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "flags.*", "test-flag-3"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Updated test linking flags to view"),
				),
			},
		},
	})
}

func TestAccViewLinks_Import(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	projectName := "view-links-test-" + projectKey
	resourceName := "launchdarkly_view_links.test"

	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
	require.NoError(t, err)
	require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")

	maintainerId := members.Items[0].Id

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewLinksCreate, projectName, projectKey, maintainerId),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"comment"}, // Comment is not returned from API
			},
		},
	})
}

func testAccCheckViewLinksExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("view links ID is not set")
		}

		client := testAccProvider.Meta().(*Client)
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
		if err != nil {
			return err
		}

		projectKey := rs.Primary.Attributes["project_key"]
		viewKey := rs.Primary.Attributes["view_key"]

		exists, err := viewExists(projectKey, viewKey, betaClient)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("view %s/%s does not exist", projectKey, viewKey)
		}

		return nil
	}
}
