package release

import (
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

func TestHashPackage(t *testing.T) {
	// Same input → same hash
	data := []byte("test zip content")
	h1 := HashPackage(data)
	h2 := HashPackage(data)
	if h1 != h2 {
		t.Errorf("HashPackage not deterministic: %s != %s", h1, h2)
	}
	// SHA256 hex is 64 chars
	if len(h1) != 64 {
		t.Errorf("HashPackage length = %d, want 64", len(h1))
	}

	// Different input → different hash
	h3 := HashPackage([]byte("other content"))
	if h1 == h3 {
		t.Errorf("HashPackage collision for different inputs")
	}

	// Empty input
	h4 := HashPackage([]byte{})
	if len(h4) != 64 {
		t.Errorf("HashPackage empty input length = %d, want 64", len(h4))
	}
}

func TestBuildResourceHashes(t *testing.T) {
	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{ID: 100, Name: "agent1"},
			{ID: 200, Name: "agent2"},
		},
		Plugins: []*export.ExportedPlugin{
			{ID: 300, Name: "plugin1"},
		},
		Workflows:         []*export.ExportedWorkflow{},
		Variables:         []*export.ExportedVariable{},
		SpaceModels:       []*export.ExportedSpaceModel{},
		KnowledgeBases:    []*export.ExportedKnowledge{},
		Folders:           []*export.ExportedFolder{},
		ExternalKnowledge: []*export.ExportedExternalKnowledge{},
	}

	hashes, err := BuildResourceHashes(resources)
	if err != nil {
		t.Fatalf("BuildResourceHashes error: %v", err)
	}

	// Should contain entries for agents and plugins
	if _, ok := hashes["agent:100"]; !ok {
		t.Error("missing hash for agent:100")
	}
	if _, ok := hashes["agent:200"]; !ok {
		t.Error("missing hash for agent:200")
	}
	if _, ok := hashes["plugin:300"]; !ok {
		t.Error("missing hash for plugin:300")
	}

	// Different agents → different hashes
	if hashes["agent:100"] == hashes["agent:200"] {
		t.Error("agent:100 and agent:200 should have different hashes")
	}

	// Deterministic: same input → same hashes
	hashes2, _ := BuildResourceHashes(resources)
	for k, v := range hashes {
		if hashes2[k] != v {
			t.Errorf("hash for %s not deterministic: %s != %s", k, v, hashes2[k])
		}
	}
}

func TestBuildResourceHashesNil(t *testing.T) {
	hashes, err := BuildResourceHashes(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("expected empty hashes for nil resources, got %d entries", len(hashes))
	}
}
