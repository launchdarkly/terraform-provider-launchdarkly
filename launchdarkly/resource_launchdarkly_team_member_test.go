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
	custom_roles = [launchdarkly_custom_role.test_2.key]
}
`
)

func TestAccTeamMember_CreateGeneric(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
		},
	})
}

func TestAccTeamMember_UpdateGeneric(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
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
		},
	})
}

func TestAccTeamMember_CreateWithCustomRole(t *testing.T) {
	roleKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleResourceName := "launchdarkly_custom_role.test"
	resourceName := "launchdarkly_team_member.custom_role_test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMemberCustomRoleCreate, roleKey, roleKey, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, EMAIL, fmt.Sprintf("%s+wbteste2e@launchdarkly.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, FIRST_NAME, "first"),
					resource.TestCheckResourceAttr(resourceName, LAST_NAME, "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey),
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

func TestAccTeamMember_UpdateWithCustomRole(t *testing.T) {
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
		Providers: testAccProviders,
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
					resource.TestCheckResourceAttr(resourceName, "custom_roles.0", roleKey2),
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
