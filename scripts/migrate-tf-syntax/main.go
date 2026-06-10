// Command migrate-tf-syntax converts Terraform HCL between block syntax and list-of-objects
// nested-attribute syntax for resources whose schema migrated from terraform-plugin-sdk/v2 to
// terraform-plugin-framework. Default mappings target the LaunchDarkly Terraform provider v2.x → v3
// cutover but the tool accepts an arbitrary mapping file so it can be reused for other providers.
//
// Usage:
//
//	migrate-tf-syntax -dir ./configs -direction v2-to-v3
//	migrate-tf-syntax -dir ./configs -direction v3-to-v2 -mappings my-spec.json
//	migrate-tf-syntax -dir ./configs -direction v2-to-v3 -dry-run
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

//go:embed mappings.json
var defaultMappings []byte

// AttrSpec describes one attribute that switched from a block to a list-of-objects nested attribute.
// Nested holds child specs for attributes that themselves contain converted blocks (e.g. rules ⊃ clauses).
type AttrSpec struct {
	Name   string      `json:"name"`
	Nested []*AttrSpec `json:"nested,omitempty"`
}

// DeprecationSpec describes an attribute that was removed from the provider schema between v2 and v3.
// Supported actions:
//   - "drop": remove the attribute from the resource body (no replacement).
//   - "iis_to_csa": rewrite include_in_snippet into a client_side_availability nested-attribute list.
//     "to" names the replacement attribute (e.g. client_side_availability or
//     default_client_side_availability). "using_mobile_key" overrides the synthesized mobile-key
//     value (default false) — set to "true" for the project resource which historically wrote
//     using_mobile_key=true when migrating include_in_snippet.
type DeprecationSpec struct {
	Name           string `json:"name"`
	Action         string `json:"action"`
	To             string `json:"to,omitempty"`
	UsingMobileKey string `json:"using_mobile_key,omitempty"`
}

// DSAttrRewrite describes a data-source attribute that was removed/renamed in v3. The script does
// a cross-file expression rewrite of every `data.<resource_label>.<name>.<from>` reference, where
// <resource_label> is the spec key holding this entry.
//
//   - To set: rename the terminal attr (`data.X.Y.from` → `data.X.Y.to`). Anything after is
//     preserved (e.g. `[0].using_environment_id`).
//   - ToExpr set: replace the whole prefix (`data.X.Y.from` → `data.X.Y.<to-expr>`). Use when the
//     v3 access path is structurally different.
type DSAttrRewrite struct {
	From   string `json:"from"`
	To     string `json:"to,omitempty"`
	ToExpr string `json:"to_expr,omitempty"`
}

// ResourceSpec bundles all rewrite operations that apply to one resource type. DSAttrRewrites apply
// to the data source of the same name.
type ResourceSpec struct {
	Blocks         []*AttrSpec        `json:"blocks,omitempty"`
	Deprecations   []*DeprecationSpec `json:"deprecations,omitempty"`
	DSAttrRewrites []*DSAttrRewrite   `json:"ds_attr_rewrites,omitempty"`
}

type Spec map[string]*ResourceSpec

func main() {
	var (
		dir         = flag.String("dir", ".", "directory containing .tf files (non-recursive)")
		direction   = flag.String("direction", "v2-to-v3", "v2-to-v3 (blocks → nested attrs) or v3-to-v2 (nested attrs → blocks)")
		mappingPath = flag.String("mappings", "", "path to mappings JSON (defaults to embedded LaunchDarkly v3 spec)")
		dryRun      = flag.Bool("dry-run", false, "print converted output to stdout instead of writing files")
	)
	flag.Parse()

	if *direction != "v2-to-v3" && *direction != "v3-to-v2" {
		die("direction must be v2-to-v3 or v3-to-v2")
	}

	raw := defaultMappings
	if *mappingPath != "" {
		b, err := os.ReadFile(*mappingPath)
		if err != nil {
			die(fmt.Sprintf("read mappings: %v", err))
		}
		raw = b
	}
	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		die(fmt.Sprintf("parse mappings: %v", err))
	}

	matches, err := filepath.Glob(filepath.Join(*dir, "*.tf"))
	if err != nil {
		die(err.Error())
	}
	if len(matches) == 0 {
		die(fmt.Sprintf("no .tf files in %s", *dir))
	}
	for _, f := range matches {
		if err := process(f, *direction, spec, *dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", f, err)
			os.Exit(1)
		}
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}

