// Package main implements the API-coverage drift report described in
// .claude/plans/AUTOGEN_PIPELINE.md (stage 1).
//
// It diffs the endpoint families (OpenAPI tags) of LaunchDarkly's public
// OpenAPI spec against the resources and data sources registered in the
// provider, using a curated mapping file as the source of truth for which
// families are covered, intentionally ignored, or pending triage.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
	"gopkg.in/yaml.v3"
)

const providerTypeName = "launchdarkly"

// untaggedFamily is the synthetic family name for operations the spec leaves
// untagged. specOperations and familySlice must agree on this sentinel so that
// `driftreport -family "<untagged>"` returns the same paths the report lists.
const untaggedFamily = "<untagged>"

var specMethods = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true, "delete": true,
}

// Family statuses accepted in mapping.yaml.
const (
	statusCovered = "covered"
	statusPartial = "partial"
	statusIgnored = "ignored"
	statusTriage  = "triage"
)

var validStatuses = map[string]bool{
	statusCovered: true, statusPartial: true, statusIgnored: true, statusTriage: true,
}

type Mapping struct {
	Version  int             `yaml:"version"`
	Families []MappingFamily `yaml:"families"`
}

type MappingFamily struct {
	Tag                   string              `yaml:"tag"`
	Status                string              `yaml:"status"`
	Resources             []ResourceEntry     `yaml:"resources,omitempty"`
	NewResourceCandidates []ResourceCandidate `yaml:"new_resource_candidates,omitempty"`
	IgnoredOperations     []IgnoredOperation  `yaml:"ignored_operations,omitempty"`
	TriageOperations      []string            `yaml:"triage_operations,omitempty"`
	Reason                string              `yaml:"reason,omitempty"`
	Notes                 string              `yaml:"notes,omitempty"`
}

// ResourceCandidate is a curated NET-NEW resource the provider does not yet
// model but should, covering a cluster of a partial family's operations.
// Unlike triage_operations (decision still pending), a candidate is decided:
// it is ready for the stage-2 scaffolder to implement as a brand-new resource.
// That is safe — scaffolding a new resource only ADDS a type and never touches
// the family's existing resources, so there is no state-compatibility risk
// (the same reason a wholly new family is scaffoldable). Candidate operations
// count as claimed, so they do not surface as unclaimed drift; the report lists
// them under scaffoldable_resources. Once implemented, move the operations to a
// real resources entry and drop the candidate.
type ResourceCandidate struct {
	Name       string   `yaml:"name"`
	Operations []string `yaml:"operations"`
}

// ResourceEntry is one resource claim under a family. A bare YAML string is a
// tag-level claim; the object shape additionally claims the specific
// operationIds the resource implements (required for partial families).
type ResourceEntry struct {
	Name       string   `yaml:"name"`
	Operations []string `yaml:"operations,omitempty"`
}

func (r *ResourceEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return value.Decode(&r.Name)
	}
	// Alias the type to avoid recursing into this method.
	type plain ResourceEntry
	var p plain
	if err := value.Decode(&p); err != nil {
		return err
	}
	*r = ResourceEntry(p)
	return nil
}

// IgnoredOperation marks a spec operation in a partial family as deliberately
// unmodeled (bulk/UI-only/runtime endpoints). Mirrors the family-level
// ignored-needs-reason rule.
//
// triage_operations is the op-level analogue of a triage family: acknowledged,
// coverage decision pending. Triage ops are listed in the report but don't
// fail the run, and need no reason — the pending decision is the point.
type IgnoredOperation struct {
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
}

