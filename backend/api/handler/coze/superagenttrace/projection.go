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
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	crossMessage "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/message"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
)

const (
	EventToolStarted      = "tool.started"
	EventToolCompleted    = "tool.completed"
	EventToolFailed       = "tool.failed"
	EventPlanUpdated      = "plan.updated"
	EventContextCompacted = "context.compacted"
)

var toolOutputOffloadPattern = regexp.MustCompile(`full\s+([0-9]+)\s+bytes\s+saved\s+to\s+(/workspace/\.agent/tooloutputs/[^\s\]]+\.json)`)

type functionCallPayload struct {
	ID       string `json:"id"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type projectionContext struct {
	toolCalls map[string]toolCallContext
}

type toolCallContext struct {
	ToolName         string
	Plugin           string
	RequestMessageID string
	RequestEventID   string
	CreatedAt        int64
}

type Data struct {
	ConversationID string  `json:"conversation_id"`
	Runs           []Run   `json:"runs"`
	Events         []Event `json:"events"`
}

type Run struct {
	RunID          string                   `json:"run_id"`
	ConversationID string                   `json:"conversation_id"`
	AgentID        string                   `json:"agent_id"`
	Status         string                   `json:"status"`
	Error          *agentrunEntity.RunError `json:"error,omitempty"`
	CreatedAt      int64                    `json:"created_at"`
	UpdatedAt      int64                    `json:"updated_at"`
	CompletedAt    int64                    `json:"completed_at,omitempty"`
	FailedAt       int64                    `json:"failed_at,omitempty"`
}

type Event struct {
	ID             string            `json:"id"`
	Event          string            `json:"event"`
	Kind           string            `json:"kind"`
	RunID          string            `json:"run_id,omitempty"`
	MessageID      string            `json:"message_id,omitempty"`
	ConversationID string            `json:"conversation_id"`
	AgentID        string            `json:"agent_id,omitempty"`
	Role           string            `json:"role,omitempty"`
	Type           string            `json:"type,omitempty"`
	Content        string            `json:"content,omitempty"`
	ContentType    string            `json:"content_type,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Status         string            `json:"status,omitempty"`
	CreatedAt      int64             `json:"created_at"`
	UpdatedAt      int64             `json:"updated_at,omitempty"`
}

func BuildData(conversationID int64, runRecords []*agentrunEntity.RunRecordMeta, messages []*msgEntity.Message) *Data {
	data := &Data{
		ConversationID: strconv.FormatInt(conversationID, 10),
		Runs:           make([]Run, 0, len(runRecords)),
		Events:         make([]Event, 0, len(runRecords)*2+len(messages)*2),
	}
	for _, runRecord := range runRecords {
		if runRecord == nil {
			continue
		}
		data.Runs = append(data.Runs, buildRun(runRecord))
		data.Events = append(data.Events, buildRunEvents(runRecord)...)
	}
	ctx := buildProjectionContext(messages)
	for _, message := range messages {
		data.Events = append(data.Events, buildMessageEvents(ctx, message)...)
	}
	sort.SliceStable(data.Events, func(i, j int) bool {
		if data.Events[i].CreatedAt == data.Events[j].CreatedAt {
			return eventPriority(data.Events[i].Event) < eventPriority(data.Events[j].Event)
		}
		return data.Events[i].CreatedAt < data.Events[j].CreatedAt
	})
	return data
}

func buildRun(runRecord *agentrunEntity.RunRecordMeta) Run {
	return Run{
		RunID:          strconv.FormatInt(runRecord.ID, 10),
		ConversationID: strconv.FormatInt(runRecord.ConversationID, 10),
		AgentID:        strconv.FormatInt(runRecord.AgentID, 10),
		Status:         string(runRecord.Status),
		Error:          runRecord.Error,
		CreatedAt:      runRecord.CreatedAt,
		UpdatedAt:      runRecord.UpdatedAt,
		CompletedAt:    runRecord.CompletedAt,
		FailedAt:       runRecord.FailedAt,
	}
}

