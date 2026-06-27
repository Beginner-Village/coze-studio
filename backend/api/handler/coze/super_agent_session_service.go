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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/common"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	superAgentSessionTitleExtKey = "super_agent_session_title"
	superAgentSessionDefaultPage = 1
	superAgentSessionDefaultSize = 20
	superAgentSessionMaxPageSize = 100
	superAgentSessionMaxTitleLen = 128
)

type superAgentListSessionsRequest struct {
	SpaceID     int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID     int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID       int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	ConnectorID *int64 `form:"connector_id" json:"connector_id,string,omitempty"`
	User        string `form:"user_id" json:"user_id,omitempty"`
	ClientID    string `form:"client_id" json:"client_id,omitempty"`
	Page        int    `form:"page" json:"page,omitempty"`
	PageSize    int    `form:"page_size" json:"page_size,omitempty"`
}

type superAgentCreateSessionRequest struct {
	SpaceID     int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID     int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID       int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	ConnectorID *int64 `form:"connector_id" json:"connector_id,string,omitempty"`
	Title       string `form:"title" json:"title,omitempty"`
	User        string `form:"user_id" json:"user_id,omitempty"`
	ClientID    string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentGetSessionRequest struct {
	ConversationID           int64   `form:"conversation_id" json:"conversation_id,string,omitempty"`
	SpaceID                  int64   `form:"space_id" json:"space_id,string,omitempty"`
	AgentID                  int64   `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID                    int64   `form:"bot_id" json:"bot_id,string,omitempty"`
	ConnectorID              *string `form:"connector_id" json:"connector_id,omitempty"`
	RunID                    *int64  `form:"run_id" json:"run_id,string,omitempty"`
	TraceLimit               int32   `form:"trace_limit" json:"trace_limit,omitempty"`
	RunLimit                 int32   `form:"run_limit" json:"run_limit,omitempty"`
	MessageLimit             int32   `form:"message_limit" json:"message_limit,omitempty"`
	ArtifactLimit            int32   `form:"artifact_limit" json:"artifact_limit,omitempty"`
	WorkspacePath            string  `form:"workspace_path" json:"workspace_path,omitempty"`
	WorkspaceRecursive       bool    `form:"workspace_recursive" json:"workspace_recursive,omitempty"`
	IncludeSnapshot          *bool   `form:"include_snapshot" json:"include_snapshot,omitempty"`
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
	User                     string  `form:"user_id" json:"user_id,omitempty"`
	ClientID                 string  `form:"client_id" json:"client_id,omitempty"`
}

type superAgentRenameSessionRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	Title          string `form:"title" json:"title,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentDeleteSessionRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentListSessionsResponse struct {
	Code int                       `json:"code"`
	Msg  string                    `json:"msg"`
	Data superAgentSessionListData `json:"data"`
}

type superAgentCreateSessionResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data superAgentCreateSessionData `json:"data"`
}

type superAgentGetSessionResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data superAgentGetSessionData `json:"data"`
}

type superAgentCreateSessionData struct {
	Session superAgentSessionItem `json:"session"`
}

type superAgentGetSessionData struct {
	Session  superAgentSessionItem          `json:"session"`
	Snapshot *superAgentHarnessSnapshotData `json:"snapshot,omitempty"`
}

