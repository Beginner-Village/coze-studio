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
)

type workflowCanvasFakeTool struct{ name string }

func (f workflowCanvasFakeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}

func (f workflowCanvasFakeTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "", nil
}

func TestWorkflowCanvasModeEnabled(t *testing.T) {
	for _, ext := range []map[string]string{
		{workflowCanvasModeExtKey: "true"},
		{workflowCanvasModeExtKey: "1"},
		{workflowCanvasModeExtKey: " yes "},
		{workflowCanvasModeExtKey: "ON"},
	} {
		if !workflowCanvasModeEnabled(ext) {
			t.Fatalf("expected workflow canvas mode for ext=%v", ext)
		}
	}

	for _, ext := range []map[string]string{
		nil,
		{},
		{workflowCanvasModeExtKey: "false"},
		{workflowCanvasModeExtKey: "canvas"},
	} {
		if workflowCanvasModeEnabled(ext) {
			t.Fatalf("did not expect workflow canvas mode for ext=%v", ext)
		}
	}
}

func TestFilterWorkflowCanvasTools(t *testing.T) {
	in := []tool.BaseTool{
		workflowCanvasFakeTool{name: "workflow_canvas_add_node"},
		workflowCanvasFakeTool{name: "update_plan"},
		workflowCanvasFakeTool{name: "web_search"},
		workflowCanvasFakeTool{name: "workflow_canvas_connect"},
	}

	out := filterWorkflowCanvasTools(context.Background(), in)
	if len(out) != 2 {
		t.Fatalf("expected 2 workflow canvas tools, got %d", len(out))
	}

	got := map[string]bool{}
	for _, tl := range out {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info err: %v", err)
		}
		got[info.Name] = true
	}

	for _, want := range []string{"workflow_canvas_add_node", "workflow_canvas_connect"} {
		if !got[want] {
			t.Fatalf("expected %q in filtered tools, got %v", want, got)
		}
	}
	for _, gone := range []string{"update_plan", "web_search"} {
		if got[gone] {
			t.Fatalf("expected %q to be filtered out, got %v", gone, got)
		}
	}
}
