package release

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input                       string
		major, minor, patch int
		ok                          bool
	}{
		{"v1.0.0", 1, 0, 0, true},
		{"v2.3.4", 2, 3, 4, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v10.20.30", 10, 20, 30, true},
		{"1.0.0", 0, 0, 0, false},      // missing v prefix
		{"v1.0", 0, 0, 0, false},        // missing patch
		{"v1.0.0-beta", 0, 0, 0, false}, // pre-release not supported
		{"", 0, 0, 0, false},
		{"vx.y.z", 0, 0, 0, false},
	}

	for _, tt := range tests {
		major, minor, patch, ok := parseSemver(tt.input)
		if ok != tt.ok {
			t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok {
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("parseSemver(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		}
	}
}

func TestDiffIDList(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	fromIDs := []int64{1, 2, 3}
	toIDs := []int64{2, 3, 4}
	fromHashes := map[string]string{
		"agent:1": "hash_a1",
		"agent:2": "hash_a2",
		"agent:3": "hash_a3",
	}
	toHashes := map[string]string{
		"agent:2": "hash_a2",     // same
		"agent:3": "hash_a3_new", // changed
		"agent:4": "hash_a4",
	}

	diffIDList(diff, "agents", fromIDs, toIDs, fromHashes, toHashes, "agent")

	// Added: 4
	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 4 {
		t.Errorf("Added = %v, want [{ID:4}]", diff.Added["agents"])
	}

	// Removed: 1
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}

	// Modified: 3 (hash changed)
	if len(diff.Modified["agents"]) != 1 || diff.Modified["agents"][0].ID != 3 {
		t.Errorf("Modified = %v, want [{ID:3}]", diff.Modified["agents"])
	}
}

func TestDiffIDListNoHashes(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{1, 2}, []int64{2, 3}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 3 {
		t.Errorf("Added = %v, want [{ID:3}]", diff.Added["agents"])
	}
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}
	// No hashes → no modified detection
	if len(diff.Modified["agents"]) != 0 {
		t.Errorf("Modified = %v, want empty (no hashes)", diff.Modified["agents"])
	}
}

func TestDiffIDListEmpty(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{}, []int64{}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 0 || len(diff.Removed["agents"]) != 0 || len(diff.Modified["agents"]) != 0 {
		t.Error("expected empty diff for empty inputs")
	}
}
