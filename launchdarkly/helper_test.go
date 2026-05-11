package launchdarkly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs403ApprovalRequiredFromParts(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		body   string
		errMsg string
		want   bool
	}{
		{
			name:   "403 status with 'approval is required' body",
			status: 403,
			body:   `{"code":"forbidden","message":"approval is required"}`,
			errMsg: "403 Forbidden",
			want:   true,
		},
		{
			name:   "403 status with capitalized Approval in body",
			status: 403,
			body:   `{"message":"This change requires Approval before it can be applied"}`,
			errMsg: "403 Forbidden",
			want:   true,
		},
		{
			name:   "403 status without approval keyword (e.g. token RBAC)",
			status: 403,
			body:   `{"code":"forbidden","message":"access denied"}`,
			errMsg: "403 Forbidden",
			want:   false,
		},
		{
			name:   "200 status — never a gate",
			status: 200,
			body:   `{"approval":"approved"}`,
			errMsg: "",
			want:   false,
		},
		{
			name:   "404 status — never a gate even if body mentions approval",
			status: 404,
			body:   `{"message":"approval workflow not found"}`,
			errMsg: "404 Not Found",
			want:   false,
		},
		{
			name:   "no status, error msg carries 403 + approval",
			status: 0,
			body:   "",
			errMsg: "403 Forbidden: approval is required for this change",
			want:   true,
		},
		{
			name:   "no status, body carries 403 + approval",
			status: 0,
			body:   `{"status":403,"message":"approval required"}`,
			errMsg: "request failed",
			want:   true,
		},
		{
			name:   "empty everything",
			status: 0,
			body:   "",
			errMsg: "",
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, is403ApprovalRequiredFromParts(tc.status, tc.body, tc.errMsg))
		})
	}
}
