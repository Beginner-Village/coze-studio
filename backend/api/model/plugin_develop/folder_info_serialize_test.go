package plugin_develop

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFolderInfoResourceIDsSerializeAsStrings guards against a regression where
// FolderInfo.ResourceIDs was typed []int64 and serialized as JSON numbers. Large
// int64 resource IDs lose precision when parsed by JS, and the frontend matches
// them against the string res_id from the resource list, so they MUST be strings.
func TestFolderInfoResourceIDsSerializeAsStrings(t *testing.T) {
	f := FolderInfo{
		ID:          7532755646102372352,
		SpaceID:     7532755646102372352,
		Name:        "tmp",
		ResourceIDs: []string{"7532755646102372352", "1"},
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, `"resource_ids":["7532755646102372352","1"]`) {
		t.Fatalf("resource_ids must serialize as a JSON string array, got: %s", got)
	}
	// id already uses ,string — keep it consistent so the frontend can match.
	if !strings.Contains(got, `"id":"7532755646102372352"`) {
		t.Fatalf("id must serialize as a JSON string, got: %s", got)
	}
}
