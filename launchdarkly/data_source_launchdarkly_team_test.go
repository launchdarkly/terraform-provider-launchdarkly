package launchdarkly

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

func testAccDataSourceTeamConfig(teamKey string) string {
	return fmt.Sprintf(`
data "launchdarkly_team" "test" {
  key = "%s"
}
`, teamKey)
}

func testAccDataSourceTeamCreate(client *Client, teamKey string) (*ldapi.Team, error) {
	teamPostInput := ldapi.TeamPostInput{
		Key:  teamKey,
		Name: teamKey,
		RoleAttributes: &map[string][]string{
			"adminPermissions": []string{"production", "everything"},
		},
	}
	team, resp, err := client.ld.TeamsApi.PostTeam(client.ctx).TeamPostInput(teamPostInput).Execute()
	if err != nil {
		log.Printf("Error when calling `TeamsApi.PostTeam``:\nTeam: %v\nResponse: %v\nError:%v\n", team, resp, err)
		return nil, err
	}
	return team, nil
}

func testAccDataSourceTeamDelete(client *Client, teamKey string) error {
	_, err := client.ld.TeamsApi.DeleteTeam(client.ctx, teamKey).Execute()
	if err != nil {
		return err
	}
	return nil
}

func TestAccDataSourceTeam_noMatchReturnsError(t *testing.T) {
	key := "false-teeth"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceTeamConfig(key),
				ExpectError: regexp.MustCompile(`404 Not Found`),
			},
		},
	})
}

func TestAccDataSourceTeam_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	// Populate account with dummy team
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S)
	require.NoError(t, err)
	teamKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	team, createErr := testAccDataSourceTeamCreate(client, teamKey)
	require.NoError(t, createErr)

	resourceName := "data.launchdarkly_team.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamConfig(teamKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, KEY),
					resource.TestCheckResourceAttr(resourceName, KEY, *team.Key),
					resource.TestCheckResourceAttr(resourceName, NAME, *team.Name),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, *team.Description),
					resource.TestCheckResourceAttr(resourceName, ID, *team.Key),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.key", "adminPermissions"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.0", "production"),
					resource.TestCheckResourceAttr(resourceName, "role_attributes.0.values.1", "everything"),
				),
			},
		},
	})
	deleteErr := testAccDataSourceTeamDelete(client, teamKey)
	require.NoError(t, deleteErr)
}
