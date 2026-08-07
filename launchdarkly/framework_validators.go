package launchdarkly

// framework_validators.go houses shared validator.String implementations
// (key, id, tag, op, length). New validators added here should be
// exercised by framework_validators_test.go with at least one positive
// and one negative case.

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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

// viewKeyValidator returns a String validator for view keys. LaunchDarkly
// normalises view keys to lowercase server-side, so an uppercase key in
// configuration can never round-trip: the API stores the lowercased form,
// Read pulls that back, and the resulting permanent diff forces a replace on
// every plan. Rejecting it up front turns that into a plan-time error that
// names the fix.
//
// Applies to every attribute a user can type a view key into, not just
// launchdarkly_view.key — a reference site (view_links.view_key,
// feature_flag.view_keys, ...) that names an uppercase key would otherwise
// fail later as an opaque 404.
func viewKeyValidator() validator.String {
	return compositeStringValidator{
		validators: []validator.String{
			keyValidator(),
			lowercaseValidator("LaunchDarkly normalises view keys to lowercase"),
		},
	}
}

// lowercaseValidator rejects any string containing uppercase characters. Kept
// separate from the key regex so the diagnostic can name the real problem and
// suggest the normalised value, rather than emitting the generic
// "letters, numbers, '.', '-', or '_'" message for a value that already
// satisfies those rules.
//
// reason is folded into the diagnostic to explain why this particular
// attribute must be lowercase; pass "" to omit it.
func lowercaseValidator(reason string) validator.String {
	return caseValidator{reason: reason}
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

// caseValidator enforces that a string contains no uppercase characters.
type caseValidator struct {
	reason string
}

func (v caseValidator) Description(context.Context) string {
	return "Must be lowercase"
}

func (v caseValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v caseValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	got := req.ConfigValue.ValueString()
	lowered := strings.ToLower(got)
	if got == lowered {
		return
	}
	detail := fmt.Sprintf("expected %s to be lowercase, got %q", req.Path, got)
	if v.reason != "" {
		detail += ". " + v.reason
	}
	detail += fmt.Sprintf(". Use %q instead", lowered)
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", detail)
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
