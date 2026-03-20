package launchdarkly

import (
	"fmt"
	"time"
)

// aiTestCooldown adds a brief delay between AI config / variation tests.
// These tests use resource.Test (serial) instead of resource.ParallelTest
// because the AI Config API creates feature flags internally. The flag creation
// endpoint has a tight rate limit that returns 429, but the AI Config API handler
// translates this to a 400, bypassing the retry client. Serial execution with
// cooldown pauses avoids these transient failures.
func aiTestCooldown() {
	time.Sleep(2 * time.Second)
}

// withAITestProject wraps a Terraform config string with a random project resource
// for use in AI config/tool/variation/model config acceptance tests.
func withAITestProject(projectKey, resource string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
	key  = "%s"
	name = "AI Config Test Project"
	environments {
		name  = "Test Environment"
		key   = "test-env"
		color = "000000"
	}
}

%s`, projectKey, resource)
}
