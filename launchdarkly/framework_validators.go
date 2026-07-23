package launchdarkly

// framework_validators.go houses shared validator.String implementations
// (key, id, tag, op, length). New validators added here should be
// exercised by framework_validators_test.go with at least one positive
// and one negative case.

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// keyPattern is the canonical regex for LD resource keys.
var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// idPattern matches a 24-character hex ID (LD's UUID-style identifier).
var idPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)

// tagPattern is the per-element tag validator. The 1-64 length cap is
// enforced separately in the validator body.
var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`)

// operators enumerates every clause operator LD accepts.
var operators = []string{
	"in",
	"endsWith",
	"startsWith",
	"matches",
	"contains",
	"lessThan",
	"greaterThan",
	"lessThanOrEqual",
	"greaterThanOrEqual",
	"before",
	"after",
	"segmentMatch",
	"semVerEqual",
	"semVerLessThan",
	"semVerGreaterThan",
}

// keyValidator returns a String validator enforcing the LD key pattern.
func keyValidator() validator.String {
	return regexValidator{
		pattern: keyPattern,
		desc:    "Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	}
}

// keyAndLengthValidator combines the key regex with a min/max length check.
func keyAndLengthValidator(minLength, maxLength int) validator.String {
	return compositeStringValidator{
		validators: []validator.String{
			keyValidator(),
			stringLenBetween(minLength, maxLength),
		},
	}
}

// idValidator enforces a 24-character hex LD ID.
func idValidator() validator.String {
	return regexValidator{
		pattern: idPattern,
		desc:    "Must be a 24 character hexadecimal string",
	}
}

// tagValidator enforces tag-element rules: 1-64 chars, alphanumeric and
// .-_ only. Applied per element.
func tagValidator() validator.String {
	return compositeStringValidator{
		validators: []validator.String{
			stringLenBetween(1, 64),
			regexValidator{
				pattern: tagPattern,
				desc:    "Must contain only letters, numbers, '.', '-', or '_' and be at most 64 characters",
			},
		},
	}
}

// opValidator restricts a string attribute to the LD clause-operator enum.
func opValidator() validator.String {
	return oneOfValidator{allowed: operators}
}

// regexValidator validates a string against a regex pattern.
type regexValidator struct {
	pattern *regexp.Regexp
	desc    string
}

func (v regexValidator) Description(context.Context) string         { return v.desc }
func (v regexValidator) MarkdownDescription(context.Context) string { return v.desc }

func (v regexValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !v.pattern.MatchString(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("invalid value for %s (%s)", req.Path, v.desc),
		)
	}
}

// lengthValidator enforces a closed [minLength, maxLength] interval on
// the number of bytes in a string.
type lengthValidator struct {
	minLength int
	maxLength int
}

func stringLenBetween(minLength, maxLength int) validator.String {
	return lengthValidator{minLength: minLength, maxLength: maxLength}
}

func (v lengthValidator) Description(context.Context) string {
	return fmt.Sprintf("must be between %d and %d characters", v.minLength, v.maxLength)
}

func (v lengthValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v lengthValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if len(s) < v.minLength || len(s) > v.maxLength {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid length",
			fmt.Sprintf("expected length of %s to be in the range (%d - %d), got %d", req.Path, v.minLength, v.maxLength, len(s)),
		)
	}
}

// oneOfValidator restricts a string attribute to a fixed enum.
type oneOfValidator struct {
	allowed []string
}

func (v oneOfValidator) Description(context.Context) string {
	return fmt.Sprintf("must be one of: %v", v.allowed)
}

func (v oneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	got := req.ConfigValue.ValueString()
	for _, candidate := range v.allowed {
		if candidate == got {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		fmt.Sprintf("expected %s to be one of %v, got %s", req.Path, v.allowed, got),
	)
}

// compositeStringValidator runs a sequence of validators against the same
// value, accumulating diagnostics so all failures surface at once (matching
// validation.All semantics).
type compositeStringValidator struct {
	validators []validator.String
}

func (v compositeStringValidator) Description(ctx context.Context) string {
	parts := make([]string, 0, len(v.validators))
	for _, child := range v.validators {
		parts = append(parts, child.Description(ctx))
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "; "
		}
		out += p
	}
	return out
}

func (v compositeStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v compositeStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	for _, child := range v.validators {
		var childResp validator.StringResponse
		child.ValidateString(ctx, req, &childResp)
		resp.Diagnostics.Append(childResp.Diagnostics...)
	}
}
