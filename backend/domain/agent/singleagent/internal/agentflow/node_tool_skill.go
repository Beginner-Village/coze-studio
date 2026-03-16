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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// readSkillTool implements tool.InvokableTool for the read_skill function.
// It loads full skill instructions on demand (progressive disclosure pattern).
type readSkillTool struct {
	spaceID       int64
	skillInfoList []*singleagent.SkillReference
	skillCache    map[int64]*entity.Skill
	mu            sync.Mutex
}

type readSkillRequest struct {
	SkillName string `json:"skill_name" jsonschema:"description=The name of the skill to read detailed instructions for"`
}

func newReadSkillTool(spaceID int64, skillInfoList []*singleagent.SkillReference) tool.InvokableTool {
	return &readSkillTool{
		spaceID:       spaceID,
		skillInfoList: skillInfoList,
		skillCache:    make(map[int64]*entity.Skill),
	}
}

func (t *readSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_skill",
		Desc: "Read the detailed instructions for a specific skill. Call this when a user's request matches one of the available skills listed in the system prompt.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "The name of the skill to read detailed instructions for",
				Required: true,
			},
		}),
	}, nil
}

func (t *readSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var req readSkillRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if req.SkillName == "" {
		return "Error: skill_name is required", nil
	}

	// Find matching skill reference by name
	var matchedRef *singleagent.SkillReference
	for _, ref := range t.skillInfoList {
		if strings.EqualFold(ref.SkillName, req.SkillName) {
			matchedRef = ref
			break
		}
	}
	if matchedRef == nil {
		return fmt.Sprintf("Error: skill '%s' not found. Available skills: %s",
			req.SkillName, t.listSkillNames()), nil
	}

	// Load from cache or DB
	skill, err := t.loadSkill(ctx, matchedRef.SkillID)
	if err != nil {
		logs.CtxErrorf(ctx, "[readSkillTool] failed to load skill %d: %v", matchedRef.SkillID, err)
		return fmt.Sprintf("Error: failed to load skill '%s'", req.SkillName), nil
	}
	if skill == nil {
		return fmt.Sprintf("Error: skill '%s' has been deleted or is unavailable", req.SkillName), nil
	}

	// Resolve resource references in the prompt
	resolvedPrompt := resolveResourceReferences(skill.Prompt)

	return fmt.Sprintf("=== Skill: %s ===\n%s", skill.Name, resolvedPrompt), nil
}

func (t *readSkillTool) loadSkill(ctx context.Context, skillID int64) (*entity.Skill, error) {
	t.mu.Lock()
	if cached, ok := t.skillCache[skillID]; ok {
		t.mu.Unlock()
		return cached, nil
	}
	t.mu.Unlock()

	svc := crossskill.DefaultSVC()
	if svc == nil {
		return nil, fmt.Errorf("skill service not initialized")
	}

	skill, err := svc.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}

	if skill != nil {
		t.mu.Lock()
		t.skillCache[skillID] = skill
		t.mu.Unlock()
	}

	return skill, nil
}

func (t *readSkillTool) listSkillNames() string {
	names := make([]string, 0, len(t.skillInfoList))
	for _, ref := range t.skillInfoList {
		names = append(names, ref.SkillName)
	}
	return strings.Join(names, ", ")
}

// resourceRefRegex matches resource references in skill prompts.
// Format: {type:name|id:xxx}
// Examples:
//   - {workflow:退款流程|id:345678}
//   - {plugin:CRM系统|id:456}
//   - {knowledge:FAQ库|id:123}
var resourceRefRegex = regexp.MustCompile(`\{(workflow|plugin|knowledge):([^}|]+)(?:\|id:(\w+))?\}`)

// resolveResourceReferences replaces resource reference placeholders with
// human-readable descriptions that the agent can act on.
func resolveResourceReferences(prompt string) string {
	return resourceRefRegex.ReplaceAllStringFunc(prompt, func(match string) string {
		submatches := resourceRefRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		resourceType := submatches[1]
		resourceName := submatches[2]
		// resourceID := "" // will be used for actual resolution in the future
		// if len(submatches) >= 4 {
		// 	resourceID = submatches[3]
		// }

		// Generate descriptive text based on resource type
		switch resourceType {
		case "workflow":
			return fmt.Sprintf("【Workflow: %s】\n  Call this workflow using the corresponding tool when needed.", resourceName)
		case "plugin":
			return fmt.Sprintf("【Plugin: %s】\n  Use the corresponding plugin tool to interact with this service.", resourceName)
		case "knowledge":
			return fmt.Sprintf("【Knowledge Base: %s】\n  Query this knowledge base for relevant information.", resourceName)
		default:
			return match
		}
	})
}

// newSkillTools creates the read_skill tool if skills are configured.
func newSkillTools(spaceID int64, skillInfoList []*singleagent.SkillReference) []tool.InvokableTool {
	if len(skillInfoList) == 0 {
		return nil
	}
	return []tool.InvokableTool{newReadSkillTool(spaceID, skillInfoList)}
}
