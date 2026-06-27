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

package superagenttrace

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	crossMessage "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/message"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
)

func TestBuildDataDerivesHarnessToolAndPlanEvents(t *testing.T) {
	trace := BuildData(
		789,
		[]*agentrunEntity.RunRecordMeta{
			{
				ID:             123,
				ConversationID: 789,
				AgentID:        456,
				Status:         agentrunEntity.RunStatusCompleted,
				CreatedAt:      1000,
				UpdatedAt:      4000,
				CompletedAt:    4000,
			},
		},
		[]*msgEntity.Message{
			{
				ID:             2001,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Content:        `{"function":{"name":"update_plan","arguments":{"plan":[{"content":"inspect","status":"in_progress"}]}}}`,
				Ext: map[string]string{
					"call_id":        "call-plan",
					"tool_name":      "update_plan",
					"plugin_request": `{"plan":[{"content":"inspect","status":"in_progress"}]}`,
				},
				CreatedAt: 1100,
				UpdatedAt: 1100,
			},
			{
				ID:             2002,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Tool,
				MessageType:    crossMessage.MessageTypeToolResponse,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "Plan updated (0/1 done):\n[~] inspect\n",
				Ext: map[string]string{
					"call_id":       "call-plan",
					"plugin_status": "0",
				},
				CreatedAt: 1200,
				UpdatedAt: 1200,
			},
			{
				ID:             2003,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Ext: map[string]string{
					"call_id":        "call-bash",
					"tool_name":      "run_bash",
					"plugin_request": `{"command":"npm test"}`,
				},
				CreatedAt: 1300,
				UpdatedAt: 1300,
			},
			{
				ID:             2004,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Tool,
				MessageType:    crossMessage.MessageTypeToolResponse,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "PASS\n",
				Ext: map[string]string{
					"call_id":       "call-bash",
					"plugin_status": "0",
				},
				CreatedAt: 1400,
				UpdatedAt: 1400,
			},
		},
	)

	planEvent := requireTraceEvent(t, trace.Events, EventPlanUpdated)
	require.Equal(t, "plan", planEvent.Kind)
	require.Equal(t, "123", planEvent.RunID)
	require.Equal(t, "2001", planEvent.MessageID)
	require.Equal(t, "update_plan", planEvent.Metadata["tool_name"])
	require.Contains(t, planEvent.Content, `"status":"in_progress"`)

	toolStarted := requireTraceEvent(t, trace.Events, EventToolStarted)
	require.Equal(t, "tool", toolStarted.Kind)
	require.Equal(t, "update_plan", toolStarted.Metadata["tool_name"])
	require.Equal(t, "call-plan", toolStarted.Metadata["call_id"])
	require.Contains(t, toolStarted.Content, `"plan"`)

	toolCompleted := requireTraceEvent(t, trace.Events, EventToolCompleted)
	require.Equal(t, "tool", toolCompleted.Kind)
	require.Equal(t, "success", toolCompleted.Status)
	require.Equal(t, "call-plan", toolCompleted.Metadata["call_id"])
	require.Equal(t, "update_plan", toolCompleted.Metadata["tool_name"])
	require.Equal(t, "2001", toolCompleted.Metadata["request_message_id"])
	require.Equal(t, "message:2001:tool.started", toolCompleted.Metadata["request_event_id"])
	require.Equal(t, "100", toolCompleted.Metadata["duration_ms"])
	require.Contains(t, toolCompleted.Content, "Plan updated")

	runBashStarted := requireTraceEventWithMetadata(t, trace.Events, EventToolStarted, "call_id", "call-bash")
	require.Equal(t, "run_bash", runBashStarted.Metadata["tool_name"])
	require.Contains(t, runBashStarted.Content, "npm test")

	runBashCompleted := requireTraceEventWithMetadata(t, trace.Events, EventToolCompleted, "call_id", "call-bash")
	require.Equal(t, "success", runBashCompleted.Status)
	require.Equal(t, "run_bash", runBashCompleted.Metadata["tool_name"])
	require.Equal(t, "2003", runBashCompleted.Metadata["request_message_id"])
	require.Equal(t, "message:2003:tool.started", runBashCompleted.Metadata["request_event_id"])
	require.Equal(t, "100", runBashCompleted.Metadata["duration_ms"])
	require.Contains(t, runBashCompleted.Content, "PASS")
}

