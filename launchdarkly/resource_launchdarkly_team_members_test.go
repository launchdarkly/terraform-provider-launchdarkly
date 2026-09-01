package launchdarkly

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v24"
)

const (
	// A batch of three members, two of them on a team created in the same
	// config, so team assignment travels with member creation.
	testAccTeamMembersCreate = `
resource "launchdarkly_team" "batch_test" {
	key  = "%s"
	name = "Bulk members acceptance test"
}

resource "launchdarkly_team_members" "test" {
	members = {
		"%s+one@launchdarkly.com" = {
			first_name = "Member"
			last_name  = "One"
			role       = "reader"
			team_keys  = [launchdarkly_team.batch_test.key]
		}
		"%s+two@launchdarkly.com" = {
			role      = "writer"
			team_keys = [launchdarkly_team.batch_test.key]
		}
		"%s+three@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	// Adds one member, drops "three", and promotes "two" to admin. This is a
	// partial removal, which deletion protection must allow.
	testAccTeamMembersUpdate = `
resource "launchdarkly_team" "batch_test" {
	key  = "%s"
	name = "Bulk members acceptance test"
}

resource "launchdarkly_team_members" "test" {
	members = {
		"%s+one@launchdarkly.com" = {
			first_name = "Member"
			last_name  = "One"
			role       = "reader"
			team_keys  = [launchdarkly_team.batch_test.key]
		}
		"%s+two@launchdarkly.com" = {
			role      = "admin"
			team_keys = [launchdarkly_team.batch_test.key]
		}
		"%s+four@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	// Replaces every managed member, which deletion protection must refuse.
	testAccTeamMembersFullReplace = `
resource "launchdarkly_team" "batch_test" {
	key  = "%s"
	name = "Bulk members acceptance test"
}

resource "launchdarkly_team_members" "test" {
	members = {
		"%s+five@launchdarkly.com" = {
			role = "reader"
		}
		"%s+six@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	// Same membership as the update step but with protection disabled, so the
	// test can tear itself down.
	testAccTeamMembersUnprotected = `
resource "launchdarkly_team" "batch_test" {
	key  = "%s"
	name = "Bulk members acceptance test"
}

resource "launchdarkly_team_members" "test" {
	deletion_protection = false

	members = {
		"%s+one@launchdarkly.com" = {
			first_name = "Member"
			last_name  = "One"
			role       = "reader"
			team_keys  = [launchdarkly_team.batch_test.key]
		}
		"%s+two@launchdarkly.com" = {
			role      = "admin"
			team_keys = [launchdarkly_team.batch_test.key]
		}
		"%s+four@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	testAccTeamMembersProtected = `
resource "launchdarkly_team_members" "protected" {
	members = {
		"%s+protected@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	testAccTeamMembersUnprotectedSingle = `
resource "launchdarkly_team_members" "protected" {
	deletion_protection = false

	members = {
		"%s+protected@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`

	testAccTeamMembersAdoptDisabled = `
resource "launchdarkly_team_members" "adopt" {
	deletion_protection = false

	members = {
		"%s" = {
			role = "writer"
		}
	}
}
`

	testAccTeamMembersAdoptEnabled = `
resource "launchdarkly_team_members" "adopt" {
	adopt_existing      = true
	deletion_protection = false

	members = {
		"%s" = {
			role = "writer"
		}
	}
}
`

	testAccTeamMembersImport = `
resource "launchdarkly_team_members" "import_test" {
	deletion_protection = false

	members = {
		"%s+imp1@launchdarkly.com" = {
			role = "reader"
		}
		"%s+imp2@launchdarkly.com" = {
			role = "reader"
		}
	}
}
`
)

// TestAccTeamMembers_CreateUpdate covers the main lifecycle: a batched create,
// a stable follow-up plan, a partial update, the whole-batch replacement guard,
// and finally a destroy with protection disabled.
func TestAccTeamMembers_CreateUpdate(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamKey := strings.ToLower(acctest.RandStringFromCharSet(10, acctest.CharSetAlpha))
	resourceName := "launchdarkly_team_members.test"

	memberOne := fmt.Sprintf("%s+one@launchdarkly.com", name)
	memberTwo := fmt.Sprintf("%s+two@launchdarkly.com", name)
	memberThree := fmt.Sprintf("%s+three@launchdarkly.com", name)
	memberFour := fmt.Sprintf("%s+four@launchdarkly.com", name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMembersDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMembersCreate, teamKey, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "members.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "true"),
					resource.TestCheckResourceAttr(resourceName, "adopt_existing", "false"),
					// email defaults to the map key.
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.email", memberOne), memberOne),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.role", memberOne), "reader"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.first_name", memberOne), "Member"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.team_keys.#", memberOne), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.role", memberTwo), "writer"),
					resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("members.%s.id", memberThree)),
				),
			},
			{
				// A create that left computed values unresolved would show up
				// here: PlanOnly fails the test if the re-plan is not empty.
				Config:   fmt.Sprintf(testAccTeamMembersCreate, teamKey, name, name, name),
				PlanOnly: true,
			},
			{
				Config: fmt.Sprintf(testAccTeamMembersUpdate, teamKey, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "members.%", "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.role", memberTwo), "admin"),
					resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("members.%s.id", memberFour)),
					testAccCheckTeamMemberEmailAbsent(memberThree),
				),
			},
			{
				// Replacing every member is refused while protection is on, and
				// the refusal happens during planning.
				Config:      fmt.Sprintf(testAccTeamMembersFullReplace, teamKey, name, name),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Refusing to replace every member"),
			},
			{
				Config: fmt.Sprintf(testAccTeamMembersUnprotected, teamKey, name, name, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "false"),
				),
			},
		},
	})
}