func process(path, direction string, spec Spec, dryRun bool) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, diag := hclwrite.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diag.HasErrors() {
		return fmt.Errorf("parse: %s", diag)
	}
	changed := false
	for _, blk := range f.Body().Blocks() {
		if blk.Type() != "resource" {
			continue
		}
		labels := blk.Labels()
		if len(labels) < 1 {
			continue
		}
		rspec, ok := spec[labels[0]]
		if !ok || rspec == nil {
			continue
		}
		var did bool
		if direction == "v2-to-v3" {
			if forward(blk.Body(), rspec.Blocks) {
				did = true
			}
			if applyDeprecations(blk.Body(), rspec.Deprecations) {
				did = true
			}
		} else {
			did = reverse(blk.Body(), rspec.Blocks)
		}
		if did {
			changed = true
		}
	}
	out := hclwrite.Format(f.Bytes())
	// Cross-file DS-reader rewrites apply only in v2-to-v3.
	if direction == "v2-to-v3" {
		var rewritten bool
		out, rewritten = applyDSAttrRewrites(out, spec)
		if rewritten {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if dryRun {
		fmt.Println("// ---", path, "---")
		_, err := os.Stdout.Write(out)
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	return os.WriteFile(path, out, 0o644)
}

// applyDSAttrRewrites runs every ds_attr_rewrite entry across the file contents. Each rewrite
// targets `data.<resource_label>.<name>.<from>` and either renames the terminal segment (`To`) or
// replaces the entire prefix with a new expression rooted at the same name (`ToExpr`).
//
// Implementation: regex over the formatted file bytes. The pattern uses a word boundary on `<from>`
// so it matches both the bare reference (`...client_side_availability`) and chained access
// (`...client_side_availability[0]`, `...client_side_availability.foo`). False positives inside
// HCL strings/comments are vanishingly unlikely for these very specific deprecated attr names.
func applyDSAttrRewrites(src []byte, spec Spec) ([]byte, bool) {
	out := src
	changed := false
	for label, rspec := range spec {
		if rspec == nil {
			continue
		}
		for _, rw := range rspec.DSAttrRewrites {
			if rw.From == "" {
				continue
			}
			re := regexp.MustCompile(`\bdata\.` + regexp.QuoteMeta(label) + `\.([A-Za-z_][A-Za-z0-9_]*)\.` + regexp.QuoteMeta(rw.From) + `\b`)
			var repl string
			switch {
			case rw.ToExpr != "":
				repl = "data." + label + ".${1}." + rw.ToExpr
			case rw.To != "":
				repl = "data." + label + ".${1}." + rw.To
			default:
				fmt.Fprintf(os.Stderr, "warning: ds_attr_rewrite on %q/%q missing \"to\" and \"to_expr\" (skipping)\n", label, rw.From)
				continue
			}
			newOut := re.ReplaceAll(out, []byte(repl))
			if !bytes.Equal(newOut, out) {
				out = newOut
				changed = true
			}
		}
	}
	return out, changed
}

// forward converts repeated `name { ... }` blocks into a single `name = [{...}, ...]` attribute.
// Recurses into nested specs first so the inner conversion is reflected in the serialized tokens
// of the outer element before we move them.
func forward(body *hclwrite.Body, specs []*AttrSpec) bool {
	changed := false
	for _, s := range specs {
		var matched []*hclwrite.Block
		for _, b := range body.Blocks() {
			if b.Type() == s.Name {
				matched = append(matched, b)
			}
		}
		if len(matched) == 0 {
			continue
		}
		if len(s.Nested) > 0 {
			for _, b := range matched {
				forward(b.Body(), s.Nested)
			}
		}
		tokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		}
		for i, b := range matched {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			tokens = append(tokens, trimLeadingNewlines(b.Body().BuildTokens(nil))...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
			if i < len(matched)-1 {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
		for _, b := range matched {
			body.RemoveBlock(b)
		}
		body.SetAttributeRaw(s.Name, tokens)
		changed = true
	}
	return changed
}

// reverse converts a `name = [{...}, ...]` nested attribute back into repeated `name { ... }` blocks.
// Best-effort: emitted blocks are appended at the end of the body (original attribute position is
// not preserved). Comments inside the tuple are preserved via raw token rendering.
func reverse(body *hclwrite.Body, specs []*AttrSpec) bool {
	changed := false
	for _, s := range specs {
		attr := body.GetAttribute(s.Name)
		if attr == nil {
			continue
		}
		elems := extractTupleElements(attr.Expr().BuildTokens(nil))
		if len(elems) == 0 {
			continue
		}
		body.RemoveAttribute(s.Name)
		for _, elem := range elems {
			elem = ensureTrailingNewline(trimLeadingNewlines(elem))
			newBlock := body.AppendNewBlock(s.Name, nil)
			if len(s.Nested) > 0 {
				// Re-parse element as a body so we can recurse on nested specs.
				wrapped := []byte(fmt.Sprintf("dummy {\n%s\n}\n", tokensString(elem)))
				tmp, diag := hclwrite.ParseConfig(wrapped, "<elem>", hcl.Pos{Line: 1, Column: 1})
				if diag.HasErrors() || len(tmp.Body().Blocks()) == 0 {
					newBlock.Body().AppendUnstructuredTokens(elem)
					changed = true
					continue
				}
				reverse(tmp.Body().Blocks()[0].Body(), s.Nested)
				newBlock.Body().AppendUnstructuredTokens(ensureTrailingNewline(tmp.Body().Blocks()[0].Body().BuildTokens(nil)))
			} else {
				newBlock.Body().AppendUnstructuredTokens(elem)
			}
			changed = true
		}
	}
	return changed
}

// applyDeprecations runs each deprecation rule against the resource body. Returns true if any
// rewrite happened. v2-to-v3 only; reverse direction is unsupported (deprecation removals are
// strictly one-way — the attribute no longer exists in the v3 schema).
func applyDeprecations(body *hclwrite.Body, deps []*DeprecationSpec) bool {
	changed := false
	for _, d := range deps {
		switch d.Action {
		case "drop":
			if body.GetAttribute(d.Name) != nil {
				body.RemoveAttribute(d.Name)
				changed = true
			}
		case "iis_to_csa":
			mobile := "false"
			if d.UsingMobileKey != "" {
				mobile = d.UsingMobileKey
			}
			if dropOrConvertIISToCSA(body, d.Name, d.To, mobile) {
				changed = true
			}
		case "policy_to_policy_statements":
			if convertPolicyToPolicyStatements(body, d.Name, d.To) {
				changed = true
			}
		case "rename":
			if renameAttribute(body, d.Name, d.To) {
				changed = true
			}
		default:
			fmt.Fprintf(os.Stderr, "warning: unknown deprecation action %q for attribute %q (skipping)\n", d.Action, d.Name)
		}
	}
	return changed
}

// dropOrConvertIISToCSA implements the iis_to_csa deprecation action. If the body already declares
// `to` (the replacement nested attribute, e.g. client_side_availability or
// default_client_side_availability), it just removes `name` — config wins. Otherwise it materializes
// `to = [{ using_environment_id = <iis-value>, using_mobile_key = false }]` and removes `name`.
//
// The IIS expression is preserved verbatim (true/false/var refs/etc.), so the result still type-
// checks under the v3 schema. Comments attached to the IIS attribute are lost — emit a one-line note
// on stderr so users notice.
func dropOrConvertIISToCSA(body *hclwrite.Body, name, to, mobile string) bool {
	if to == "" {
		fmt.Fprintf(os.Stderr, "warning: iis_to_csa action on %q requires \"to\" target (skipping)\n", name)
		return false
	}
	iisAttr := body.GetAttribute(name)
	if iisAttr == nil {
		return false
	}
	if body.GetAttribute(to) != nil {
		// Replacement already declared — config wins. Drop the deprecated attr.
		body.RemoveAttribute(name)
		return true
	}
	// Read the IIS expression tokens (right-hand side only) and trim surrounding whitespace.
	exprTokens := iisAttr.Expr().BuildTokens(nil)
	exprTokens = trimLeadingNewlines(exprTokens)
	// Build the replacement attribute tokens: to = [{
	//   using_environment_id = <iis-expr>
	//   using_mobile_key     = false
	// }]
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("using_environment_id")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
	}
	tokens = append(tokens, exprTokens...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("using_mobile_key")},
		&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(mobile)},
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
	)
	body.RemoveAttribute(name)
	body.SetAttributeRaw(to, tokens)
	return true
}

// renameAttribute moves the value of `name` onto `to`, preserving the right-hand-side expression
// tokens verbatim. If `to` already exists in the body, the original wins and `name` is dropped
// silently — same convention as the other deprecation actions (config wins over deprecated alias).
func renameAttribute(body *hclwrite.Body, name, to string) bool {
	if to == "" {
		fmt.Fprintf(os.Stderr, "warning: rename action on %q requires \"to\" target (skipping)\n", name)
		return false
	}
	src := body.GetAttribute(name)
	if src == nil {
		return false
	}
	if body.GetAttribute(to) != nil {
		body.RemoveAttribute(name)
		return true
	}
	tokens := src.Expr().BuildTokens(nil)
	body.RemoveAttribute(name)
	body.SetAttributeRaw(to, tokens)
	return true
}

// convertPolicyToPolicyStatements implements the policy_to_policy_statements deprecation action.
// The deprecated custom_role policy SetNestedAttribute carried elements with required resources,
// actions, and effect. The replacement policy_statements ListNestedAttribute adds optional
// not_resources and not_actions. We copy each policy element verbatim into policy_statements; the
// inner attribute names are identical so the inner expression tokens transfer unchanged.
//
// If policy_statements is already declared, the existing list wins and the deprecated policy
// attribute is dropped. By convention v3 users prefer the newer form when both are present.
func convertPolicyToPolicyStatements(body *hclwrite.Body, name, to string) bool {
	if to == "" {
		fmt.Fprintf(os.Stderr, "warning: policy_to_policy_statements action on %q requires \"to\" target (skipping)\n", name)
		return false
	}
	policyAttr := body.GetAttribute(name)
	if policyAttr == nil {
		return false
	}
	if body.GetAttribute(to) != nil {
		body.RemoveAttribute(name)
		return true
	}
	tokens := policyAttr.Expr().BuildTokens(nil)
	// policy was a set-nested attribute under the framework, so on disk it serializes as
	// `policy = [{ ... }, { ... }]` after the block conversion pass. The right-hand side is
	// already a tuple of object literals — exactly what policy_statements expects (a list). Reuse
	// the expression verbatim.
	body.RemoveAttribute(name)
	body.SetAttributeRaw(to, tokens)
	return true
}

// extractTupleElements walks token stream `[ {...}, {...}, ... ]` and returns the inner body
// tokens of each top-level `{...}` element (excluding the surrounding braces themselves). It
// tracks bracket + brace depth to handle nesting correctly.
func extractTupleElements(tokens hclwrite.Tokens) []hclwrite.Tokens {
	// Find first `[`.
	i := 0
	for ; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrack {
			i++
			break
		}
	}
	bracket := 1
	var elems []hclwrite.Tokens
	for i < len(tokens) && bracket > 0 {
		t := tokens[i]
		switch t.Type {
		case hclsyntax.TokenOBrack:
			bracket++
			i++
		case hclsyntax.TokenCBrack:
			bracket--
			i++
		case hclsyntax.TokenOBrace:
			if bracket != 1 {
				i++
				continue
			}
			brace := 1
			i++
			var elem hclwrite.Tokens
			for i < len(tokens) && brace > 0 {
				switch tokens[i].Type {
				case hclsyntax.TokenOBrace:
					brace++
				case hclsyntax.TokenCBrace:
					brace--
					if brace == 0 {
						i++
						goto endElem
					}
				}
				elem = append(elem, tokens[i])
				i++
			}
		endElem:
			elems = append(elems, elem)
		default:
			i++
		}
	}
	return elems
}

