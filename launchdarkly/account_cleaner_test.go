package launchdarkly

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	dummyProject = "dummy-project"
)

func TestCleanAccount(t *testing.T) {
	// Uncomment this if you really want to wipe the account.
	t.SkipNow()

	fmt.Println("****** DANGER!!!! ******")
	fmt.Println("We're about to clean your account!!! pausing 10 seconds so you can kill this in case it was run by mistake!!")
	time.Sleep(10 * time.Second)

	require.NoError(t, cleanAccount())
}

func cleanAccount() error {
	c, err := NewClient(os.Getenv(launchDarklyApiKeyEnvVar))
	if err != nil {
		return err
	}

	err = c.cleanProjects()
	if err != nil {
		return err
	}
	err = c.cleanTeamMembers()
	if err != nil {
		return err
	}

	err = c.cleanCustomRoles()
	if err != nil {
		return err
	}
	return nil
}

// cleanProjects ensures exactly one project with name and key 'dummy-project' exists for an account.
// LD requires at least one project in an account.
func (c *Client) cleanProjects() error {
	// make sure we have a dummy project
	_, response, err := c.ld.ProjectsApi.GetProject(c.ctx, dummyProject)

	if response.StatusCode == 404 {
		_, _, err = c.ld.ProjectsApi.PostProject(c.ctx, ldapi.ProjectBody{Name: dummyProject, Key: dummyProject})
		if err != nil {
			return handleLdapiErr(err)
		}
	} else {
		if err != nil {
			return err
		}
	}
	projects, _, err := c.ld.ProjectsApi.GetProjects(c.ctx)
	if err != nil {
		return handleLdapiErr(err)
	}

	// delete all but dummy project
	for _, p := range projects.Items {
		if p.Key != dummyProject {
			_, err := c.ld.ProjectsApi.DeleteProject(c.ctx, p.Key)
			if err != nil {
				return handleLdapiErr(err)
			}
		}
	}
	return nil
}

// cleanTeamMembers ensures the only team member is the account owner
func (c *Client) cleanTeamMembers() error {
	members, _, err := c.ld.TeamMembersApi.GetMembers(c.ctx)
	if err != nil {
		return handleLdapiErr(err)
	}
	for _, m := range members.Items {
		if *m.Role != ldapi.OWNER {
			_, err := c.ld.TeamMembersApi.DeleteMember(c.ctx, m.Id)
			if err != nil {
				return handleLdapiErr(err)
			}
		}
	}
	return nil
}

// cleanCustomRoles deletes all custom roles
func (c *Client) cleanCustomRoles() error {
	roles, _, err := c.ld.CustomRolesApi.GetCustomRoles(c.ctx)
	if err != nil {
		return handleLdapiErr(err)
	}

	for _, r := range roles.Items {
		_, err := c.ld.CustomRolesApi.DeleteCustomRole(c.ctx, r.Id)
		if err != nil {
			return handleLdapiErr(err)
		}
	}
	return nil
}