func buildRunEvents(runRecord *agentrunEntity.RunRecordMeta) []Event {
	events := []Event{
		{
			ID:             "run:" + strconv.FormatInt(runRecord.ID, 10) + ":created",
			Event:          string(agentrunEntity.RunEventCreated),
			Kind:           "run",
			RunID:          strconv.FormatInt(runRecord.ID, 10),
			ConversationID: strconv.FormatInt(runRecord.ConversationID, 10),
			AgentID:        strconv.FormatInt(runRecord.AgentID, 10),
			Status:         string(runRecord.Status),
			CreatedAt:      runRecord.CreatedAt,
			UpdatedAt:      runRecord.UpdatedAt,
		},
	}
	if terminal := terminalRunEvent(runRecord); terminal != "" {
		createdAt := runRecord.UpdatedAt
		if runRecord.CompletedAt > 0 {
			createdAt = runRecord.CompletedAt
		}
		if runRecord.FailedAt > 0 {
			createdAt = runRecord.FailedAt
		}
		events = append(events, Event{
			ID:             "run:" + strconv.FormatInt(runRecord.ID, 10) + ":" + string(runRecord.Status),
			Event:          terminal,
			Kind:           "run",
			RunID:          strconv.FormatInt(runRecord.ID, 10),
			ConversationID: strconv.FormatInt(runRecord.ConversationID, 10),
			AgentID:        strconv.FormatInt(runRecord.AgentID, 10),
			Status:         string(runRecord.Status),
			CreatedAt:      createdAt,
			UpdatedAt:      runRecord.UpdatedAt,
		})
	}
	return events
}

func terminalRunEvent(runRecord *agentrunEntity.RunRecordMeta) string {
	switch runRecord.Status {
	case agentrunEntity.RunStatusCompleted:
		return string(agentrunEntity.RunEventCompleted)
	case agentrunEntity.RunStatusFailed:
		return string(agentrunEntity.RunEventFailed)
	case agentrunEntity.RunStatusCancelled:
		return string(agentrunEntity.RunEventCancelled)
	case agentrunEntity.RunStatusExpired:
		return string(agentrunEntity.RunEventExpired)
	case agentrunEntity.RunStatusRequiredAction:
		return string(agentrunEntity.RunEventRequiredAction)
	default:
		return ""
	}
}

func buildProjectionContext(messages []*msgEntity.Message) projectionContext {
	ctx := projectionContext{
		toolCalls: map[string]toolCallContext{},
	}
	for _, message := range messages {
		if message == nil || message.MessageType != crossMessage.MessageTypeFunctionCall {
			continue
		}
		callID := strings.TrimSpace(toolMetadata(message)["call_id"])
		if callID == "" {
			continue
		}
		ctx.toolCalls[callID] = toolCallContext{
			ToolName:         toolName(message),
			Plugin:           strings.TrimSpace(message.Ext["plugin"]),
			RequestMessageID: strconv.FormatInt(message.ID, 10),
			RequestEventID:   toolStartedEventID(message.ID),
			CreatedAt:        message.CreatedAt,
		}
	}
	return ctx
}

func buildMessageEvents(ctx projectionContext, message *msgEntity.Message) []Event {
	if message == nil || message.MessageType == crossMessage.MessageTypeVerbose {
		return nil
	}
	events := make([]Event, 0, 3)
	if generic := buildGenericMessageEvent(message); generic != nil {
		events = append(events, *generic)
	}
	switch message.MessageType {
	case crossMessage.MessageTypeFunctionCall:
		events = append(events, buildToolStartedEvent(message))
		if toolName(message) == "update_plan" {
			events = append(events, buildPlanUpdatedEvent(message))
		}
	case crossMessage.MessageTypeToolResponse:
		events = append(events, buildToolResponseEvent(ctx, message))
	}
	return events
}