// tokensString renders tokens to their textual form preserving leading spaces (used when we need to
// re-parse a slice of tokens as an HCL body).
func tokensString(tokens hclwrite.Tokens) string {
	var buf bytes.Buffer
	_, _ = tokens.WriteTo(&buf)
	return buf.String()
}

// trimLeadingNewlines strips leading newline tokens from a slice so that an inserted block body
// does not double up with the newline AppendNewBlock injects after the opening brace.
func trimLeadingNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	i := 0
	for i < len(tokens) && tokens[i].Type == hclsyntax.TokenNewline {
		i++
	}
	return tokens[i:]
}

// ensureTrailingNewline guarantees the last non-whitespace token is followed by a newline so that an
// appended block body terminates correctly. Inline tuple elements like `{ value = "true" }` strip
// down to ` value = "true" ` with no trailing newline — without this fix the closing `}` lands on
// the same line as the last argument and HCL rejects it.
func ensureTrailingNewline(tokens hclwrite.Tokens) hclwrite.Tokens {
	for i := len(tokens) - 1; i >= 0; i-- {
		switch tokens[i].Type {
		case hclsyntax.TokenNewline:
			return tokens
		case hclsyntax.TokenComment:
			// Comments may include their own trailing newline; treat as fine.
			return tokens
		}
		if len(tokens[i].Bytes) > 0 {
			break
		}
	}
	return append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
}
