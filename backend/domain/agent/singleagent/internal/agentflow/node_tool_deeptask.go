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

// node_tool_deeptask.go 实现「路线 A」：把 eino 自带的 DeepAgent
// (adk/prebuilt/deep) 通过 adk.NewAgentTool 包成一个 `deep_task` 工具，
// 挂给现有 ReAct agent。模型遇到复杂多步任务时调用 deep_task 委托给 DeepAgent，
// 由它自主规划(write_todos) + 起隔离上下文的子 agent(task) + 复用我们的沙箱工具
// 跑到完成再回灌结果。
//
// 关键：这是 eino 原生支持的「agent-as-tool」路径，不需要 adk<->compose 桥接，
// 不动现有 streaming/interrupt 链路。由 DEEP_TASK_ENABLED 开关控制，默认 OFF，
// 关闭时对普通 agent 零影响。注意：超级 agent（isSuperAgent）会无条件启用 deep_task，
// env 开关只作用于普通 agent。

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

const deepTaskEnvKey = "DEEP_TASK_ENABLED"

// deepTaskEnabled 报告是否启用 deep_task 工具（默认关）。
func deepTaskEnabled() bool {
	v := strings.TrimSpace(os.Getenv(deepTaskEnvKey))
	return v == "1" || strings.EqualFold(v, "true")
}

const deepTaskName = "deep_task"

// deepTaskDescription 是回给模型的工具说明，引导模型在「需要多步、探索、
// 并行调查的复杂任务」时才委托给它。
const deepTaskDescription = "Delegate a complex, multi-step coding or file task to an autonomous sub-agent. " +
	"It plans the work as a todo list, can spawn its own isolated sub-agents, and has full sandbox access " +
	"(run shell commands, read/write/edit files, grep, glob). Use it for tasks that need many steps, " +
	"exploration, or parallel investigation — not for a single quick question. " +
	"Pass one clear, self-contained task description; the sub-agent runs to completion and returns a final summary."

// newDeepTaskTool 构建 DeepAgent 并用 adk.NewAgentTool 包成一个 deep_task 工具。
// DeepAgent 复用同一个 chatModel 和同一批 agentTools（沙箱/插件/工作流…），
// 因此它的子 agent 在同一个 Docker 沙箱里执行。注意：传入的 agentTools 不应包含
// deep_task 自身，否则会递归。
func newDeepTaskTool(
	ctx context.Context,
	chatModel chatmodel.ToolCallingChatModel,
	agentTools []tool.BaseTool,
) (tool.BaseTool, error) {
	cfg := &deep.Config{
		Name:        deepTaskName,
		Description: deepTaskDescription,
		ChatModel:   chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
		},
	}
	da, err := deep.New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return adk.NewAgentTool(ctx, da), nil
}
