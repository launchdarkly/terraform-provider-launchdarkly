package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testAccViewCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key = launchdarkly_project.test.key
	key         = "%s"
	name        = "%s"
	description = "%s"
	tags        = ["test"]
	maintainer_id = "%s"
}
`

	testAccViewUpdate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_view" "test" {
	project_key       = launchdarkly_project.test.key
	key               = "%s"
	name              = "%s"
	description       = "%s"
	generate_sdk_keys = true
	tags              = ["test", "updated"]
	maintainer_id     = "%s"
}
`

	testAccViewWithMaintainer = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_team" "test_team" {
	key         = "%s"
	name        = "Test Team"
	description = "Team to maintain views"
	custom_role_keys = []
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "%s"
	name          = "%s"
	description   = "%s"
	maintainer_id = "%s"
	tags          = ["test"]
}
`

	testAccViewWithTeamMaintainer = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

resource "launchdarkly_team" "test_team" {
	key         = "%s"
	name        = "Test Team"
	description = "Team to maintain views"
	custom_role_keys = []
}

resource "launchdarkly_view" "test" {
	project_key         = launchdarkly_project.test.key
	key                 = "%s"
	name                = "%s"
	description         = "%s"
	maintainer_team_key = launchdarkly_team.test_team.key
	tags                = ["test"]
}
`
)

func TestAccView_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewName := "Test View"
	viewDescription := "Test view description"
	updatedViewName := "Updated Test View"
	updatedViewDescription := "Updated test view description"
	resourceName := "launchdarkly_view.test"

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
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckViewDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewCreate, projectKey, viewKey, viewName, viewDescription, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, viewKey),
					resource.TestCheckResourceAttr(resourceName, NAME, viewName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, viewDescription),
					resource.TestCheckResourceAttr(resourceName, GENERATE_SDK_KEYS, "false"),
					resource.TestCheckResourceAttr(resourceName, ARCHIVED, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewUpdate, projectKey, viewKey, updatedViewName, updatedViewDescription, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, updatedViewName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, updatedViewDescription),
					resource.TestCheckResourceAttr(resourceName, GENERATE_SDK_KEYS, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccViewCreate, projectKey, viewKey, viewName, viewDescription, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, viewKey),
					resource.TestCheckResourceAttr(resourceName, NAME, viewName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, viewDescription),
					resource.TestCheckResourceAttr(resourceName, GENERATE_SDK_KEYS, "false"),
					resource.TestCheckResourceAttr(resourceName, ARCHIVED, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccView_WithMaintainer(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	viewName := "Test View"
	viewDescription := "Test view description"
	resourceName := "launchdarkly_view.test"

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
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckViewDestroy,
		Steps: []resource.TestStep{
			// Set the view to be maintained by a team
			{
				Config: fmt.Sprintf(testAccViewWithTeamMaintainer, projectKey, teamKey, viewKey, viewName, viewDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, teamKey),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set the view to be maintained by an individual - but keep the team in our TF config because it would otherwise be deleted before being removed as a maintainer
			{
				Config: fmt.Sprintf(testAccViewWithMaintainer, projectKey, teamKey, viewKey, viewName, viewDescription, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, maintainerId),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set the view to be maintained by a team
			{
				Config: fmt.Sprintf(testAccViewWithTeamMaintainer, projectKey, teamKey, viewKey, viewName, viewDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, teamKey),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_ID, ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccView_InvalidKey(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	invalidViewKey := "invalid key with spaces"
	viewName := "Test View"
	viewDescription := "Test view description"

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
				Config:      fmt.Sprintf(testAccViewCreate, projectKey, invalidViewKey, viewName, viewDescription, maintainerId),
				ExpectError: regexp.MustCompile("invalid value for key"),
			},
		},
	})
}

func testAccCheckViewExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("view ID is not set")
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		viewKey := rs.Primary.Attributes[KEY]

		client := testAccProvider.Meta().(*Client)
		_, _, err := getView(client, projectKey, viewKey)
		if err != nil {
			return fmt.Errorf("received an error getting view. %s", err)
		}
		return nil
	}
}

func testAccCheckViewDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_view" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		viewKey := rs.Primary.Attributes[KEY]

		_, res, err := getView(client, projectKey, viewKey)
		if isStatusNotFound(res) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("view still exists")
	}
	return nil
}
