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

	"github.com/cloudwego/eino/components/tool"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
)

// gateSuperAgentTools 按超级体能力开关(SuperAgentToolConfig)剔除被关闭的工具。
// tc 为 nil 时全部放行(默认全开，保持向后兼容)。sandboxOff=true(沙箱总开关关闭，
// 即「纯 MCP 模式」)时，依赖沙箱执行的 web_search/web_fetch 一并剔除。
// 该门控只作用于超级体的工具集，普通单智能体不会调用此函数。
func gateSuperAgentTools(ctx context.Context, tools []tool.InvokableTool, tc *crossagent.SuperAgentToolConfig, sandboxOff bool) []tool.InvokableTool {
	blocked := map[string]bool{}
	if sandboxOff || !tc.WebSearchEnabled() {
		blocked["web_search"] = true
	}
	if sandboxOff || !tc.WebFetchEnabled() {
		blocked["web_fetch"] = true
	}
	if !tc.SkillManageEnabled() {
		blocked["skill_manage"] = true
	}
	if !tc.RunBashEnabled() {
		blocked["run_bash"] = true
	}
	if len(blocked) == 0 {
		return tools
	}
	out := make([]tool.InvokableTool, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err == nil && blocked[info.Name] {
			continue
		}
		out = append(out, t)
	}
	return out
}
