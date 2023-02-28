package launchdarkly

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccDestinationCreateKinesis = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "%s"
	name        = "kinesis-dest"
	kind        = "kinesis"
	config = {
		region      = "us-east-1"
		role_arn    = "arn:aws:iam::123456789012:role/marketingadmin"
		stream_name = "cat-stream"
	}
	on = true
	tags = [ "terraform" ]
}
`
	testAccDestinationCreatePubsub = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key     = "%s"
	name        = "pubsub-dest"
	kind        = "google-pubsub"
	config = {
		project = "test-project"
		topic   = "test-topic"
	}
	tags = [ "terraform" ]
}
`
	testAccDestinationCreateMparticleDeprecated = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "mparticle-dest"
	kind = "mparticle"
	config = {
		api_key       = "apiKeyfromMParticle"
		secret        = "mParticleSecret"
		user_identity = "customer_id"
		environment   = "production"
	}
	on = true
}
`
	testAccDestinationCreateMparticle = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "mparticle-dest"
	kind = "mparticle"
	config = {
		api_key       = "apiKeyfromMParticle"
		secret        = "mParticleSecret"
		user_identities = <<EOF
			[{"ldContextKind":"user","mparticleUserIdentity":"customer_id"}]
		EOF
		environment   = "production"
	}
	on = true
}
`

	testAccDestinationCreateMalformedMparticle = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "mparticle-dest"
	kind = "mparticle"
	config = {
		api_key       = "apiKeyfromMParticle"
		secret        = "mParticleSecret"
		user_identities = <<EOF
			[{"ldContextKind"="user","mparticleUserIdentity"="customer_id"}]
		EOF
		environment   = "production"
	}
	on = true
}
`
	testAccDestinationCreateSegmentDeprecated = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name    = "segment-dest"
	kind    = "segment"
	config  = {
		write_key = "super-secret-write-key"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationCreateSegment = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name    = "segment-dest"
	kind    = "segment"
	config  = {
		write_key = "super-secret-write-key"
		anonymous_id_context_kind = "anonymousUser"
		user_id_context_kind = "user"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationCreateAzureEventHubs = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name    = "azure-event-hubs-dest"
	kind    = "azure-event-hubs"
	config  = {
		namespace = "namespace"
		name = "name"
		policy_name = "policy-name"
		policy_key = "super-secret-policy-key"
	}
	on = true
	tags = [ "terraform" ]
}
`

	testAccDestinationUpdateKinesis = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "updated-kinesis-dest"
	kind = "kinesis"
	config = {
		region = "us-west-1",
		role_arn = "arn:aws:iam::123456789012:role/marketingadmin",
		stream_name = "cat-stream"
	}
	on = false
	tags = [ "terraform", "updated" ]
}
`
	testAccDestinationUpdatePubsub = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "updated-pubsub-dest"
	kind = "google-pubsub"
	config = {
		"project": "renamed-project",
		"topic": "test-topic"
	}
	on = true
	tags = [ "terraform", "updated" ]
}
`

	testAccDestinationUpdateMparticleDeprecated = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "updated-mparticle-dest"
	kind = "mparticle"
	config = {
		api_key = "updatedApiKey"
		secret = "updatedSecret"
		user_identity = "customer_id"
		environment = "production"
	}
	on = true
}
`

	testAccDestinationUpdateMparticle = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "updated-mparticle-dest"
	kind = "mparticle"
	config = {
		api_key = "updatedApiKey"
		secret = "updatedSecret"
		user_identities = <<EOF
			[{"ldContextKind":"user","mparticleUserIdentity":"customer_id"},
			{"ldContextKind":"device","mparticleUserIdentity":"google"}]
		EOF
		environment = "production"
	}
	on = true
}
`

	testAccDestinationUpdateSegmentDeprecated = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "segment-dest"
	kind = "segment"
	config = {
		write_key = "updated-write-key"
	}
	tags = [ "terraform" ]
}
`

	testAccDestinationUpdateSegment = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name = "segment-dest"
	kind = "segment"
	config = {
		write_key = "updated-write-key"
		anonymous_id_context_kind = "updatedAnonymousUser"
		user_id_context_kind = "updatedUser"
	}
	tags = [ "terraform" ]
}
`

	testAccDestinationUpdateAzureEventHubs = `
resource "launchdarkly_destination" "test" {
	project_key = launchdarkly_project.test.key
	env_key = "%s"
	name    = "updated-azure-event-hubs-dest"
	kind    = "azure-event-hubs"
	config  = {
		namespace = "namespace"
		name = "updated-name"
		policy_name = "updated-policy-name"
		policy_key = "updated-policy-key"
	}
	on = false
	tags = [ "terraform" ]
}
`
)

func TestAccDestination_CreateMparticleDeprecated(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateMparticleDeprecated, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.api_key", "apiKeyfromMParticle"),
				),
			},
		},
	})
}

func TestAccDestination_CreateMalformedMparticle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateMalformedMparticle, envKey)),
				ExpectError: regexp.MustCompile(`config field "user_identities" for destination kind "mparticle" is not valid: invalid character '=' after object key`),
			},
		},
	})
}

