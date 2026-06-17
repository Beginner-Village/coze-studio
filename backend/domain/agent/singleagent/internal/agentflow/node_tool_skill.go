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
	"sort"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// readSkillTool implements tool.InvokableTool for the read_skill function.
// It loads full skill instructions on demand (progressive disclosure pattern).
type readSkillTool struct {
	spaceID       int64
	sandboxKey    string
	skillInfoList []*singleagent.SkillReference
	skillCache    map[int64]*entity.Skill
	mu            sync.Mutex
}

type readSkillRequest struct {
	SkillName string `json:"skill_name" jsonschema:"description=The name of the skill to read detailed instructions for"`
}

func newReadSkillTool(spaceID int64, sandboxKey string, skillInfoList []*singleagent.SkillReference) tool.InvokableTool {
	return &readSkillTool{
		spaceID:       spaceID,
		sandboxKey:    sandboxKey,
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

	// 解析技能内联脚本文件（<skill-file path="...">...</skill-file>）。
	cleaned, files := parseSkillFiles(skill.Prompt)

	// 把脚本注入沙箱 /skills/<name>/，让模型可用 run_bash 执行（L3 可执行脚本）。
	var injectNote string
	if len(files) > 0 {
		if svc := crosssandbox.DefaultSVC(); svc != nil && t.sandboxKey != "" {
			if err := svc.SyncSkill(ctx, t.sandboxKey, skill.Name, files); err != nil {
				logs.CtxWarnf(ctx, "[readSkillTool] inject skill files for %s failed: %v", skill.Name, err)
			} else {
				paths := make([]string, 0, len(files))
				for rel := range files {
					paths = append(paths, "/skills/"+skill.Name+"/"+rel)
				}
				sort.Strings(paths)
				injectNote = fmt.Sprintf("\n\n[Sandbox] The following skill files are available; run them with run_bash:\n- %s", strings.Join(paths, "\n- "))
			}
		}
	}

	// Resolve resource references in the prompt
	resolvedPrompt := resolveResourceReferences(cleaned)

	return fmt.Sprintf("=== Skill: %s ===\n%s%s", skill.Name, resolvedPrompt, injectNote), nil
}

// skillFileRegex 匹配技能 Prompt 内联脚本块：<skill-file path="scripts/run.py">...</skill-file>
var skillFileRegex = regexp.MustCompile(`(?s)<skill-file\s+path="([^"]+)"\s*>\n?(.*?)\n?</skill-file>`)

// parseSkillFiles 从 prompt 抽出内联脚本文件，返回去掉文件块后的正文与 {相对路径: 内容}。
func parseSkillFiles(prompt string) (string, map[string][]byte) {
	matches := skillFileRegex.FindAllStringSubmatch(prompt, -1)
	if len(matches) == 0 {
		return prompt, nil
	}
	files := make(map[string][]byte, len(matches))
	for _, mt := range matches {
		path := strings.TrimSpace(mt[1])
		if path == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			continue
		}
		files[path] = []byte(mt[2])
	}
	cleaned := strings.TrimSpace(skillFileRegex.ReplaceAllString(prompt, ""))
	return cleaned, files
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

	// Enforce space isolation: a skill bound to this agent must belong to the
	// agent's space. Treat a cross-space skill as not found to prevent reading
	// skills from other spaces via a forged/leaked skill id.
	if skill != nil && skill.SpaceID != t.spaceID {
		logs.CtxWarnf(ctx, "[readSkillTool] skill %d space mismatch (skill.SpaceID=%d, agent.SpaceID=%d), denying access",
			skillID, skill.SpaceID, t.spaceID)
		return nil, nil
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

// syncBoundSkillsToSandbox 把超级智能体绑定的技能 eager 落盘到沙箱 /skills/<name>/。
// 每个技能写出 SKILL.md(技能正文)+ 其内联脚本(<skill-file path>),让 agent 用通用
// list_files/read_file/run_bash 自己读取技能文件夹并执行 —— 标准的 Claude Code 式技能。
// SyncSkill 自带内容 hash 去重,同版本跳过,因此每次会话调用代价很低。
func syncBoundSkillsToSandbox(ctx context.Context, sandboxKey string, spaceID int64, skillInfoList []*singleagent.SkillReference) {
	svc := crosssandbox.DefaultSVC()
	skillSvc := crossskill.DefaultSVC()
	if svc == nil || skillSvc == nil || sandboxKey == "" || len(skillInfoList) == 0 {
		return
	}
	for _, ref := range skillInfoList {
		skill, err := skillSvc.GetSkill(ctx, ref.SkillID)
		if err != nil || skill == nil {
			continue
		}
		// 跨空间技能不落盘(空间隔离)。
		if skill.SpaceID != spaceID {
			continue
		}

		var files map[string][]byte
		if len(skill.Files) > 0 {
			// 真·文件夹技能:整棵文件树(SKILL.md + scripts/ + references/ + assets/)原样落盘,保留子目录。
			files = make(map[string][]byte, len(skill.Files))
			for rel, content := range skill.Files {
				files[rel] = []byte(content)
			}
			if _, ok := files["SKILL.md"]; !ok {
				files["SKILL.md"] = []byte(resolveResourceReferences(skill.Prompt))
			}
		} else {
			// 兼容旧技能:从 prompt 里抽 <skill-file> 内联脚本 + SKILL.md 正文。
			cleaned, inlineFiles := parseSkillFiles(skill.Prompt)
			files = inlineFiles
			if files == nil {
				files = make(map[string][]byte, 1)
			}
			files["SKILL.md"] = []byte(resolveResourceReferences(cleaned))
		}
		if err := svc.SyncSkill(ctx, sandboxKey, skill.Name, files); err != nil {
			logs.CtxWarnf(ctx, "[syncBoundSkillsToSandbox] sync skill %s failed: %v", skill.Name, err)
		}
	}
}

// newSkillTools creates the read_skill tool if skills are configured.
func newSkillTools(spaceID int64, sandboxKey string, skillInfoList []*singleagent.SkillReference) []tool.InvokableTool {
	if len(skillInfoList) == 0 {
		return nil
	}
	return []tool.InvokableTool{newReadSkillTool(spaceID, sandboxKey, skillInfoList)}
}
