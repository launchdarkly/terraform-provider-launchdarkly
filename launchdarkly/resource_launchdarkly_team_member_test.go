package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func testAccTeamMemberCreate(rName string) string {
	return fmt.Sprintf(`
resource "launchdarkly_team_member" "test" {
	email = "%s@example.com"
	first_name = "first"
	last_name = "last"
	role = "admin"
	custom_roles = []
}
`, rName)
}

func testAccTeamMemberUpdate(rName string) string {
	return fmt.Sprintf(`
resource "launchdarkly_team_member" "test" {
	email = "%s@example.com"
	first_name = "first"
	last_name = "last"
	role = "writer"
	custom_roles = []
}
`, rName)
}

func testAccTeamMemberCustomRoleCreate(roleKey, rName string) string {
	return fmt.Sprintf(`
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
		email = "%s@example.com"
		first_name = "first"
		last_name = "last"
		custom_roles = [launchdarkly_custom_role.test.key]
	}
	`, roleKey, roleKey, rName)
}

func testAccTeamMemberCustomRoleUpdate(roleKey1, roleKey2, rName string) string {
	return fmt.Sprintf(`
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
		email = "%s@example.com"
		first_name = "first"
		last_name = "last"
		custom_roles = [launchdarkly_custom_role.test_2.key]
	}
	`, roleKey1, roleKey1, roleKey2, roleKey2, rName)
}

func TestAccTeamMember_Create(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamMemberCreate(randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "role", "admin"),
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

func TestAccTeamMember_Update(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamMemberCreate(randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "role", "admin"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "0"),
				),
			},
			{
				Config: testAccTeamMemberUpdate(randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "role", "writer"),
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
				Config: testAccTeamMemberCustomRoleCreate(roleKey, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(roleKey), roleKey),
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
				Config: testAccTeamMemberCustomRoleCreate(roleKey1, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName1),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(roleKey1), roleKey1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamMemberCustomRoleUpdate(roleKey1, roleKey2, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomRoleExists(roleResourceName2),
					testAccCheckMemberExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "email", fmt.Sprintf("%s@example.com", randomName)),
					resource.TestCheckResourceAttr(resourceName, "first_name", "first"),
					resource.TestCheckResourceAttr(resourceName, "last_name", "last"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_roles."+testAccMemberCustomRolePropertyKey(roleKey2), roleKey2),
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

func testAccMemberCustomRolePropertyKey(roleKey string) string {
	return fmt.Sprintf("%d", hashcode.String(roleKey))
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
		_, _, err := client.ld.TeamMembersApi.GetMember(client.ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("received an error getting team member. %s", err)
		}
		return nil
	}
}
