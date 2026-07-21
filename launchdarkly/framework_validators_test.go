package launchdarkly

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runStringValidator(t *testing.T, v validator.String, value types.String) *validator.StringResponse {
	t.Helper()
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: value,
	}
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), req, resp)
	return resp
}

func TestKeyValidator(t *testing.T) {
	good := []string{"a", "abc", "my-key", "my_key", "0abc", "Z.dot.path"}
	for _, s := range good {
		resp := runStringValidator(t, keyValidator(), types.StringValue(s))
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected %q to pass, got %v", s, resp.Diagnostics)
		}
	}
	bad := []string{"-leading-dash", ".dot-first", "has space", "has/slash", ""}
	for _, s := range bad {
		resp := runStringValidator(t, keyValidator(), types.StringValue(s))
		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected %q to fail key validation", s)
		}
	}
}

func TestKeyValidator_NullUnknownSkip(t *testing.T) {
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := runStringValidator(t, keyValidator(), v)
		if resp.Diagnostics.HasError() {
			t.Fatalf("null/unknown values must not error, got %v", resp.Diagnostics)
		}
	}
}

func TestKeyAndLengthValidator(t *testing.T) {
	v := keyAndLengthValidator(3, 10)
	resp := runStringValidator(t, v, types.StringValue("abc"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected 'abc' to pass, got %v", resp.Diagnostics)
	}
	resp = runStringValidator(t, v, types.StringValue("ab"))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected too-short value to fail")
	}
	resp = runStringValidator(t, v, types.StringValue(strings.Repeat("x", 11)))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected too-long value to fail")
	}
	resp = runStringValidator(t, v, types.StringValue("-bad"))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected pattern-failing value to fail even within length")
	}
}

func TestIDValidator(t *testing.T) {
	resp := runStringValidator(t, idValidator(), types.StringValue("abcdef0123456789abcdef01"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected 24-char hex to pass, got %v", resp.Diagnostics)
	}
	resp = runStringValidator(t, idValidator(), types.StringValue("abc"))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected short id to fail")
	}
	resp = runStringValidator(t, idValidator(), types.StringValue("zzzdef0123456789abcdef01"))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected non-hex chars to fail")
	}
}

func TestTagValidator(t *testing.T) {
	v := tagValidator()
	resp := runStringValidator(t, v, types.StringValue("env-prod"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected valid tag, got %v", resp.Diagnostics)
	}
	resp = runStringValidator(t, v, types.StringValue(""))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected empty tag to fail length")
	}
	resp = runStringValidator(t, v, types.StringValue(strings.Repeat("a", 65)))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected 65-char tag to fail length")
	}
	resp = runStringValidator(t, v, types.StringValue("has space"))
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected disallowed-char tag to fail")
	}
}

func TestOpValidator(t *testing.T) {
	v := opValidator()
	for _, op := range []string{"in", "endsWith", "semVerEqual"} {
		resp := runStringValidator(t, v, types.StringValue(op))
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected %q to be a valid op, got %v", op, resp.Diagnostics)
		}
	}
	for _, op := range []string{"", "equals", "EQUALS"} {
		resp := runStringValidator(t, v, types.StringValue(op))
		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected %q to be rejected", op)
		}
	}
}
