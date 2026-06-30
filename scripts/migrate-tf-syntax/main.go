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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

//go:embed mappings.json
var defaultMappings []byte

// AttrSpec describes one attribute that switched from a block to a nested attribute.
// Nested holds child specs for attributes that themselves contain converted blocks (e.g. rules ⊃ clauses).
// Object marks a genuine single-object attribute (provider v3.x SingleNestedAttribute): forward emits
// `name = { ... }` and reverse parses that object back to a block, instead of the list-of-objects
// `name = [{ ... }]` shape used by List/SetNestedAttribute. See REL-14237.
// MapKey names an inner attribute hoisted to the map key for a MapNestedAttribute (provider v3.x):
// forward emits `name = { <keyval> = { ...rest } }` keyed by each block's MapKey attribute (which is
// dropped from the inner object), and reverse expands the map back to repeated blocks, re-injecting
// `MapKey = <key>`. Only literal-string keys are hoisted automatically; non-literal keys warn+skip.
// Mutually exclusive with Object. See REL-14236 (launchdarkly_project environments).
type AttrSpec struct {
	Name   string      `json:"name"`
	Nested []*AttrSpec `json:"nested,omitempty"`
	Object bool        `json:"object,omitempty"`
	MapKey string      `json:"map_key,omitempty"`
}

// DeprecationSpec describes an attribute that was removed from the provider schema between v2 and v3.
// Supported actions:
//   - "drop": remove the attribute from the resource body (no replacement).
//   - "iis_to_csa": rewrite include_in_snippet into a client_side_availability nested-attribute list.
//     "to" names the replacement attribute (e.g. client_side_availability or
//     default_client_side_availability). "using_mobile_key" overrides the synthesized mobile-key
//     value (default false) — set to "true" for the project resource which historically wrote
//     using_mobile_key=true when migrating include_in_snippet.
//   - "rename": move the value of "name" onto "to" verbatim (e.g. policy_statements → inline_roles).
//   - "policy_to_policy_statements": copy the custom_role policy list onto "to" (policy_statements).
//   - "ensure_boolean_variations": synthesize the required variations attribute on a feature_flag whose
//     variation_type is the literal "boolean" and that omitted variations (v2 allowed this, v3 does
//     not). The value is the LaunchDarkly invariant [{ value = "true" }, { value = "false" }]. "name"
//     and "to" are unused. Skips (and warns) when variation_type is a non-literal expression.
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
//   - StripIndex: when the v3 target is a single object (not a list), drop a single trailing list
//     index (`[0]` or `.0`) that immediately follows the matched attribute, so a v2 list access like
//     `data.X.Y.from[0].z` becomes `data.X.Y.to.z`. Applies to To and ToExpr rewrites alike.
type DSAttrRewrite struct {
	From       string `json:"from"`
	To         string `json:"to,omitempty"`
	ToExpr     string `json:"to_expr,omitempty"`
	StripIndex bool   `json:"strip_index,omitempty"`
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
		dir         = flag.String("dir", ".", "directory containing .tf files")
		direction   = flag.String("direction", "v2-to-v3", "v2-to-v3 (blocks → nested attrs) or v3-to-v2 (nested attrs → blocks)")
		mappingPath = flag.String("mappings", "", "path to mappings JSON (defaults to embedded LaunchDarkly v3 spec)")
		dryRun      = flag.Bool("dry-run", false, "print converted output to stdout instead of writing files")
		recursive   = flag.Bool("recursive", false, "descend into subdirectories (skips .terraform and .git); covers local modules")
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

	matches, err := collectTFFiles(*dir, *recursive)
	if err != nil {
		die(err.Error())
	}
	if len(matches) == 0 {
		die(fmt.Sprintf("no .tf files in %s", *dir))
	}
	// Collect each project's environment keys (in source order) up front so we
	// can resolve positional `environments[N]` references to their map keys in
	// migration warnings. Done before conversion, while configs are still v2.
	projectEnvKeys := map[string][]string{}
	if *direction == "v2-to-v3" {
		projectEnvKeys = collectProjectEnvKeys(matches)
	}
	for _, f := range matches {
		if err := process(f, *direction, spec, *dryRun, projectEnvKeys); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", f, err)
			os.Exit(1)
		}
	}
	if warningCount > 0 {
		fmt.Fprintf(os.Stderr, "%d warning(s) emitted: the flagged attributes need manual conversion\n", warningCount)
	}
	if synthesizedBoolVars > 0 {
		fmt.Fprintf(os.Stderr, "note: synthesized default true/false variations for %d boolean flag(s). "+
			"Provider v3 preserves any variation name/description set outside Terraform.\n", synthesizedBoolVars)
	}
}

