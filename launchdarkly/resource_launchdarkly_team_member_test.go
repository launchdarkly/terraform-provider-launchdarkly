package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccTeamMemberCreate = `
resource "launchdarkly_team_member" "test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	role = "admin"
	custom_roles = []
}
`
	testAccTeamMemberUpdate = `
resource "launchdarkly_team_member" "test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	role = "no_access"
	custom_roles = []
}
`

	testAccTeamMemberCustomRoleCreate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on staging environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}

resource "launchdarkly_team_member" "custom_role_test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	custom_roles = [launchdarkly_custom_role.test.key]
}
`
	testAccTeamMemberCustomRoleUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on staging environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/staging"]
	}
}

resource "launchdarkly_custom_role" "test_2" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on production environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/production"]
	}
}

resource "launchdarkly_team_member" "custom_role_test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	custom_roles = [launchdarkly_custom_role.test.key]
}
`
	testAccTeamMemberCustomRoleWithRoleAttributes = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on testAttribute environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/$${roleAttribute/testAttribute}"]
	}
}

resource "launchdarkly_team_member" "custom_role_test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	custom_roles = [launchdarkly_custom_role.test.key]
	role_attributes {
		key = "testAttribute"
		values = ["staging", "production"]
	}
	role_attributes {
		key = "nonexistentAttribute"
		values = ["someValue"]
	}
}
`
	testAccTeamMemberCustomRoleWithRoleAttributesUpdate = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on testAttribute environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/$${roleAttribute/testAttribute}"]
	}
}

resource "launchdarkly_team_member" "custom_role_test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	custom_roles = [launchdarkly_custom_role.test.key]
	role_attributes {
		key = "newAttribute"
		values = ["value1", "value2"]
	}
	role_attributes {
		key = "testAttribute"
		values = ["staging"]
	}
}
`
	testAccTeamMemberCustomRoleWithRoleAttributesRemove = `
resource "launchdarkly_custom_role" "test" {
	key = "%s"
	name = "Updated - %s"
	description= "Allow all actions on testAttribute environments"
	policy_statements {
		actions = ["*"]	
		effect = "allow"
		resources = ["proj/*:env/$${roleAttribute/testAttribute}"]
	}
}

resource "launchdarkly_team_member" "custom_role_test" {
	email = "%s+wbteste2e@launchdarkly.com"
	first_name = "first"
	last_name = "last"
	custom_roles = [launchdarkly_custom_role.test.key]
}
`
)

func TestAccTeamMember_CreateAndUpdateGeneric(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTeamMemberDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMemberCreate, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, ROLE, "admin"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccTeamMemberUpdate, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, ROLE, "no_access"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "0"),
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

func TestAccTeamMember_WithCustomRole(t *testing.T) {
	roleKey1 := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleKey2 := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleResourceName1 := "launchdarkly_custom_role.test"
	roleResourceName2 := "launchdarkly_custom_role.test_2"
	resourceName := "launchdarkly_team_member.custom_role_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckTeamMemberDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleCreate, roleKey1, roleKey1, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName1),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleUpdate, roleKey1, roleKey1, roleKey2, roleKey2, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName2),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// delete launchdarkly_custom_role.test_2, udpate launchdarkly_custom_role.test with role attributes
				// and add role attribute values to the team member
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleWithRoleAttributes, roleKey1, roleKey1, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName1),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey1),

					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.key", "testAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.0", "staging"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.1", "production"),
					// we allow the setting of role attributes to be set even if they do not otherwise exist
					// on a custom role
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "nonexistentAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "someValue"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// remove the nonexistentAttribute block, reorder testAttribute block and add a newAttribute block,
				// and remove the production value from the testAttribute block
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleWithRoleAttributesUpdate, roleKey1, roleKey1, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName1),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey1),

					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.key", "testAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.1.values.0", "staging"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "newAttribute"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "value1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.1", "value2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// remove role attributes from the team member
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleWithRoleAttributesRemove, roleKey1, roleKey1, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName1),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey1),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "0"),
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

func testAccCheckMemberExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("team member ID is not set")
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err := client.ld.AccountMembersApi.GetMember(client.ctx, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting team member. %s", err)
		}
		return nil
	}
}

// testAccCheckTeamMemberDestroy verifies the team member has been destroyed
func testAccCheckTeamMemberDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_team_member" {
			continue
		}

		_, res, err := client.ld.AccountMembersApi.GetMember(client.ctx, rs.Primary.ID).Execute()

		if isStatusNotFound(res) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("team member %s still exists", rs.Primary.ID)
	}
	return nil
}