func buildGenericMessageEvent(message *msgEntity.Message) *Event {
	event := string(agentrunEntity.RunEventMessageCompleted)
	if message.MessageType == crossMessage.MessageTypeQuestion || message.MessageType == crossMessage.MessageTypeAck {
		event = string(agentrunEntity.RunEventAck)
	}
	return &Event{
		ID:             "message:" + strconv.FormatInt(message.ID, 10),
		Event:          event,
		Kind:           "message",
		RunID:          strconv.FormatInt(message.RunID, 10),
		MessageID:      strconv.FormatInt(message.ID, 10),
		ConversationID: strconv.FormatInt(message.ConversationID, 10),
		AgentID:        strconv.FormatInt(message.AgentID, 10),
		Role:           string(message.Role),
		Type:           string(message.MessageType),
		Content:        message.Content,
		ContentType:    string(message.ContentType),
		Metadata:       copyMetadata(message.Ext),
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}

func buildToolStartedEvent(message *msgEntity.Message) Event {
	return Event{
		ID:             toolStartedEventID(message.ID),
		Event:          EventToolStarted,
		Kind:           "tool",
		RunID:          strconv.FormatInt(message.RunID, 10),
		MessageID:      strconv.FormatInt(message.ID, 10),
		ConversationID: strconv.FormatInt(message.ConversationID, 10),
		AgentID:        strconv.FormatInt(message.AgentID, 10),
		Role:           string(message.Role),
		Type:           string(message.MessageType),
		Content:        toolRequestContent(message),
		ContentType:    string(message.ContentType),
		Metadata:       toolMetadata(message),
		Status:         "running",
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}

func buildPlanUpdatedEvent(message *msgEntity.Message) Event {
	return Event{
		ID:             "message:" + strconv.FormatInt(message.ID, 10) + ":plan.updated",
		Event:          EventPlanUpdated,
		Kind:           "plan",
		RunID:          strconv.FormatInt(message.RunID, 10),
		MessageID:      strconv.FormatInt(message.ID, 10),
		ConversationID: strconv.FormatInt(message.ConversationID, 10),
		AgentID:        strconv.FormatInt(message.AgentID, 10),
		Role:           string(message.Role),
		Type:           string(message.MessageType),
		Content:        toolRequestContent(message),
		ContentType:    string(message.ContentType),
		Metadata:       toolMetadata(message),
		Status:         "updated",
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}

func buildToolResponseEvent(ctx projectionContext, message *msgEntity.Message) Event {
	status := "success"
	event := EventToolCompleted
	if isFailedToolResponse(message.Ext, message.Content) {
		status = "failed"
		event = EventToolFailed
	}
	metadata := toolResponseMetadata(ctx, message)
	return Event{
		ID:             "message:" + strconv.FormatInt(message.ID, 10) + ":" + event,
		Event:          event,
		Kind:           "tool",
		RunID:          strconv.FormatInt(message.RunID, 10),
		MessageID:      strconv.FormatInt(message.ID, 10),
		ConversationID: strconv.FormatInt(message.ConversationID, 10),
		AgentID:        strconv.FormatInt(message.AgentID, 10),
		Role:           string(message.Role),
		Type:           string(message.MessageType),
		Content:        message.Content,
		ContentType:    string(message.ContentType),
		Metadata:       metadata,
		Status:         status,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
	}
}

func toolStartedEventID(messageID int64) string {
	return "message:" + strconv.FormatInt(messageID, 10) + ":tool.started"
}

func toolResponseMetadata(ctx projectionContext, message *msgEntity.Message) map[string]string {
	metadata := copyMetadata(message.Ext)
	metadata = addToolOutputOffloadMetadata(metadata, message.Content)
	callID := strings.TrimSpace(metadata["call_id"])
	if callID == "" {
		return metadata
	}
	call, ok := ctx.toolCalls[callID]
	if !ok {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	if call.ToolName != "" && strings.TrimSpace(metadata["tool_name"]) == "" {
		metadata["tool_name"] = call.ToolName
	}
	if call.Plugin != "" && strings.TrimSpace(metadata["plugin"]) == "" {
		metadata["plugin"] = call.Plugin
	}
	if call.RequestMessageID != "" {
		metadata["request_message_id"] = call.RequestMessageID
	}
	if call.RequestEventID != "" {
		metadata["request_event_id"] = call.RequestEventID
	}
	if call.CreatedAt > 0 && message.CreatedAt >= call.CreatedAt {
		metadata["duration_ms"] = strconv.FormatInt(message.CreatedAt-call.CreatedAt, 10)
	}
	if exitCode, ok := toolResponseExitCode(message.Content); ok {
		metadata["exit_code"] = exitCode
	}
	return metadata
}

func addToolOutputOffloadMetadata(metadata map[string]string, content string) map[string]string {
	path, bytesValue := toolOutputOffloadReference(content)
	if path == "" {
		return metadata
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["tool_output_path"] = path
	metadata["tool_output_offloaded"] = "true"
	if bytesValue != "" {
		metadata["tool_output_bytes"] = bytesValue
	}
	return metadata
}

func toolOutputOffloadReference(content string) (string, string) {
	matches := toolOutputOffloadPattern.FindStringSubmatch(content)
	if len(matches) != 3 {
		return "", ""
	}
	return matches[2], matches[1]
}

func toolRequestContent(message *msgEntity.Message) string {
	if value := strings.TrimSpace(message.Ext["plugin_request"]); value != "" {
		return value
	}
	if _, arguments, _ := extractFunctionCall(message.Content); arguments != "" {
		return arguments
	}
	return message.Content
}

func toolName(message *msgEntity.Message) string {
	if value := strings.TrimSpace(message.Ext["tool_name"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(message.Ext["plugin"]); value != "" {
		return value
	}
	name, _, _ := extractFunctionCall(message.Content)
	return name
}

func toolMetadata(message *msgEntity.Message) map[string]string {
	metadata := copyMetadata(message.Ext)
	name, _, callID := extractFunctionCall(message.Content)
	if name != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		if strings.TrimSpace(metadata["tool_name"]) == "" {
			metadata["tool_name"] = name
		}
	}
	if callID != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		if strings.TrimSpace(metadata["call_id"]) == "" {
			metadata["call_id"] = callID
		}
	}
	return metadata
}

func extractFunctionCall(content string) (name string, arguments string, callID string) {
	var payload functionCallPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return "", "", ""
	}
	name = strings.TrimSpace(payload.Function.Name)
	callID = strings.TrimSpace(payload.ID)
	arguments = compactFunctionArguments(payload.Function.Arguments)
	return name, arguments, callID
}

func compactFunctionArguments(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err == nil {
		return compacted.String()
	}
	return string(raw)
}

func isFailedToolResponse(metadata map[string]string, content string) bool {
	status := strings.ToLower(strings.TrimSpace(metadata["plugin_status"]))
	if status != "" && status != "0" && status != "success" && status != "succeeded" && status != "ok" {
		return true
	}
	if exitCode, ok := toolResponseExitCode(content); ok && exitCode != "0" {
		return true
	}
	return false
}

func toolResponseExitCode(content string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "exit_code" {
			continue
		}
		exitCode := strings.TrimSpace(value)
		if exitCode == "" {
			return "", false
		}
		return exitCode, true
	}
	return "", false
}

func copyMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}

func eventPriority(event string) int {
	switch event {
	case string(agentrunEntity.RunEventCreated):
		return 0
	case string(agentrunEntity.RunEventAck):
		return 1
	case EventToolStarted:
		return 2
	case EventPlanUpdated:
		return 3
	case EventToolCompleted, EventToolFailed:
		return 4
	case string(agentrunEntity.RunEventMessageCompleted):
		return 5
	default:
		return 6
	}
}
