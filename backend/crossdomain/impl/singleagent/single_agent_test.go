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

package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"

	crossagent "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/agent"
)

func TestBuildSingleAgentStreamExecuteReqPreservesExt(t *testing.T) {
	svc := &impl{}
	req := &crossagent.AgentRuntime{
		AgentID:        123,
		UserID:         "456",
		ConversationID: 789,
		Input:          schema.UserMessage("搭一个客服流程"),
		Ext: map[string]string{
			"workflow_canvas_mode": "true",
			"workflow_id":          "wf-1",
		},
	}

	got := svc.buildSingleAgentStreamExecuteReq(context.Background(), req)
	if got.Ext["workflow_canvas_mode"] != "true" {
		t.Fatalf("expected workflow_canvas_mode ext to be preserved, got %v", got.Ext)
	}
	if got.Ext["workflow_id"] != "wf-1" {
		t.Fatalf("expected workflow_id ext to be preserved, got %v", got.Ext)
	}
}
