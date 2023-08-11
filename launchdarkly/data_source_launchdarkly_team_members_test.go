package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v12"
	"github.com/stretchr/testify/require"
)

func testAccDataSourceTeamMembersConfig(emails string) string {
	return fmt.Sprintf(`
data "launchdarkly_team_members" "test" {
  emails = %s
  ignore_missing = false
}
`, emails)
}

func testAccDataSourceTeamMembersConfigIgnoreMissing(emails string) string {
	return fmt.Sprintf(`
data "launchdarkly_team_members" "test" {
  emails = %s
  ignore_missing = true
}
`, emails)
}

func TestAccDataSourceTeamMembers_noMatchReturnsError(t *testing.T) {
	emails := `["does-not-exist+wbteste2e@launchdarkly.com"]`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceTeamMembersConfig(emails),
				ExpectError: regexp.MustCompile(`Error: No team member found for email: does-not-exist\+wbteste2e@launchdarkly.com`),
			},
		},
	})
}

func TestAccDataSourceTeamMembers_noMatchReturnsNoErrorIfIgnoreMissing(t *testing.T) {
	emails := `["does-not-exist+wbteste2e@launchdarkly.com"]`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamMembersConfigIgnoreMissing(emails),
			},
		},
	})
}

func TestAccDataSourceTeamMembers_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	// Populate account with dummy team members to ensure pagination is working
	teamMemberCount := 15
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)

	teamMembers := make([]ldapi.Member, 0, teamMemberCount)
	for i := 0; i < teamMemberCount; i++ {
		randomEmail := fmt.Sprintf("%s+wbteste2e@launchdarkly.com", acctest.RandStringFromCharSet(10, "abcdefghijklmnopqrstuvwxyz012346789+"))
		member, err := testAccDataSourceTeamMemberCreate(client, randomEmail)
		require.NoError(t, err)
		teamMembers = append(teamMembers, *member)
	}

	resourceName := "data.launchdarkly_team_members.test"
	testMember := teamMembers[teamMemberCount-1]
	testMember2 := teamMembers[teamMemberCount-2]
	testMember3 := teamMembers[teamMemberCount-3]
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamMembersConfig(fmt.Sprintf(`["%s","%s","%s"]`, testMember.Email, testMember2.Email, testMember3.Email)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, IGNORE_MISSING),
					resource.TestCheckResourceAttr(resourceName, "team_members.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "team_members.0.email", testMember.Email),
					resource.TestCheckResourceAttr(resourceName, "team_members.0.first_name", *testMember.FirstName),
					resource.TestCheckResourceAttr(resourceName, "team_members.0.last_name", *testMember.LastName),
					resource.TestCheckResourceAttr(resourceName, "team_members.0.id", testMember.Id),
					resource.TestCheckResourceAttr(resourceName, "team_members.0.role", testMember.Role),
					resource.TestCheckResourceAttr(resourceName, "team_members.1.email", testMember2.Email),
					resource.TestCheckResourceAttr(resourceName, "team_members.1.first_name", *testMember2.FirstName),
					resource.TestCheckResourceAttr(resourceName, "team_members.1.last_name", *testMember2.LastName),
					resource.TestCheckResourceAttr(resourceName, "team_members.1.id", testMember2.Id),
					resource.TestCheckResourceAttr(resourceName, "team_members.1.role", testMember2.Role),
					resource.TestCheckResourceAttr(resourceName, "team_members.2.email", testMember3.Email),
					resource.TestCheckResourceAttr(resourceName, "team_members.2.first_name", *testMember3.FirstName),
					resource.TestCheckResourceAttr(resourceName, "team_members.2.last_name", *testMember3.LastName),
					resource.TestCheckResourceAttr(resourceName, "team_members.2.id", testMember3.Id),
					resource.TestCheckResourceAttr(resourceName, "team_members.2.role", testMember3.Role),
				),
			},
		},
	})
	for _, member := range teamMembers {
		err := testAccDataSourceTeamMemberDelete(client, member.Id)
		require.NoError(t, err)
	}
}