func TestAccDestination_CreateAndUpdateKinesis(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateKinesis, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "kinesis-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "kinesis"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.role_arn", "arn:aws:iam::123456789012:role/marketingadmin"),
					resource.TestCheckResourceAttr(resourceName, "config.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "config.stream_name", "cat-stream"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform")),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateKinesis, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "updated-kinesis-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "kinesis"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "config.role_arn", "arn:aws:iam::123456789012:role/marketingadmin"),
					resource.TestCheckResourceAttr(resourceName, "config.region", "us-west-1"),
					resource.TestCheckResourceAttr(resourceName, "config.stream_name", "cat-stream"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
		},
	})
}

func TestAccDestination_CreateAndUpdatePubsub(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreatePubsub, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "pubsub-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "google-pubsub"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "config.project", "test-project"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdatePubsub, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "updated-pubsub-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "google-pubsub"),
					resource.TestCheckResourceAttr(resourceName, "config.project", "renamed-project"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tags"},
			},
		},
	})
}

func TestAccDestination_CreateAndUpdateMparticle(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateMparticle, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.secret", "mParticleSecret"),
					resource.TestCheckResourceAttr(resourceName, "config.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "config.api_key", "apiKeyfromMParticle"),
					resource.TestCheckResourceAttr(resourceName, "config.user_identities", "[{\"ldContextKind\":\"user\",\"mparticleUserIdentity\":\"customer_id\"}]"),
					resource.TestCheckResourceAttr(resourceName, "config.user_identity", "customer_id"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.api_key", "config.secret"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateMparticle, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "updated-mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.secret", "updatedSecret"),
					resource.TestCheckResourceAttr(resourceName, "config.environment", "production"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.api_key", "config.secret"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateMparticleDeprecated, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "updated-mparticle-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "mparticle"),
					resource.TestCheckResourceAttr(resourceName, "config.secret", "updatedSecret"),
					resource.TestCheckResourceAttr(resourceName, "config.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "config.user_identity", "customer_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.api_key", "config.secret"},
			},
		},
	})
}

func TestAccDestination_CreateAndUpdateSegmentDeprecated(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateSegmentDeprecated, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "segment"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "super-secret-write-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.write_key", "tags"}, // TODO investigate why tags never works - is it broken?
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateSegmentDeprecated, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "segment"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"), // should default to false when removed
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "updated-write-key"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.write_key", "tags"},
			},
		},
	})
}

func TestAccDestination_CreateAndUpdateSegment(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateSegment, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "segment"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "super-secret-write-key"),
					resource.TestCheckResourceAttr(resourceName, "config.anonymous_id_context_kind", "anonymousUser"),
					resource.TestCheckResourceAttr(resourceName, "config.user_id_context_kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.write_key", "tags"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateSegment, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "segment-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "segment"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"), // should default to false when removed
					resource.TestCheckResourceAttr(resourceName, "config.write_key", "updated-write-key"),
					resource.TestCheckResourceAttr(resourceName, "config.anonymous_id_context_kind", "updatedAnonymousUser"),
					resource.TestCheckResourceAttr(resourceName, "config.user_id_context_kind", "updatedUser"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.write_key", "tags"},
			},
		},
	})
}

func TestAccDestination_CreateAndUpdateAzureEventHubs(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	envKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_destination.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationCreateAzureEventHubs, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "azure-event-hubs-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "azure-event-hubs"),
					resource.TestCheckResourceAttr(resourceName, "config.namespace", "namespace"),
					resource.TestCheckResourceAttr(resourceName, "config.name", NAME),
					resource.TestCheckResourceAttr(resourceName, "config.policy_name", "policy-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_key", "super-secret-policy-key"),
					resource.TestCheckResourceAttr(resourceName, ON, "true"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.policy_key", "tags"},
			},
			{
				Config: withRandomProjectAndEnv(projectKey, envKey, fmt.Sprintf(testAccDestinationUpdateAzureEventHubs, envKey)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("launchdarkly_project.test"),
					testAccCheckDestinationExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, ENV_KEY, envKey),
					resource.TestCheckResourceAttr(resourceName, NAME, "updated-azure-event-hubs-dest"),
					resource.TestCheckResourceAttr(resourceName, KIND, "azure-event-hubs"),
					resource.TestCheckResourceAttr(resourceName, "config.namespace", "namespace"),
					resource.TestCheckResourceAttr(resourceName, "config.name", "updated-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_name", "updated-policy-name"),
					resource.TestCheckResourceAttr(resourceName, "config.policy_key", "updated-policy-key"),
					resource.TestCheckResourceAttr(resourceName, ON, "false"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "terraform"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config.policy_key", "tags"},
			},
		},
	})
}

func testAccCheckDestinationExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("destination ID not set")
		}
		projKey, ok := rs.Primary.Attributes[PROJECT_KEY]
		if !ok {
			return fmt.Errorf("destination project key not found: %s", resourceName)
		}
		envKey, ok := rs.Primary.Attributes[ENV_KEY]
		if !ok {
			return fmt.Errorf("destination environment key not found: %s", resourceName)
		}
		client := testAccProvider.Meta().(*Client)
		_, _, destID, err := destinationImportIDtoKeys(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, _, err = client.ld.DataExportDestinationsApi.GetDestination(client.ctx, projKey, envKey, destID).Execute()
		if err != nil {
			return fmt.Errorf("error getting destination: %s", err)
		}
		return nil
	}
}
