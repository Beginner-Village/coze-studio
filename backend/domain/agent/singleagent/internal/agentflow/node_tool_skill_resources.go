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

package agentflow

import (
	"context"
	"strconv"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// skillResourceRef represents a single resource reference in a skill prompt.
type skillResourceRef struct {
	ResourceType string // "workflow", "plugin", "knowledge"
	Name         string
	ID           int64
}

// parseSkillResourceRefs extracts all resource references from a prompt string.
// Reuses resourceRefRegex defined in node_tool_skill.go.
func parseSkillResourceRefs(prompt string) []skillResourceRef {
	matches := resourceRefRegex.FindAllStringSubmatch(prompt, -1)
	refs := make([]skillResourceRef, 0, len(matches))
	for _, m := range matches {
		if len(m) < 4 || m[3] == "" {
			continue
		}
		id, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			continue
		}
		refs = append(refs, skillResourceRef{
			ResourceType: m[1],
			Name:         m[2],
			ID:           id,
		})
	}
	return refs
}

// resolveSkillResources batch-loads skills, parses resource references from their
// prompts, and returns deduplicated workflow IDs that are not already bound.
// Errors are handled with fail-soft strategy: failures log warnings but do not
// prevent the agent from starting.
func resolveSkillResources(
	ctx context.Context,
	skillInfoList []*singleagent.SkillReference,
	existingWorkflowIDs map[int64]struct{},
) (workflowIDs []int64, err error) {
	if len(skillInfoList) == 0 {
		return nil, nil
	}

	svc := crossskill.DefaultSVC()
	if svc == nil {
		logs.CtxWarnf(ctx, "[resolveSkillResources] skill service not initialized, skipping")
		return nil, nil
	}

	skillIDs := make([]int64, 0, len(skillInfoList))
	for _, ref := range skillInfoList {
		skillIDs = append(skillIDs, ref.SkillID)
	}

	skills, err := svc.MGetSkills(ctx, skillIDs)
	if err != nil {
		logs.CtxErrorf(ctx, "[resolveSkillResources] MGetSkills failed: %v", err)
		return nil, err
	}

	seen := make(map[int64]struct{})
	var result []int64

	for _, skill := range skills {
		if skill == nil || skill.Prompt == "" {
			continue
		}
		refs := parseSkillResourceRefs(skill.Prompt)
		for _, ref := range refs {
			if ref.ResourceType != "workflow" {
				continue
			}
			if _, exists := existingWorkflowIDs[ref.ID]; exists {
				continue
			}
			if _, dup := seen[ref.ID]; dup {
				continue
			}
			seen[ref.ID] = struct{}{}
			result = append(result, ref.ID)
		}
	}

	return result, nil
}
