package launchdarkly

import (
	"errors"
	"fmt"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v23"
)

func TestIsApprovalRequiredErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain approval message", errors.New(`{"code":"forbidden","message":"approval is required"}`), true},
		{"wrapped approval message", fmt.Errorf("patch failed: %w", errors.New("approval is required")), true},
		{"unrelated forbidden", errors.New(`{"code":"forbidden","message":"insufficient permissions"}`), false},
		{"unrelated error", errors.New("connection reset"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isApprovalRequiredErr(tt.err); got != tt.want {
				t.Fatalf("isApprovalRequiredErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAppendSegmentTargetingOps(t *testing.T) {
	rule := ldapi.UserSegmentRule{}
	target := ldapi.SegmentTarget{}

	t.Run("all empty yields no ops", func(t *testing.T) {
		ops := appendSegmentTargetingOps(nil, nil, nil, nil, nil, nil)
		if len(ops) != 0 {
			t.Fatalf("expected 0 ops for empty targeting, got %d: %+v", len(ops), ops)
		}
	})

	t.Run("only non-empty collections produce ops", func(t *testing.T) {
		ops := appendSegmentTargetingOps(nil,
			[]string{"u1"},                // included
			nil,                           // excluded (empty -> skipped)
			[]ldapi.UserSegmentRule{rule}, // rules
			nil,                           // includedContexts (empty -> skipped)
			[]ldapi.SegmentTarget{target}, // excludedContexts
		)
		gotPaths := map[string]bool{}
		for _, op := range ops {
			gotPaths[op.Path] = true
		}
		want := []string{"/included", "/rules", "/excludedContexts"}
		if len(ops) != len(want) {
			t.Fatalf("expected %d ops, got %d: %+v", len(want), len(ops), ops)
		}
		for _, p := range want {
			if !gotPaths[p] {
				t.Errorf("missing expected op path %q (got %v)", p, gotPaths)
			}
		}
		if gotPaths["/excluded"] || gotPaths["/includedContexts"] {
			t.Errorf("empty collections should not emit ops, got %v", gotPaths)
		}
	})
}
