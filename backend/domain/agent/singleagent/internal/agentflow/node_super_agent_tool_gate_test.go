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
