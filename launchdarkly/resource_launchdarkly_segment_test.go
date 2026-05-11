package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
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
	rules {
		clauses {
			attribute = "test_att"
			op = "in"
			values = ["test"]
		}
		clauses {
			attribute = "test_att_1"
			op = "endsWith"
			values = ["test2"]
			negate = true
			context_kind = "user"
		}
		weight = 50000
		bucket_by = "bucket"
		rollout_context_kind = "other"
	}
}`

	testAccSegmentUpdateWithContextTargets = `
	resource "launchdarkly_segment" "test" {
		key         = "segmentKey1"
		project_key = launchdarkly_project.test.key
		env_key     = "test"
		name        = "segment name"
		description = "segment description"
		included = ["user1", "user2"]
		included_contexts {
			values = ["account1", "account2"]
			context_kind = "account"
		}
		included_contexts {
			values = ["other_value"]
			context_kind = "other"
		}
		excluded_contexts {
			values = ["bad_account"]
			context_kind = "account"
		}
		rules {
			clauses {
				attribute = "test_att"
				op = "in"
				values = ["test"]
			}
			clauses {
				attribute = "test_att_1"
				op = "endsWith"
				values = ["test2"]
				negate = true
				context_kind = "user"
			}
			weight = 50000
			bucket_by = "bucket"
		}
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
	rules {
		clauses {
			attribute = "test_att"
			op        = "endsWith"
			values    = ["test"]
			negate    = false
		}
	}
	rules {
		clauses {
			attribute  = "is_vip"
			op         = "in"
			values     = [true]
			value_type = "boolean"
			negate     = false
			context_kind = "account"
		}
		clauses {
			attribute  = "answer"
			op         = "in"
			values     = [42, 84.68]
			value_type = "number"
			negate     = true
			context_kind = "survey"
		}
	}
}`

	testAccSegmentCreateWithContextTargets = `
resource "launchdarkly_segment" "test" {
	key         = "segmentKey1"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "segment name"
	excluded = ["user1", "user2"]
	included_contexts {
		values = ["account1"]
		context_kind = "account"
	}
	excluded_contexts {
		values = ["bad_account"]
		context_kind = "account"
	}
	excluded_contexts {
		values = ["meanie", "beanie"]
		context_kind = "eanies"
	}
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
	rules {
		clauses {
			attribute  = "anonymous"
			op         = "in"
			negate     = false
			values = [
				true
			]
		}
	}
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
		Providers: testAccProviders,
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
		Providers: testAccProviders,
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
		Providers: testAccProviders,
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
		Providers: testAccProviders,
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
		Providers: testAccProviders,
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
		client := testAccProvider.Meta().(*Client)
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
	environments {
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
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
	environments {
		key   = "test-env"
		name  = "Test Environment"
		color = "010101"
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
		client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		members, _, err := client.ld.AccountMembersApi.GetMembers(client.ctx).Execute()
		require.NoError(t, err)
		require.True(t, len(members.Items) > 0, "This test requires at least one member in the account")
		maintainerID = members.Items[0].Id
	}
	testAccSegmentWithViewKeys = fmt.Sprintf(testAccSegmentWithViewKeysTemplate, projectKey, maintainerID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckProjectDestroy,
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

// TestAccSegment_MinimalCreateNoPatch validates the issue-#370 fix path: a
// minimal segment config (no rules / included / excluded / *_contexts) must
// succeed without a post-create PATCH. Under segment approvals this is the
// difference between create succeeding and the call failing with
// `403 approval is required`.
//
// This test does not enable segment approvals on the env — the provider's
// approval_settings schema today configures flag approvals only (different
// LD infra, per maintainer note on issue #339). It exercises the new
// segmentPostCreatePatchOps "skip PATCH entirely" branch end-to-end against
// real LD. The approval-gate scenario itself is covered manually; see the
// plan at .claude/plans/issue-370-segment-approval-create.md.
func TestAccSegment_MinimalCreateNoPatch(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_segment.minimal"
	config := fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	lifecycle {
		ignore_changes = [environments]
	}
	name = "testProject"
	key = "%s"
	environments {
		name  = "testEnvironment"
		key   = "test"
		color = "000000"
	}
}

resource "launchdarkly_segment" "minimal" {
	key         = "minimal-segment"
	project_key = launchdarkly_project.test.key
	env_key     = "test"
	name        = "minimal segment"
	description = "POST-only fields, no PATCH expected"
	tags        = ["tf-test"]
	unbounded   = false
}
`, projectKey)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckSegmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, "minimal-segment"),
					resource.TestCheckResourceAttr(resourceName, PROJECT_KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, "test"),
					resource.TestCheckResourceAttr(resourceName, NAME, "minimal segment"),
					resource.TestCheckResourceAttr(resourceName, DESCRIPTION, "POST-only fields, no PATCH expected"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tf-test"),
					resource.TestCheckResourceAttr(resourceName, UNBOUNDED, "false"),
					resource.TestCheckResourceAttrSet(resourceName, CREATION_DATE),
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

// TestAccSegment_CreateUnderApproval is the segment-approval-specific
// scenario from issue #370. It is currently skipped because segment
// approvals cannot be enabled via the provider's `approval_settings` block
// (that block gates flag approvals only, per the maintainer comment on
// issue #339). Until the LD API exposes segment-approval configuration in a
// way the provider can target, this scenario must be verified manually:
//
//  1. terraform apply project + env from the config below.
//  2. In the LD UI, enable "Require approvals" for segment changes on the
//     test env.
//  3. terraform apply the minimal segment. Expect: succeeds (Issue #370 fix).
//  4. terraform apply a segment with rules. Expect: fails with a recoverable
//     diagnostic naming the import command, and the resource is in state.
//
// See .claude/plans/issue-370-segment-approval-create.md for the full
// verification protocol.
func TestAccSegment_CreateUnderApproval(t *testing.T) {
	t.Skip("segment approvals are not configurable via the provider's approval_settings block (flag-only, per issue #339). Verified manually; see .claude/plans/issue-370-segment-approval-create.md.")
}
