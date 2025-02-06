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
  name = "%s"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/$${roleAttribute/developer-envs}"]
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
  role_attributes {
	key = "fake-attribute"
	values = ["fake-value"]
  }
  role_attributes {
	key = "developer-envs"
	values = ["development", "production"]
  }
  role_attributes {
	key = "another-fake-attribute"
	values = ["another-fake-value"]
  }
}
`
	testAccTeamUpdateNameDescriptionRoleAttributes = `
resource "launchdarkly_custom_role" "terraform_team_test" {
  key = "%s"
  name = "%s"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/$${roleAttribute/developer-envs}"]
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
  role_attributes {
	key = "developer-envs"
	values = ["development"]
  }
  role_attributes {
	key = "fake-attribute"
	values = ["faker-value", "fake-value"]
  }
}
`
	testAccTeamUpdateRoles = `
resource "launchdarkly_custom_role" "terraform_team_test" {
  key = "%s"
  name = "%s"
  base_permissions = "no_access"
  policy {
    actions = ["*"]
    effect = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_custom_role" "other_team_test" {
  key = "%s"
  name = "Other test role %s"
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
	testAccTeamUpdateMembersMaintainersRoleAttributes = `
resource "launchdarkly_custom_role" "other_team_test" {
  key = "%s"
  name = "Other test role %s"
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
  role_attributes {
	key = "testAttribute"
	values = ["testValue"]
  }
}
`
)

func TestAccTeam_CreateAndUpdate(t *testing.T) {
	randomTeamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomNewTeamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomRoleOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomRoleTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	randomEmailOne := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailTwo := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomEmailThree := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team.test"

	// custom role names must also be unique
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamCreate, randomRoleOne, randomRoleOne, randomEmailOne, randomEmailTwo, randomTeamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "waterbear"),
					resource.TestCheckResourceAttr(resourceName, KEY, randomTeamKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The best integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleOne),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "another-fake-attribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "another-fake-value"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.key", "developer-envs"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.0", "development"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.1", "production"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.2.key", "fake-attribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.2.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.2.values.0", "fake-value"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},

			{
				Config: fmt.Sprintf(testAccTeamUpdateNameDescriptionRoleAttributes, randomRoleOne, randomRoleOne, randomEmailOne, randomEmailTwo, randomTeamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, KEY, randomTeamKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleOne),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "developer-envs"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "development"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.key", "fake-attribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.0", "faker-value"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.1", "fake-value"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccTeamUpdateRoles, randomRoleOne, randomRoleOne, randomRoleTwo, randomRoleTwo, randomEmailOne, randomEmailTwo, randomTeamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, KEY, randomTeamKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleTwo),
					resource.TestCheckResourceAttr(resourceName, "roleAttributes.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccTeamUpdateMembersMaintainersRoleAttributes, randomRoleTwo, randomRoleTwo, randomEmailOne, randomEmailTwo, randomEmailThree, randomTeamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, KEY, randomTeamKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleTwo),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "testAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "testValue"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Check the team key can be updated (with force new)
			{
				Config: fmt.Sprintf(testAccTeamUpdateMembersMaintainersRoleAttributes, randomRoleTwo, randomRoleTwo, randomEmailOne, randomEmailTwo, randomEmailThree, randomNewTeamKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "Integrations"),
					resource.TestCheckResourceAttr(resourceName, KEY, randomNewTeamKey),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "The BEST integrations squad"),
					resource.TestCheckResourceAttr(resourceName, "member_ids.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintainers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_role_keys.0", randomRoleTwo),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "testAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "testValue"),
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