func (m *Mapping) validate() error {
	seen := map[string]bool{}
	for _, f := range m.Families {
		if f.Tag == "" {
			return fmt.Errorf("mapping entry with empty tag")
		}
		if seen[f.Tag] {
			return fmt.Errorf("duplicate mapping entry for tag %q", f.Tag)
		}
		seen[f.Tag] = true
		if !validStatuses[f.Status] {
			return fmt.Errorf("tag %q: invalid status %q", f.Tag, f.Status)
		}
		if f.Status == statusIgnored && f.Reason == "" {
			return fmt.Errorf("tag %q: ignored entries require a reason", f.Tag)
		}
		if (f.Status == statusCovered || f.Status == statusPartial) && len(f.Resources) == 0 {
			return fmt.Errorf("tag %q: %s entries must list at least one resource", f.Tag, f.Status)
		}
		if err := f.validateOperations(); err != nil {
			return err
		}
	}
	return nil
}

// validateOperations enforces the v2 operation-level rules: partial families
// require operation lists (that is what distinguishes them from covered);
// non-partial families must not carry them (op lists on a covered family
// would silently never be checked).
func (f *MappingFamily) validateOperations() error {
	if f.Status != statusPartial {
		for _, r := range f.Resources {
			if len(r.Operations) > 0 {
				return fmt.Errorf("tag %q: resource %q lists operations but family status is %q (only partial families carry operation lists)", f.Tag, r.Name, f.Status)
			}
		}
		if len(f.IgnoredOperations) > 0 {
			return fmt.Errorf("tag %q: ignored_operations set but family status is %q (only partial families carry operation lists)", f.Tag, f.Status)
		}
		if len(f.TriageOperations) > 0 {
			return fmt.Errorf("tag %q: triage_operations set but family status is %q (only partial families carry operation lists)", f.Tag, f.Status)
		}
		if len(f.NewResourceCandidates) > 0 {
			return fmt.Errorf("tag %q: new_resource_candidates set but family status is %q (only partial families carry operation lists)", f.Tag, f.Status)
		}
		return nil
	}
	claimed := map[string]bool{}
	for _, r := range f.Resources {
		if r.Name == "" {
			return fmt.Errorf("tag %q: resource entry with empty name", f.Tag)
		}
		if len(r.Operations) == 0 {
			return fmt.Errorf("tag %q: partial families require an operations list on every resource (resource %q has none)", f.Tag, r.Name)
		}
		for _, op := range r.Operations {
			if op == "" {
				return fmt.Errorf("tag %q: resource %q lists an empty operationId", f.Tag, r.Name)
			}
			if claimed[op] {
				return fmt.Errorf("tag %q: operation %q claimed more than once", f.Tag, op)
			}
			claimed[op] = true
		}
	}
	for _, ig := range f.IgnoredOperations {
		if ig.ID == "" || ig.Reason == "" {
			return fmt.Errorf("tag %q: ignored_operations entries require both id and reason", f.Tag)
		}
		if claimed[ig.ID] {
			return fmt.Errorf("tag %q: operation %q claimed more than once", f.Tag, ig.ID)
		}
		claimed[ig.ID] = true
	}
	for _, op := range f.TriageOperations {
		if op == "" {
			return fmt.Errorf("tag %q: triage_operations lists an empty operationId", f.Tag)
		}
		if claimed[op] {
			return fmt.Errorf("tag %q: operation %q claimed more than once", f.Tag, op)
		}
		claimed[op] = true
	}
	implemented := map[string]bool{}
	for _, r := range f.Resources {
		implemented[r.Name] = true
	}
	for _, c := range f.NewResourceCandidates {
		if c.Name == "" {
			return fmt.Errorf("tag %q: new_resource_candidates entry with empty name", f.Tag)
		}
		if implemented[c.Name] {
			return fmt.Errorf("tag %q: new_resource_candidate %q is already an implemented resource (move its operations to that resource's entry instead)", f.Tag, c.Name)
		}
		if len(c.Operations) == 0 {
			return fmt.Errorf("tag %q: new_resource_candidate %q must list at least one operation", f.Tag, c.Name)
		}
		for _, op := range c.Operations {
			if op == "" {
				return fmt.Errorf("tag %q: new_resource_candidate %q lists an empty operationId", f.Tag, c.Name)
			}
			if claimed[op] {
				return fmt.Errorf("tag %q: operation %q claimed more than once", f.Tag, op)
			}
			claimed[op] = true
		}
	}
	return nil
}