type superAgentSessionListData struct {
	Sessions []superAgentSessionItem `json:"sessions"`
	HasMore  bool                    `json:"has_more"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type superAgentRenameSessionResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data superAgentRenameSessionData `json:"data"`
}

type superAgentDeleteSessionResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data superAgentDeleteSessionData `json:"data"`
}

type superAgentRenameSessionData struct {
	Session superAgentSessionItem `json:"session"`
}

type superAgentDeleteSessionData struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
}

type superAgentSessionItem struct {
	SessionID      string       `json:"session_id"`
	ConversationID string       `json:"conversation_id"`
	SectionID      string       `json:"section_id,omitempty"`
	AgentID        string       `json:"agent_id"`
	ConnectorID    string       `json:"connector_id"`
	Scene          common.Scene `json:"scene"`
	Title          string       `json:"title"`
	Renamable      bool         `json:"renamable"`
	CreatedAt      int64        `json:"created_at"`
	UpdatedAt      int64        `json:"updated_at"`
}

// SuperAgentCreateSession creates a durable work session for a super-agent app server client.
// @router /api/super-agent/sessions/create [POST]
func SuperAgentCreateSession(ctx context.Context, c *app.RequestContext) {
	var req superAgentCreateSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	botID := req.BotID
	if botID == 0 {
		botID = req.AgentID
	}
	if botID == 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) > superAgentSessionMaxTitleLen {
		invalidParamRequestResponse(c, fmt.Sprintf("title must be no longer than %d characters", superAgentSessionMaxTitleLen))
		return
	}
	connectorID := consts.CozeConnectorID
	if req.ConnectorID != nil && *req.ConnectorID > 0 {
		connectorID = *req.ConnectorID
	}

	// 始终写入标题 ext 标记(空标题用默认值),作为「超级体会话」的标识 ——
	// list 据此把它与同 scene 的 bot 默认调试会话区分开(见 isSuperAgentSession)。
	if title == "" {
		title = "新会话"
	}
	ext := map[string]any{
		superAgentSessionTitleExtKey: title,
	}
	encodedExt, err := json.Marshal(ext)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	// 会话用 Scene_Playground 创建,与前端调试聊天区(scene=Playground)对齐:
	// agent_run.checkConversation 要求 conversation.Scene == 请求 scene 才认这个会话,
	// 否则回退到 bot 的默认调试会话 —— 这正是历史上「切换会话后聊天落到固定会话、
	// 切回来历史为空」的根因。统一到 Playground 后,每个会话才有独立、可回显的历史。
	session, err := conversation.ConversationSVC.ConversationDomainSVC.Create(ctx, &convEntity.CreateMeta{
		AgentID:     botID,
		UserID:      userID,
		ConnectorID: connectorID,
		Scene:       common.Scene_Playground,
		Ext:         string(encodedExt),
	})
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if session == nil {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationNotFound))
		return
	}
	c.JSON(http.StatusOK, &superAgentCreateSessionResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentCreateSessionData{
			Session: buildSuperAgentSessionItem(session, userID),
		},
	})
}

// SuperAgentListSessions lists durable work sessions for a super-agent app server client.
// @router /api/super-agent/sessions/list [POST]
func SuperAgentListSessions(ctx context.Context, c *app.RequestContext) {
	var req superAgentListSessionsRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	botID := req.BotID
	if botID == 0 {
		botID = req.AgentID
	}
	if botID == 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	page, pageSize := normalizeSuperAgentSessionPage(req.Page, req.PageSize)
	connectorID := consts.CozeConnectorID
	if req.ConnectorID != nil && *req.ConnectorID > 0 {
		connectorID = *req.ConnectorID
	}

	sessions, hasMore, err := conversation.ConversationSVC.ConversationDomainSVC.List(ctx, &convEntity.ListMeta{
		UserID:      userID,
		ConnectorID: connectorID,
		Scene:       common.Scene_Playground,
		AgentID:     botID,
		Limit:       pageSize,
		Page:        page,
	})
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}

	items := make([]superAgentSessionItem, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		// Scene_Playground 下同时存在 bot 的默认调试会话(无 ext 标记)与超级体会话
		// (带 super_agent_session_title 标记)。侧栏只展示显式创建的超级体会话,
		// 排除默认调试会话,避免出现一个无名的「幽灵会话」。
		if !isSuperAgentSession(session) {
			continue
		}
		items = append(items, buildSuperAgentSessionItem(session, userID))
	}
	c.JSON(http.StatusOK, &superAgentListSessionsResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentSessionListData{
			Sessions: items,
			HasMore:  hasMore,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// SuperAgentGetSession returns a durable work session and optional harness snapshot.
// @router /api/super-agent/sessions/get [POST]
func SuperAgentGetSession(ctx context.Context, c *app.RequestContext) {
	var req superAgentGetSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}

	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, req.ConversationID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if currentConversation == nil {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationNotFound))
		return
	}
	if currentConversation.CreatorID != userID {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied")))
		return
	}

	data := superAgentGetSessionData{
		Session: buildSuperAgentSessionItem(currentConversation, userID),
	}
	if includeSnapshotComponent(req.IncludeSnapshot, false) {
		agentID := req.AgentID
		if agentID == 0 {
			agentID = req.BotID
		}
		if agentID == 0 {
			agentID = currentConversation.AgentID
		}
		snapshot, err := buildSuperAgentHarnessSnapshot(ctx, superAgentHarnessSnapshotRequest{
			SpaceID:                  req.SpaceID,
			AgentID:                  agentID,
			BotID:                    req.BotID,
			ConnectorID:              req.ConnectorID,
			ConversationID:           req.ConversationID,
			RunID:                    req.RunID,
			TraceLimit:               req.TraceLimit,
			RunLimit:                 req.RunLimit,
			MessageLimit:             req.MessageLimit,
			ArtifactLimit:            req.ArtifactLimit,
			WorkspacePath:            req.WorkspacePath,
			WorkspaceRecursive:       req.WorkspaceRecursive,
			IncludeHarness:           req.IncludeHarness,
			IncludeMessages:          req.IncludeMessages,
			IncludeRuns:              req.IncludeRuns,
			IncludeWorkspace:         req.IncludeWorkspace,
			IncludeContext:           req.IncludeContext,
			IncludeToolOutputs:       req.IncludeToolOutputs,
			IncludeToolOutputContent: req.IncludeToolOutputContent,
			IncludeArtifacts:         req.IncludeArtifacts,
			IncludeTrace:             req.IncludeTrace,
			IncludeApprovals:         req.IncludeApprovals,
		})
		if err != nil {
			internalServerErrorResponse(ctx, c, err)
			return
		}
		data.Snapshot = snapshot
	}

	c.JSON(http.StatusOK, &superAgentGetSessionResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// SuperAgentRenameSession persists a user-facing title for a super-agent conversation.
// @router /api/super-agent/sessions/rename [POST]
func SuperAgentRenameSession(ctx context.Context, c *app.RequestContext) {
	var req superAgentRenameSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		invalidParamRequestResponse(c, "title is required")
		return
	}
	if utf8.RuneCountInString(title) > superAgentSessionMaxTitleLen {
		invalidParamRequestResponse(c, fmt.Sprintf("title must be no longer than %d characters", superAgentSessionMaxTitleLen))
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}

	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, req.ConversationID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if currentConversation == nil {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationNotFound))
		return
	}
	if currentConversation.CreatorID != userID {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied")))
		return
	}

	ext := decodeSuperAgentSessionExt(currentConversation.Ext)
	ext[superAgentSessionTitleExtKey] = title
	encodedExt, err := json.Marshal(ext)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if err := conversation.ConversationSVC.ConversationDomainSVC.UpdateExt(ctx, currentConversation.ID, string(encodedExt)); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	currentConversation.Ext = string(encodedExt)
	c.JSON(http.StatusOK, &superAgentRenameSessionResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentRenameSessionData{
			Session: buildSuperAgentSessionItem(currentConversation, userID),
		},
	})
}

// SuperAgentDeleteSession deletes a durable work session owned by the current user.
// @router /api/super-agent/sessions/delete [POST]
func SuperAgentDeleteSession(ctx context.Context, c *app.RequestContext) {
	var req superAgentDeleteSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}

	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, req.ConversationID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	// Idempotent delete: a session whose underlying conversation is already gone
	// (already deleted, orphaned sidebar entry, etc.) is treated as successfully
	// deleted instead of failing with "conversation not found".
	if err != nil || currentConversation == nil || currentConversation.ID <= 0 {
		respondSuperAgentDeleteSessionOK(c, req.ConversationID)
		return
	}
	if currentConversation.CreatorID != userID {
		internalServerErrorResponse(ctx, c, errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied")))
		return
	}
	if err := conversation.ConversationSVC.ConversationDomainSVC.Delete(ctx, currentConversation.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	clearSuperAgentDeletedSessionHarnessState(ctx, currentConversation)
	respondSuperAgentDeleteSessionOK(c, currentConversation.ID)
}

func respondSuperAgentDeleteSessionOK(c *app.RequestContext, conversationID int64) {
	sessionID := strconv.FormatInt(conversationID, 10)
	c.JSON(http.StatusOK, &superAgentDeleteSessionResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentDeleteSessionData{
			SessionID:      sessionID,
			ConversationID: sessionID,
		},
	})
}

func clearSuperAgentDeletedSessionHarnessState(ctx context.Context, currentConversation *convEntity.Conversation) {
	if currentConversation == nil || currentConversation.ID <= 0 || currentConversation.AgentID <= 0 || application.SingleAgentSVC == nil {
		return
	}
	cleanupCtx := ctx
	if ctxutil.GetApiAuthFromCtx(cleanupCtx) == nil && ctxutil.GetUIDFromCtx(cleanupCtx) == nil {
		cleanupCtx = ctxcache.Init(cleanupCtx)
		ctxcache.Store(cleanupCtx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: currentConversation.CreatorID})
	}
	_, err := application.SingleAgentSVC.ClearSuperAgentHarnessContext(cleanupCtx, &developer_api.SuperAgentHarnessContextClearRequest{
		AgentID:        currentConversation.AgentID,
		ConversationID: currentConversation.ID,
	})
	if err != nil {
		logs.CtxWarnf(ctx, "[SuperAgentDeleteSession] clear harness context for conversation %d failed: %v", currentConversation.ID, err)
	}
	_, err = application.SingleAgentSVC.DeleteSuperAgentWorkspaceFile(cleanupCtx, &developer_api.DeleteSandboxFileRequest{
		AgentID: currentConversation.AgentID,
		Path:    fmt.Sprintf("/workspace/.agent/sessions/%d/plan.json", currentConversation.ID),
	})
	if err != nil {
		logs.CtxWarnf(ctx, "[SuperAgentDeleteSession] clear harness plan for conversation %d failed: %v", currentConversation.ID, err)
	}
	_, err = application.SingleAgentSVC.DeleteSuperAgentWorkspaceFile(cleanupCtx, &developer_api.DeleteSandboxFileRequest{
		AgentID: currentConversation.AgentID,
		Path:    fmt.Sprintf("/workspace/.agent/tooloutputs/sessions/%d", currentConversation.ID),
	})
	if err != nil {
		logs.CtxWarnf(ctx, "[SuperAgentDeleteSession] clear harness tool outputs for conversation %d failed: %v", currentConversation.ID, err)
	}
}

func resolveSuperAgentSessionUserID(ctx context.Context, explicit string) (int64, error) {
	user := strings.TrimSpace(explicit)
	if strings.HasPrefix(user, "api-user-") {
		user = strings.TrimPrefix(user, "api-user-")
	}
	if user != "" {
		userID, err := strconv.ParseInt(user, 10, 64)
		if err != nil || userID <= 0 {
			return 0, errors.New("user_id must be a positive integer")
		}
		return userID, nil
	}
	if apiAuth := ctxutil.GetApiAuthFromCtx(ctx); apiAuth != nil && apiAuth.UserID > 0 {
		return apiAuth.UserID, nil
	}
	if userID := ctxutil.GetUIDFromCtx(ctx); userID != nil && *userID > 0 {
		return *userID, nil
	}
	return 0, errors.New("user_id is required")
}

func normalizeSuperAgentSessionPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = superAgentSessionDefaultPage
	}
	if pageSize <= 0 {
		pageSize = superAgentSessionDefaultSize
	}
	if pageSize > superAgentSessionMaxPageSize {
		pageSize = superAgentSessionMaxPageSize
	}
	return page, pageSize
}

func buildSuperAgentSessionItem(session *convEntity.Conversation, viewerID int64) superAgentSessionItem {
	if session == nil {
		return superAgentSessionItem{}
	}
	sessionID := strconv.FormatInt(session.ID, 10)
	return superAgentSessionItem{
		SessionID:      sessionID,
		ConversationID: sessionID,
		SectionID:      formatSuperAgentOptionalID(session.SectionID),
		AgentID:        strconv.FormatInt(session.AgentID, 10),
		ConnectorID:    strconv.FormatInt(session.ConnectorID, 10),
		Scene:          session.Scene,
		Title:          resolveSuperAgentSessionTitle(session),
		Renamable:      session.CreatorID == viewerID,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}
}

// isSuperAgentSession 判断一个会话是否为显式创建的超级体会话(带标题 ext 标记)。
// 用于把它与同 scene(Playground)下的 bot 默认调试会话区分开。
func isSuperAgentSession(session *convEntity.Conversation) bool {
	if session == nil {
		return false
	}
	ext := decodeSuperAgentSessionExt(session.Ext)
	_, ok := ext[superAgentSessionTitleExtKey]
	return ok
}

func resolveSuperAgentSessionTitle(session *convEntity.Conversation) string {
	ext := decodeSuperAgentSessionExt(session.Ext)
	if rawTitle, ok := ext[superAgentSessionTitleExtKey]; ok {
		if title, ok := rawTitle.(string); ok && strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
	}
	if session.ID > 0 {
		return "会话 " + strconv.FormatInt(session.ID, 10)
	}
	return "未命名会话"
}

func decodeSuperAgentSessionExt(ext string) map[string]any {
	decoded := make(map[string]any)
	if strings.TrimSpace(ext) == "" {
		return decoded
	}
	if err := json.Unmarshal([]byte(ext), &decoded); err != nil {
		return make(map[string]any)
	}
	return decoded
}

func formatSuperAgentOptionalID(id int64) string {
	if id <= 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
