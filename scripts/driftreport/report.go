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
	Tag       string   `yaml:"tag"`
	Status    string   `yaml:"status"`
	Resources []string `yaml:"resources,omitempty"`
	Reason    string   `yaml:"reason,omitempty"`
	Notes     string   `yaml:"notes,omitempty"`
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

// specFamilies extracts tag -> sorted unique paths from raw OpenAPI JSON.
func specFamilies(rawSpec []byte) (map[string][]string, error) {
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
	families := map[string]map[string]bool{}
	for path, ops := range spec.Paths {
		for method, rawOp := range ops {
			if !specMethods[strings.ToLower(method)] {
				continue
			}
			var op struct {
				Tags []string `json:"tags"`
			}
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return nil, fmt.Errorf("parsing operation %s %s: %w", method, path, err)
			}
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{"<untagged>"}
			}
			for _, tag := range tags {
				if families[tag] == nil {
					families[tag] = map[string]bool{}
				}
				families[tag][path] = true
			}
		}
	}
	out := make(map[string][]string, len(families))
	for tag, paths := range families {
		sorted := make([]string, 0, len(paths))
		for p := range paths {
			sorted = append(sorted, p)
		}
		sort.Strings(sorted)
		out[tag] = sorted
	}
	return out, nil
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
	NewFamilies       []FamilyDetail `json:"new_families"`
	StaleFamilies     []string       `json:"stale_families"`
	UnmappedResources []string       `json:"unmapped_resources"`

	// Informational.
	TriageFamilies []string       `json:"triage_families"`
	StatusCounts   map[string]int `json:"status_counts"`
	TotalFamilies  int            `json:"total_families"`
}

type FamilyDetail struct {
	Tag   string   `json:"tag"`
	Paths []string `json:"paths"`
}

func (r *Report) HasDrift() bool {
	return len(r.NewFamilies) > 0 || len(r.StaleFamilies) > 0 || len(r.UnmappedResources) > 0
}

func buildReport(families map[string][]string, mapping *Mapping, resources, dataSources []string, specSource string) *Report {
	report := &Report{
		GeneratedAt:   time.Now().UTC(),
		SpecSource:    specSource,
		StatusCounts:  map[string]int{},
		TotalFamilies: len(families),
		// Initialized so empty lists serialize as [] rather than null in JSON.
		NewFamilies:       []FamilyDetail{},
		StaleFamilies:     []string{},
		UnmappedResources: []string{},
		TriageFamilies:    []string{},
	}

	mapped := map[string]MappingFamily{}
	for _, f := range mapping.Families {
		mapped[f.Tag] = f
	}

	// Spec tags absent from the mapping => new families.
	for tag, paths := range families {
		if _, ok := mapped[tag]; !ok {
			report.NewFamilies = append(report.NewFamilies, FamilyDetail{Tag: tag, Paths: paths})
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

	// Registered types never referenced by any family => mapping is incomplete.
	referenced := map[string]bool{}
	for _, f := range mapping.Families {
		for _, r := range f.Resources {
			referenced[r] = true
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

func renderMarkdown(w io.Writer, r *Report) {
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

	if len(r.TriageFamilies) > 0 {
		fmt.Fprintf(w, "## Families pending triage (acknowledged, decision pending)\n\n")
		for _, t := range r.TriageFamilies {
			fmt.Fprintf(w, "- %s\n", t)
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
}
