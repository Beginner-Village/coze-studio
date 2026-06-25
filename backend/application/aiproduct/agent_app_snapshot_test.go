/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aiproduct

import (
	"testing"
)

func TestBuildAgentSnapshotFreezesSkillVersions(t *testing.T) {
	in := AgentSnapshotInput{
		ModelID:      "1",
		ModelParams:  map[string]any{"max_tokens": 8192},
		Prompt:       "you are a helper",
		Capabilities: map[string]any{"sandbox": true, "skill_manage": false},
		MCPServers:   []map[string]any{{"name": "docs", "type": "streamable_http"}},
		Skills: []SnapshotSkill{
			{SkillID: 765, Name: "pdf-tools", Version: "4", PackageHash: "sha256:abc"},
		},
	}

	snap := BuildAgentSnapshot(in)

	skills, ok := snap["skill_set"].([]map[string]any)
	if !ok || len(skills) != 1 {
		t.Fatalf("skill_set wrong: %#v", snap["skill_set"])
	}
	if skills[0]["skill_version"] != "4" || skills[0]["package_hash"] != "sha256:abc" {
		t.Fatalf("skill not pinned: %#v", skills[0])
	}
	if snap["prompt"] != "you are a helper" {
		t.Fatalf("prompt missing: %#v", snap["prompt"])
	}
	if caps := snap["capabilities"].(map[string]any); caps["skill_manage"] != false {
		t.Fatalf("capabilities not frozen: %#v", caps)
	}
}
