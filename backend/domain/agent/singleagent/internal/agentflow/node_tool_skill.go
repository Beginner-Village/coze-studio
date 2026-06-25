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
	"crypto/sha256"
	"encoding/hex"
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
		return argParseErrMsg(err), nil
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

	instructions, files := prepareSkillRuntimeFiles(skill)

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

	return fmt.Sprintf("=== Skill: %s ===\n%s%s", skill.Name, instructions, injectNote), nil
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

func prepareSkillRuntimeFiles(skill *entity.Skill) (string, map[string][]byte) {
	if len(skill.Files) > 0 {
		files := make(map[string][]byte, len(skill.Files)+1)
		for rel, content := range skill.Files {
			if isSafeSkillRelativePath(rel) {
				files[rel] = []byte(content)
			}
		}
		instructions := skill.Prompt
		if content, ok := files["SKILL.md"]; ok {
			instructions = string(content)
		}
		instructions = resolveResourceReferences(strings.TrimSpace(instructions))
		files["SKILL.md"] = []byte(instructions)
		return instructions, files
	}

	cleaned, inlineFiles := parseSkillFiles(skill.Prompt)
	instructions := resolveResourceReferences(strings.TrimSpace(cleaned))
	if inlineFiles == nil {
		inlineFiles = make(map[string][]byte, 1)
	}
	inlineFiles["SKILL.md"] = []byte(instructions)
	return instructions, inlineFiles
}

func isSafeSkillRelativePath(path string) bool {
	path = strings.TrimSpace(path)
	return path != "" &&
		!strings.HasPrefix(path, "/") &&
		!strings.Contains(path, "..") &&
		!strings.Contains(path, "\\")
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
// skillManifestPath 记录「已同步的技能集合指纹」。只要绑定技能内容不变,
// 后续每条消息只读一次该文件即跳过整段同步——等价于「容器创建时同步一次,
// 之后不再同步,技能变更时才重新同步」。配合 /skills 持久化卷,容器重建也直接命中。
const skillManifestPath = "/skills/.manifest"

type boundSkillFiles struct {
	name  string
	files map[string][]byte
}

func syncBoundSkillsToSandbox(ctx context.Context, sandboxKey string, spaceID int64, skillInfoList []*singleagent.SkillReference) {
	svc := crosssandbox.DefaultSVC()
	skillSvc := crossskill.DefaultSVC()
	if svc == nil || skillSvc == nil || sandboxKey == "" || len(skillInfoList) == 0 {
		return
	}

	// 1) 收集技能文件并计算集合指纹(GetSkill 为 DB/缓存读,廉价;不涉及 docker)。
	prepared := make([]boundSkillFiles, 0, len(skillInfoList))
	for _, ref := range skillInfoList {
		skill, err := skillSvc.GetSkill(ctx, ref.SkillID)
		if err != nil || skill == nil || skill.SpaceID != spaceID {
			continue // 跨空间技能不落盘(空间隔离)
		}

		_, files := prepareSkillRuntimeFiles(skill)
		prepared = append(prepared, boundSkillFiles{name: skill.Name, files: files})
	}
	if len(prepared) == 0 {
		return
	}
	manifest := computeSkillsManifest(prepared)

	// 2) 指纹未变 → 整段跳过(只 1 次读,不写、不调 SyncSkill)。
	if cur, err := svc.ReadFile(ctx, sandboxKey, skillManifestPath); err == nil &&
		strings.TrimSpace(string(cur)) == manifest {
		return
	}

	// 3) 首次或技能变更 → 全量同步后写入指纹。
	for _, p := range prepared {
		if err := svc.SyncSkill(ctx, sandboxKey, p.name, p.files); err != nil {
			logs.CtxWarnf(ctx, "[syncBoundSkillsToSandbox] sync skill %s failed: %v", p.name, err)
			return // 同步失败不落指纹,下条消息重试
		}
	}
	if err := svc.WriteFile(ctx, sandboxKey, skillManifestPath, []byte(manifest)); err != nil {
		logs.CtxWarnf(ctx, "[syncBoundSkillsToSandbox] write manifest failed: %v", err)
	}
}

// computeSkillsManifest 对(技能名 + 文件名 + 文件内容)做稳定排序后求 sha256,
// 任意技能内容变化都会改变指纹。
func computeSkillsManifest(skills []boundSkillFiles) string {
	sort.Slice(skills, func(i, j int) bool { return skills[i].name < skills[j].name })
	h := sha256.New()
	for _, s := range skills {
		h.Write([]byte(s.name))
		h.Write([]byte{0})
		names := make([]string, 0, len(s.files))
		for n := range s.files {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			h.Write([]byte(n))
			h.Write([]byte{0})
			h.Write(s.files[n])
			h.Write([]byte{0})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// newSkillTools creates the read_skill tool if skills are configured.
func newSkillTools(spaceID int64, sandboxKey string, skillInfoList []*singleagent.SkillReference) []tool.InvokableTool {
	if len(skillInfoList) == 0 {
		return nil
	}
	return []tool.InvokableTool{newReadSkillTool(spaceID, sandboxKey, skillInfoList)}
}
