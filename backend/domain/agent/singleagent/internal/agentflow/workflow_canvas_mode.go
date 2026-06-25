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
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

const (
	workflowCanvasModeExtKey = "workflow_canvas_mode"
	workflowCanvasToolPrefix = "workflow_canvas_"
)

func workflowCanvasModeEnabled(ext map[string]string) bool {
	v := strings.ToLower(strings.TrimSpace(ext[workflowCanvasModeExtKey]))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func filterWorkflowCanvasTools(ctx context.Context, tools []tool.BaseTool) []tool.BaseTool {
	out := make([]tool.BaseTool, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil || info == nil || !strings.HasPrefix(info.Name, workflowCanvasToolPrefix) {
			continue
		}
		out = append(out, t)
	}
	return out
}