func loadMapping(path string) (*Mapping, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading mapping file: %w", err)
	}
	var m Mapping
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing mapping file: %w", err)
	}
	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("invalid mapping file: %w", err)
	}
	return &m, nil
}

// SpecOperation is one operation (method + path) in the spec, keyed for
// mapping claims by its operationId.
type SpecOperation struct {
	Path        string
	Method      string
	OperationID string
}

// key identifies the operation for mapping claims. The LD spec assigns an
// operationId to every operation; "METHOD path" is a defensive fallback so a
// spec hygiene slip surfaces as an unclaimed op instead of a silent skip.
func (o SpecOperation) key() string {
	if o.OperationID != "" {
		return o.OperationID
	}
	return o.Method + " " + o.Path
}

// specOperations extracts tag -> operations from raw OpenAPI JSON, sorted by
// path then method.
func specOperations(rawSpec []byte) (map[string][]SpecOperation, error) {
	// Path items mix operation objects with non-operation keys of other JSON
	// types (e.g. "parameters" is an array), so decode in two stages.
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}
	if len(spec.Paths) == 0 {
		return nil, fmt.Errorf("OpenAPI spec contains no paths")
	}
	families := map[string][]SpecOperation{}
	for path, ops := range spec.Paths {
		for method, rawOp := range ops {
			if !specMethods[strings.ToLower(method)] {
				continue
			}
			var op struct {
				Tags        []string `json:"tags"`
				OperationID string   `json:"operationId"`
			}
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return nil, fmt.Errorf("parsing operation %s %s: %w", method, path, err)
			}
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{untaggedFamily}
			}
			for _, tag := range tags {
				families[tag] = append(families[tag], SpecOperation{
					Path:        path,
					Method:      strings.ToUpper(method),
					OperationID: op.OperationID,
				})
			}
		}
	}
	for _, ops := range families {
		sort.Slice(ops, func(i, j int) bool {
			if ops[i].Path != ops[j].Path {
				return ops[i].Path < ops[j].Path
			}
			return ops[i].Method < ops[j].Method
		})
	}
	return families, nil
}

// uniquePaths flattens a family's operations to its sorted unique paths.
func uniquePaths(ops []SpecOperation) []string {
	seen := map[string]bool{}
	var out []string
	for _, op := range ops {
		if !seen[op.Path] {
			seen[op.Path] = true
			out = append(out, op.Path)
		}
	}
	sort.Strings(out)
	return out
}

// familySlice extracts a compact JSON description of one endpoint family —
// paths, methods, operationIds, and summaries — for use as scaffolding-agent
// context (stage 2 of the autogen pipeline). Schema details are deliberately
// omitted; the agent reads the full spec for those.
func familySlice(rawSpec []byte, tag string) ([]byte, error) {
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	type operation struct {
		Method      string `json:"method"`
		OperationID string `json:"operationId,omitempty"`
		Summary     string `json:"summary,omitempty"`
	}
	slice := struct {
		Tag   string                 `json:"tag"`
		Paths map[string][]operation `json:"paths"`
	}{Tag: tag, Paths: map[string][]operation{}}

	for path, ops := range spec.Paths {
		for method, rawOp := range ops {
			if !specMethods[strings.ToLower(method)] {
				continue
			}
			var op struct {
				Tags        []string `json:"tags"`
				OperationID string   `json:"operationId"`
				Summary     string   `json:"summary"`
			}
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return nil, fmt.Errorf("parsing operation %s %s: %w", method, path, err)
			}
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{untaggedFamily}
			}
			for _, t := range tags {
				if t == tag {
					slice.Paths[path] = append(slice.Paths[path], operation{
						Method:      strings.ToUpper(method),
						OperationID: op.OperationID,
						Summary:     op.Summary,
					})
				}
			}
		}
	}
	if len(slice.Paths) == 0 {
		return nil, fmt.Errorf("no endpoints found for family %q", tag)
	}
	for _, ops := range slice.Paths {
		sort.Slice(ops, func(i, j int) bool { return ops[i].Method < ops[j].Method })
	}
	return json.MarshalIndent(slice, "", "  ")
}