// collectTFFiles returns the .tf files under dir. Non-recursive mode matches the historical
// single-directory glob. Recursive mode walks subdirectories so local modules are converted in the
// same pass, skipping .terraform (provider/module cache — not user-owned code) and .git.
func collectTFFiles(dir string, recursive bool) ([]string, error) {
	if !recursive {
		return filepath.Glob(filepath.Join(dir, "*.tf"))
	}
	var matches []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".terraform", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

// warningCount tracks non-fatal conversion warnings (e.g. dynamic blocks) so main can summarize.
var warningCount int

// synthesizedBoolVars counts boolean feature flags where variations were auto-added, so main can print
// a single transparency note. The converter writes value-only variations; provider v3 preserves any
// variation name/description set outside Terraform when the config omits them, so no follow-up is needed.
var synthesizedBoolVars int

func warnf(format string, args ...interface{}) {
	warningCount++
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// envIndexRefRe matches positional references to a project's environments,
// e.g. launchdarkly_project.<label>.environments[0] or [*].
var envIndexRefRe = regexp.MustCompile(`launchdarkly_project\.([A-Za-z0-9_-]+)\.environments\[\s*([0-9]+|\*)\s*\]`)

// collectProjectEnvKeys parses the given files and returns, per
// launchdarkly_project resource label, the environment keys in source order.
// Used to resolve positional environment references to their map keys in
// migration warnings. Reads the v2 (pre-conversion) block form.
func collectProjectEnvKeys(files []string) map[string][]string {
	out := map[string][]string{}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, diag := hclwrite.ParseConfig(b, path, hcl.Pos{Line: 1, Column: 1})
		if diag.HasErrors() {
			continue
		}
		for _, blk := range f.Body().Blocks() {
			if blk.Type() != "resource" || len(blk.Labels()) != 2 || blk.Labels()[0] != "launchdarkly_project" {
				continue
			}
			var keys []string
			for _, env := range blk.Body().Blocks() {
				if env.Type() != "environments" {
					continue
				}
				keyAttr := env.Body().GetAttribute("key")
				if keyAttr == nil {
					continue
				}
				if k, ok := stringLiteralValue(keyAttr.Expr().BuildTokens(nil)); ok && len(k) == 3 {
					keys = append(keys, string(k[1].Bytes))
				}
			}
			if len(keys) > 0 {
				out[blk.Labels()[1]] = keys
			}
		}
	}
	return out
}

// warnEnvIndexRefs warns on every positional `environments[N]` (or `[*]`)
// reference, since environments is a map in v3 and must be addressed by key.
// It resolves N to the env key when known. Detection-only: the tool never
// edits these references (auto-rewriting arbitrary expressions risks silently
// corrupting configs; the manual fix is one token).
func warnEnvIndexRefs(path string, src []byte, projectEnvKeys map[string][]string) {
	for i, line := range strings.Split(string(src), "\n") {
		for _, m := range envIndexRefRe.FindAllStringSubmatch(line, -1) {
			label, idx := m[1], m[2]
			loc := fmt.Sprintf("%s:%d", path, i+1)
			if idx == "*" {
				warnf("%s: `launchdarkly_project.%s.environments[*]` is a list splat, but environments is now a map — use `values(launchdarkly_project.%s.environments)[*]...` (or `keys(...)`)", loc, label, label)
				continue
			}
			n, _ := strconv.Atoi(idx)
			if keys := projectEnvKeys[label]; n < len(keys) {
				warnf("%s: `launchdarkly_project.%s.environments[%s]` must become `launchdarkly_project.%s.environments[%q]` — environments is now a map keyed by env key", loc, label, idx, label, keys[n])
			} else {
				warnf("%s: `launchdarkly_project.%s.environments[%s]` uses a list index, but environments is now a map — replace `[%s]` with `[\"<env_key>\"]`", loc, label, idx, idx)
			}
		}
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}

func process(path, direction string, spec Spec, dryRun bool, projectEnvKeys map[string][]string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Detection-only: flag positional environments[N] references for manual
	// fix-up (the tool converts the block to a map but never edits references).
	if direction == "v2-to-v3" {
		warnEnvIndexRefs(path, src, projectEnvKeys)
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
		where := fmt.Sprintf("%s: resource %q", path, strings.Join(labels, "."))
		var did bool
		if direction == "v2-to-v3" {
			if forward(blk.Body(), rspec.Blocks, where) {
				did = true
			}
			if applyDeprecations(blk.Body(), rspec.Deprecations, where) {
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
			idxSuffix := ""
			if rw.StripIndex {
				// Also consume a single trailing list index so a v2 list access
				// (`...from[0].z` / `...from.0.z`) collapses to the v3 object
				// access (`...to.z`). The index is outside any capture group and
				// absent from the replacement, so it is dropped.
				idxSuffix = `(?:\[0\]|\.0)?`
			}
			re := regexp.MustCompile(`\bdata\.` + regexp.QuoteMeta(label) + `\.([A-Za-z_][A-Za-z0-9_]*)\.` + regexp.QuoteMeta(rw.From) + `\b` + idxSuffix)
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
//
// `dynamic "name" { ... }` generator blocks cannot be converted mechanically — they need a for
// expression (`name = [for x in ... : { ... }]`) that only the author can write. When one is found
// for a mapped name, the whole attribute is skipped (converting only the static siblings would
// leave an attribute and a dynamic block for the same name, which v3 rejects) and a warning points
// at the spot. `where` carries the file and resource address for that warning.
func forward(body *hclwrite.Body, specs []*AttrSpec, where string) bool {
	changed := false
	for _, s := range specs {
		var matched []*hclwrite.Block
		dynamic := false
		for _, b := range body.Blocks() {
			if b.Type() == s.Name {
				matched = append(matched, b)
			}
			if b.Type() == "dynamic" && len(b.Labels()) > 0 && b.Labels()[0] == s.Name {
				dynamic = true
			}
		}
		if dynamic {
			warnf("%s: dynamic %q block cannot be converted automatically; rewrite it by hand as %s = [for ... : { ... }]", where, s.Name, s.Name)
			continue
		}
		if len(matched) == 0 {
			continue
		}
		if len(s.Nested) > 0 {
			for _, b := range matched {
				forward(b.Body(), s.Nested, where)
			}
		}
		var tokens hclwrite.Tokens
		if s.Object {
			// Single-object attribute (v3 SingleNestedAttribute): emit `name = { ... }`.
			// A valid v2 config has exactly one block (the SDKv2 schema was MaxItems:1);
			// if more are present we take the first and warn rather than emit invalid HCL.
			if len(matched) > 1 {
				warnf("%s: %q is a single-object attribute but %d blocks were found; using the first", where, s.Name, len(matched))
			}
			tokens = hclwrite.Tokens{
				{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
				{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			}
			tokens = append(tokens, trimLeadingNewlines(matched[0].Body().BuildTokens(nil))...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
		} else if s.MapKey != "" {
			// Map attribute keyed by an inner field (v3 MapNestedAttribute):
			// emit `name = { <keyval> = { ...rest } }`, hoisting each block's
			// MapKey attribute to the map key and dropping it from the object.
			//
			// The `key` attribute stays inside each object (it's Optional+
			// Computed in v3 and equals the map key); we only read it to build
			// the map key. Validate EVERY block's key first so a missing or
			// non-literal key aborts the whole attribute with the file
			// untouched (no blocks are mutated either way).
			keyExprs := make([]hclwrite.Tokens, 0, len(matched))
			seenKeys := make(map[string]bool, len(matched))
			skip := false
			for _, b := range matched {
				keyAttr := b.Body().GetAttribute(s.MapKey)
				if keyAttr == nil {
					warnf("%s: %q block is missing the %q attribute needed to key the v3 map; convert by hand", where, s.Name, s.MapKey)
					skip = true
					break
				}
				keyExpr, ok := stringLiteralValue(keyAttr.Expr().BuildTokens(nil))
				if !ok {
					warnf("%s: %q block's %q is not a literal string; cannot key the v3 map automatically — convert by hand", where, s.Name, s.MapKey)
					skip = true
					break
				}
				// Duplicate keys would collapse into one map entry (last wins),
				// silently dropping a block. Abort and leave it for the author.
				if lit := string(keyExpr[1].Bytes); seenKeys[lit] {
					warnf("%s: %q has duplicate %s %q across blocks; a map cannot hold duplicate keys — convert by hand", where, s.Name, s.MapKey, lit)
					skip = true
					break
				} else {
					seenKeys[lit] = true
				}
				keyExprs = append(keyExprs, keyExpr)
			}
			if skip {
				continue
			}
			mapTokens := hclwrite.Tokens{
				{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
				{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			}
			for i, b := range matched {
				mapTokens = append(mapTokens, keyExprs[i]...)
				mapTokens = append(mapTokens,
					&hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
					&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
					&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
				)
				mapTokens = append(mapTokens, trimLeadingNewlines(b.Body().BuildTokens(nil))...)
				mapTokens = append(mapTokens,
					&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
					&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
				)
			}
			mapTokens = append(mapTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
			tokens = mapTokens
		} else {
			tokens = hclwrite.Tokens{
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
		}
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
		if s.Object {
			// v3 single object `name = { ... }` → v2 block `name { ... }`.
			elem := extractObjectBody(attr.Expr().BuildTokens(nil))
			if len(elem) == 0 {
				continue
			}
			body.RemoveAttribute(s.Name)
			newBlock := body.AppendNewBlock(s.Name, nil)
			newBlock.Body().AppendUnstructuredTokens(ensureTrailingNewline(trimLeadingNewlines(elem)))
			changed = true
			continue
		}
		if s.MapKey != "" {
			// v3 map `name = { <key> = { key = <key> ... } }` → repeated v2
			// blocks `name { key = <key> ... }`. The `key` attribute is kept
			// inside the object by the forward pass, so it is already present;
			// only a hand-written v3 map that omitted it needs it re-injected
			// from the map key.
			entries := extractMapEntries(attr.Expr().BuildTokens(nil))
			if len(entries) == 0 {
				continue
			}
			body.RemoveAttribute(s.Name)
			for _, e := range entries {
				bodyTokens := e.body
				wrapped := []byte(fmt.Sprintf("dummy {\n%s\n}\n", tokensString(ensureTrailingNewline(trimLeadingNewlines(e.body)))))
				tmp, diag := hclwrite.ParseConfig(wrapped, "<elem>", hcl.Pos{Line: 1, Column: 1})
				if !diag.HasErrors() && len(tmp.Body().Blocks()) > 0 {
					eb := tmp.Body().Blocks()[0].Body()
					if len(s.Nested) > 0 {
						reverse(eb, s.Nested)
					}
					if eb.GetAttribute(s.MapKey) == nil {
						eb.SetAttributeRaw(s.MapKey, e.key)
					}
					bodyTokens = eb.BuildTokens(nil)
				}
				newBlock := body.AppendNewBlock(s.Name, nil)
				newBlock.Body().AppendUnstructuredTokens(ensureTrailingNewline(trimLeadingNewlines(bodyTokens)))
				changed = true
			}
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
func applyDeprecations(body *hclwrite.Body, deps []*DeprecationSpec, where string) bool {
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
		case "ensure_boolean_variations":
			if ensureBooleanVariations(body, where) {
				changed = true
			}
		default:
			fmt.Fprintf(os.Stderr, "warning: unknown deprecation action %q for attribute %q (skipping)\n", d.Action, d.Name)
		}
	}
	return changed
}

// ensureBooleanVariations implements the ensure_boolean_variations action. v2 let a
// launchdarkly_feature_flag with variation_type = "boolean" omit the variations block; v3 requires
// variations for every flag. A LaunchDarkly boolean flag's variations are an invariant — exactly two,
// [{ value = "true" }, { value = "false" }] in that order — so the synthesized value is deterministic
// and reconstructs exactly what v2 created implicitly.
//
// It fires only when variations is absent AND variation_type is the literal string "boolean". If
// variations is already present (a v2 config that named its boolean variations) it is left untouched.
// If variation_type is a non-literal expression (a var/local) the value cannot be resolved statically,
// so it warns and leaves the flag for the author — the same warn-when-in-doubt policy as dynamic blocks.
func ensureBooleanVariations(body *hclwrite.Body, where string) bool {
	if body.GetAttribute("variations") != nil {
		return false
	}
	vt := body.GetAttribute("variation_type")
	if vt == nil {
		return false
	}
	if strings.TrimSpace(tokensString(vt.Expr().BuildTokens(nil))) != `"boolean"` {
		warnf("%s: feature flag is missing the required \"variations\" attribute and variation_type is not a literal \"boolean\"; v3 requires variations for every flag — add them by hand", where)
		return false
	}
	body.SetAttributeRaw("variations", booleanVariationTokens())
	synthesizedBoolVars++
	return true
}

// booleanVariationTokens builds the token stream for `[{ value = "true" }, { value = "false" }]`.
// hclwrite.Format (run over the whole file in process) normalizes the indentation afterward.
func booleanVariationTokens() hclwrite.Tokens {
	str := func(s string) hclwrite.Tokens {
		return hclwrite.Tokens{
			{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
			{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(s)},
			{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
		}
	}
	elem := func(v string) hclwrite.Tokens {
		t := hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			{Type: hclsyntax.TokenIdent, Bytes: []byte("value")},
			{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
		}
		t = append(t, str(v)...)
		return append(t,
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
		)
	}
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}
	tokens = append(tokens, elem("true")...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")},
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	)
	tokens = append(tokens, elem("false")...)
	return append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
	)
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
	// Build the replacement single-object attribute tokens (v3 models
	// client_side_availability / default_client_side_availability as
	// SingleNestedAttribute — see REL-14237): to = {
	//   using_environment_id = <iis-expr>
	//   using_mobile_key     = false
	// }
	tokens := hclwrite.Tokens{
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

// stringLiteralValue reports whether tokens form a single quoted-string literal
// (`"..."`) and, if so, returns the clean quote/lit/quote token slice (with any
// leading indentation stripped) suitable for use as a v3 map key. Non-literal
// expressions (idents, function calls, interpolations) return false.
func stringLiteralValue(tokens hclwrite.Tokens) (hclwrite.Tokens, bool) {
	var nonWs hclwrite.Tokens
	for _, t := range tokens {
		if t.Type == hclsyntax.TokenNewline {
			continue
		}
		nonWs = append(nonWs, t)
	}
	if len(nonWs) == 3 &&
		nonWs[0].Type == hclsyntax.TokenOQuote &&
		nonWs[1].Type == hclsyntax.TokenQuotedLit &&
		nonWs[2].Type == hclsyntax.TokenCQuote {
		clean := hclwrite.Tokens{
			{Type: hclsyntax.TokenOQuote, Bytes: nonWs[0].Bytes},
			{Type: hclsyntax.TokenQuotedLit, Bytes: nonWs[1].Bytes},
			{Type: hclsyntax.TokenCQuote, Bytes: nonWs[2].Bytes},
		}
		return clean, true
	}
	return nil, false
}

// mapEntry is one `<key> = { ... }` pair of a v3 map attribute.
type mapEntry struct {
	key  hclwrite.Tokens
	body hclwrite.Tokens
}

// extractMapEntries walks token stream `{ <key> = {...}, <key> = {...}, ... }`
// and returns each entry's key tokens and the inner body tokens of its `{...}`
// value (excluding the value's surrounding braces). Used by the reverse
// direction for map-nested attributes (MapKey specs).
func extractMapEntries(tokens hclwrite.Tokens) []mapEntry {
	i := 0
	for ; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrace {
			i++
			break
		}
	}
	var entries []mapEntry
	for i < len(tokens) {
		for i < len(tokens) && (tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComma) {
			i++
		}
		if i >= len(tokens) || tokens[i].Type == hclsyntax.TokenCBrace {
			break
		}
		var key hclwrite.Tokens
		for i < len(tokens) && tokens[i].Type != hclsyntax.TokenEqual {
			if tokens[i].Type == hclsyntax.TokenOBrace || tokens[i].Type == hclsyntax.TokenCBrace {
				break
			}
			if tokens[i].Type != hclsyntax.TokenNewline {
				key = append(key, &hclwrite.Token{Type: tokens[i].Type, Bytes: tokens[i].Bytes})
			}
			i++
		}
		if i >= len(tokens) || tokens[i].Type != hclsyntax.TokenEqual {
			break
		}
		i++ // consume '='
		for i < len(tokens) && tokens[i].Type == hclsyntax.TokenNewline {
			i++
		}
		if i >= len(tokens) || tokens[i].Type != hclsyntax.TokenOBrace {
			break
		}
		brace := 1
		i++
		var bodyToks hclwrite.Tokens
		for i < len(tokens) && brace > 0 {
			switch tokens[i].Type {
			case hclsyntax.TokenOBrace:
				brace++
			case hclsyntax.TokenCBrace:
				brace--
				if brace == 0 {
					i++
					goto done
				}
			}
			bodyToks = append(bodyToks, tokens[i])
			i++
		}
	done:
		entries = append(entries, mapEntry{key: key, body: bodyToks})
	}
	return entries
}

// extractObjectBody returns the inner tokens of a single object expression `{ ... }` (excluding the
// outer braces). Used by the reverse direction for single-object (SingleNestedAttribute) attributes,
// which serialize as `name = { ... }` rather than the `name = [{ ... }]` tuple shape.
func extractObjectBody(tokens hclwrite.Tokens) hclwrite.Tokens {
	i := 0
	for ; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenOBrace {
			i++
			break
		}
	}
	brace := 1
	var elem hclwrite.Tokens
	for i < len(tokens) && brace > 0 {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			brace++
		case hclsyntax.TokenCBrace:
			brace--
			if brace == 0 {
				return elem
			}
		}
		elem = append(elem, tokens[i])
		i++
	}
	return elem
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