func TestBuildDataExtractsFunctionCallArgumentsWhenPluginRequestIsMissing(t *testing.T) {
	trace := BuildData(
		789,
		[]*agentrunEntity.RunRecordMeta{
			{
				ID:             123,
				ConversationID: 789,
				AgentID:        456,
				Status:         agentrunEntity.RunStatusCompleted,
				CreatedAt:      1000,
				UpdatedAt:      4000,
				CompletedAt:    4000,
			},
		},
		[]*msgEntity.Message{
			{
				ID:             2001,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Content:        `{"index":0,"id":"call-plan","type":"function","function":{"name":"update_plan","arguments":"{\"plan\":[{\"content\":\"inspect\",\"status\":\"in_progress\"}]}"} }`,
				Ext: map[string]string{
					"call_id": "call-plan",
				},
				CreatedAt: 1100,
				UpdatedAt: 1100,
			},
		},
	)

	toolStarted := requireTraceEvent(t, trace.Events, EventToolStarted)
	require.Equal(t, "update_plan", toolStarted.Metadata["tool_name"])
	require.JSONEq(t, `{"plan":[{"content":"inspect","status":"in_progress"}]}`, toolStarted.Content)

	planEvent := requireTraceEvent(t, trace.Events, EventPlanUpdated)
	require.Equal(t, "plan", planEvent.Kind)
	require.Equal(t, "update_plan", planEvent.Metadata["tool_name"])
	require.JSONEq(t, `{"plan":[{"content":"inspect","status":"in_progress"}]}`, planEvent.Content)
}

func TestBuildDataLinksOffloadedToolOutputPath(t *testing.T) {
	trace := BuildData(
		789,
		[]*agentrunEntity.RunRecordMeta{
			{
				ID:             123,
				ConversationID: 789,
				AgentID:        456,
				Status:         agentrunEntity.RunStatusCompleted,
				CreatedAt:      1000,
				UpdatedAt:      4000,
				CompletedAt:    4000,
			},
		},
		[]*msgEntity.Message{
			{
				ID:             2001,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Ext: map[string]string{
					"call_id":        "call-offload",
					"tool_name":      "run_bash",
					"plugin_request": `{"command":"python3 - <<'PY'\nprint('x'*220000)\nPY"}`,
				},
				CreatedAt: 1100,
				UpdatedAt: 1100,
			},
			{
				ID:             2002,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Tool,
				MessageType:    crossMessage.MessageTypeToolResponse,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "head\n\n...[output truncated; full 123456 bytes saved to /workspace/.agent/tooloutputs/sessions/789/abcd1234ef567890.json — use read_file/grep on that path to view more]...\n\ntail\n",
				Ext: map[string]string{
					"call_id":       "call-offload",
					"plugin_status": "0",
				},
				CreatedAt: 1300,
				UpdatedAt: 1300,
			},
		},
	)

	completed := requireTraceEvent(t, trace.Events, EventToolCompleted)
	require.Equal(t, "run_bash", completed.Metadata["tool_name"])
	require.Equal(t, "2001", completed.Metadata["request_message_id"])
	require.Equal(t, "message:2001:tool.started", completed.Metadata["request_event_id"])
	require.Equal(t, "200", completed.Metadata["duration_ms"])
	require.Equal(t, "/workspace/.agent/tooloutputs/sessions/789/abcd1234ef567890.json", completed.Metadata["tool_output_path"])
	require.Equal(t, "123456", completed.Metadata["tool_output_bytes"])
	require.Equal(t, "true", completed.Metadata["tool_output_offloaded"])
}