func fetchSpec(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return os.ReadFile(source)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("fetching spec: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching spec: unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// registeredTypes enumerates resource and data source type names from the
// provider's own registration lists — authoritative, cannot drift from code.
func registeredTypes() (resources, dataSources []string) {
	ctx := context.Background()
	p := launchdarkly.NewPluginProvider("driftreport")()
	for _, newRes := range p.Resources(ctx) {
		var resp resource.MetadataResponse
		newRes().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerTypeName}, &resp)
		resources = append(resources, resp.TypeName)
	}
	for _, newDS := range p.DataSources(ctx) {
		var resp datasource.MetadataResponse
		newDS().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerTypeName}, &resp)
		dataSources = append(dataSources, resp.TypeName)
	}
	sort.Strings(resources)
	sort.Strings(dataSources)
	return resources, dataSources
}

type Report struct {
	GeneratedAt time.Time `json:"generated_at"`
	SpecSource  string    `json:"spec_source"`

	// Drift signals (any non-empty => exit code 2).
	NewFamilies          []FamilyDetail      `json:"new_families"`
	StaleFamilies        []string            `json:"stale_families"`
	UnmappedResources    []string            `json:"unmapped_resources"`
	UnclaimedOperations  []OperationDetail   `json:"unclaimed_operations"`
	StaleOperations      []OperationDetail   `json:"stale_operations"`
	RegisteredCandidates []CandidateConflict `json:"registered_candidates"`

	// Informational.
	TriageFamilies        []string               `json:"triage_families"`
	TriageOperations      []OperationDetail      `json:"triage_operations"`
	ScaffoldableResources []ScaffoldableResource `json:"scaffoldable_resources"`
	StatusCounts          map[string]int         `json:"status_counts"`
	TotalFamilies         int                    `json:"total_families"`
}

// CandidateConflict is a new_resource_candidate whose name is ALREADY a
// registered provider type — a curation mistake (a candidate must be net-new).
// It is drift: the operations belong on a real resources entry, not a candidate,
// and it must never be emitted as scaffoldable (that would re-scaffold an
// existing resource).
type CandidateConflict struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
}

// ScaffoldableResource is a curated net-new resource (from a partial family's
// new_resource_candidates) that is ready for the stage-2 scaffolder. It is
// informational: it never fails the run — it is a backlog signal a human
// dispatches. Operations carries only the spec operations that still exist;
// any candidate operation missing from the spec surfaces under StaleOperations.
type ScaffoldableResource struct {
	Tag        string            `json:"tag"`
	Name       string            `json:"name"`
	Operations []OperationDetail `json:"operations"`
}

type FamilyDetail struct {
	Tag   string   `json:"tag"`
	Paths []string `json:"paths"`
}

// OperationDetail is one operation-level drift item in a partial family.
// Stale entries (operationId in the mapping but gone from the spec) carry
// only Tag and OperationID.
type OperationDetail struct {
	Tag         string `json:"tag"`
	OperationID string `json:"operation_id"`
	Method      string `json:"method,omitempty"`
	Path        string `json:"path,omitempty"`
}

func (r *Report) HasDrift() bool {
	return len(r.NewFamilies) > 0 || len(r.StaleFamilies) > 0 || len(r.UnmappedResources) > 0 ||
		len(r.UnclaimedOperations) > 0 || len(r.StaleOperations) > 0 || len(r.RegisteredCandidates) > 0
}

