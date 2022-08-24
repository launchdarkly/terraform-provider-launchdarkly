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
  key = "%s"
  name = "Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s+wbteste2e@launchdarkly.com"
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
	testAccTeamUpdateNameDescription = `
resource "launchdarkly_custom_role" "terraform_team_test" {
  key = "%s"
  name = "Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_one]
}

resource "launchdarkly_team" "test" {
  key = "%s"
  name = "Integrations"
  description = "The BEST integrations squad"
  member_ids = [launchdarkly_team_member.test_member_one.id]
  maintainers = [launchdarkly_team_member.test_member_two.id]
  custom_role_keys = [launchdarkly_custom_role.terraform_team_test.key]
}
`
	testAccTeamUpdateRoles = `
resource "launchdarkly_custom_role" "terraform_team_test" {
  key = "%s"
  name = "Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_custom_role" "other_team_test" {
  key = "%s"
  name = "Other Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_one]
}

resource "launchdarkly_team" "test" {
  key = "%s"
  name = "Integrations"
  description = "The BEST integrations squad"
  member_ids = [launchdarkly_team_member.test_member_one.id]
  maintainers = [launchdarkly_team_member.test_member_two.id]
  custom_role_keys = [launchdarkly_custom_role.other_team_test.key]
}
`
	testAccTeamUpdateMembersMaintainers = `
resource "launchdarkly_custom_role" "other_team_test" {
  key = "%s"
  name = "Other Terraform Teams test role"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_team_member" "test_member_one" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
}

resource "launchdarkly_team_member" "test_member_two" {
  email = "%s+wbteste2e@launchdarkly.com"
  role = "reader"
  depends_on = [launchdarkly_team_member.test_member_one]
}

resource "launchdarkly_team_member" "test_member_three" {
  email = "%s+wbteste2e@launchdarkly.com"
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

func TestAccTeam_Create(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomRole := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomEmailOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := fmt.Sprintf("launchdarkly_team.test")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamCreate, randomRole, randomEmailOne, randomEmailTwo, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "waterbear"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The best integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRole),
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

func TestAccTeam_Update(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomRoleOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomRoleTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomEmailOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailThree := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := fmt.Sprintf("launchdarkly_team.test")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamCreate, randomRoleOne, randomEmailOne, randomEmailTwo, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
				)},
			{
				Config: fmt.Sprintf(testAccTeamUpdateNameDescription, randomRoleOne, randomEmailOne, randomEmailTwo, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleOne),
				),
			},
			{
				Config: fmt.Sprintf(testAccTeamUpdateRoles, randomRoleOne, randomRoleTwo, randomEmailOne, randomEmailTwo, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleTwo),
				),
			},
			{
				Config: fmt.Sprintf(testAccTeamUpdateMembersMaintainers, randomRoleTwo, randomEmailOne, randomEmailTwo, randomEmailThree, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleTwo),
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
