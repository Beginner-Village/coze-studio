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
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze/superagenttrace"
	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

type superAgentHarnessSnapshotRequest struct {
	SpaceID                  int64   `form:"space_id" json:"space_id,string,omitempty"`
	AgentID                  int64   `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID                    int64   `form:"bot_id" json:"bot_id,string,omitempty"`
	ConnectorID              *string `form:"connector_id" json:"connector_id,omitempty"`
	ConversationID           int64   `form:"conversation_id" json:"conversation_id,string,omitempty"`
	RunID                    *int64  `form:"run_id" json:"run_id,string,omitempty"`
	TraceLimit               int32   `form:"trace_limit" json:"trace_limit,omitempty"`
	RunLimit                 int32   `form:"run_limit" json:"run_limit,omitempty"`
	MessageLimit             int32   `form:"message_limit" json:"message_limit,omitempty"`
	ArtifactLimit            int32   `form:"artifact_limit" json:"artifact_limit,omitempty"`
	WorkspacePath            string  `form:"workspace_path" json:"workspace_path,omitempty"`
	WorkspaceRecursive       bool    `form:"workspace_recursive" json:"workspace_recursive,omitempty"`
	IncludeHarness           *bool   `form:"include_harness" json:"include_harness,omitempty"`
	IncludeMessages          *bool   `form:"include_messages" json:"include_messages,omitempty"`
	IncludeRuns              *bool   `form:"include_runs" json:"include_runs,omitempty"`
	IncludeWorkspace         *bool   `form:"include_workspace" json:"include_workspace,omitempty"`
	IncludeContext           *bool   `form:"include_context" json:"include_context,omitempty"`
	IncludeToolOutputs       *bool   `form:"include_tool_outputs" json:"include_tool_outputs,omitempty"`
	IncludeToolOutputContent *bool   `form:"include_tool_output_content" json:"include_tool_output_content,omitempty"`
	IncludeArtifacts         *bool   `form:"include_artifacts" json:"include_artifacts,omitempty"`
	IncludeTrace             *bool   `form:"include_trace" json:"include_trace,omitempty"`
	IncludeApprovals         *bool   `form:"include_approvals" json:"include_approvals,omitempty"`
	IncludeResume            *bool   `form:"include_resume" json:"include_resume,omitempty"`
}

type superAgentHarnessSnapshotResponse struct {
	Code int                            `json:"code"`
	Msg  string                         `json:"msg"`
	Data *superAgentHarnessSnapshotData `json:"data"`
}

type superAgentHarnessResumeResponse struct {
	Code int                          `json:"code"`
	Msg  string                       `json:"msg"`
	Data *superAgentHarnessResumeData `json:"data"`
}

type superAgentHarnessSnapshotData struct {
	Components        []string                                      `json:"components"`
	AgentID           string                                        `json:"agent_id,omitempty"`
	ConversationID    string                                        `json:"conversation_id,omitempty"`
	Messages          *superAgentMessageListData                    `json:"messages,omitempty"`
	Runs              *superAgentRunListData                        `json:"runs,omitempty"`
	Workspace         *developer_api.ListSandboxFilesData           `json:"workspace,omitempty"`
	Harness           *developer_api.SuperAgentHarnessStateData     `json:"harness,omitempty"`
	Context           *developer_api.SuperAgentHarnessContextState  `json:"context,omitempty"`
	ToolOutputs       *application.SuperAgentHarnessToolOutputsData `json:"tool_outputs,omitempty"`
	Artifacts         *application.SuperAgentArtifactListData       `json:"artifacts,omitempty"`
	Trace             *superagenttrace.Data                         `json:"trace,omitempty"`
	Approvals         []superAgentApprovalItem                      `json:"approvals,omitempty"`
	ApprovalDecisions []superAgentApprovalDecisionSnapshotItem      `json:"approval_decisions,omitempty"`
	Resume            *superAgentHarnessResumeData                  `json:"resume,omitempty"`
}

type superAgentHarnessResumeData struct {
	Version            string   `json:"version"`
	ConversationID     string   `json:"conversation_id,omitempty"`
	AgentID            string   `json:"agent_id,omitempty"`
	Summary            string   `json:"summary,omitempty"`
	SummaryPath        string   `json:"summary_path,omitempty"`
	SummaryExists      bool     `json:"summary_exists"`
	PlanPath           string   `json:"plan_path,omitempty"`
	PlanExists         bool     `json:"plan_exists"`
	PlanStatus         string   `json:"plan_status,omitempty"`
	PlanItemCount      int      `json:"plan_item_count,omitempty"`
	RecentMessageCount int      `json:"recent_message_count,omitempty"`
	MessageIDs         []string `json:"message_ids,omitempty"`
	ToolOutputRoot     string   `json:"tool_output_root,omitempty"`
	ToolOutputCount    int      `json:"tool_output_count,omitempty"`
	ToolOutputPaths    []string `json:"tool_output_paths,omitempty"`
	ArtifactRoot       string   `json:"artifact_root,omitempty"`
	ArtifactCount      int      `json:"artifact_count,omitempty"`
	ArtifactPaths      []string `json:"artifact_paths,omitempty"`
	TraceEventCount    int      `json:"trace_event_count,omitempty"`
	ApprovalCount      int      `json:"approval_count,omitempty"`
	Components         []string `json:"components,omitempty"`
	Prompt             string   `json:"prompt"`
}

type superAgentApprovalDecisionSnapshotItem struct {
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
	DecisionPath   string `json:"decision_path"`
	Size           int64  `json:"size,omitempty"`
	Mtime          int64  `json:"mtime,omitempty"`
}

// SuperAgentGetHarnessState .
// @router /api/super-agent/harness/state [POST]
func SuperAgentGetHarnessState(ctx context.Context, c *app.RequestContext) {
	var req developer_api.SuperAgentHarnessStateRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.GetSuperAgentHarnessState(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentClearHarnessContext .
// @router /api/super-agent/harness/context/clear [POST]
func SuperAgentClearHarnessContext(ctx context.Context, c *app.RequestContext) {
	var req developer_api.SuperAgentHarnessContextClearRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ClearSuperAgentHarnessContext(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentUpdateHarnessPlan .
// @router /api/super-agent/harness/plan [POST]
func SuperAgentUpdateHarnessPlan(ctx context.Context, c *app.RequestContext) {
	var req developer_api.SuperAgentHarnessPlanUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.UpdateSuperAgentHarnessPlan(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentListHarnessToolOutputs .
// @router /api/super-agent/harness/tool-outputs [POST]
func SuperAgentListHarnessToolOutputs(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentHarnessToolOutputsRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ListSuperAgentHarnessToolOutputs(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentCleanupHarnessToolOutputs .
// @router /api/super-agent/harness/cleanup [POST]
func SuperAgentCleanupHarnessToolOutputs(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentHarnessCleanupRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.CleanupSuperAgentHarnessToolOutputs(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentGetHarnessSnapshot returns a workbench snapshot for App Server clients.
// @router /api/super-agent/harness/snapshot [POST]
func SuperAgentGetHarnessSnapshot(ctx context.Context, c *app.RequestContext) {
	var req superAgentHarnessSnapshotRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 && snapshotAgentID(req) <= 0 {
		invalidParamRequestResponse(c, "conversation_id or agent_id is required")
		return
	}
	resp, err := buildSuperAgentHarnessSnapshot(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, &superAgentHarnessSnapshotResponse{
		Code: 0,
		Msg:  "success",
		Data: resp,
	})
}

// SuperAgentGetHarnessResume returns a lightweight session resume handoff for App Server clients.
// @router /api/super-agent/harness/resume [POST]
func SuperAgentGetHarnessResume(ctx context.Context, c *app.RequestContext) {
	var req superAgentHarnessSnapshotRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	applySuperAgentHarnessResumeDefaults(&req)
	snapshot, err := buildSuperAgentHarnessSnapshot(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	resume := snapshot.Resume
	if resume == nil {
		resume = buildSuperAgentHarnessResume(snapshot)
	}
	c.JSON(consts.StatusOK, &superAgentHarnessResumeResponse{
		Code: 0,
		Msg:  "success",
		Data: resume,
	})
}

func buildSuperAgentHarnessSnapshot(ctx context.Context, req superAgentHarnessSnapshotRequest) (*superAgentHarnessSnapshotData, error) {
	agentID := snapshotAgentID(req)
	if req.ConversationID <= 0 && agentID <= 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "conversation_id or agent_id is required"))
	}

	data := &superAgentHarnessSnapshotData{
		Components: make([]string, 0, 5),
	}
	if agentID > 0 {
		data.AgentID = strconv.FormatInt(agentID, 10)
	}
	if req.ConversationID > 0 {
		data.ConversationID = strconv.FormatInt(req.ConversationID, 10)
	}

	if req.ConversationID > 0 && includeSnapshotComponent(req.IncludeMessages, true) {
		messages, err := buildSuperAgentMessageList(ctx, superAgentListMessagesRequest{
			ConversationID: req.ConversationID,
			Limit:          int(req.MessageLimit),
			OrderBy:        "ASC",
		})
		if err != nil {
			return nil, err
		}
		data.Messages = messages
		if data.AgentID == "" && messages != nil {
			data.AgentID = messages.AgentID
		}
		if agentID <= 0 && data.AgentID != "" {
			if parsedAgentID, parseErr := strconv.ParseInt(data.AgentID, 10, 64); parseErr == nil && parsedAgentID > 0 {
				agentID = parsedAgentID
				req.AgentID = parsedAgentID
			}
		}
		data.Components = append(data.Components, "messages")
	}

	if req.ConversationID > 0 && includeSnapshotComponent(req.IncludeRuns, true) {
		runs, err := buildSuperAgentRunHistory(ctx, superAgentListRunsRequest{
			ConversationID: req.ConversationID,
			Limit:          req.RunLimit,
		})
		if err != nil {
			return nil, err
		}
		data.Runs = runs
		data.Components = append(data.Components, "runs")
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeWorkspace, true) {
		workspacePath := strings.TrimSpace(req.WorkspacePath)
		if workspacePath == "" {
			workspacePath = "/workspace"
		}
		workspace, err := application.SingleAgentSVC.ListSuperAgentWorkspaceFiles(ctx, &developer_api.ListSandboxFilesRequest{
			SpaceID:     req.SpaceID,
			AgentID:     req.AgentID,
			BotID:       req.BotID,
			ConnectorID: req.ConnectorID,
			Path:        workspacePath,
			Recursive:   req.WorkspaceRecursive,
		})
		if err != nil {
			return nil, err
		}
		if workspace != nil {
			data.Workspace = workspace.Data
			data.Components = append(data.Components, "workspace")
		}
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeHarness, true) {
		harnessState, err := application.SingleAgentSVC.GetSuperAgentHarnessState(ctx, &developer_api.SuperAgentHarnessStateRequest{
			SpaceID:        req.SpaceID,
			AgentID:        req.AgentID,
			BotID:          req.BotID,
			ConversationID: req.ConversationID,
			ConnectorID:    req.ConnectorID,
		})
		if err != nil {
			return nil, err
		}
		if harnessState != nil {
			data.Harness = harnessState.Data
			data.Components = append(data.Components, "harness")
		}
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeContext, true) {
		contextState, err := application.SingleAgentSVC.GetSuperAgentHarnessContext(ctx, &developer_api.SuperAgentHarnessStateRequest{
			SpaceID:        req.SpaceID,
			AgentID:        req.AgentID,
			BotID:          req.BotID,
			ConversationID: req.ConversationID,
			ConnectorID:    req.ConnectorID,
		})
		if err != nil {
			return nil, err
		}
		if contextState != nil {
			data.Context = contextState
			data.Components = append(data.Components, "context")
		}
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeToolOutputs, true) {
		toolOutputs, err := application.SingleAgentSVC.ListSuperAgentHarnessToolOutputs(ctx, &application.SuperAgentHarnessToolOutputsRequest{
			SpaceID:        req.SpaceID,
			AgentID:        req.AgentID,
			BotID:          req.BotID,
			ConversationID: req.ConversationID,
			ConnectorID:    req.ConnectorID,
			Recursive:      true,
			IncludeContent: includeSnapshotComponent(req.IncludeToolOutputContent, false),
		})
		if err != nil {
			return nil, err
		}
		if toolOutputs != nil {
			data.ToolOutputs = toolOutputs.Data
			data.Components = append(data.Components, "tool_outputs")
		}
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeArtifacts, true) {
		artifacts, err := application.SingleAgentSVC.ListSuperAgentArtifacts(ctx, &application.SuperAgentArtifactListRequest{
			AgentID:     req.AgentID,
			BotID:       req.BotID,
			ConnectorID: req.ConnectorID,
			Limit:       req.ArtifactLimit,
		})
		if err != nil {
			return nil, err
		}
		if artifacts != nil {
			data.Artifacts = artifacts.Data
			data.Components = append(data.Components, "artifacts")
		}
	}

	if req.ConversationID > 0 && includeSnapshotComponent(req.IncludeTrace, true) {
		trace, err := buildSuperAgentTraceData(ctx, superAgentTraceGetRequest{
			ConversationID: req.ConversationID,
			RunID:          req.RunID,
			Limit:          req.TraceLimit,
		})
		if err != nil {
			return nil, err
		}
		appendContextCompactedTraceEvent(trace, data.Context, data.ConversationID, data.AgentID)
		data.Trace = trace
		data.Components = append(data.Components, "trace")
	}

	if req.ConversationID > 0 && includeSnapshotComponent(req.IncludeApprovals, true) {
		approvals, err := listSuperAgentPendingApprovals(ctx, superAgentListApprovalsRequest{
			ConversationID: req.ConversationID,
			AgentID:        agentID,
			RunID:          snapshotRunID(req.RunID),
			Limit:          req.TraceLimit,
		})
		if err != nil {
			return nil, err
		}
		data.Approvals = approvals
		data.Components = append(data.Components, "approvals")
	}

	if agentID > 0 && includeSnapshotComponent(req.IncludeApprovals, true) {
		decisions, err := listSuperAgentApprovalDecisionSnapshots(ctx, req, agentID)
		if err != nil {
			return nil, err
		}
		data.ApprovalDecisions = decisions
		data.Components = append(data.Components, "approval_decisions")
	}

	if includeSnapshotComponent(req.IncludeResume, true) {
		data.Resume = buildSuperAgentHarnessResume(data)
		data.Components = append(data.Components, "resume")
	}

	return data, nil
}

func applySuperAgentHarnessResumeDefaults(req *superAgentHarnessSnapshotRequest) {
	defaultSnapshotBool(&req.IncludeMessages, true)
	defaultSnapshotBool(&req.IncludeRuns, false)
	defaultSnapshotBool(&req.IncludeWorkspace, false)
	defaultSnapshotBool(&req.IncludeHarness, true)
	defaultSnapshotBool(&req.IncludeContext, true)
	defaultSnapshotBool(&req.IncludeToolOutputs, true)
	defaultSnapshotBool(&req.IncludeToolOutputContent, false)
	defaultSnapshotBool(&req.IncludeArtifacts, true)
	defaultSnapshotBool(&req.IncludeTrace, false)
	defaultSnapshotBool(&req.IncludeApprovals, false)
	defaultSnapshotBool(&req.IncludeResume, true)
}

func defaultSnapshotBool(flag **bool, value bool) {
	if *flag != nil {
		return
	}
	v := value
	*flag = &v
}

func buildSuperAgentHarnessResume(data *superAgentHarnessSnapshotData) *superAgentHarnessResumeData {
	resume := &superAgentHarnessResumeData{
		Version:        "v1",
		ConversationID: data.ConversationID,
		AgentID:        data.AgentID,
		Components:     append([]string(nil), data.Components...),
	}
	if data.Context != nil {
		resume.SummaryPath = data.Context.SummaryPath
		resume.SummaryExists = data.Context.SummaryExists
		if data.Context.Summary != nil {
			resume.Summary = data.Context.Summary.Summary
			if strings.TrimSpace(data.Context.Summary.SummaryPath) != "" {
				resume.SummaryPath = data.Context.Summary.SummaryPath
			}
			resume.ArtifactPaths = appendUniqueStrings(resume.ArtifactPaths, data.Context.Summary.Artifacts...)
		}
	}
	if data.Harness != nil && data.Harness.Plan != nil {
		resume.PlanPath = data.Harness.Plan.Path
		resume.PlanExists = data.Harness.Plan.Exists
		resume.PlanItemCount, resume.PlanStatus = summarizeSuperAgentPlan(data.Harness.Plan)
	}
	if data.Messages != nil {
		resume.RecentMessageCount = len(data.Messages.Messages)
		for _, message := range data.Messages.Messages {
			resume.MessageIDs = appendUniqueStrings(resume.MessageIDs, message.MessageID)
		}
	}
	if data.ToolOutputs != nil {
		resume.ToolOutputRoot = data.ToolOutputs.Root
		for _, entry := range data.ToolOutputs.Entries {
			if entry == nil {
				continue
			}
			resume.ToolOutputPaths = appendUniqueStrings(resume.ToolOutputPaths, entry.Path)
		}
		for _, file := range data.ToolOutputs.Files {
			if file == nil || file.IsDir {
				continue
			}
			resume.ToolOutputPaths = appendUniqueStrings(resume.ToolOutputPaths, file.Path)
		}
		resume.ToolOutputCount = len(resume.ToolOutputPaths)
	}
	if data.Artifacts != nil {
		resume.ArtifactRoot = data.Artifacts.Root
		for _, artifact := range data.Artifacts.Artifacts {
			resume.ArtifactPaths = appendUniqueStrings(resume.ArtifactPaths, artifact.Path)
		}
	}
	resume.ArtifactCount = len(resume.ArtifactPaths)
	if data.Trace != nil {
		resume.TraceEventCount = len(data.Trace.Events)
	}
	resume.ApprovalCount = len(data.Approvals)
	resume.Prompt = formatSuperAgentHarnessResumePrompt(resume)
	return resume
}

func summarizeSuperAgentPlan(plan *developer_api.SuperAgentHarnessPlanState) (int, string) {
	if plan == nil || !plan.Exists {
		return 0, "missing"
	}
	content := strings.TrimSpace(plan.Content)
	if content == "" {
		return 0, "empty"
	}
	var items []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		return 0, "available"
	}
	if len(items) == 0 {
		return 0, "empty"
	}
	hasPending := false
	for _, item := range items {
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if status == "in_progress" {
			return len(items), "in_progress"
		}
		if status != "completed" && status != "complete" && status != "done" {
			hasPending = true
		}
	}
	if hasPending {
		return len(items), "pending"
	}
	return len(items), "completed"
}

func formatSuperAgentHarnessResumePrompt(resume *superAgentHarnessResumeData) string {
	if resume == nil {
		return ""
	}
	parts := []string{
		"Super-agent session resume handoff.",
	}
	if resume.ConversationID != "" {
		parts = append(parts, "Conversation: "+resume.ConversationID)
	}
	if resume.AgentID != "" {
		parts = append(parts, "Agent: "+resume.AgentID)
	}
	if resume.SummaryPath != "" {
		parts = append(parts, "Context summary: "+resume.SummaryPath)
	}
	if strings.TrimSpace(resume.Summary) != "" {
		parts = append(parts, "Summary:\n"+strings.TrimSpace(resume.Summary))
	}
	if resume.PlanPath != "" {
		parts = append(parts, "Plan: "+resume.PlanPath+" ("+resume.PlanStatus+")")
	}
	if len(resume.MessageIDs) > 0 {
		parts = append(parts, "Recent message ids: "+strings.Join(resume.MessageIDs, ", "))
	}
	if len(resume.ToolOutputPaths) > 0 {
		parts = append(parts, "Tool outputs: "+strings.Join(resume.ToolOutputPaths, ", "))
	}
	if len(resume.ArtifactPaths) > 0 {
		parts = append(parts, "Artifacts: "+strings.Join(resume.ArtifactPaths, ", "))
	}
	return strings.Join(parts, "\n")
}

func appendUniqueStrings(values []string, next ...string) []string {
	if len(next) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values)+len(next))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func listSuperAgentApprovalDecisionSnapshots(ctx context.Context, req superAgentHarnessSnapshotRequest, agentID int64) ([]superAgentApprovalDecisionSnapshotItem, error) {
	items := make([]superAgentApprovalDecisionSnapshotItem, 0)
	if application.SingleAgentSVC == nil || agentID <= 0 {
		return items, nil
	}
	listResp, err := application.SingleAgentSVC.ListSuperAgentWorkspaceFiles(ctx, &developer_api.ListSandboxFilesRequest{
		SpaceID:     req.SpaceID,
		AgentID:     req.AgentID,
		BotID:       req.BotID,
		ConnectorID: req.ConnectorID,
		Path:        superAgentApprovalDecisionRoot,
		Recursive:   false,
	})
	if err != nil || listResp == nil || listResp.Data == nil {
		return items, nil
	}

	conversationFilter := ""
	if req.ConversationID > 0 {
		conversationFilter = strconv.FormatInt(req.ConversationID, 10)
	}
	runFilter := snapshotRunID(req.RunID)

	for _, file := range listResp.Data.Files {
		if file == nil || file.IsDir || strings.TrimSpace(file.Path) == "" || !strings.HasSuffix(file.Path, ".json") {
			continue
		}
		readResp, err := application.SingleAgentSVC.ReadSuperAgentWorkspaceFile(ctx, &developer_api.ReadSandboxFileRequest{
			SpaceID:     req.SpaceID,
			AgentID:     req.AgentID,
			BotID:       req.BotID,
			ConnectorID: req.ConnectorID,
			Path:        file.Path,
		})
		if err != nil || readResp == nil || readResp.Data == nil || readResp.Data.IsBinary {
			continue
		}
		var record superAgentApprovalDecisionRecord
		if err := json.Unmarshal([]byte(readResp.Data.Content), &record); err != nil {
			continue
		}
		if conversationFilter != "" && record.ConversationID != conversationFilter {
			continue
		}
		if runFilter != "" && record.RunID != runFilter {
			continue
		}
		items = append(items, superAgentApprovalDecisionSnapshotItem{
			Version:        record.Version,
			ApprovalID:     record.ApprovalID,
			RunID:          record.RunID,
			ConversationID: record.ConversationID,
			AgentID:        record.AgentID,
			BotID:          record.BotID,
			Decision:       record.Decision,
			Status:         record.Status,
			Cancelled:      record.Cancelled,
			Note:           record.Note,
			Source:         record.Source,
			ResolvedAt:     record.ResolvedAt,
			DecisionPath:   file.Path,
			Size:           file.Size,
			Mtime:          file.Mtime,
		})
	}
	return items, nil
}

func appendContextCompactedTraceEvent(trace *superagenttrace.Data, contextState *developer_api.SuperAgentHarnessContextState, conversationID, agentID string) {
	if trace == nil || contextState == nil || !contextState.SummaryExists || contextState.Summary == nil {
		return
	}
	summary := contextState.Summary
	if strings.TrimSpace(summary.Summary) == "" && strings.TrimSpace(summary.SummaryPath) == "" {
		return
	}
	if conversationID == "" {
		conversationID = trace.ConversationID
	}
	metadata := map[string]string{}
	if summary.Trigger != "" {
		metadata["trigger"] = summary.Trigger
	}
	if summary.SummaryPath != "" {
		metadata["summary_path"] = summary.SummaryPath
	}
	if summary.OriginalMessages > 0 {
		metadata["original_messages"] = strconv.Itoa(summary.OriginalMessages)
	}
	if summary.CompactedMessages > 0 {
		metadata["compacted_messages"] = strconv.Itoa(summary.CompactedMessages)
	}
	if summary.RetainedMessages > 0 {
		metadata["retained_messages"] = strconv.Itoa(summary.RetainedMessages)
	}
	if summary.OriginalBytes > 0 {
		metadata["original_bytes"] = strconv.Itoa(summary.OriginalBytes)
	}
	if summary.MaxBytes > 0 {
		metadata["max_bytes"] = strconv.Itoa(summary.MaxBytes)
	}
	eventID := "context:compacted:" + summary.SummaryPath
	if superAgentTraceHasEventID(trace, eventID) {
		return
	}
	trace.Events = append(trace.Events, superagenttrace.Event{
		ID:             eventID,
		Event:          superagenttrace.EventContextCompacted,
		Kind:           "context",
		RunID:          summary.RunID,
		MessageID:      summary.MessageID,
		ConversationID: conversationID,
		AgentID:        agentID,
		Content:        summary.Summary,
		ContentType:    "text",
		Metadata:       metadata,
		Status:         "compacted",
		CreatedAt:      summary.UpdatedAt,
		UpdatedAt:      summary.UpdatedAt,
	})
}

func appendPlanUpdatedTraceEvent(trace *superagenttrace.Data, plan *developer_api.SuperAgentHarnessPlanState, conversationID, agentID string) {
	if trace == nil || plan == nil || !plan.Exists || strings.TrimSpace(plan.Content) == "" {
		return
	}
	if conversationID == "" {
		conversationID = trace.ConversationID
	}
	metadata := map[string]string{
		"source": "harness_state",
	}
	if plan.Path != "" {
		metadata["plan_path"] = plan.Path
	}
	if plan.Size > 0 {
		metadata["size"] = strconv.FormatInt(plan.Size, 10)
	}
	if plan.TotalSize > 0 {
		metadata["total_size"] = strconv.FormatInt(plan.TotalSize, 10)
	}
	if plan.Mtime > 0 {
		metadata["mtime"] = strconv.FormatInt(plan.Mtime, 10)
	}
	if plan.IsTruncated {
		metadata["is_truncated"] = "true"
	}
	eventID := "plan:updated:" + plan.Path
	if superAgentTraceHasEventID(trace, eventID) {
		return
	}
	trace.Events = append(trace.Events, superagenttrace.Event{
		ID:             eventID,
		Event:          superagenttrace.EventPlanUpdated,
		Kind:           "plan",
		ConversationID: conversationID,
		AgentID:        agentID,
		Content:        plan.Content,
		ContentType:    "json",
		Metadata:       metadata,
		Status:         "updated",
		CreatedAt:      plan.Mtime,
		UpdatedAt:      plan.Mtime,
	})
}

func superAgentTraceHasEventID(trace *superagenttrace.Data, eventID string) bool {
	if trace == nil || eventID == "" {
		return false
	}
	for _, event := range trace.Events {
		if event.ID == eventID {
			return true
		}
	}
	return false
}

func snapshotAgentID(req superAgentHarnessSnapshotRequest) int64 {
	if req.AgentID != 0 {
		return req.AgentID
	}
	return req.BotID
}

func snapshotRunID(runID *int64) string {
	if runID == nil || *runID <= 0 {
		return ""
	}
	return strings.TrimSpace(strconv.FormatInt(*runID, 10))
}

func includeSnapshotComponent(flag *bool, defaultValue bool) bool {
	if flag == nil {
		return defaultValue
	}
	return *flag
}