func buildReport(families map[string][]SpecOperation, mapping *Mapping, resources, dataSources []string, specSource string) *Report {
	report := &Report{
		GeneratedAt:   time.Now().UTC(),
		SpecSource:    specSource,
		StatusCounts:  map[string]int{},
		TotalFamilies: len(families),
		// Initialized so empty lists serialize as [] rather than null in JSON.
		NewFamilies:           []FamilyDetail{},
		StaleFamilies:         []string{},
		UnmappedResources:     []string{},
		UnclaimedOperations:   []OperationDetail{},
		StaleOperations:       []OperationDetail{},
		TriageFamilies:        []string{},
		TriageOperations:      []OperationDetail{},
		ScaffoldableResources: []ScaffoldableResource{},
		RegisteredCandidates:  []CandidateConflict{},
	}

	mapped := map[string]MappingFamily{}
	for _, f := range mapping.Families {
		mapped[f.Tag] = f
	}

	// Registered provider types — a new_resource_candidate naming one is a
	// curation mistake (candidates must be net-new), so it is reported as drift
	// rather than as scaffoldable.
	registered := map[string]bool{}
	for _, t := range append(append([]string{}, resources...), dataSources...) {
		registered[t] = true
	}

	// Spec tags absent from the mapping => new families.
	for tag, ops := range families {
		if _, ok := mapped[tag]; !ok {
			report.NewFamilies = append(report.NewFamilies, FamilyDetail{Tag: tag, Paths: uniquePaths(ops)})
		}
	}
	sort.Slice(report.NewFamilies, func(i, j int) bool {
		return report.NewFamilies[i].Tag < report.NewFamilies[j].Tag
	})

	// Mapping entries whose tag vanished from the spec => stale (likely renamed).
	for _, f := range mapping.Families {
		report.StatusCounts[f.Status]++
		if _, ok := families[f.Tag]; !ok {
			report.StaleFamilies = append(report.StaleFamilies, f.Tag)
		}
		if f.Status == statusTriage {
			report.TriageFamilies = append(report.TriageFamilies, f.Tag)
		}
	}
	sort.Strings(report.StaleFamilies)
	sort.Strings(report.TriageFamilies)

	// Operation-level diff for partial families: every spec operation must be
	// claimed by a resource or deliberately ignored, and every claim must
	// still exist in the spec. Skipped when the family tag itself is stale —
	// the stale-family signal already covers it.
	for _, f := range mapping.Families {
		if f.Status != statusPartial {
			continue
		}
		specOps, tagInSpec := families[f.Tag]
		if !tagInSpec {
			continue
		}
		claimed := map[string]bool{}
		for _, r := range f.Resources {
			for _, op := range r.Operations {
				claimed[op] = true
			}
		}
		for _, ig := range f.IgnoredOperations {
			claimed[ig.ID] = true
		}
		triage := map[string]bool{}
		for _, op := range f.TriageOperations {
			claimed[op] = true
			triage[op] = true
		}
		// Curated net-new-resource operations count as claimed so they do not
		// surface as unclaimed drift; they are emitted under scaffoldable_resources.
		for _, c := range f.NewResourceCandidates {
			for _, op := range c.Operations {
				claimed[op] = true
			}
		}
		specByKey := map[string]SpecOperation{}
		inSpec := map[string]bool{}
		for _, op := range specOps {
			specByKey[op.key()] = op
			inSpec[op.key()] = true
			detail := OperationDetail{
				Tag:         f.Tag,
				OperationID: op.key(),
				Method:      op.Method,
				Path:        op.Path,
			}
			switch {
			case triage[op.key()]:
				report.TriageOperations = append(report.TriageOperations, detail)
			case !claimed[op.key()]:
				report.UnclaimedOperations = append(report.UnclaimedOperations, detail)
			}
		}
		// Emit curated net-new resources as scaffoldable. Only spec-present ops
		// are listed; a candidate op missing from the spec is caught as a stale
		// claim by the loop below (candidate ops are in `claimed`).
		for _, c := range f.NewResourceCandidates {
			// A candidate naming an already-registered type is a curation
			// mistake: the resource exists, so it is not net-new and must not be
			// advertised as scaffoldable. Surface it as drift instead.
			if registered[c.Name] {
				report.RegisteredCandidates = append(report.RegisteredCandidates, CandidateConflict{Tag: f.Tag, Name: c.Name})
				continue
			}
			sr := ScaffoldableResource{Tag: f.Tag, Name: c.Name, Operations: []OperationDetail{}}
			for _, opID := range c.Operations {
				if so, ok := specByKey[opID]; ok {
					sr.Operations = append(sr.Operations, OperationDetail{Tag: f.Tag, OperationID: opID, Method: so.Method, Path: so.Path})
				}
			}
			sortOperationDetails(sr.Operations)
			report.ScaffoldableResources = append(report.ScaffoldableResources, sr)
		}
		for op := range claimed {
			if !inSpec[op] {
				report.StaleOperations = append(report.StaleOperations, OperationDetail{Tag: f.Tag, OperationID: op})
			}
		}
	}
	sortOperationDetails(report.UnclaimedOperations)
	sortOperationDetails(report.TriageOperations)
	sort.Slice(report.StaleOperations, func(i, j int) bool {
		a, b := report.StaleOperations[i], report.StaleOperations[j]
		if a.Tag != b.Tag {
			return a.Tag < b.Tag
		}
		return a.OperationID < b.OperationID
	})
	sort.Slice(report.ScaffoldableResources, func(i, j int) bool {
		a, b := report.ScaffoldableResources[i], report.ScaffoldableResources[j]
		if a.Tag != b.Tag {
			return a.Tag < b.Tag
		}
		return a.Name < b.Name
	})
	sort.Slice(report.RegisteredCandidates, func(i, j int) bool {
		a, b := report.RegisteredCandidates[i], report.RegisteredCandidates[j]
		if a.Tag != b.Tag {
			return a.Tag < b.Tag
		}
		return a.Name < b.Name
	})

	// Registered types never referenced by any family => mapping is incomplete.
	referenced := map[string]bool{}
	for _, f := range mapping.Families {
		for _, r := range f.Resources {
			referenced[r.Name] = true
		}
	}
	seen := map[string]bool{}
	for _, t := range append(append([]string{}, resources...), dataSources...) {
		if !referenced[t] && !seen[t] {
			report.UnmappedResources = append(report.UnmappedResources, t)
			seen[t] = true
		}
	}
	sort.Strings(report.UnmappedResources)

	return report
}

