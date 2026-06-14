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

// skillResources holds the deduplicated resource IDs referenced by skill prompts,
// split by resource type so the builder can mount each kind with the right tool
// constructor.
type skillResources struct {
	WorkflowIDs  []int64
	PluginIDs    []int64
	KnowledgeIDs []int64
}

// resolveSkillResources batch-loads skills, parses resource references from their
// prompts, and returns the deduplicated workflow / plugin / knowledge IDs that are
// referenced. Workflow IDs already bound to the agent are skipped.
// Errors are handled with fail-soft strategy: failures log warnings but do not
// prevent the agent from starting.
func resolveSkillResources(
	ctx context.Context,
	skillInfoList []*singleagent.SkillReference,
	existingWorkflowIDs map[int64]struct{},
) (*skillResources, error) {
	if len(skillInfoList) == 0 {
		return &skillResources{}, nil
	}

	svc := crossskill.DefaultSVC()
	if svc == nil {
		logs.CtxWarnf(ctx, "[resolveSkillResources] skill service not initialized, skipping")
		return &skillResources{}, nil
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

	res := &skillResources{}
	seenWf := make(map[int64]struct{})
	seenPlugin := make(map[int64]struct{})
	seenKnowledge := make(map[int64]struct{})

	for _, skill := range skills {
		if skill == nil || skill.Prompt == "" {
			continue
		}
		refs := parseSkillResourceRefs(skill.Prompt)
		for _, ref := range refs {
			switch ref.ResourceType {
			case "workflow":
				if _, exists := existingWorkflowIDs[ref.ID]; exists {
					continue
				}
				if _, dup := seenWf[ref.ID]; dup {
					continue
				}
				seenWf[ref.ID] = struct{}{}
				res.WorkflowIDs = append(res.WorkflowIDs, ref.ID)
			case "plugin":
				if _, dup := seenPlugin[ref.ID]; dup {
					continue
				}
				seenPlugin[ref.ID] = struct{}{}
				res.PluginIDs = append(res.PluginIDs, ref.ID)
			case "knowledge":
				if _, dup := seenKnowledge[ref.ID]; dup {
					continue
				}
				seenKnowledge[ref.ID] = struct{}{}
				res.KnowledgeIDs = append(res.KnowledgeIDs, ref.ID)
			}
		}
	}

	return res, nil
}
