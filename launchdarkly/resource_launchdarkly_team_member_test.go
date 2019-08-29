package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func testAccTeamMemberCreate(rName string) string {
	return fmt.Sprintf(`
resource "launchdarkly_team_member" "teamMember1" {
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
resource "launchdarkly_team_member" "teamMember1" {
	email = "%s@example.com"
	first_name = "first"
	last_name = "last"
	role = "writer"
	custom_roles = []
}
`, rName)
}

func TestAccTeamMember_Create(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.teamMember1"
	resource.Test(t, resource.TestCase{
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
		},
	})
}

func TestAccTeamMember_Update(t *testing.T) {
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_member.teamMember1"
	resource.Test(t, resource.TestCase{
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