func sortOperationDetails(ops []OperationDetail) {
	sort.Slice(ops, func(i, j int) bool {
		a, b := ops[i], ops[j]
		if a.Tag != b.Tag {
			return a.Tag < b.Tag
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
}

// errWriter captures the first write error so renderMarkdown can stay
// fmt.Fprintf-based without checking every call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	if err != nil {
		ew.err = err
	}
	return n, err
}

func renderMarkdown(out io.Writer, r *Report) error {
	w := &errWriter{w: out}
	fmt.Fprintf(w, "# LaunchDarkly API coverage drift report\n\n")
	fmt.Fprintf(w, "Generated %s from `%s`. %d endpoint families in spec.\n\n",
		r.GeneratedAt.Format(time.RFC3339), r.SpecSource, r.TotalFamilies)

	if !r.HasDrift() {
		fmt.Fprintf(w, "**No drift detected.** All spec families are classified in the mapping file.\n\n")
	} else {
		fmt.Fprintf(w, "**Drift detected.** Update `scripts/driftreport/mapping.yaml` to classify the items below.\n\n")
	}

	if len(r.NewFamilies) > 0 {
		fmt.Fprintf(w, "## New endpoint families (not in mapping)\n\n")
		for _, f := range r.NewFamilies {
			fmt.Fprintf(w, "### %s\n\n", f.Tag)
			for _, p := range f.Paths {
				fmt.Fprintf(w, "- `%s`\n", p)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	if len(r.StaleFamilies) > 0 {
		fmt.Fprintf(w, "## Stale mapping entries (tag no longer in spec — renamed or removed)\n\n")
		for _, t := range r.StaleFamilies {
			fmt.Fprintf(w, "- %s\n", t)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.UnmappedResources) > 0 {
		fmt.Fprintf(w, "## Registered provider types not referenced by any family\n\n")
		for _, t := range r.UnmappedResources {
			fmt.Fprintf(w, "- `%s`\n", t)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.RegisteredCandidates) > 0 {
		fmt.Fprintf(w, "## Candidates that already exist (curation error)\n\n")
		fmt.Fprintf(w, "These `new_resource_candidates` name an already-registered provider type, so they are not net-new. Move each one's operations to a `resources` entry instead of `new_resource_candidates`.\n\n")
		for _, c := range r.RegisteredCandidates {
			fmt.Fprintf(w, "- %s: `%s`\n", c.Tag, c.Name)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.UnclaimedOperations) > 0 {
		fmt.Fprintf(w, "## Unclaimed operations in partial families\n\n")
		fmt.Fprintf(w, "Claim each operation on a resource entry or add it to the family's `ignored_operations`.\n\n")
		lastTag := ""
		for _, op := range r.UnclaimedOperations {
			if op.Tag != lastTag {
				fmt.Fprintf(w, "### %s\n\n", op.Tag)
				lastTag = op.Tag
			}
			fmt.Fprintf(w, "- `%s %s` (`%s`)\n", op.Method, op.Path, op.OperationID)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.StaleOperations) > 0 {
		fmt.Fprintf(w, "## Stale operation claims (operationId no longer in spec — renamed or removed)\n\n")
		for _, op := range r.StaleOperations {
			fmt.Fprintf(w, "- %s: `%s`\n", op.Tag, op.OperationID)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.ScaffoldableResources) > 0 {
		fmt.Fprintf(w, "## Scaffoldable new resources in partial families (curated — ready for stage 2)\n\n")
		fmt.Fprintf(w, "Net-new resources within a partial family; the stage-2 scaffolder can implement each without touching the family's existing resources.\n\n")
		for _, sr := range r.ScaffoldableResources {
			fmt.Fprintf(w, "### %s → `%s`\n\n", sr.Tag, sr.Name)
			for _, op := range sr.Operations {
				fmt.Fprintf(w, "- `%s %s` (`%s`)\n", op.Method, op.Path, op.OperationID)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	if len(r.TriageFamilies) > 0 {
		fmt.Fprintf(w, "## Families pending triage (acknowledged, decision pending)\n\n")
		for _, t := range r.TriageFamilies {
			fmt.Fprintf(w, "- %s\n", t)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.TriageOperations) > 0 {
		fmt.Fprintf(w, "## Operations pending triage in partial families (acknowledged, decision pending)\n\n")
		lastTag := ""
		for _, op := range r.TriageOperations {
			if op.Tag != lastTag {
				fmt.Fprintf(w, "### %s\n\n", op.Tag)
				lastTag = op.Tag
			}
			fmt.Fprintf(w, "- `%s %s` (`%s`)\n", op.Method, op.Path, op.OperationID)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "## Mapping status summary\n\n")
	statuses := make([]string, 0, len(r.StatusCounts))
	for s := range r.StatusCounts {
		statuses = append(statuses, s)
	}
	sort.Strings(statuses)
	for _, s := range statuses {
		fmt.Fprintf(w, "- %s: %d\n", s, r.StatusCounts[s])
	}
	return w.err
}