func TestBuildDataDerivesFailedToolEventForNonZeroPluginStatus(t *testing.T) {
	trace := BuildData(
		789,
		[]*agentrunEntity.RunRecordMeta{
			{
				ID:             123,
				ConversationID: 789,
				AgentID:        456,
				Status:         agentrunEntity.RunStatusCompleted,
				CreatedAt:      1000,
				UpdatedAt:      4000,
				CompletedAt:    4000,
			},
		},
		[]*msgEntity.Message{
			{
				ID:             2001,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Ext: map[string]string{
					"call_id":        "call-fail",
					"tool_name":      "run_bash",
					"plugin_request": `{"command":"bash -lc 'exit 2'"}`,
				},
				CreatedAt: 1100,
				UpdatedAt: 1100,
			},
			{
				ID:             2002,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Tool,
				MessageType:    crossMessage.MessageTypeToolResponse,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "exit status 2\n",
				Ext: map[string]string{
					"call_id":       "call-fail",
					"plugin_status": "2",
				},
				CreatedAt: 1300,
				UpdatedAt: 1300,
			},
		},
	)

	failed := requireTraceEvent(t, trace.Events, EventToolFailed)
	require.Equal(t, "tool", failed.Kind)
	require.Equal(t, "failed", failed.Status)
	require.Equal(t, "run_bash", failed.Metadata["tool_name"])
	require.Equal(t, "2001", failed.Metadata["request_message_id"])
	require.Equal(t, "message:2001:tool.started", failed.Metadata["request_event_id"])
	require.Equal(t, "200", failed.Metadata["duration_ms"])
	require.Contains(t, failed.Content, "exit status 2")
	requireTraceEventMissingWithMetadata(t, trace.Events, EventToolCompleted, "call_id", "call-fail")
}

func TestBuildDataDerivesFailedRunBashEventFromExitCodeContent(t *testing.T) {
	trace := BuildData(
		789,
		[]*agentrunEntity.RunRecordMeta{
			{
				ID:             123,
				ConversationID: 789,
				AgentID:        456,
				Status:         agentrunEntity.RunStatusCompleted,
				CreatedAt:      1000,
				UpdatedAt:      4000,
				CompletedAt:    4000,
			},
		},
		[]*msgEntity.Message{
			{
				ID:             2001,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeFunctionCall,
				ContentType:    crossMessage.ContentTypeText,
				Ext: map[string]string{
					"call_id":        "call-bash-exit",
					"tool_name":      "run_bash",
					"plugin_request": `{"command":"bash -lc 'exit 2'"}`,
				},
				CreatedAt: 1100,
				UpdatedAt: 1100,
			},
			{
				ID:             2002,
				ConversationID: 789,
				RunID:          123,
				AgentID:        456,
				Role:           schema.Tool,
				MessageType:    crossMessage.MessageTypeToolResponse,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "exit_code: 2\nstdout:\n\nstderr:\n",
				Ext: map[string]string{
					"call_id": "call-bash-exit",
				},
				CreatedAt: 1300,
				UpdatedAt: 1300,
			},
		},
	)

	failed := requireTraceEvent(t, trace.Events, EventToolFailed)
	require.Equal(t, "failed", failed.Status)
	require.Equal(t, "run_bash", failed.Metadata["tool_name"])
	require.Equal(t, "2", failed.Metadata["exit_code"])
	require.Equal(t, "2001", failed.Metadata["request_message_id"])
	require.Equal(t, "message:2001:tool.started", failed.Metadata["request_event_id"])
	require.Equal(t, "200", failed.Metadata["duration_ms"])
	require.Contains(t, failed.Content, "exit_code: 2")
	requireTraceEventMissingWithMetadata(t, trace.Events, EventToolCompleted, "call_id", "call-bash-exit")
}

func requireTraceEvent(t *testing.T, events []Event, eventName string) Event {
	t.Helper()
	for _, event := range events {
		if event.Event == eventName {
			return event
		}
	}
	require.Failf(t, "missing trace event", "event %s not found in %#v", eventName, events)
	return Event{}
}

func requireTraceEventMissingWithMetadata(t *testing.T, events []Event, eventName, key, value string) {
	t.Helper()
	for _, event := range events {
		if event.Event == eventName && event.Metadata[key] == value {
			require.Failf(t, "unexpected trace event", "event %s with %s=%s found in %#v", eventName, key, value, events)
		}
	}
}

func requireTraceEventWithMetadata(t *testing.T, events []Event, eventName, key, value string) Event {
	t.Helper()
	for _, event := range events {
		if event.Event == eventName && event.Metadata[key] == value {
			return event
		}
	}
	require.Failf(t, "missing trace event", "event %s with %s=%s not found in %#v", eventName, key, value, events)
	return Event{}
}
