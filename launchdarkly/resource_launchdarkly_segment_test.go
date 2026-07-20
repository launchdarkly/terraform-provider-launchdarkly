package launchdarkly

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

const (
	testAccSegmentCreate = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", "segmentTag2"]
	included    = ["user1", "user2"]
	excluded    = ["user3", "user4"]
}`

	testAccSegmentUpdate = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", ".segmentTag2"]
	included    = ["user1", "user2", "user3", "user4"]
	excluded    = []
	rules = [{
		clauses = [{
			attribute = "test_att"
			op = "in"
			values = ["test"]
		}, {
			attribute = "test_att_1"
			op = "endsWith"
			values = ["test2"]
			negate = true
			context_kind = "user"
		}]
		weight = 50000
		bucket_by = "bucket"
		rollout_context_kind = "other"
	}]
}`

	testAccSegmentUpdateWithContextTargets = `
	resource "launchdarkly_segment" "test" {
		key         = "segmentKey1"
		project_key = launchdarkly_project.test.key
		env_key     = "test"
		name        = "segment name"
		description = "segment description"
		included = ["user1", "user2"]
		included_contexts = [{
			values = ["account1", "account2"]
			context_kind = "account"
		}, {
			values = ["other_value"]
			context_kind = "other"
		}]
		excluded_contexts = [{
			values = ["bad_account"]
			context_kind = "account"
		}]
		rules = [{
			clauses = [{
				attribute = "test_att"
				op = "in"
				values = ["test"]
			}, {
				attribute = "test_att_1"
				op = "endsWith"
				values = ["test2"]
				negate = true
				context_kind = "user"
			}]
			weight = 50000
			bucket_by = "bucket"
		}]
	}`

	testAccSegmentCreateWithRules = `
resource "launchdarkly_segment" "test" {
    key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
  	name        = "segment name"
	description = "segment description"
	tags        = ["segmentTag1", "segmentTag2"]
	included    = ["user1", "user2"]
	excluded    = ["user3", "user4"]
	unbounded = false
	rules = [{
		clauses = [{
			attribute = "test_att"
			op        = "endsWith"
			values    = ["test"]
			negate    = false
		}]
	}, {
		clauses = [{
			attribute  = "is_vip"
			op         = "in"
			values     = [true]
			value_type = "boolean"
			negate     = false
			context_kind = "account"
		}, {
			attribute  = "answer"
			op         = "in"
			values     = [42, 84.68]
			value_type = "number"
			negate     = true
			context_kind = "survey"
		}]
	}]
}`

	testAccSegmentCreateWithContextTargets = `
resource "launchdarkly_segment" "test" {
	key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "segment name"
	excluded = ["user1", "user2"]
	included_contexts = [{
		values = ["account1"]
		context_kind = "account"
	}]
	excluded_contexts = [{
		values = ["bad_account"]
		context_kind = "account"
	}, {
		values = ["meanie", "beanie"]
		context_kind = "eanies"
	}]
}`

	testAccSegmentCreateWithUnbounded = `
resource "launchdarkly_segment" "test" {
	key                    = "segmentKey1"
	project_key            = launchdarkly_project.test.key
	env_key                = "test"
	name                   = "segment name"
	description            = "segment description"
	tags                   = ["segmentTag1", "segmentTag2"]
	unbounded              = true
	unbounded_context_kind = "device"
}`

	testAccSegmentCreateWithUnboundedUpdate = `
resource "launchdarkly_segment" "test" {
	key                    = "segmentKey1"
	project_key            = launchdarkly_project.test.key
	env_key                = "test"
	name                   = "segment name"
	description            = "segment description"
	tags                   = ["segmentTag1", "segmentTag2"]
	unbounded              = true
	unbounded_context_kind = "account"
}`

	testAccSegmentWithAnonymousUser = `
resource "launchdarkly_segment" "anon" {
	key 					= "anonymousSegment"
	project_key            	= launchdarkly_project.test.key
	env_key                	= "test"
	name 					= "anonymous segment"
	rules = [{
		clauses = [{
			attribute  = "anonymous"
			op         = "in"
			negate     = false
			values = [
				true
			]
		}]
	}]
}
`
)

func TestAccSegment_CreateAndUpdate(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", ".segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "included.2", "user3"),
					resource.TestCheckResourceAttr(resourceName, "included.3", "user4"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.values.0", "other_value"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "user"), // should default when missing
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
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

func TestAccSegment_Unbounded(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithUnbounded),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "true"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED_CONTEXT_KIND, "device"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithUnboundedUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "true"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED_CONTEXT_KIND, "account"),
				),
			},
		},
	})
}