// TestAccTeamMembers_DeletionProtection checks that a destroy is refused while
// protection is enabled and succeeds once it is turned off.
func TestAccTeamMembers_DeletionProtection(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_members.protected"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMembersDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMembersProtected, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "true"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccTeamMembersProtected, name),
				Destroy:     true,
				ExpectError: regexp.MustCompile("deletion protection is enabled"),
			},
			{
				Config: fmt.Sprintf(testAccTeamMembersUnprotectedSingle, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "deletion_protection", "false"),
				),
			},
		},
	})
}

// TestAccTeamMembers_AdoptExisting checks the conflict policy for an email that
// already belongs to a member of the account. The pre-existing member is
// created through the API rather than with launchdarkly_team_member, because
// managing the same person with two resources is exactly what the
// documentation warns against and would make teardown order significant.
func TestAccTeamMembers_AdoptExisting(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	email := strings.ToLower(fmt.Sprintf("%s+adopt@launchdarkly.com", name))
	resourceName := "launchdarkly_team_members.adopt"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSeedMember(t, email)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMembersDestroy,
		Steps: []resource.TestStep{
			{
				// Default behavior: refuse, and say which emails conflict.
				Config:      fmt.Sprintf(testAccTeamMembersAdoptDisabled, email),
				ExpectError: regexp.MustCompile("already exist in this account"),
			},
			{
				// Opting in takes over the member and reconciles their role.
				Config: fmt.Sprintf(testAccTeamMembersAdoptEnabled, email),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "adopt_existing", "true"),
					resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("members.%s.id", email)),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.role", email), "writer"),
				),
			},
		},
	})
}

// TestAccTeamMembers_Import checks that a comma-separated list of member IDs
// imports into an email-keyed map.
func TestAccTeamMembers_Import(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_team_members.import_test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMembersDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccTeamMembersImport, name, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamMembersExist(resourceName),
					resource.TestCheckResourceAttr(resourceName, "members.%", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTeamMembersImportIDs(resourceName),
				// ImportStateVerify matches resources by ID, but this resource's
				// batch ID is generated, so an imported batch legitimately has a
				// different one from the batch that created the members. Assert
				// the import contract directly instead.
				ImportStateCheck: testAccCheckTeamMembersImportedState(
					fmt.Sprintf("%s+imp1@launchdarkly.com", name),
					fmt.Sprintf("%s+imp2@launchdarkly.com", name),
				),
			},
		},
	})
}

