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

package coze

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
)

const superAgentApprovalRunPrefix = "run:"
const superAgentApprovalDecisionRoot = "/workspace/.agent/approvals"

type superAgentListApprovalsRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	SpaceID        int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID        int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID          int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	RunID          string `form:"run_id" json:"run_id,omitempty"`
	Status         string `form:"status" json:"status,omitempty"`
	Limit          int32  `form:"limit" json:"limit,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentResolveApprovalRequest struct {
	ApprovalID     string `form:"approval_id" json:"approval_id,omitempty"`
	Decision       string `form:"decision" json:"decision,omitempty"`
	Note           string `form:"note" json:"note,omitempty"`
	RunID          string `form:"run_id" json:"run_id,omitempty"`
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	SpaceID        int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID        int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID          int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentListApprovalsResponse struct {
	Code int                        `json:"code"`
	Msg  string                     `json:"msg"`
	Data superAgentApprovalListData `json:"data"`
}

type superAgentApprovalListData struct {
	ConversationID string                   `json:"conversation_id"`
	RunID          string                   `json:"run_id,omitempty"`
	Approvals      []superAgentApprovalItem `json:"approvals"`
}

type superAgentApprovalItem struct {
	ApprovalID     string                       `json:"approval_id"`
	RunID          string                       `json:"run_id"`
	ConversationID string                       `json:"conversation_id"`
	AgentID        string                       `json:"agent_id"`
	MessageID      string                       `json:"message_id,omitempty"`
	Status         string                       `json:"status"`
	Type           string                       `json:"type"`
	Summary        string                       `json:"summary"`
	RequiredAction *messageModel.RequiredAction `json:"required_action,omitempty"`
	ToolCallIDs    []string                     `json:"tool_call_ids,omitempty"`
	CreatedAt      int64                        `json:"created_at"`
	UpdatedAt      int64                        `json:"updated_at"`
}

type superAgentResolveApprovalResponse struct {
	Code int                           `json:"code"`
	Msg  string                        `json:"msg"`
	Data superAgentResolveApprovalData `json:"data"`
}

type superAgentResolveApprovalData struct {
	ApprovalID   string `json:"approval_id"`
	RunID        string `json:"run_id,omitempty"`
	Decision     string `json:"decision"`
	Status       string `json:"status"`
	Resolved     bool   `json:"resolved"`
	Cancelled    bool   `json:"cancelled"`
	Note         string `json:"note,omitempty"`
	DecisionPath string `json:"decision_path,omitempty"`
	Persisted    bool   `json:"persisted"`
}

// SuperAgentListApprovals lists pending App Server approvals for a conversation.
// @router /api/super-agent/approvals/list [POST]
func SuperAgentListApprovals(ctx context.Context, c *app.RequestContext) {
	var req superAgentListApprovalsRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}

	approvals, err := listSuperAgentPendingApprovals(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentListApprovalsResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentApprovalListData{
			ConversationID: strconv.FormatInt(req.ConversationID, 10),
			RunID:          strings.TrimSpace(req.RunID),
			Approvals:      approvals,
		},
	})
}

func listSuperAgentPendingApprovals(ctx context.Context, req superAgentListApprovalsRequest) ([]superAgentApprovalItem, error) {
	approvals := make([]superAgentApprovalItem, 0)
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "" && status != "pending" {
		return approvals, nil
	}
	if conversation.ConversationSVC.AgentRunDomainSVC == nil {
		return approvals, nil
	}

	limit := req.Limit
	if limit <= 0 || limit > superAgentTraceMaxPageSize {
		limit = superAgentTraceMaxPageSize
	}
	records, err := conversation.ConversationSVC.AgentRunDomainSVC.List(ctx, &agentrunEntity.ListRunRecordMeta{
		ConversationID: req.ConversationID,
		AgentID:        req.AgentID,
		Limit:          limit,
		OrderBy:        "desc",
	})
	if err != nil {
		return nil, err
	}

	filterRunID := strings.TrimSpace(req.RunID)
	pendingRecords := make([]*agentrunEntity.RunRecordMeta, 0, len(records))
	pendingRunIDs := make([]int64, 0, len(records))
	for _, record := range records {
		if record == nil || record.Status != agentrunEntity.RunStatusRequiredAction {
			continue
		}
		runID := strconv.FormatInt(record.ID, 10)
		if filterRunID != "" && filterRunID != runID {
			continue
		}
		pendingRecords = append(pendingRecords, record)
		pendingRunIDs = append(pendingRunIDs, record.ID)
	}

	details, err := loadSuperAgentApprovalDetails(ctx, req.ConversationID, pendingRunIDs)
	if err != nil {
		return nil, err
	}
	for _, record := range pendingRecords {
		runID := strconv.FormatInt(record.ID, 10)
		detail := details[record.ID]
		approvals = append(approvals, superAgentApprovalItem{
			ApprovalID:     superAgentApprovalRunPrefix + runID,
			RunID:          runID,
			ConversationID: strconv.FormatInt(record.ConversationID, 10),
			AgentID:        strconv.FormatInt(record.AgentID, 10),
			MessageID:      detail.MessageID,
			Status:         "pending",
			Type:           "run.required_action",
			Summary:        "Run requires approval before continuing",
			RequiredAction: detail.RequiredAction,
			ToolCallIDs:    detail.ToolCallIDs,
			CreatedAt:      record.CreatedAt,
			UpdatedAt:      record.UpdatedAt,
		})
	}
	return approvals, nil
}

type superAgentApprovalDetail struct {
	MessageID      string
	RequiredAction *messageModel.RequiredAction
	ToolCallIDs    []string
	CreatedAt      int64
}

func loadSuperAgentApprovalDetails(ctx context.Context, conversationID int64, runIDs []int64) (map[int64]superAgentApprovalDetail, error) {
	details := make(map[int64]superAgentApprovalDetail, len(runIDs))
	if len(runIDs) == 0 || conversation.ConversationSVC.MessageDomainSVC == nil {
		return details, nil
	}
	messages, err := conversation.ConversationSVC.MessageDomainSVC.GetByRunIDs(ctx, conversationID, runIDs)
	if err != nil {
		return nil, err
	}
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		detail, ok := buildSuperAgentApprovalDetail(msg)
		if !ok {
			continue
		}
		existing, exists := details[msg.RunID]
		if exists && existing.CreatedAt > detail.CreatedAt {
			continue
		}
		details[msg.RunID] = detail
	}
	return details, nil
}

func buildSuperAgentApprovalDetail(msg *msgEntity.Message) (superAgentApprovalDetail, bool) {
	detail := superAgentApprovalDetail{
		MessageID: strconv.FormatInt(msg.ID, 10),
		CreatedAt: msg.CreatedAt,
	}
	detail.RequiredAction = msg.RequiredAction
	if detail.RequiredAction == nil && msg.Ext != nil {
		if raw := strings.TrimSpace(msg.Ext[string(msgEntity.ExtKeyRequiresAction)]); raw != "" {
			var requiredAction messageModel.RequiredAction
			if err := json.Unmarshal([]byte(raw), &requiredAction); err == nil {
				detail.RequiredAction = &requiredAction
			}
		}
	}
	detail.ToolCallIDs = collectSuperAgentApprovalToolCallIDs(detail.RequiredAction, msg.Ext)
	if detail.RequiredAction == nil && len(detail.ToolCallIDs) == 0 {
		return superAgentApprovalDetail{}, false
	}
	return detail, true
}

func collectSuperAgentApprovalToolCallIDs(requiredAction *messageModel.RequiredAction, ext map[string]string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if requiredAction != nil && requiredAction.SubmitToolOutputs != nil {
		for _, toolCall := range requiredAction.SubmitToolOutputs.ToolCalls {
			if toolCall != nil {
				add(toolCall.ID)
			}
		}
	}
	if ext != nil {
		for _, part := range strings.FieldsFunc(ext[string(msgEntity.ExtKeyToolCallsIDs)], func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
		}) {
			add(part)
		}
	}
	return ids
}

type superAgentApprovalDecisionRecord struct {
	Version        string `json:"version"`
	ApprovalID     string `json:"approval_id"`
	RunID          string `json:"run_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	BotID          string `json:"bot_id,omitempty"`
	Decision       string `json:"decision"`
	Status         string `json:"status"`
	Cancelled      bool   `json:"cancelled"`
	Note           string `json:"note,omitempty"`
	Source         string `json:"source"`
	ResolvedAt     int64  `json:"resolved_at"`
}