func TestAccSegment_WithAnonymousClause(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.anon"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentWithAnonymousUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "anonymousSegment"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "anonymous segment"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "anonymous"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "true"),
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

func TestAccSegment_WithRules(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithRules),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.value_type", "string"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"), // this should automatically populate even if not set
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.attribute", "is_vip"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.0", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.value_type", "boolean"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.attribute", "answer"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.0", "42"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.values.1", "84.68"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.value_type", "number"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.1.context_kind", "survey"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove the rules block and confirm the rules were actually deleted
			{
				Config: withRandomProject(projectKey, testAccSegmentCreate), // this is the same but without the rules
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "segmentTag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "segmentTag2"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user3"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user4"),
					resource.TestCheckNoResourceAttr(resourceName, "included_contexts.#"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded_contexts.#"),
					resource.TestCheckNoResourceAttr(resourceName, fmt.Sprintf("%s.#", RULES)),
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

func TestAccSegment_WithTargetingByContext(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccSegmentCreateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "included.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.0", "meanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.1", "beanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.context_kind", "eanies"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: withRandomProject(projectKey, testAccSegmentUpdateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "segment description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "included.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "included.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "excluded.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.1", "account2"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.values.0", "other_value"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.1.context_kind", "other"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.weight", "50000"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.bucket_by", "bucket"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.rollout_context_kind", "user"), // should default from the API if not set
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "test_att"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "in"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.attribute", "test_att_1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.op", "endsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.values.0", "test2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.negate", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.1.context_kind", "user"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // check that it reverts well as well
				Config: withRandomProject(projectKey, testAccSegmentCreateWithContextTargets),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "segmentKey1"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment name"),
					resource.TestCheckResourceAttr(resourceName, "excluded.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "excluded.1", "user2"),
					resource.TestCheckNoResourceAttr(resourceName, "included.#"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.values.0", "account1"),
					resource.TestCheckResourceAttr(resourceName, "included_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.values.0", "bad_account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.0.context_kind", "account"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.0", "meanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.values.1", "beanie"),
					resource.TestCheckResourceAttr(resourceName, "excluded_contexts.1.context_kind", "eanies"),
					resource.TestCheckNoResourceAttr(resourceName, "rules.#"),
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

func testAccCheckSegmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		segmentKey, ok := rs.Primary.Attributes[KEY]
		if !ok {
			return fmt.Errorf("environment key not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("project key not found: %s", resourceName)
		}
		client := mustTestAccClient()
		_, _, err := client.ld.SegmentsApi.GetSegment(client.ctx, projKey, envKey, segmentKey).Execute()
		if err != nil {
			return fmt.Errorf("received an error getting environment. %s", err)
		}
		return nil
	}
}

// TestAccSegment_ViewAssociationRequired tests that creating a segment without view_keys
// fails when the project requires view association for new segments
func TestAccSegment_ViewAssociationRequired(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.test"
	testAccSegmentWithViewKeys := ""

	// Config with project requiring view association but segment without view_keys (should fail)
	testAccSegmentWithoutViewKeys := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_segments = true
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "010101"
		}
	}
}

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-no-views"
	name        = "Test Segment Without Views"
}
`, projectKey)

	// Config with project requiring view association and segment with view_keys (should succeed)
	testAccSegmentWithViewKeysTemplate := `
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "View Requirement Test"
	require_view_association_for_new_segments = true
	environments = {
		"test-env" = {
			name  = "Test Environment"
			color = "010101"
		}
	}
}

resource "launchdarkly_view" "test" {
	project_key   = launchdarkly_project.test.key
	key           = "test-view"
	name          = "Test View"
	maintainer_id = "%s"
}

resource "launchdarkly_segment" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "test-segment-with-views"
	name        = "Test Segment With Views"
	view_keys   = [launchdarkly_view.test.key]
}
`
	maintainerID := "507f1f77bcf86cd799439011"
	if os.Getenv("TF_ACC") != "" {
		testAccPreCheck(t)
		maintainerID = firstMemberIDForTest(t)
	}
	testAccSegmentWithViewKeys = fmt.Sprintf(testAccSegmentWithViewKeysTemplate, projectKey, maintainerID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Step 1: Verify segment without view_keys fails when project requires it
			{
				Config:      testAccSegmentWithoutViewKeys,
				ExpectError: regexp.MustCompile(`requires new segments to be associated with at least one view`),
			},
			// Step 2: Verify segment with view_keys succeeds when project requires it
			{
				Config: testAccSegmentWithViewKeys,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "view_keys.#", "1"),
				),
			},
		},
	})
}

// TestAccSegment_ApprovalRequired covers the targeting-free half of issue #370:
// when segment approvals are enabled for an environment, a segment with no
// targeting must still create, because the create path skips the gated, no-op
// targeting PATCH. Segment approvals are not exposed through the provider
// schema, so they are toggled out-of-band via the beta approval-settings API in
// a PreConfig step. Runs serially because it mutates environment-scoped approval
// settings.
//
// The complementary "targeting change is blocked under approvals" assertion
// lives in TestAccSegment_ApprovalRequiredWithoutBypass: the default acceptance
// token now carries the bypassRequiredSegmentApproval permission, so it can no
// longer demonstrate the blocked path — a token without that permission is
// required.
func TestAccSegment_ApprovalRequired(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := "test-env"
	bareResourceName := "launchdarkly_segment.bare"

	projectOnly := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "Segment Approvals Test"
	environments = {
		"%s" = {
			name  = "Test Environment"
			color = "010101"
		}
	}
}
`, projectKey, envKey)

	bareSegment := projectOnly + `
resource "launchdarkly_segment" "bare" {
	project_key = launchdarkly_project.test.key
	env_key     = "test-env"
	key         = "approval-bare"
	name        = "bare under approvals"
	description = "no targeting; must create under approvals"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			// Step 1: create the project + environment (no segment yet).
			{
				Config: projectOnly,
			},
			// Step 2: enable segment approvals, then a targeting-free segment
			// must still create successfully.
			{
				PreConfig: func() { enableSegmentApprovalsForTest(t, projectKey, envKey) },
				Config:    bareSegment,
				Check:     resource.ComposeTestCheckFunc(testAccCheckSegmentExists(bareResourceName)),
			},
		},
	})
}

// enableSegmentApprovalsForTest turns on segment approvals for the given
// environment via the beta approval-settings API. Segment approvals are not
// configurable through the provider schema, so the test toggles them directly.
func enableSegmentApprovalsForTest(t *testing.T, projectKey, envKey string) {
	t.Helper()
	host := os.Getenv(LAUNCHDARKLY_API_HOST)
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	body := fmt.Sprintf(`{"environmentKey":%q,"resourceKind":"segment","required":true,"serviceKind":"launchdarkly","minNumApprovals":1}`, envKey)
	req, err := http.NewRequest(http.MethodPatch, host+"/api/v2/approval-requests/projects/"+projectKey+"/settings", strings.NewReader(body))
	if err != nil {
		t.Fatalf("build approval-settings request: %s", err)
	}
	req.Header.Set("Authorization", os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN))
	req.Header.Set("LD-API-Version", "beta")
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("enable segment approvals: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("enable segment approvals for %s/%s returned %d: %s", projectKey, envKey, resp.StatusCode, string(b))
	}
}

// TestAccSegment_ApprovalBypass is the positive counterpart to
// TestAccSegment_ApprovalRequiredWithoutBypass (issue #370 / REL-15009). It
// verifies that when the token driving the provider holds a role that includes
// the "bypassRequiredSegmentApproval" action, the provider CAN apply segment
// targeting changes in an environment where segment approvals are required —
// the "approval is required" gate is bypassed rather than surfaced as an error.
//
// It mints a dedicated, project-scoped service token whose inline role grants
// the bypass action and drives a provider instance authenticated with it. The
// scaffold (project, environment, approval settings, token) runs out-of-band
// with the admin acceptance client because the scoped token is intentionally
// not permitted to create account-level resources like projects. Runs serially
// because it mutates environment-scoped approval settings.
func TestAccSegment_ApprovalBypass(t *testing.T) {
	projectKey, envKey, tokenSecret := segmentApprovalScopedSetup(t, "tf-acc-segment-bypass",
		func(projectKey, envKey string) []ldapi.StatementPost {
			// Full management of the project (the create path also reads the
			// environment for view reconciliation) plus the explicit bypass
			// action on segments. "*" alongside the explicit action guards
			// against the wildcard omitting the newly added action.
			return []ldapi.StatementPost{
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s", projectKey)}, []string{"*"}),
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s:env/*", projectKey)}, []string{"*"}),
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s:env/*:segment/*", projectKey)}, []string{"*", "bypassRequiredSegmentApproval"}),
			}
		})

	resourceName := "launchdarkly_segment.target"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: segmentApprovalTargetingConfig(tokenSecret, projectKey, envKey, "approval-bypass"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "a@b.com"),
				),
			},
		},
	})
}

// TestAccSegment_ApprovalRequiredWithoutBypass is the negative counterpart to
// TestAccSegment_ApprovalBypass and preserves the "targeting change is blocked
// under approvals" coverage from issue #370 that the default acceptance token
// can no longer demonstrate (it now carries the bypass permission). It mints a
// service token that can fully manage segments but is explicitly denied the
// bypassRequiredSegmentApproval action, then confirms that applying a targeting
// change still fails with the actionable "approval is required" error. Runs
// serially because it mutates environment-scoped approval settings.
func TestAccSegment_ApprovalRequiredWithoutBypass(t *testing.T) {
	projectKey, envKey, tokenSecret := segmentApprovalScopedSetup(t, "tf-acc-segment-nobypass",
		func(projectKey, envKey string) []ldapi.StatementPost {
			// Full segment management, but an explicit deny of the bypass
			// action (deny overrides allow), so this token remains subject to
			// the approval gate.
			return []ldapi.StatementPost{
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s", projectKey)}, []string{"*"}),
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s:env/*", projectKey)}, []string{"*"}),
				segmentInlineStatement("allow", []string{fmt.Sprintf("proj/%s:env/*:segment/*", projectKey)}, []string{"*"}),
				segmentInlineStatement("deny", []string{fmt.Sprintf("proj/%s:env/*:segment/*", projectKey)}, []string{"bypassRequiredSegmentApproval"}),
			}
		})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: segmentApprovalTargetingConfig(tokenSecret, projectKey, envKey, "approval-blocked"),
				// Terraform word-wraps diagnostics, so tolerate whitespace
				// (including newlines) between words.
				ExpectError: regexp.MustCompile(`approval\s+is\s+required`),
			},
		},
	})
}

// segmentApprovalScopedSetup scaffolds a project + environment with segment
// approvals required, then mints a service token whose inline role is built by
// statementsFn (called with the generated project and environment keys). It
// registers cleanup of the token and project and returns the project key, the
// environment key, and the token secret. The scaffold uses the admin
// acceptance client because scoped tokens cannot create projects. Because this
// runs live API calls before resource.Test would otherwise skip, it gates on
// TF_ACC so `make test` (unit) stays green.
func segmentApprovalScopedSetup(t *testing.T, tokenName string, statementsFn func(projectKey, envKey string) []ldapi.StatementPost) (projectKey, envKey, tokenSecret string) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}

	client := mustTestAccClient()
	projectKey = acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey = "test-env"

	if _, err := testAccProjectScaffoldCreate(client, ldapi.ProjectPost{
		Key:  projectKey,
		Name: "Segment Approval Scoped Test",
		Environments: []ldapi.EnvironmentPost{{
			Key:   envKey,
			Name:  "Test Environment",
			Color: "010101",
		}},
	}); err != nil {
		t.Fatalf("failed to scaffold project %q: %s", projectKey, err)
	}
	t.Cleanup(func() { _ = testAccProjectScaffoldDelete(client, projectKey) })

	// Require segment approvals for the environment (out-of-band beta API).
	enableSegmentApprovalsForTest(t, projectKey, envKey)

	token, _, err := client.ld.AccessTokensApi.PostToken(client.ctx).AccessTokenPost(ldapi.AccessTokenPost{
		Name:         ldapi.PtrString(tokenName + "-" + projectKey),
		ServiceToken: ldapi.PtrBool(true),
		InlineRole:   statementsFn(projectKey, envKey),
	}).Execute()
	if err != nil {
		t.Fatalf("failed to create scoped service token: %s", handleLdapiErr(err))
	}
	t.Cleanup(func() { _, _ = client.ld.AccessTokensApi.DeleteToken(client.ctx, token.Id).Execute() })
	if token.Token == nil || *token.Token == "" {
		t.Fatal("scoped service token response did not include a token secret")
	}
	return projectKey, envKey, *token.Token
}

// segmentApprovalTargetingConfig builds an HCL config that authenticates the
// provider with the given (scoped) token and manages a segment with a targeting
// rule in an approval-gated environment. Whether the apply succeeds or fails
// depends solely on whether the token can bypass segment approvals.
func segmentApprovalTargetingConfig(tokenSecret, projectKey, envKey, segmentKey string) string {
	return fmt.Sprintf(`
provider "launchdarkly" {
	access_token = %q
}

resource "launchdarkly_segment" "target" {
	project_key = %q
	env_key     = %q
	key         = %q
	name        = "targeting under approvals"
	rules = [{
		clauses = [{
			attribute    = "email"
			op           = "in"
			values       = ["a@b.com"]
			context_kind = "user"
		}]
	}]
}
`, tokenSecret, projectKey, envKey, segmentKey)
}

// segmentInlineStatement builds a single-resource-kind policy statement for an
// inline access-token role. Policy statements may not mix resource kinds, so
// callers pass one kind's specifiers per call.
func segmentInlineStatement(effect string, resources, actions []string) ldapi.StatementPost {
	s := ldapi.StatementPost{Effect: effect}
	s.SetResources(resources)
	s.SetActions(actions)
	return s
}