// testAccCheckTeamMembersImportedState asserts what an import records. Import
// deliberately captures only the attributes this resource reconciles, so names
// and team keys must come back null even when the members have them.
func testAccCheckTeamMembersImportedState(emails ...string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported instance, got %d", len(states))
		}
		attrs := states[0].Attributes
		if got := attrs["members.%"]; got != fmt.Sprintf("%d", len(emails)) {
			return fmt.Errorf("expected %d imported members, got %q", len(emails), got)
		}
		if got := attrs["deletion_protection"]; got != "true" {
			return fmt.Errorf("imported batch should be protected, got deletion_protection=%q", got)
		}
		if got := attrs["adopt_existing"]; got != "false" {
			return fmt.Errorf("imported batch should not adopt, got adopt_existing=%q", got)
		}
		for _, email := range emails {
			if got := attrs[fmt.Sprintf("members.%s.email", email)]; got != email {
				return fmt.Errorf("member %s: email recorded as %q", email, got)
			}
			if got := attrs[fmt.Sprintf("members.%s.id", email)]; got == "" {
				return fmt.Errorf("member %s: no member ID recorded", email)
			}
			if got := attrs[fmt.Sprintf("members.%s.role", email)]; got != "reader" {
				return fmt.Errorf("member %s: role recorded as %q, want reader", email, got)
			}
			for _, unreconciled := range []string{"first_name", "last_name", "team_keys.#"} {
				if got, present := attrs[fmt.Sprintf("members.%s.%s", email, unreconciled)]; present && got != "" && got != "0" {
					return fmt.Errorf("member %s: %s should not be imported, got %q", email, unreconciled, got)
				}
			}
		}
		return nil
	}
}

// testAccTeamMembersImportIDs builds the comma-separated member ID list that
// this resource accepts as an import ID.
func testAccTeamMembersImportIDs(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		ids := make([]string, 0, 2)
		for key, value := range rs.Primary.Attributes {
			if strings.HasPrefix(key, "members.") && strings.HasSuffix(key, ".id") && value != "" {
				ids = append(ids, value)
			}
		}
		if len(ids) == 0 {
			return "", fmt.Errorf("no member IDs found in state for %s", resourceName)
		}
		return strings.Join(ids, ","), nil
	}
}

// testAccSeedMember creates a member outside Terraform, for the adoption test.
// It registers cleanup so the member is removed even if the test fails before
// Terraform takes ownership.
func testAccSeedMember(t *testing.T, email string) {
	t.Helper()
	client := mustTestAccClient()
	role := "reader"
	members, _, err := client.ld.AccountMembersApi.PostMembers(client.ctx).
		NewMemberForm([]ldapi.NewMemberForm{{Email: email, Role: &role}}).Execute()
	if err != nil {
		t.Fatalf("failed to seed member %s: %v", email, handleLdapiErr(err))
	}
	if len(members.Items) != 1 {
		t.Fatalf("expected 1 seeded member, got %d", len(members.Items))
	}
	id := members.Items[0].Id
	t.Cleanup(func() {
		res, err := client.ld.AccountMembersApi.DeleteMember(client.ctx, id).Execute()
		if err != nil && !isStatusNotFound(res) {
			t.Logf("failed to clean up seeded member %s: %v", email, handleLdapiErr(err))
		}
	})
}

