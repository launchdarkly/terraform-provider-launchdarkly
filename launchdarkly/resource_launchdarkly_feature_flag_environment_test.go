package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	testAccFeatureFlagEnvironmentBasic = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "basic" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	targeting_enabled = false
  	flag_fallthrough {
    	variation = 1
  	}
}
`

	testAccFeatureFlagEnvironmentUpdate = `	
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 0
	}
	variations {
		value = 10
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "basic" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	targeting_enabled = true
	track_events = true
	user_targets {
		values = []
	}
	user_targets {
		values = ["user1", "user2"]
	}
	user_targets {
		values = []
	}
	rules {
		clauses {
			attribute = "country"
			op        = "startsWith"
			values    = ["great", "amazing"]
			negate    = false
		}
		variation = 0
	}
	rules {
		clauses {
			attribute = "name"
			op        = "startsWith"
			values    = ["h"]
			negate    = false
		}
		rollout_weights = [90000, 10000, 0]
		bucket_by = "email"
	}

	flag_fallthrough {
		rollout_weights = [60000, 40000, 0]
		bucket_by = "email"
	}
}
`

	testAccFeatureFlagEnvironmentPrereq = `
resource "launchdarkly_feature_flag" "bool" {
	project_key = launchdarkly_project.test.key
	key = "bool-flag"
	name = "boolean flag"
	variation_type = "boolean"
}

resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "prereq" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	targeting_enabled = true
	  
	prerequisites {
		flag_key = launchdarkly_feature_flag.bool.key
		variation = 0
	}
}
`

	testAccInvalidFallthroughBucketBy = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "invalid_bucket_by" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	targeting_enabled = true
	  
	flag_fallthrough {
		bucket_by = "email"
	}
}
`

	testAccInvalidRuleBucketBy = `
resource "launchdarkly_feature_flag" "basic" {
	project_key = launchdarkly_project.test.key
	key = "basic-flag"
	name = "Basic feature flag"
	variation_type = "number"
	variations {
		value = 10
	}
	variations {
		value = 20
	}
	variations {
		value = 30
	}
}

resource "launchdarkly_feature_flag_environment" "invalid_bucket_by" {
	flag_id 		  = launchdarkly_feature_flag.basic.id
	env_key 		  = "test"
	targeting_enabled = true
	  
	rules {
		clauses {
			attribute = "name"
			op        = "startsWith"
			values    = ["h"]
			negate    = false
		}
		variation = 0
		bucket_by = "name"
	}
}
`
)

func TestAccFeatureFlagEnvironment_Basic(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "targeting_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.variation", "1"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_Update(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "targeting_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "0"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "targeting_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "track_events", "true"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout_weights.0", "60000"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout_weights.1", "40000"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.0.values.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.1.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.1.values.0", "user1"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.1.values.1", "user2"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.2.values.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.variation", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.attribute", "country"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.0", "great"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.values.1", "amazing"),
					resource.TestCheckResourceAttr(resourceName, "rules.0.clauses.0.negate", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.0", "90000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.1", "10000"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.rollout_weights.2", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.bucket_by", "email"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.attribute", "name"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.op", "startsWith"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.values.0", "h"),
					resource.TestCheckResourceAttr(resourceName, "rules.1.clauses.0.negate", "false"),
				),
			},
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentBasic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "targeting_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.variation", "1"),
					resource.TestCheckResourceAttr(resourceName, "flag_fallthrough.0.rollout.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "user_targets.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "rules.#", "0"),
				),
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_InvalidBucketBy(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	// resourceName := "launchdarkly_feature_flag_environment.invalid_bucket_by"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProject(projectKey, testAccInvalidFallthroughBucketBy),
				ExpectError: regexp.MustCompile("cannot use bucket_by argument with variation, only with rollout_weights"),
			},
			{
				Config:      withRandomProject(projectKey, testAccInvalidRuleBucketBy),
				ExpectError: regexp.MustCompile("cannot use bucket_by argument with variation, only with rollout_weights"),
			},
		},
	})
}

func TestAccFeatureFlagEnvironment_Prereq(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_feature_flag_environment.prereq"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProject(projectKey, testAccFeatureFlagEnvironmentPrereq),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFeatureFlagEnvironmentExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "targeting_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "prerequisites.0.flag_key", "bool-flag"),
				),
			},
		},
	})
}

func testAccCheckFeatureFlagEnvironmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		flagId, ok := rs.Primary.Attributes[FLAG_ID]
		if !ok {
			return fmt.Errorf("feature flag id not found: %s", resourceName)
		}
		projKey, flagKey, err := flagIdToKeys(flagId)
		if err != nil {
			return err
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("environent key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projKey, flagKey, &ldapi.GetFeatureFlagOpts{Env: optional.NewString(envKey)})
		if err != nil {
			return fmt.Errorf("received an error getting feature flag environment. %s", err)
		}
		return nil
	}
}