// SuperAgentResolveApproval records an App Server approval decision.
// @router /api/super-agent/approvals/resolve [POST]
func SuperAgentResolveApproval(ctx context.Context, c *app.RequestContext) {
	var req superAgentResolveApprovalRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	approvalID := strings.TrimSpace(req.ApprovalID)
	if approvalID == "" {
		invalidParamRequestResponse(c, "approval_id is required")
		return
	}
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != "approve" && decision != "reject" && decision != "cancel" {
		invalidParamRequestResponse(c, "decision must be approve, reject, or cancel")
		return
	}

	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = strings.TrimPrefix(approvalID, superAgentApprovalRunPrefix)
	}
	cancelled := false
	status := "approved"
	switch decision {
	case "reject":
		status = "rejected"
	case "cancel":
		status = "cancelled"
		cancelled = conversation.CancelActiveAgentRun(runID)
	}
	decisionPath, persisted, err := persistSuperAgentApprovalDecision(ctx, req, runID, decision, status, cancelled)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentResolveApprovalResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentResolveApprovalData{
			ApprovalID:   approvalID,
			RunID:        runID,
			Decision:     decision,
			Status:       status,
			Resolved:     true,
			Cancelled:    cancelled,
			Note:         strings.TrimSpace(req.Note),
			DecisionPath: decisionPath,
			Persisted:    persisted,
		},
	})
}

