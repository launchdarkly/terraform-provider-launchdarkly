package launchdarkly

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ciWorkflowFiles are the workflows that shard the acceptance suite across a
// test_case matrix. Paths are relative to this package directory.
var ciWorkflowFiles = []string{
	filepath.Join("..", ".github", "workflows", "test.yml"),
	filepath.Join("..", ".github", "workflows", "test-fork-pr.yml"),
}

var acceptanceTestFuncRe = regexp.MustCompile(`(?m)^func (TestAcc\w+)\(`)

// ciWorkflow is the minimal shape needed to reach the acceptance matrix.
type ciWorkflow struct {
	Jobs map[string]struct {
		Strategy struct {
			Matrix struct {
				TestCase []string `yaml:"test_case"`
			} `yaml:"matrix"`
		} `yaml:"strategy"`
	} `yaml:"jobs"`
}

// TestCIMatrixCoversAcceptanceTests guards the CI acceptance matrices against
// drift from the real TestAcc* function names.
//
// Each matrix entry is handed to `go test -run` as an *unanchored* regex, so
// both drift directions fail silently:
//
//   - an entry that matches no test function runs zero tests and reports a
//     green job (four TestAccTeamMember_* entries did exactly this until
//     REL-15194);
//   - a test function matched by no entry never executes in CI at all, so its
//     config rots unnoticed until someone turns it on.
//
// Neither is visible from a passing workflow run, hence this check. It runs in
// the build job, needs no LaunchDarkly credentials, and is skipped by nothing.
func TestCIMatrixCoversAcceptanceTests(t *testing.T) {
	names := acceptanceTestNames(t)
	if len(names) == 0 {
		t.Fatal("found no TestAcc* functions in this package; the discovery regex is broken")
	}

	matrices := make(map[string][]string, len(ciWorkflowFiles))
	for _, path := range ciWorkflowFiles {
		matrices[path] = acceptanceMatrix(t, path)
	}

	for path, entries := range matrices {
		t.Run(filepath.Base(path), func(t *testing.T) {
			checkMatrix(t, path, entries, names)
		})
	}

	// The fork-PR workflow duplicates the matrix because pull_request_target
	// needs its own job. Nothing in GitHub Actions keeps the two lists in
	// sync, and in practice only test.yml got updated as resources were added.
	t.Run("workflows agree", func(t *testing.T) {
		base := ciWorkflowFiles[0]
		for _, other := range ciWorkflowFiles[1:] {
			for _, entry := range missing(matrices[base], matrices[other]) {
				t.Errorf("%q is in %s but not %s; keep the two test_case lists identical", entry, base, other)
			}
			for _, entry := range missing(matrices[other], matrices[base]) {
				t.Errorf("%q is in %s but not %s; keep the two test_case lists identical", entry, other, base)
			}
		}
	})
}

// checkMatrix asserts a single test_case list covers every acceptance test,
// contains no entry that matches nothing, and has no duplicates.
func checkMatrix(t *testing.T, path string, entries, names []string) {
	t.Helper()

	patterns := make([]*regexp.Regexp, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if seen[entry] {
			t.Errorf("%s: duplicate test_case entry %q; GitHub Actions would run the same shard twice", path, entry)
			continue
		}
		seen[entry] = true

		re, err := regexp.Compile(entry)
		if err != nil {
			t.Errorf("%s: test_case entry %q is not a valid regex, so `go test -run` will reject it: %s", path, entry, err)
			continue
		}
		patterns = append(patterns, re)

		if !matchesAny(re, names) {
			t.Errorf("%s: test_case entry %q matches no TestAcc* function; that shard silently runs zero tests and reports success. Fix the entry or delete it.", path, entry)
		}
	}

	for _, name := range names {
		matched := false
		for _, re := range patterns {
			if re.MatchString(name) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("%s: %s is matched by no test_case entry, so it never runs in CI. Add an entry that matches it.", path, name)
		}
	}
}

// acceptanceTestNames scans the package's own _test.go files for acceptance
// test declarations. This deliberately reads source rather than using
// reflection: test functions are not addressable by name at runtime, and the
// matrix entries are matched against source-level names anyway.
//
// Scanning only this package is deliberate too. Every TestAcc* function in the
// repo lives here, and a repo-wide walk would also descend into .claude/
// worktrees and pick up copies of these same files. If acceptance tests ever
// move into another package, extend this to walk those directories explicitly
// rather than globbing from the repo root.
func acceptanceTestNames(t *testing.T) []string {
	t.Helper()

	testFiles, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("globbing test files: %s", err)
	}

	var names []string
	for _, file := range testFiles {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %s", file, err)
		}
		for _, match := range acceptanceTestFuncRe.FindAllStringSubmatch(string(src), -1) {
			names = append(names, match[1])
		}
	}
	sort.Strings(names)
	return names
}

// acceptanceMatrix returns the test_case list from the workflow's matrix job.
func acceptanceMatrix(t *testing.T, path string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %s", path, err)
	}
	var wf ciWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parsing %s: %s", path, err)
	}

	var entries []string
	for _, job := range wf.Jobs {
		entries = append(entries, job.Strategy.Matrix.TestCase...)
	}
	if len(entries) == 0 {
		t.Fatalf("%s declares no test_case matrix; if the acceptance sharding moved, update ciWorkflowFiles", path)
	}
	return entries
}

func matchesAny(re *regexp.Regexp, names []string) bool {
	for _, name := range names {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// missing returns the entries of a that are absent from b.
func missing(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, entry := range b {
		inB[strings.TrimSpace(entry)] = true
	}
	var out []string
	for _, entry := range a {
		if !inB[strings.TrimSpace(entry)] {
			out = append(out, entry)
		}
	}
	sort.Strings(out)
	return out
}
