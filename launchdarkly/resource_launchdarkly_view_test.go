package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccViewCreate = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
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
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
	}
}

resource "launchdarkly_view" "test" {
	project_key       = launchdarkly_project.test.key
	key               = "%s"
	name              = "%s"
	description       = "%s"
	tags              = ["test", "updated"]
	maintainer_id     = "%s"
}
`

	testAccViewWithMaintainer = `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Test project"
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
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
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "000000"
		}
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

	maintainerId := firstMemberIDForTest(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckViewDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccViewCreate, projectKey, viewKey, viewName, viewDescription, maintainerId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, KEY, viewKey),
					resource.TestCheckResourceAttr(resourceName, NAME, viewName),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, viewDescription),
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

	maintainerId := firstMemberIDForTest(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckViewDestroy,
		Steps: []resource.TestStep{
			// Set the view to be maintained by a team
			{
				Config: fmt.Sprintf(testAccViewWithTeamMaintainer, projectKey, teamKey, viewKey, viewName, viewDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckViewExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, MAINTAINER_TEAM_KEY, teamKey),
					// Framework-served resource: the unused maintainer side is
					// null (absent), not "". The view Read picks one side per
					// API response; the other stays unset. See
					// resource_view_framework.go (default both to null, then
					// switch on Maintainer.Kind).
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
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
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_TEAM_KEY),
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
					resource.TestCheckNoResourceAttr(resourceName, MAINTAINER_ID),
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

	maintainerId := firstMemberIDForTest(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

		client := mustTestAccClient()
		betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			return fmt.Errorf("failed to create beta client: %v", err)
		}

		_, _, err = getView(betaClient, projectKey, viewKey)
		if err != nil {
			return fmt.Errorf("received an error getting view. %s", err)
		}
		return nil
	}
}

func testAccCheckViewDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return fmt.Errorf("failed to create beta client: %v", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_view" {
			continue
		}
		projectKey := rs.Primary.Attributes[PROJECT_KEY]
		viewKey := rs.Primary.Attributes[KEY]

		_, res, err := getView(betaClient, projectKey, viewKey)
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
