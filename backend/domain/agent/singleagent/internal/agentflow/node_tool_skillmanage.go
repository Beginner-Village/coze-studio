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

// node_tool_skillmanage.go 实现「路线 P3」：超级智能体的**技能自创**。
//
// 超级 agent 可以把标准技能文件夹写到 /skills/<name>/,
// 沉淀成可复用的能力,之后再 list/read/维护。工具通过 registerSuperAgentExtension
// 注册,只挂给超级 agent。

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9-]+$`)

func init() {
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &skillManageTool{key: deps.SandboxKey}
	})
}

type skillManageTool struct{ key string }

type skillManageRequest struct {
	Action     string `json:"action" jsonschema:"description=One of: create, write_file, edit, remove_file, delete, list, read, diff"`
	Name       string `json:"name" jsonschema:"description=Skill name (lowercase letters, digits, hyphens)"`
	Path       string `json:"path" jsonschema:"description=Relative standard skill file path when action=write_file/edit/remove_file/read/diff"`
	Content    string `json:"content" jsonschema:"description=File content when action=create/write_file/diff"`
	OldString  string `json:"old_string" jsonschema:"description=Exact text to replace when action=edit"`
	NewString  string `json:"new_string" jsonschema:"description=Replacement text when action=edit"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"description=Replace every occurrence when action=edit; default false"`
}

func (t *skillManageTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "skill_manage",
		Desc: "Create, list, read, write, or delete standard files for your own reusable skills (each lives under /skills/<name>). " +
			"action=create writes content as /skills/<name>/SKILL.md (content should be Markdown with frontmatter); " +
			"action=write_file writes one standard skill file under SKILL.md, scripts/, references/, templates/, or assets/; " +
			"action=edit edits one standard skill file by exact string replacement; " +
			"action=remove_file removes one standard skill file; action=delete removes the whole skill folder; " +
			"action=list lists existing skill names, or a skill package file tree when name is provided; " +
			"action=read returns a skill file; action=diff compares a skill file with proposed content. " +
			"Use this to author a capability once and reuse it later.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":      {Type: schema.String, Desc: "One of: create, write_file, edit, remove_file, delete, list, read, diff", Required: true},
			"name":        {Type: schema.String, Desc: "Skill name (lowercase letters, digits, hyphens). Required for create/write_file/edit/remove_file/delete/read/diff; optional for list", Required: false},
			"path":        {Type: schema.String, Desc: "Relative standard skill file path when action=write_file/edit/remove_file/read/diff; read defaults to SKILL.md", Required: false},
			"content":     {Type: schema.String, Desc: "File content when action=create/write_file/diff", Required: false},
			"old_string":  {Type: schema.String, Desc: "Exact text to replace when action=edit", Required: false},
			"new_string":  {Type: schema.String, Desc: "Replacement text when action=edit", Required: false},
			"replace_all": {Type: schema.Boolean, Desc: "Replace all occurrences when action=edit; default false", Required: false},
		}),
	}, nil
}

func (t *skillManageTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req skillManageRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	action := strings.TrimSpace(req.Action)
	name := strings.TrimSpace(req.Name)

	switch action {
	case "create":
		if name == "" {
			return "Error: name is required for create", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		if strings.TrimSpace(req.Content) == "" {
			return "Error: content is required for create", nil
		}
		path := "/skills/" + name + "/SKILL.md"
		if err := svc.WriteFile(ctx, t.key, path, []byte(req.Content)); err != nil {
			return fmt.Sprintf("Error creating skill: %v", err), nil
		}
		return fmt.Sprintf("Created skill %s.", name), nil

	case "write_file":
		if name == "" {
			return "Error: name is required for write_file", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		relPath, err := normalizeStandardSkillFilePath(req.Path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		target := "/skills/" + name + "/" + relPath
		if err := svc.WriteFile(ctx, t.key, target, []byte(req.Content)); err != nil {
			return fmt.Sprintf("Error writing skill file: %v", err), nil
		}
		return fmt.Sprintf("Wrote skill file %s for %s.", relPath, name), nil

	case "edit":
		if name == "" {
			return "Error: name is required for edit", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		relPath, err := normalizeStandardSkillFilePath(req.Path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		target := "/skills/" + name + "/" + relPath
		n, err := svc.EditFile(ctx, t.key, target, req.OldString, req.NewString, req.ReplaceAll)
		if err != nil {
			return fmt.Sprintf("Error editing skill file: %v", err), nil
		}
		return fmt.Sprintf("Edited skill file %s for %s (%d replacement(s)).", relPath, name, n), nil

	case "remove_file":
		if name == "" {
			return "Error: name is required for remove_file", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		relPath, err := normalizeStandardSkillFilePath(req.Path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		if relPath == "SKILL.md" {
			return "Error: SKILL.md is the skill entry file and cannot be removed with remove_file; use delete to remove the whole skill", nil
		}
		target := "/skills/" + name + "/" + relPath
		if _, err := svc.Exec(ctx, t.key, "rm -f -- "+skillManageShellQuote(target), 10); err != nil {
			return fmt.Sprintf("Error removing skill file: %v", err), nil
		}
		return fmt.Sprintf("Removed skill file %s for %s.", relPath, name), nil

	case "delete":
		if name == "" {
			return "Error: name is required for delete", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		target := "/skills/" + name
		if _, err := svc.Exec(ctx, t.key, "rm -rf -- "+skillManageShellQuote(target), 10); err != nil {
			return fmt.Sprintf("Error deleting skill: %v", err), nil
		}
		return fmt.Sprintf("Deleted skill %s.", name), nil

	case "list":
		if name != "" {
			if !skillNameRe.MatchString(name) {
				return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
			}
			cmd := "cd " + skillManageShellQuote("/skills/"+name) + " 2>/dev/null && find . -type f | sed 's#^\\./##' | sort"
			res, err := svc.Exec(ctx, t.key, cmd, 0)
			if err != nil {
				return fmt.Sprintf("Error listing skill files: %v", err), nil
			}
			out := strings.TrimSpace(res.Stdout)
			if out == "" {
				return fmt.Sprintf("(skill %s has no files)", name), nil
			}
			return out, nil
		}
		res, err := svc.Exec(ctx, t.key, "ls -1 /skills 2>/dev/null", 0)
		if err != nil {
			return fmt.Sprintf("Error listing skills: %v", err), nil
		}
		out := strings.TrimSpace(res.Stdout)
		if out == "" {
			return "(no skills yet)", nil
		}
		return out, nil

	case "read":
		if name == "" {
			return "Error: name is required for read", nil
		}
		relPath := "SKILL.md"
		if strings.TrimSpace(req.Path) != "" {
			var err error
			relPath, err = normalizeStandardSkillFilePath(req.Path)
			if err != nil {
				return fmt.Sprintf("Error: %v", err), nil
			}
		}
		b, err := svc.ReadFile(ctx, t.key, "/skills/"+name+"/"+relPath)
		if err != nil {
			return fmt.Sprintf("Error reading skill: %v", err), nil
		}
		if len(b) == 0 {
			return fmt.Sprintf("(skill file %s/%s not found or empty)", name, relPath), nil
		}
		return truncateForModel(string(b)), nil

	case "diff":
		if name == "" {
			return "Error: name is required for diff", nil
		}
		if !skillNameRe.MatchString(name) {
			return "Error: name must match [a-z0-9-]+ (lowercase letters, digits, hyphens)", nil
		}
		relPath, err := normalizeStandardSkillFilePath(req.Path)
		if err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		b, err := svc.ReadFile(ctx, t.key, "/skills/"+name+"/"+relPath)
		if err != nil {
			return fmt.Sprintf("Error reading skill file for diff: %v", err), nil
		}
		return renderSkillFileDiff(relPath, string(b), req.Content), nil

	default:
		return "Error: action must be one of create, write_file, edit, remove_file, delete, list, read, diff", nil
	}
}

func normalizeStandardSkillFilePath(p string) (string, error) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if p == "" {
		return "", fmt.Errorf("path is required for write_file")
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return "", fmt.Errorf("path must not contain '..'")
		}
	}
	p = strings.TrimPrefix(path.Clean("/"+strings.TrimLeft(p, "/")), "/")
	if p == "." || p == "" {
		return "", fmt.Errorf("path is required for write_file")
	}
	if p == "SKILL.md" ||
		strings.HasPrefix(p, "scripts/") ||
		strings.HasPrefix(p, "references/") ||
		strings.HasPrefix(p, "templates/") ||
		strings.HasPrefix(p, "assets/") {
		return p, nil
	}
	return "", fmt.Errorf("path must be SKILL.md or under scripts/, references/, templates/, assets/")
}

func skillManageShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func renderSkillFileDiff(relPath, current, proposed string) string {
	if current == proposed {
		return fmt.Sprintf("No changes for %s.", relPath)
	}
	currentLines := splitSkillDiffLines(current)
	proposedLines := splitSkillDiffLines(proposed)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- current/%s\n", relPath))
	b.WriteString(fmt.Sprintf("+++ proposed/%s\n", relPath))
	maxLines := len(currentLines)
	if len(proposedLines) > maxLines {
		maxLines = len(proposedLines)
	}
	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		hasOld := i < len(currentLines)
		hasNew := i < len(proposedLines)
		if hasOld {
			oldLine = currentLines[i]
		}
		if hasNew {
			newLine = proposedLines[i]
		}
		if hasOld && hasNew && oldLine == newLine {
			b.WriteString(" " + oldLine + "\n")
			continue
		}
		if hasOld {
			b.WriteString("-" + oldLine + "\n")
		}
		if hasNew {
			b.WriteString("+" + newLine + "\n")
		}
	}
	return truncateForModel(b.String())
}

func splitSkillDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