func persistSuperAgentApprovalDecision(ctx context.Context, req superAgentResolveApprovalRequest, runID, decision, status string, cancelled bool) (string, bool, error) {
	if application.SingleAgentSVC == nil {
		return "", false, nil
	}
	agentID := req.AgentID
	if agentID == 0 {
		agentID = req.BotID
	}
	if agentID <= 0 {
		return "", false, nil
	}
	decisionPath := superAgentApprovalDecisionPath(req.ApprovalID, runID)
	record := superAgentApprovalDecisionRecord{
		Version:        "v1",
		ApprovalID:     strings.TrimSpace(req.ApprovalID),
		RunID:          strings.TrimSpace(runID),
		ConversationID: formatSuperAgentOptionalID(req.ConversationID),
		AgentID:        formatSuperAgentOptionalID(agentID),
		BotID:          formatSuperAgentOptionalID(req.BotID),
		Decision:       decision,
		Status:         status,
		Cancelled:      cancelled,
		Note:           strings.TrimSpace(req.Note),
		Source:         "app_server",
		ResolvedAt:     time.Now().UnixMilli(),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return "", false, err
	}
	_, err = application.SingleAgentSVC.UploadSuperAgentWorkspaceFile(ctx, &developer_api.UploadSandboxFileRequest{
		SpaceID:  req.SpaceID,
		AgentID:  agentID,
		BotID:    req.BotID,
		Path:     decisionPath,
		Content:  string(payload),
		Encoding: "utf-8",
	})
	if err != nil {
		return "", false, err
	}
	return decisionPath, true, nil
}

func superAgentApprovalDecisionPath(approvalID, runID string) string {
	name := strings.TrimSpace(approvalID)
	if name == "" {
		name = strings.TrimSpace(runID)
	}
	name = sanitizeSuperAgentApprovalDecisionFilename(name)
	if name == "" {
		name = "approval"
	}
	return superAgentApprovalDecisionRoot + "/" + name + ".json"
}

func sanitizeSuperAgentApprovalDecisionFilename(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(name) {
		allowed := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == '-'
		if allowed {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), "-.")
}