// testAccCheckTeamMembersExist verifies every member recorded in state is
// really present in LaunchDarkly.
func testAccCheckTeamMembersExist(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		client := mustTestAccClient()
		checked := 0
		for key, id := range rs.Primary.Attributes {
			if !strings.HasPrefix(key, "members.") || !strings.HasSuffix(key, ".id") || id == "" {
				continue
			}
			if _, _, err := client.ld.AccountMembersApi.GetMember(client.ctx, id).Execute(); err != nil {
				return fmt.Errorf("member %s (%s) not found: %v", id, key, handleLdapiErr(err))
			}
			checked++
		}
		if checked == 0 {
			return fmt.Errorf("no member IDs recorded in state for %s", resourceName)
		}
		return nil
	}
}

// testAccCheckTeamMemberEmailAbsent verifies a member removed from the batch
// was actually deleted, rather than merely dropped from state.
func testAccCheckTeamMemberEmailAbsent(email string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := mustTestAccClient()
		members, err := getTeamMembersByEmail(client, []string{email})
		if err != nil {
			return fmt.Errorf("failed to check for member %s: %v", email, err)
		}
		for _, m := range members {
			if strings.EqualFold(m.Email, email) {
				return fmt.Errorf("member %s still exists but should have been deleted", email)
			}
		}
		return nil
	}
}

func testAccCheckTeamMembersDestroy(s *terraform.State) error {
	client := mustTestAccClient()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "launchdarkly_team_members" {
			continue
		}
		for key, id := range rs.Primary.Attributes {
			if !strings.HasPrefix(key, "members.") || !strings.HasSuffix(key, ".id") || id == "" {
				continue
			}
			_, res, err := client.ld.AccountMembersApi.GetMember(client.ctx, id).Execute()
			if isStatusNotFound(res) {
				continue
			}
			if err != nil {
				return err
			}
			return fmt.Errorf("team member %s still exists", id)
		}
	}
	return nil
}

// TestAccTeamMembers_ManyTeamsPerMember covers heavy team fan-out: members
// carrying a dozen team assignments each, the shape large enterprise accounts
// use (app x tier teams, each granting a role bundle). Team assignment must
// stay one grouped request per team and converge to a stable plan.
func TestAccTeamMembers_ManyTeamsPerMember(t *testing.T) {
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	teamPrefix := strings.ToLower(acctest.RandStringFromCharSet(10, acctest.CharSetAlpha))
	resourceName := "launchdarkly_team_members.fanout"

	const teamCount = 12
	teamBlocks := make([]string, 0, teamCount)
	teamRefs := make([]string, 0, teamCount)
	for i := 0; i < teamCount; i++ {
		teamBlocks = append(teamBlocks, fmt.Sprintf(`
resource "launchdarkly_team" "fanout_%d" {
	key  = "%s-%d"
	name = "Fanout %d"
}`, i, teamPrefix, i, i))
		teamRefs = append(teamRefs, fmt.Sprintf("launchdarkly_team.fanout_%d.key", i))
	}
	allTeams := strings.Join(teamRefs, ", ")

	memberOne := fmt.Sprintf("%s+fanone@launchdarkly.com", name)
	memberTwo := fmt.Sprintf("%s+fantwo@launchdarkly.com", name)
	memberThree := fmt.Sprintf("%s+fanthree@launchdarkly.com", name)

	config := fmt.Sprintf(`
%s

resource "launchdarkly_team_members" "fanout" {
	members = {
		"%s" = { role = "reader", team_keys = [%s] }
		"%s" = { role = "reader", team_keys = [%s] }
		"%s" = { role = "writer", team_keys = [%s] }
	}
	deletion_protection = false
}`, strings.Join(teamBlocks, "\n"), memberOne, allTeams, memberTwo, allTeams, memberThree, allTeams)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTeamMembersDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "members.%", "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.team_keys.#", memberOne), fmt.Sprint(teamCount)),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("members.%s.team_keys.#", memberThree), fmt.Sprint(teamCount)),
					resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("members.%s.id", memberTwo)),
				),
			},
			{
				// The fan-out must be stable: a second plan sees no changes.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
