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

import "strconv"

type SnapshotSkill struct {
	SkillID     int64
	Name        string
	Version     string
	PackageHash string
}

type AgentSnapshotInput struct {
	ModelID      string
	ModelParams  map[string]any
	Prompt       string
	Capabilities map[string]any
	MCPServers   []map[string]any
	Skills       []SnapshotSkill
}

// BuildAgentSnapshot freezes a super-agent's identity into the agent_snapshot
// map stored in ai_product.feature. Skill versions are pinned so a published
// virtual employee never drifts when the source skills are later edited.
func BuildAgentSnapshot(in AgentSnapshotInput) map[string]any {
	skillSet := make([]map[string]any, 0, len(in.Skills))
	for _, s := range in.Skills {
		skillSet = append(skillSet, map[string]any{
			// Store as string: a snowflake skill_id exceeds float64's exact range,
			// so a JSON-number round-trip (decode → map[string]any → float64) would
			// corrupt it when the recruit flow reads the snapshot back.
			"skill_id":      strconv.FormatInt(s.SkillID, 10),
			"name":          s.Name,
			"skill_version": s.Version,
			"package_hash":  s.PackageHash,
		})
	}
	return map[string]any{
		"model":        map[string]any{"model_id": in.ModelID, "params": in.ModelParams},
		"prompt":       in.Prompt,
		"capabilities": in.Capabilities,
		"mcp_servers":  in.MCPServers,
		"skill_set":    skillSet,
	}
}
