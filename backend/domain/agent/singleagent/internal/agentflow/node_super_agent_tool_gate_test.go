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
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
)

type gateFakeTool struct{ name string }

func (f gateFakeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}
func (f gateFakeTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "", nil
}

func gateNames(t *testing.T, tc *crossagent.SuperAgentToolConfig, sandboxOff bool) map[string]bool {
	t.Helper()
	in := []tool.InvokableTool{
		gateFakeTool{"run_bash"}, gateFakeTool{"read_file"},
		gateFakeTool{"web_search"}, gateFakeTool{"web_fetch"},
		gateFakeTool{"skill_manage"},
	}
	out := gateSuperAgentTools(context.Background(), in, tc, sandboxOff)
	names := map[string]bool{}
	for _, tl := range out {
		info, _ := tl.Info(context.Background())
		names[info.Name] = true
	}
	return names
}

func bptr(b bool) *bool { return &b }

func TestGateSuperAgentTools_DefaultAllOpen(t *testing.T) {
	// nil config 与全 nil 字段都应放行全部工具(向后兼容、默认全开)。
	for _, tc := range []*crossagent.SuperAgentToolConfig{nil, {}} {
		names := gateNames(t, tc, false)
		for _, want := range []string{"run_bash", "read_file", "web_search", "web_fetch", "skill_manage"} {
			if !names[want] {
				t.Fatalf("default-all-open must keep %q; got %v", want, names)
			}
		}
	}
}

func TestGateSuperAgentTools_PerToolSwitches(t *testing.T) {
	tc := &crossagent.SuperAgentToolConfig{
		WebSearch:   bptr(false),
		RunBash:     bptr(false),
		SkillManage: bptr(false),
	}
	names := gateNames(t, tc, false)
	for _, gone := range []string{"web_search", "run_bash", "skill_manage"} {
		if names[gone] {
			t.Fatalf("disabled tool %q must be removed; got %v", gone, names)
		}
	}
	// 未关闭的保留。
	for _, kept := range []string{"read_file", "web_fetch"} {
		if !names[kept] {
			t.Fatalf("enabled tool %q must remain; got %v", kept, names)
		}
	}
}

func TestGateSuperAgentTools_PureMCPModeDropsWeb(t *testing.T) {
	// 沙箱总开关关闭(纯 MCP 模式)时，依赖沙箱的 web_search/web_fetch 必须剔除，
	// 即使它们各自的开关是开启状态。
	tc := &crossagent.SuperAgentToolConfig{WebSearch: bptr(true), WebFetch: bptr(true)}
	names := gateNames(t, tc, true)
	for _, gone := range []string{"web_search", "web_fetch"} {
		if names[gone] {
			t.Fatalf("pure-MCP mode must drop %q; got %v", gone, names)
		}
	}
}

func TestGateSuperAgentTools_SandboxOffDropsSkillManage(t *testing.T) {
	// 沙箱关闭(现场未配沙箱 / 纯 MCP)时，100% 依赖沙箱的 skill_manage 必须剔除，
	// 即使其开关处于开启状态。
	tc := &crossagent.SuperAgentToolConfig{SkillManage: bptr(true)}
	names := gateNames(t, tc, true)
	if names["skill_manage"] {
		t.Fatalf("sandboxOff must drop skill_manage; got %v", names)
	}
	// 沙箱开启且 skill_manage 开 -> 保留。
	if !gateNames(t, tc, false)["skill_manage"] {
		t.Fatalf("skill_manage must remain when sandbox on and enabled")
	}
}

func TestShouldMountDeepTask(t *testing.T) {
	// 普通体永不挂 deep_task。
	if shouldMountDeepTask(false, false, false, true) {
		t.Fatal("normal agent must never mount deep_task")
	}
	// workflow-canvas 模式永不挂。
	if shouldMountDeepTask(true, true, false, true) {
		t.Fatal("workflow-canvas mode must not mount deep_task")
	}
	// 超级体 + deep_task 开 + 沙箱开 -> 挂。
	if !shouldMountDeepTask(true, false, false, true) {
		t.Fatal("super agent with deep_task enabled and sandbox on should mount")
	}
	// 超级体 + deep_task 关 -> 不挂。
	if shouldMountDeepTask(true, false, false, false) {
		t.Fatal("deep_task disabled must not mount")
	}
	// 超级体 + deep_task 开 但沙箱总开关关(纯 MCP 模式) -> 不挂：
	// 子代理继承的工具集此时既无沙箱工具、也不含尚未注入的 MCP 工具，暴露 deep_task 会误导模型。
	if shouldMountDeepTask(true, false, true, true) {
		t.Fatal("pure-MCP mode (sandboxOff) must not mount deep_task")
	}
}
