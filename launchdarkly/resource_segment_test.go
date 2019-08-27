package launchdarkly

import "testing"

func TestSegmentAcc(t *testing.T) {
	segmentCreate := testProject + `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "dummy-project"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2"]
	excluded = ["user3", "user4"]
}`
	segmentUpdate := testProject + `
resource "launchdarkly_segment" "segment3" {
    key = "segmentKey1"
	project_key = "dummy-project"
	env_key = "test"
  	name = "segment name"
	description = "segment description"
	tags = ["segmentTag1", "segmentTag2"]
	included = ["user1", "user2", "user3", "user4"]
	excluded = []
	rules = [
        {
        clauses = [{
            attribute = "test_att",
            op = "in",
            values = ["test"],
            negate = false,
            },
            {
            attribute = "test_att_1",
            op = "endsWith",
            values = ["test2"],
            negate = true,
            }],
        weight = 50000,
        bucket_by = "bucket"
		}
	]
}`

	testAcc(t, "launchdarkly_segment.segment3", segmentCreate, segmentUpdate)
}
