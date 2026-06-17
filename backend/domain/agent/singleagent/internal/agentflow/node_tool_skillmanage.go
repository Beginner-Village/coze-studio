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
// 超级 agent 可以把一段带 frontmatter 的 Markdown 写成 /skills/<name>/SKILL.md,
// 沉淀成可复用的能力,之后再 list/read 取用。工具通过 registerSuperAgentExtension
// 注册,只挂给超级 agent。

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9-]+$`)

func init() {
	registerSuperAgentExtension(func(key string) tool.InvokableTool {
		return &skillManageTool{key: key}
	})
}

type skillManageTool struct{ key string }

type skillManageRequest struct {
	Action  string `json:"action" jsonschema:"description=One of: create, list, read"`
	Name    string `json:"name" jsonschema:"description=Skill name (lowercase letters, digits, hyphens)"`
	Content string `json:"content" jsonschema:"description=SKILL.md content (Markdown with frontmatter) when action=create"`
}

func (t *skillManageTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "skill_manage",
		Desc: "Create, list, or read your own reusable skills (each is a SKILL.md under /skills/<name>). " +
			"action=create writes content as /skills/<name>/SKILL.md (content should be Markdown with frontmatter); " +
			"action=list lists existing skill names; action=read returns a skill's SKILL.md. " +
			"Use this to author a capability once and reuse it later.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":  {Type: schema.String, Desc: "One of: create, list, read", Required: true},
			"name":    {Type: schema.String, Desc: "Skill name (lowercase letters, digits, hyphens). Required for create/read", Required: false},
			"content": {Type: schema.String, Desc: "SKILL.md content (Markdown with frontmatter) when action=create", Required: false},
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
		return "", fmt.Errorf("failed to parse arguments: %w", err)
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

	case "list":
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
		b, err := svc.ReadFile(ctx, t.key, "/skills/"+name+"/SKILL.md")
		if err != nil {
			return fmt.Sprintf("Error reading skill: %v", err), nil
		}
		if len(b) == 0 {
			return fmt.Sprintf("(skill %s not found or empty)", name), nil
		}
		return truncateForModel(string(b)), nil

	default:
		return "Error: action must be one of create, list, read", nil
	}
}
