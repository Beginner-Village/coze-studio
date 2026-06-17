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

// node_tool_memory.go 实现「路线 P4」：超级智能体的**长期记忆**。
//
// 记忆以 JSON 字符串数组的形式持久化在沙箱文件 /workspace/.agent/memory.json 里，
// 跨会话存活。memory_save 追加一条事实/偏好，memory_recall 列出全部已记内容。
// 两个工具都通过 registerSuperAgentExtension 注册,只挂给超级 agent。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

const memoryFilePath = "/workspace/.agent/memory.json"

func init() {
	registerSuperAgentExtension(func(key string) tool.InvokableTool {
		return &memorySaveTool{key: key}
	})
	registerSuperAgentExtension(func(key string) tool.InvokableTool {
		return &memoryRecallTool{key: key}
	})
}

// loadMemories 读取并解析记忆文件；文件不存在/为空时返回空切片。
func loadMemories(ctx context.Context, svc crosssandbox.Manager, key string) ([]string, error) {
	b, err := svc.ReadFile(ctx, key, memoryFilePath)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return []string{}, nil
	}
	var mems []string
	if err := json.Unmarshal(b, &mems); err != nil {
		// 文件损坏时当作空记忆,避免阻塞。
		return []string{}, nil
	}
	return mems, nil
}

// ---- memory_save ----

type memorySaveTool struct{ key string }

type memorySaveRequest struct {
	Content string `json:"content" jsonschema:"description=The fact or preference to remember"`
}

func (t *memorySaveTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "memory_save",
		Desc: "Save a fact or preference about the user to long-term memory so it persists across conversations. Pass one concise statement per call.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {Type: schema.String, Desc: "The fact or preference to remember", Required: true},
		}),
	}, nil
}

func (t *memorySaveTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req memorySaveRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "Error: content must not be empty", nil
	}
	mems, err := loadMemories(ctx, svc, t.key)
	if err != nil {
		return fmt.Sprintf("Error reading memory: %v", err), nil
	}
	mems = append(mems, content)
	out, err := json.Marshal(mems)
	if err != nil {
		return "", fmt.Errorf("failed to marshal memory: %w", err)
	}
	if err := svc.WriteFile(ctx, t.key, memoryFilePath, out); err != nil {
		return fmt.Sprintf("Error saving memory: %v", err), nil
	}
	return fmt.Sprintf("Saved. Now %d memories.", len(mems)), nil
}

// ---- memory_recall ----

type memoryRecallTool struct{ key string }

type memoryRecallRequest struct {
	Query string `json:"query" jsonschema:"description=Optional query (currently ignored; all memories are returned)"`
}

func (t *memoryRecallTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "memory_recall",
		Desc: "Recall everything saved to long-term memory about the user. Call this at the start of a conversation to load what you already know.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "Optional query (currently ignored; all memories are returned)", Required: false},
		}),
	}, nil
}

func (t *memoryRecallTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	mems, err := loadMemories(ctx, svc, t.key)
	if err != nil {
		return fmt.Sprintf("Error reading memory: %v", err), nil
	}
	if len(mems) == 0 {
		return "(no memories yet)", nil
	}
	var sb strings.Builder
	for _, m := range mems {
		sb.WriteString("- ")
		sb.WriteString(m)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
