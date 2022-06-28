package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	// Team members need to be made sequentially, not in parallel
	testAccTeamCreate = `
resource "launchdarkly_custom_role" "terraform_team_test" {
  key = "terraform_teams_test_role"
  name = "Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s@example.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s@example.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_one]
}

resource "launchdarkly_team" "test" {
  key = "%s"
  name = "waterbear"
  description = "The best integrations squad"
  member_ids = [launchdarkly_team_member.test_member_one.id]
  maintainers = [launchdarkly_team_member.test_member_two.id]
  custom_role_keys = [launchdarkly_custom_role.terraform_team_test.key]
}
`
	testAccTeamUpdate = `
resource "launchdarkly_custom_role" "other_team_test" {
  key = "other_terraform_teams_test_role"
  name = "Other Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s@example.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s@example.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_one]
}

resource "launchdarkly_team_member" "test_member_three" {
  email = "%s@example.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_two]
}

resource "launchdarkly_team" "test" {
  key = "%s"
  name = "Integrations"
  description = "The BEST integrations squad"
  member_ids = [launchdarkly_team_member.test_member_two.id, launchdarkly_team_member.test_member_three.id]
  maintainers = [launchdarkly_team_member.test_member_two.id, launchdarkly_team_member.test_member_one.id]
  custom_role_keys = [launchdarkly_custom_role.other_team_test.key]
}
`
)

func TestAccTeam_CreateUpdate(t *testing.T) {
	var randomName = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomEmailOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailThree := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	var resourceName = fmt.Sprintf("launchdarkly_team.test")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamCreate, randomEmailOne, randomEmailTwo, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "waterbear"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The best integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", "terraform_teams_test_role"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamUpdate, randomEmailOne, randomEmailTwo, randomEmailThree, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", "other_terraform_teams_test_role"),
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

func testAccCheckTeamExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("team ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.TeamsApi.GetTeam(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting team: %s", err)
		}
		return nil
	}
}
