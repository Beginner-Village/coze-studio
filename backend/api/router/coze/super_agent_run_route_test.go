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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze/superagenttrace"
	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/common"
	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	crossMessage "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/message"
	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	agentservice "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/service"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	messageEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	sbx "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type fakeSuperAgentRunDomainSVC struct {
	record *agentrunEntity.RunRecordMeta
}

type fakeSuperAgentMessageDomainSVC struct {
	messages []*messageEntity.Message
}

type fakeSuperAgentConversationDomainSVC struct {
	conversation *convEntity.Conversation
	list         []*convEntity.Conversation
	hasMore      bool
	updatedExt   string
	deletedID    int64
	createdMeta  *convEntity.CreateMeta
}

type fakeSuperAgentSingleAgentDomainSVC struct {
	agentservice.SingleAgent
	draft *agententity.SingleAgent
}

type fakeSuperAgentWorkspaceSandboxManager struct{}

type fakeSuperAgentToolOutputSandboxManager struct {
	fakeSuperAgentWorkspaceSandboxManager
}

type fakeSuperAgentContextSandboxManager struct {
	fakeSuperAgentToolOutputSandboxManager
}

type fakeSuperAgentSessionContextSandboxManager struct {
	fakeSuperAgentContextSandboxManager
}

type fakeSuperAgentApprovalDecisionSandboxManager struct {
	fakeSuperAgentWorkspaceSandboxManager
	writes map[string][]byte
}

type fakeSuperAgentApprovalDecisionListSandboxManager struct {
	fakeSuperAgentContextSandboxManager
	commands *[]string
}

type fakeSuperAgentContextClearSandboxManager struct {
	fakeSuperAgentSessionContextSandboxManager
	commands *[]string
}

func (f *fakeSuperAgentRunDomainSVC) AgentRun(context.Context, *agentrunEntity.AgentRunMeta) (*schema.StreamReader[*agentrunEntity.AgentRunResponse], error) {
	return nil, nil
}

func (f *fakeSuperAgentRunDomainSVC) Delete(context.Context, []int64) error {
	return nil
}

func (f *fakeSuperAgentRunDomainSVC) Create(context.Context, *agentrunEntity.AgentRunMeta) (*agentrunEntity.RunRecordMeta, error) {
	return f.record, nil
}

func (f *fakeSuperAgentRunDomainSVC) GetByID(context.Context, int64) (*agentrunEntity.RunRecordMeta, error) {
	return f.record, nil
}

func (f *fakeSuperAgentRunDomainSVC) List(context.Context, *agentrunEntity.ListRunRecordMeta) ([]*agentrunEntity.RunRecordMeta, error) {
	if f.record == nil {
		return nil, nil
	}
	return []*agentrunEntity.RunRecordMeta{f.record}, nil
}

func (f *fakeSuperAgentMessageDomainSVC) List(context.Context, *messageEntity.ListMeta) (*messageEntity.ListResult, error) {
	return &messageEntity.ListResult{Messages: f.messages}, nil
}

func (f *fakeSuperAgentMessageDomainSVC) ListWithoutPair(context.Context, *messageEntity.ListMeta) (*messageEntity.ListResult, error) {
	return &messageEntity.ListResult{Messages: f.messages}, nil
}

func (f *fakeSuperAgentMessageDomainSVC) PreCreate(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *fakeSuperAgentMessageDomainSVC) Create(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *fakeSuperAgentMessageDomainSVC) GetByRunIDs(context.Context, int64, []int64) ([]*messageEntity.Message, error) {
	return f.messages, nil
}

func (f *fakeSuperAgentMessageDomainSVC) GetByID(context.Context, int64) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *fakeSuperAgentMessageDomainSVC) Edit(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *fakeSuperAgentMessageDomainSVC) Delete(context.Context, *messageEntity.DeleteMeta) error {
	return nil
}

func (f *fakeSuperAgentMessageDomainSVC) Broken(context.Context, *messageEntity.BrokenMeta) error {
	return nil
}

func (f *fakeSuperAgentConversationDomainSVC) Create(_ context.Context, req *convEntity.CreateMeta) (*convEntity.Conversation, error) {
	f.createdMeta = req
	return f.conversation, nil
}

func (f *fakeSuperAgentConversationDomainSVC) GetByID(context.Context, int64) (*convEntity.Conversation, error) {
	return f.conversation, nil
}

func (f *fakeSuperAgentConversationDomainSVC) NewConversationCtx(context.Context, *convEntity.NewConversationCtxRequest) (*convEntity.NewConversationCtxResponse, error) {
	return nil, nil
}

func (f *fakeSuperAgentConversationDomainSVC) GetCurrentConversation(context.Context, *convEntity.GetCurrent) (*convEntity.Conversation, error) {
	return f.conversation, nil
}

func (f *fakeSuperAgentConversationDomainSVC) Delete(_ context.Context, id int64) error {
	f.deletedID = id
	return nil
}

func (f *fakeSuperAgentConversationDomainSVC) List(context.Context, *convEntity.ListMeta) ([]*convEntity.Conversation, bool, error) {
	return f.list, f.hasMore, nil
}

func (f *fakeSuperAgentConversationDomainSVC) UpdateExt(_ context.Context, _ int64, ext string) error {
	f.updatedExt = ext
	if f.conversation != nil {
		f.conversation.Ext = ext
	}
	return nil
}

func (f *fakeSuperAgentSingleAgentDomainSVC) GetSingleAgentDraft(context.Context, int64) (*agententity.SingleAgent, error) {
	return f.draft, nil
}

func (fakeSuperAgentWorkspaceSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "python3") {
		if strings.Contains(cmd, "/workspace/.agent/sessions/123/plan.json") {
			return &sbx.ExecResponse{Stdout: `{"exists":true,"name":"plan.json","path":"/workspace/.agent/sessions/123/plan.json","is_dir":false,"size":76,"mtime":1781900600}`}, nil
		}
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "src",
				"path":   "/workspace/src",
				"is_dir": true,
				"size":   0,
				"mtime":  int64(1781900000),
			},
			{
				"name":   "main.go",
				"path":   "/workspace/src/main.go",
				"is_dir": false,
				"size":   128,
				"mtime":  int64(1781900001),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return &sbx.ExecResponse{}, nil
}

func (fakeSuperAgentWorkspaceSandboxManager) ReadFile(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (fakeSuperAgentWorkspaceSandboxManager) WriteFile(context.Context, string, string, []byte) error {
	return nil
}

func (f *fakeSuperAgentApprovalDecisionSandboxManager) WriteFile(_ context.Context, _ string, path string, content []byte) error {
	if f.writes == nil {
		f.writes = make(map[string][]byte)
	}
	f.writes[path] = append([]byte(nil), content...)
	return nil
}

func (fakeSuperAgentWorkspaceSandboxManager) ListFiles(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (fakeSuperAgentWorkspaceSandboxManager) EditFile(context.Context, string, string, string, string, bool) (int, error) {
	return 0, nil
}

func (fakeSuperAgentWorkspaceSandboxManager) Grep(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (fakeSuperAgentWorkspaceSandboxManager) Glob(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeSuperAgentWorkspaceSandboxManager) SyncSkill(context.Context, string, string, map[string][]byte) error {
	return nil
}

func (fakeSuperAgentWorkspaceSandboxManager) CheckpointTo(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeSuperAgentWorkspaceSandboxManager) RestoreFrom(context.Context, string, string) error {
	return nil
}

func (fakeSuperAgentToolOutputSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "python3") && strings.Contains(cmd, "/workspace/.agent/tooloutputs") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "tool-call-search.json",
				"path":   "/workspace/.agent/tooloutputs/tool-call-search.json",
				"is_dir": false,
				"size":   96,
				"mtime":  int64(1781900100),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return fakeSuperAgentWorkspaceSandboxManager{}.Exec(context.Background(), "", cmd, 0)
}

func (fakeSuperAgentToolOutputSandboxManager) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	if path == "/workspace/.agent/tooloutputs/tool-call-search.json" {
		return []byte(`{"tool":"Search","args":{"query":"GLM-5.2"},"result":{"count":8}}`), nil
	}
	return nil, nil
}

func (fakeSuperAgentContextSandboxManager) ReadFile(ctx context.Context, key string, path string) ([]byte, error) {
	if path == "/workspace/.agent/context-summary.json" {
		return []byte(`{"version":"v1","summary":"User is building a Codex-like super agent harness with context compaction.","updated_at":1781900300,"message_id":"901","run_id":"765","trigger":"history_bytes_exceeded","original_messages":42,"compacted_messages":26,"retained_messages":16,"original_bytes":262144,"max_bytes":163840,"summary_path":"/workspace/.agent/context-summary.json","key_files":["/workspace/main.go"],"artifacts":["/outputs/report.html"],"next_actions":["continue context compaction"]}`), nil
	}
	return fakeSuperAgentToolOutputSandboxManager{}.ReadFile(ctx, key, path)
}

func (fakeSuperAgentSessionContextSandboxManager) ReadFile(ctx context.Context, key string, path string) ([]byte, error) {
	if path == "/workspace/.agent/sessions/123/context-summary.json" {
		return []byte(`{"version":"v1","summary":"Session 123 compacted context.","updated_at":1781900500,"message_id":"902","run_id":"766","trigger":"history_bytes_exceeded","original_messages":21,"compacted_messages":13,"retained_messages":8,"original_bytes":131072,"max_bytes":163840,"summary_path":"/workspace/.agent/sessions/123/context-summary.json","key_files":["/workspace/session.go"],"artifacts":["/outputs/session.html"],"next_actions":["continue session 123"]}`), nil
	}
	if path == "/workspace/.agent/sessions/123/plan.json" {
		return []byte(`[
  {"content":"trace should expose session plan","status":"in_progress"}
]`), nil
	}
	return fakeSuperAgentToolOutputSandboxManager{}.ReadFile(ctx, key, path)
}

func (f fakeSuperAgentApprovalDecisionListSandboxManager) Exec(ctx context.Context, key string, cmd string, timeout int) (*sbx.ExecResponse, error) {
	if f.commands != nil {
		*f.commands = append(*f.commands, cmd)
	}
	if strings.Contains(cmd, "python3") && strings.Contains(cmd, "approvals") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "run-765.json",
				"path":   "/workspace/.agent/approvals/run-765.json",
				"is_dir": false,
				"size":   220,
				"mtime":  int64(1781900400),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return fakeSuperAgentContextSandboxManager{}.Exec(ctx, key, cmd, timeout)
}

func (fakeSuperAgentApprovalDecisionListSandboxManager) ReadFile(ctx context.Context, key string, path string) ([]byte, error) {
	if path == "/workspace/.agent/approvals/run-765.json" {
		return []byte(`{"version":"v1","approval_id":"run:765","run_id":"765","conversation_id":"123","agent_id":"456","bot_id":"456","decision":"approve","status":"approved","cancelled":false,"note":"looks good","source":"app_server","resolved_at":1781900400}`), nil
	}
	return fakeSuperAgentSessionContextSandboxManager{}.ReadFile(ctx, key, path)
}

func (f fakeSuperAgentContextClearSandboxManager) Exec(ctx context.Context, key string, cmd string, timeout int) (*sbx.ExecResponse, error) {
	if f.commands != nil {
		*f.commands = append(*f.commands, cmd)
	}
	return f.fakeSuperAgentSessionContextSandboxManager.Exec(ctx, key, cmd, timeout)
}

func TestSuperAgentRunRoutesRejectMissingAgentID(t *testing.T) {
	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	for _, path := range []string{
		"/api/super-agent/runs/create",
		"/api/super-agent/runs/reply",
		"/api/super-agent/runs/stream",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{"user_id":"external-user"}`), Len: len(`{"user_id":"external-user"}`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, string(w.Result().Body()), "agent_id")
		})
	}
}

func TestSuperAgentCancelRunRouteRejectsMissingRunID(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/cancel",
		&ut.Body{Body: bytes.NewBufferString(`{"agent_id":"123","user_id":"external-user"}`), Len: len(`{"agent_id":"123","user_id":"external-user"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "run_id")
}

func TestSuperAgentGetRunRouteRejectsMissingRunID(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/get",
		&ut.Body{Body: bytes.NewBufferString(`{"agent_id":"123","user_id":"external-user"}`), Len: len(`{"agent_id":"123","user_id":"external-user"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "run_id")
}

func TestSuperAgentListRunsRouteRejectsMissingConversationID(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/list",
		&ut.Body{Body: bytes.NewBufferString(`{"agent_id":"123","user_id":"external-user"}`), Len: len(`{"agent_id":"123","user_id":"external-user"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "conversation_id")
}

func TestSuperAgentReplyRunRouteRejectsNonPositiveConversationID(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"agent_id":"123","conversation_id":"0","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/reply",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "conversation_id")
}

func TestSuperAgentCancelRunRouteIsIdempotentForInactiveRun(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"run_id":"inactive-run","agent_id":"123","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/cancel",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RunID     string `json:"run_id"`
			Status    string `json:"status"`
			Cancelled bool   `json:"cancelled"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "success", got.Msg)
	assert.Equal(t, "inactive-run", got.Data.RunID)
	assert.Equal(t, "not_active", got.Data.Status)
	assert.False(t, got.Data.Cancelled)
}

func TestSuperAgentGetRunRouteReportsInactiveRun(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"run_id":"inactive-run","agent_id":"123","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/get",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
			Active bool   `json:"active"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "success", got.Msg)
	assert.Equal(t, "inactive-run", got.Data.RunID)
	assert.Equal(t, "not_active", got.Data.Status)
	assert.False(t, got.Data.Active)
}

func TestSuperAgentGetRunRouteReportsPersistedRun(t *testing.T) {
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{
		record: &agentrunEntity.RunRecordMeta{
			ID:             765,
			ConversationID: 456,
			AgentID:        123,
			Status:         agentrunEntity.RunStatusCompleted,
			CreatedAt:      1000,
			UpdatedAt:      2000,
			CompletedAt:    2500,
		},
	}
	t.Cleanup(func() {
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
	})

	h := server.Default()
	Register(h)

	body := `{"run_id":"765","agent_id":"123","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/get",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RunID          string `json:"run_id"`
			ConversationID string `json:"conversation_id"`
			AgentID        string `json:"agent_id"`
			Status         string `json:"status"`
			Active         bool   `json:"active"`
			CreatedAt      int64  `json:"created_at"`
			UpdatedAt      int64  `json:"updated_at"`
			CompletedAt    int64  `json:"completed_at"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "success", got.Msg)
	assert.Equal(t, "765", got.Data.RunID)
	assert.Equal(t, "456", got.Data.ConversationID)
	assert.Equal(t, "123", got.Data.AgentID)
	assert.Equal(t, "completed", got.Data.Status)
	assert.False(t, got.Data.Active)
	assert.Equal(t, int64(1000), got.Data.CreatedAt)
	assert.Equal(t, int64(2000), got.Data.UpdatedAt)
	assert.Equal(t, int64(2500), got.Data.CompletedAt)
}

func TestSuperAgentGetRunRouteReportsActiveRun(t *testing.T) {
	h := server.Default()
	Register(h)

	_, cancel := context.WithCancel(context.Background())
	unregister := conversation.RegisterActiveAgentRun("active-run", cancel)
	defer unregister()

	body := `{"run_id":"active-run","agent_id":"123","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/get",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RunID     string `json:"run_id"`
			Status    string `json:"status"`
			Active    bool   `json:"active"`
			CreatedAt int64  `json:"created_at"`
			UpdatedAt int64  `json:"updated_at"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "active-run", got.Data.RunID)
	assert.Equal(t, "in_progress", got.Data.Status)
	assert.True(t, got.Data.Active)
	assert.Greater(t, got.Data.CreatedAt, int64(0))
	assert.GreaterOrEqual(t, got.Data.UpdatedAt, got.Data.CreatedAt)
}

func TestSuperAgentCancelRunRouteCancelsActiveRun(t *testing.T) {
	h := server.Default()
	Register(h)

	runCtx, cancel := context.WithCancel(context.Background())
	unregister := conversation.RegisterActiveAgentRun("active-run", cancel)
	defer unregister()

	body := `{"run_id":"active-run","agent_id":"123","user_id":"external-user"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/runs/cancel",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RunID     string `json:"run_id"`
			Status    string `json:"status"`
			Cancelled bool   `json:"cancelled"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "active-run", got.Data.RunID)
	assert.Equal(t, "cancelled", got.Data.Status)
	assert.True(t, got.Data.Cancelled)
	assert.ErrorIs(t, runCtx.Err(), context.Canceled)
}

func TestSuperAgentTraceRouteRejectsMissingConversationID(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/traces/get",
		&ut.Body{Body: bytes.NewBufferString(`{"agent_id":"123"}`), Len: len(`{"agent_id":"123"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "conversation_id")
}

func TestSuperAgentTraceRouteIncludesContextCompactionEvent(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   456,
			CreatorID: 42,
		},
	}
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{
		record: &agentrunEntity.RunRecordMeta{
			ID:             765,
			ConversationID: 123,
			AgentID:        456,
			Status:         agentrunEntity.RunStatusCompleted,
			CreatedAt:      1000,
			UpdatedAt:      2000,
			CompletedAt:    2000,
		},
	}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{}
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentSessionContextSandboxManager{})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"conversation_id":"123","limit":20}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/traces/get",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Events []superagenttrace.Event `json:"events"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	contextEvent := findSuperAgentTraceEvent(got.Data.Events, superagenttrace.EventContextCompacted)
	require.NotNil(t, contextEvent)
	assert.Equal(t, "context", contextEvent.Kind)
	assert.Equal(t, "compacted", contextEvent.Status)
	assert.Equal(t, "766", contextEvent.RunID)
	assert.Equal(t, "history_bytes_exceeded", contextEvent.Metadata["trigger"])
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", contextEvent.Metadata["summary_path"])
	planEvent := findSuperAgentTraceEvent(got.Data.Events, superagenttrace.EventPlanUpdated)
	require.NotNil(t, planEvent)
	assert.Equal(t, "plan", planEvent.Kind)
	assert.Equal(t, "updated", planEvent.Status)
	assert.Equal(t, "/workspace/.agent/sessions/123/plan.json", planEvent.Metadata["plan_path"])
	assert.Equal(t, "harness_state", planEvent.Metadata["source"])
	assert.Equal(t, "1781900600", planEvent.Metadata["mtime"])
	assert.Equal(t, int64(1781900600), planEvent.CreatedAt)
	assert.Equal(t, int64(1781900600), planEvent.UpdatedAt)
	assert.Contains(t, planEvent.Content, "trace should expose session plan")
}

func TestSuperAgentHarnessResumeRouteRejectsMissingConversationID(t *testing.T) {
	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/resume",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSuperAgentDeleteSessionIsIdempotentWhenConversationGone(t *testing.T) {
	// Deleting a session whose underlying conversation is already gone (orphaned
	// sidebar entry, double-delete, etc.) must succeed idempotently rather than
	// failing with "conversation not found".
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{conversation: nil}
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"conversation_id":"999999999","user_id":"42"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/delete",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Code int `json:"code"`
		Data struct {
			ConversationID string `json:"conversation_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Result().Body(), &got))
	assert.Equal(t, 0, got.Code, "deleting an already-gone session must be idempotent success")
	assert.Equal(t, "999999999", got.Data.ConversationID)
}

func TestSuperAgentWorkspaceRoutesRejectMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	for _, path := range []string{
		"/api/super-agent/workspace/download",
		"/api/super-agent/workspace/write",
		"/api/super-agent/workspace/move",
		"/api/super-agent/workspace/mkdir",
		"/api/super-agent/workspace/stat",
		"/api/super-agent/workspace/grep",
		"/api/super-agent/workspace/glob",
		"/api/super-agent/workspace/edit",
		"/api/super-agent/workspace/patch",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestSuperAgentSandboxRoutesRejectMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	for _, path := range []string{
		"/api/super-agent/sandbox/exec",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestSuperAgentArtifactRoutesRejectMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	for _, path := range []string{
		"/api/super-agent/artifacts/list",
		"/api/super-agent/artifacts/download",
		"/api/super-agent/artifacts/delete",
		"/api/super-agent/artifacts/move",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestSuperAgentManifestRouteReturnsAppServerContract(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/super-agent/manifest",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			ProtocolVersion string   `json:"protocol_version"`
			Capabilities    []string `json:"capabilities"`
			WorkspaceRoots  []struct {
				Path     string `json:"path"`
				Label    string `json:"label"`
				Readonly bool   `json:"readonly"`
			} `json:"workspace_roots"`
			Auth struct {
				Type   string `json:"type"`
				Header string `json:"header"`
				Scheme string `json:"scheme"`
			} `json:"auth"`
			Stream struct {
				Transport   string   `json:"transport"`
				ContentType string   `json:"content_type"`
				Events      []string `json:"events"`
				DoneEvent   string   `json:"done_event"`
				ErrorEvent  string   `json:"error_event"`
			} `json:"stream"`
			Sessions struct {
				ListRoute      string `json:"list_route"`
				CreateRoute    string `json:"create_route"`
				GetRoute       string `json:"get_route"`
				RenameRoute    string `json:"rename_route"`
				DeleteRoute    string `json:"delete_route"`
				TitleExtKey    string `json:"title_ext_key"`
				MaxTitleLength int    `json:"max_title_length"`
				RequestSchemas map[string]struct {
					Required      []string   `json:"required"`
					RequiredOneOf [][]string `json:"required_one_of"`
					Optional      []string   `json:"optional"`
				} `json:"request_schemas"`
			} `json:"sessions"`
			RuntimeConfig struct {
				GetRoute       string   `json:"get_route"`
				UpdateRoute    string   `json:"update_route"`
				DeleteRoute    string   `json:"delete_route"`
				BindableTypes  []string `json:"bindable_types"`
				SnapshotFields []string `json:"snapshot_fields"`
				RequestSchemas map[string]struct {
					Required []string `json:"required"`
					Optional []string `json:"optional"`
				} `json:"request_schemas"`
			} `json:"runtime_config"`
			Messages struct {
				ListRoute      string   `json:"list_route"`
				MaxPageSize    int      `json:"max_page_size"`
				OrderValues    []string `json:"order_values"`
				RequestSchemas map[string]struct {
					Required []string `json:"required"`
					Optional []string `json:"optional"`
				} `json:"request_schemas"`
			} `json:"messages"`
			Trace struct {
				Events      []string `json:"events"`
				MaxPageSize int32    `json:"max_page_size"`
			} `json:"trace"`
			Approvals struct {
				ListRoute      string   `json:"list_route"`
				ResolveRoute   string   `json:"resolve_route"`
				DecisionRoot   string   `json:"decision_root"`
				DecisionFormat string   `json:"decision_format"`
				StatusValues   []string `json:"status_values"`
				DecisionValues []string `json:"decision_values"`
				ResponseFields []string `json:"response_fields"`
				RequestSchemas map[string]struct {
					Required []string `json:"required"`
					Optional []string `json:"optional"`
				} `json:"request_schemas"`
			} `json:"approvals"`
			Artifacts struct {
				Root             string   `json:"root"`
				ListRoute        string   `json:"list_route"`
				DownloadRoute    string   `json:"download_route"`
				DeleteRoute      string   `json:"delete_route"`
				MoveRoute        string   `json:"move_route"`
				MetadataFields   []string `json:"metadata_fields"`
				PreviewableMIMEs []string `json:"previewable_mimes"`
				MaxListItems     int32    `json:"max_list_items"`
				RequestSchemas   map[string]struct {
					Required      []string            `json:"required"`
					RequiredOneOf [][]string          `json:"required_one_of"`
					Optional      []string            `json:"optional"`
					Aliases       map[string][]string `json:"aliases"`
				} `json:"request_schemas"`
			} `json:"artifacts"`
			Workspace struct {
				IdentifierFields []string `json:"identifier_fields"`
				ReadableRoots    []string `json:"readable_roots"`
				WritableRoots    []string `json:"writable_roots"`
				ReadMaxBytes     int64    `json:"read_max_bytes"`
				BinaryEncoding   string   `json:"binary_encoding"`
				WriteRoute       string   `json:"write_route"`
				MoveRoute        string   `json:"move_route"`
				MkdirRoute       string   `json:"mkdir_route"`
				StatRoute        string   `json:"stat_route"`
				GrepRoute        string   `json:"grep_route"`
				GlobRoute        string   `json:"glob_route"`
				EditRoute        string   `json:"edit_route"`
				PatchRoute       string   `json:"patch_route"`
				RequestSchemas   map[string]struct {
					Required []string            `json:"required"`
					Optional []string            `json:"optional"`
					Aliases  map[string][]string `json:"aliases"`
				} `json:"request_schemas"`
			} `json:"workspace"`
			Sandbox struct {
				ExecRoute      string `json:"exec_route"`
				DefaultWorkdir string `json:"default_workdir"`
				MaxTimeoutSec  int    `json:"max_timeout_sec"`
				OutputMaxBytes int    `json:"output_max_bytes"`
				RequestSchemas map[string]struct {
					Required []string            `json:"required"`
					Optional []string            `json:"optional"`
					Aliases  map[string][]string `json:"aliases"`
				} `json:"request_schemas"`
			} `json:"sandbox"`
			Products struct {
				ProductTypes            []string `json:"product_types"`
				VisibilityValues        []string `json:"visibility_values"`
				StatusValues            []string `json:"status_values"`
				SourceRefTypes          []string `json:"source_ref_types"`
				ListRoute               string   `json:"list_route"`
				GetRoute                string   `json:"get_route"`
				InstallRoute            string   `json:"install_route"`
				UpgradeRoute            string   `json:"upgrade_route"`
				UninstallRoute          string   `json:"uninstall_route"`
				MarketplaceListRoute    string   `json:"marketplace_list_route"`
				MarketplaceGetRoute     string   `json:"marketplace_get_route"`
				MarketplaceInstallRoute string   `json:"marketplace_install_route"`
				RequestSchemas          map[string]struct {
					Required []string `json:"required"`
					Optional []string `json:"optional"`
				} `json:"request_schemas"`
			} `json:"products"`
			Skills struct {
				EntryFile      string            `json:"entry_file"`
				FileRoots      []string          `json:"file_roots"`
				PublishScopes  map[string]int8   `json:"publish_scopes"`
				Routes         map[string]string `json:"routes"`
				RequestSchemas map[string]struct {
					Required []string            `json:"required"`
					Optional []string            `json:"optional"`
					Aliases  map[string][]string `json:"aliases"`
				} `json:"request_schemas"`
				Assets struct {
					Root            string   `json:"root"`
					ListRoute       string   `json:"list_route"`
					GetRoute        string   `json:"get_route"`
					UpsertRoute     string   `json:"upsert_route"`
					DeleteRoute     string   `json:"delete_route"`
					AllowedMIMEs    []string `json:"allowed_mimes"`
					ContentEncoding string   `json:"content_encoding"`
				} `json:"assets"`
				AgentTool struct {
					Name    string   `json:"name"`
					Actions []string `json:"actions"`
				} `json:"agent_tool"`
			} `json:"skills"`
			Harness struct {
				DeliverableRoot         string   `json:"deliverable_root"`
				PlanPath                string   `json:"plan_path"`
				SessionPlanPathTemplate string   `json:"session_plan_path_template"`
				ToolOutputRoot          string   `json:"tool_output_root"`
				StateRoute              string   `json:"state_route"`
				StateFields             []string `json:"state_fields"`
				PlanUpdateRoute         string   `json:"plan_update_route"`
				PlanRoute               string   `json:"plan_route"`
				ToolOutputsRoute        string   `json:"tool_outputs_route"`
				ToolOutputFields        []string `json:"tool_output_entry_fields"`
				CleanupRoute            string   `json:"cleanup_route"`
				SnapshotRoute           string   `json:"snapshot_route"`
				SnapshotFields          []string `json:"snapshot_fields"`
				ResumeRoute             string   `json:"resume_route"`
				ResumeFields            []string `json:"resume_fields"`
				SkillRuntimeRoot        string   `json:"skill_runtime_root"`
				SkillEntryFile          string   `json:"skill_entry_file"`
				SkillFileRoots          []string `json:"skill_file_roots"`
				Tools                   []struct {
					Name      string `json:"name"`
					Category  string `json:"category"`
					Available string `json:"available"`
					Mutates   bool   `json:"mutates"`
				} `json:"tools"`
			} `json:"harness"`
			ExternalAPI struct {
				BasePath             string            `json:"base_path"`
				ProtocolVersion      string            `json:"protocol_version"`
				Auth                 map[string]string `json:"auth"`
				SchemaRoute          string            `json:"schema_route"`
				SessionAuthSupported bool              `json:"session_auth_supported"`
				Transports           []string          `json:"transports"`
				IdentifierFields     []string          `json:"identifier_fields"`
				Capabilities         []string          `json:"capabilities"`
				EntryRoutes          map[string]string `json:"entry_routes"`
				RequestSchemas       map[string]struct {
					Required      []string   `json:"required"`
					RequiredOneOf [][]string `json:"required_one_of"`
					Optional      []string   `json:"optional"`
				} `json:"request_schemas"`
				ClientMetadata map[string]string `json:"client_metadata"`
			} `json:"external_api"`
			OpenAPI struct {
				Version     string `json:"version"`
				Title       string `json:"title"`
				SchemaRoute string `json:"schema_route"`
				Operations  []struct {
					OperationID   string `json:"operation_id"`
					Method        string `json:"method"`
					Path          string `json:"path"`
					Category      string `json:"category"`
					Transport     string `json:"transport"`
					Mutates       bool   `json:"mutates"`
					RequestSchema string `json:"request_schema"`
				} `json:"operations"`
			} `json:"openapi"`
			SkillPublishScopes map[string]int8   `json:"skill_publish_scopes"`
			Routes             map[string]string `json:"routes"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "super-agent.app-server.v1", got.Data.ProtocolVersion)
	assert.Contains(t, got.Data.Capabilities, "harness")
	assert.Contains(t, got.Data.Capabilities, "artifacts")
	assert.Contains(t, got.Data.Capabilities, "traces")
	assert.Equal(t, "bearer", got.Data.Auth.Type)
	assert.Equal(t, "Authorization", got.Data.Auth.Header)
	assert.Equal(t, "Bearer", got.Data.Auth.Scheme)
	assert.Equal(t, "sse", got.Data.Stream.Transport)
	assert.Equal(t, "text/event-stream", got.Data.Stream.ContentType)
	assert.Equal(t, "conversation.stream.done", got.Data.Stream.DoneEvent)
	assert.Equal(t, "conversation.error", got.Data.Stream.ErrorEvent)
	assert.Contains(t, got.Data.Stream.Events, "conversation.message.delta")
	assert.Contains(t, got.Data.Stream.Events, "conversation.message.completed")
	assert.Contains(t, got.Data.Capabilities, "sessions")
	assert.Equal(t, "POST /api/super-agent/sessions/list", got.Data.Sessions.ListRoute)
	assert.Equal(t, "POST /api/super-agent/sessions/create", got.Data.Sessions.CreateRoute)
	assert.Equal(t, "POST /api/super-agent/sessions/get", got.Data.Sessions.GetRoute)
	assert.Equal(t, "POST /api/super-agent/sessions/rename", got.Data.Sessions.RenameRoute)
	assert.Equal(t, "POST /api/super-agent/sessions/delete", got.Data.Sessions.DeleteRoute)
	assert.Equal(t, "super_agent_session_title", got.Data.Sessions.TitleExtKey)
	assert.Equal(t, 128, got.Data.Sessions.MaxTitleLength)
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.Sessions.RequestSchemas["create"].RequiredOneOf)
	assert.Contains(t, got.Data.Sessions.RequestSchemas["create"].Optional, "title")
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.Sessions.RequestSchemas["list"].RequiredOneOf)
	assert.Contains(t, got.Data.Sessions.RequestSchemas["list"].Optional, "page_size")
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.Sessions.RequestSchemas["get"].Required)
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "include_snapshot")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "message_limit")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "include_runs")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "run_limit")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "include_workspace")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "workspace_path")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "workspace_recursive")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "include_context")
	assert.Contains(t, got.Data.Sessions.RequestSchemas["get"].Optional, "include_tool_output_content")
	assert.ElementsMatch(t, []string{"conversation_id", "title"}, got.Data.Sessions.RequestSchemas["rename"].Required)
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.Sessions.RequestSchemas["delete"].Required)
	assert.Contains(t, got.Data.Capabilities, "runtime_config")
	assert.Equal(t, "POST /api/super-agent/runtime-config/get", got.Data.RuntimeConfig.GetRoute)
	assert.Equal(t, "POST /api/super-agent/runtime-config/update", got.Data.RuntimeConfig.UpdateRoute)
	assert.Equal(t, "POST /api/super-agent/runtime-config/delete", got.Data.RuntimeConfig.DeleteRoute)
	assert.Contains(t, got.Data.RuntimeConfig.BindableTypes, "standard_skill")
	assert.Contains(t, got.Data.RuntimeConfig.BindableTypes, "mcp_server")
	assert.Contains(t, got.Data.RuntimeConfig.SnapshotFields, "skills")
	assert.ElementsMatch(t, []string{"conversation_id", "space_id"}, got.Data.RuntimeConfig.RequestSchemas["update"].Required)
	assert.Contains(t, got.Data.RuntimeConfig.RequestSchemas["update"].Optional, "skill_product_ids")
	assert.Contains(t, got.Data.Capabilities, "messages")
	assert.Equal(t, "POST /api/super-agent/messages/list", got.Data.Messages.ListRoute)
	assert.Equal(t, 100, got.Data.Messages.MaxPageSize)
	assert.ElementsMatch(t, []string{"ASC", "DESC"}, got.Data.Messages.OrderValues)
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.Messages.RequestSchemas["list"].Required)
	assert.Contains(t, got.Data.Messages.RequestSchemas["list"].Optional, "limit")
	assert.Contains(t, got.Data.Messages.RequestSchemas["list"].Optional, "after_id")
	assert.Contains(t, got.Data.Messages.RequestSchemas["list"].Optional, "before_id")
	assert.Contains(t, got.Data.Trace.Events, "conversation.chat.created")
	assert.Contains(t, got.Data.Trace.Events, "conversation.message.completed")
	assert.Contains(t, got.Data.Trace.Events, superagenttrace.EventToolStarted)
	assert.Contains(t, got.Data.Trace.Events, superagenttrace.EventToolCompleted)
	assert.Contains(t, got.Data.Trace.Events, superagenttrace.EventToolFailed)
	assert.Contains(t, got.Data.Trace.Events, superagenttrace.EventPlanUpdated)
	assert.Contains(t, got.Data.Trace.Events, "context.compacted")
	assert.Equal(t, int32(100), got.Data.Trace.MaxPageSize)
	assert.Contains(t, got.Data.Capabilities, "approvals")
	assert.Equal(t, "POST /api/super-agent/approvals/list", got.Data.Approvals.ListRoute)
	assert.Equal(t, "POST /api/super-agent/approvals/resolve", got.Data.Approvals.ResolveRoute)
	assert.Equal(t, "/workspace/.agent/approvals", got.Data.Approvals.DecisionRoot)
	assert.Equal(t, "json", got.Data.Approvals.DecisionFormat)
	assert.ElementsMatch(t, []string{"pending", "approved", "rejected", "cancelled"}, got.Data.Approvals.StatusValues)
	assert.ElementsMatch(t, []string{"approve", "reject", "cancel"}, got.Data.Approvals.DecisionValues)
	assert.Contains(t, got.Data.Approvals.ResponseFields, "decision_path")
	assert.Contains(t, got.Data.Approvals.ResponseFields, "persisted")
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.Approvals.RequestSchemas["list"].Required)
	assert.Contains(t, got.Data.Approvals.RequestSchemas["list"].Optional, "run_id")
	assert.ElementsMatch(t, []string{"approval_id", "decision"}, got.Data.Approvals.RequestSchemas["resolve"].Required)
	assert.Contains(t, got.Data.Approvals.RequestSchemas["resolve"].Optional, "note")
	assert.Equal(t, "/outputs", got.Data.Artifacts.Root)
	assert.Equal(t, "POST /api/super-agent/artifacts/list", got.Data.Artifacts.ListRoute)
	assert.Equal(t, "POST /api/super-agent/artifacts/download", got.Data.Artifacts.DownloadRoute)
	assert.Equal(t, "POST /api/super-agent/artifacts/delete", got.Data.Artifacts.DeleteRoute)
	assert.Equal(t, "POST /api/super-agent/artifacts/move", got.Data.Artifacts.MoveRoute)
	assert.Contains(t, got.Data.Artifacts.MetadataFields, "sha256")
	assert.Contains(t, got.Data.Artifacts.MetadataFields, "mime")
	assert.Contains(t, got.Data.Artifacts.MetadataFields, "kind")
	assert.Contains(t, got.Data.Artifacts.MetadataFields, "preview_type")
	assert.Contains(t, got.Data.Artifacts.MetadataFields, "summary")
	assert.Contains(t, got.Data.Artifacts.PreviewableMIMEs, "text/html")
	assert.Equal(t, int32(200), got.Data.Artifacts.MaxListItems)
	assert.NotNil(t, got.Data.Artifacts.RequestSchemas["list"].Required)
	assert.NotNil(t, got.Data.Artifacts.RequestSchemas["download"].Required)
	assert.NotNil(t, got.Data.Artifacts.RequestSchemas["delete"].Required)
	assert.Empty(t, got.Data.Artifacts.RequestSchemas["list"].Required)
	assert.Empty(t, got.Data.Artifacts.RequestSchemas["download"].Required)
	assert.Empty(t, got.Data.Artifacts.RequestSchemas["delete"].Required)
	assert.Contains(t, got.Data.Artifacts.RequestSchemas["list"].Optional, "path")
	assert.Contains(t, got.Data.Artifacts.RequestSchemas["list"].Optional, "limit")
	assert.ElementsMatch(t, [][]string{{"artifact_id", "path"}}, got.Data.Artifacts.RequestSchemas["download"].RequiredOneOf)
	assert.ElementsMatch(t, [][]string{{"artifact_id", "path"}}, got.Data.Artifacts.RequestSchemas["delete"].RequiredOneOf)
	assert.ElementsMatch(t, []string{"target_path"}, got.Data.Artifacts.RequestSchemas["move"].Required)
	assert.ElementsMatch(t, [][]string{{"artifact_id", "path"}}, got.Data.Artifacts.RequestSchemas["move"].RequiredOneOf)
	assert.Contains(t, got.Data.Capabilities, "products")
	assert.Contains(t, got.Data.Products.ProductTypes, "standard_skill")
	assert.Contains(t, got.Data.Products.ProductTypes, "mcp_server")
	assert.Contains(t, got.Data.Products.VisibilityValues, "global")
	assert.Contains(t, got.Data.Products.StatusValues, "published")
	assert.Contains(t, got.Data.Products.SourceRefTypes, "skill")
	assert.Equal(t, "POST /api/super-agent/products/list", got.Data.Products.ListRoute)
	assert.Equal(t, "POST /api/super-agent/products/get", got.Data.Products.GetRoute)
	assert.Equal(t, "POST /api/super-agent/products/install", got.Data.Products.InstallRoute)
	assert.Equal(t, "POST /api/super-agent/products/upgrade", got.Data.Products.UpgradeRoute)
	assert.Equal(t, "POST /api/super-agent/products/uninstall", got.Data.Products.UninstallRoute)
	assert.Equal(t, "POST /api/super-agent/marketplace/products/list", got.Data.Products.MarketplaceListRoute)
	assert.Equal(t, "POST /api/super-agent/marketplace/products/get", got.Data.Products.MarketplaceGetRoute)
	assert.Equal(t, "POST /api/super-agent/marketplace/products/install", got.Data.Products.MarketplaceInstallRoute)
	assert.ElementsMatch(t, []string{"space_id"}, got.Data.Products.RequestSchemas["list"].Required)
	assert.Contains(t, got.Data.Products.RequestSchemas["list"].Optional, "type")
	assert.ElementsMatch(t, []string{"space_id", "product_id"}, got.Data.Products.RequestSchemas["install"].Required)
	assert.Contains(t, got.Data.Products.RequestSchemas["install"].Optional, "version")
	assert.Contains(t, got.Data.Workspace.IdentifierFields, "agent_id")
	assert.Contains(t, got.Data.Workspace.IdentifierFields, "bot_id")
	assert.Contains(t, got.Data.Workspace.ReadableRoots, "/skills")
	assert.Contains(t, got.Data.Workspace.WritableRoots, "/workspace")
	assert.Equal(t, int64(25*1024*1024), got.Data.Workspace.ReadMaxBytes)
	assert.Equal(t, "base64", got.Data.Workspace.BinaryEncoding)
	assert.Equal(t, "POST /api/super-agent/workspace/write", got.Data.Workspace.WriteRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/move", got.Data.Workspace.MoveRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/mkdir", got.Data.Workspace.MkdirRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/stat", got.Data.Workspace.StatRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/grep", got.Data.Workspace.GrepRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/glob", got.Data.Workspace.GlobRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/edit", got.Data.Workspace.EditRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/patch", got.Data.Workspace.PatchRoute)
	assert.ElementsMatch(t, []string{"patch"}, got.Data.Workspace.RequestSchemas["patch"].Required)
	assert.Contains(t, got.Data.Workspace.RequestSchemas["patch"].Optional, "workdir")
	assert.Contains(t, got.Data.Workspace.RequestSchemas["patch"].Optional, "work_dir")
	assert.ElementsMatch(t, []string{"work_dir"}, got.Data.Workspace.RequestSchemas["patch"].Aliases["workdir"])
	assert.Equal(t, "POST /api/super-agent/sandbox/exec", got.Data.Sandbox.ExecRoute)
	assert.Equal(t, "/workspace", got.Data.Sandbox.DefaultWorkdir)
	assert.Equal(t, 300, got.Data.Sandbox.MaxTimeoutSec)
	assert.Equal(t, 64*1024, got.Data.Sandbox.OutputMaxBytes)
	assert.ElementsMatch(t, []string{"command"}, got.Data.Sandbox.RequestSchemas["exec"].Required)
	assert.Contains(t, got.Data.Sandbox.RequestSchemas["exec"].Optional, "workdir")
	assert.Contains(t, got.Data.Sandbox.RequestSchemas["exec"].Optional, "work_dir")
	assert.ElementsMatch(t, []string{"work_dir"}, got.Data.Sandbox.RequestSchemas["exec"].Aliases["workdir"])
	assert.Equal(t, "SKILL.md", got.Data.Skills.EntryFile)
	assert.ElementsMatch(t, []string{"SKILL.md", "scripts/", "references/", "templates/", "assets/"}, got.Data.Skills.FileRoots)
	assert.Equal(t, int8(1), got.Data.Skills.PublishScopes["private"])
	assert.Equal(t, int8(2), got.Data.Skills.PublishScopes["space"])
	assert.Equal(t, int8(3), got.Data.Skills.PublishScopes["global"])
	assert.Equal(t, "POST /api/super-agent/skills/create", got.Data.Skills.Routes["create"])
	assert.Equal(t, "GET /api/super-agent/skills/list", got.Data.Skills.Routes["list"])
	assert.Equal(t, "GET /api/super-agent/marketplace/list", got.Data.Skills.Routes["marketplace.list"])
	assert.Equal(t, "GET /api/super-agent/marketplace/get", got.Data.Skills.Routes["marketplace.get"])
	assert.Equal(t, "POST /api/super-agent/marketplace/install", got.Data.Skills.Routes["marketplace.install"])
	assert.Equal(t, "GET /api/super-agent/skills/assets/list", got.Data.Skills.Routes["assets.list"])
	assert.Equal(t, "GET /api/super-agent/skills/assets/get", got.Data.Skills.Routes["assets.get"])
	assert.Equal(t, "POST /api/super-agent/skills/assets/upsert", got.Data.Skills.Routes["assets.upsert"])
	assert.Equal(t, "POST /api/super-agent/skills/validate-package", got.Data.Skills.Routes["validate_package"])
	assert.Equal(t, "POST /api/super-agent/skills/import", got.Data.Skills.Routes["import"])
	assert.Equal(t, "POST /api/super-agent/skills/import-runtime", got.Data.Skills.Routes["import_runtime"])
	assert.Equal(t, "POST /api/super-agent/skills/export", got.Data.Skills.Routes["export"])
	assert.ElementsMatch(t, []string{"space_id", "name", "files"}, got.Data.Skills.RequestSchemas["create"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["create"].Optional, "description")
	assert.Contains(t, got.Data.Skills.RequestSchemas["create"].Optional, "prompt")
	assert.Contains(t, got.Data.Skills.RequestSchemas["create"].Optional, "icon_uri")
	assert.ElementsMatch(t, []string{"space_id", "content"}, got.Data.Skills.RequestSchemas["import"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["import"].Optional, "filename")
	assert.Contains(t, got.Data.Skills.RequestSchemas["import"].Optional, "icon_uri")
	assert.ElementsMatch(t, []string{"content"}, got.Data.Skills.RequestSchemas["validate_package"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["validate_package"].Optional, "filename")
	assert.ElementsMatch(t, []string{"agent_id", "name"}, got.Data.Skills.RequestSchemas["import_runtime"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["import_runtime"].Optional, "publish_scope")
	assert.Contains(t, got.Data.Skills.RequestSchemas["import_runtime"].Optional, "skill_id")
	assert.Contains(t, got.Data.Skills.RequestSchemas["import_runtime"].Optional, "connector_id")
	assert.ElementsMatch(t, []string{"space_id", "skill_id"}, got.Data.Skills.RequestSchemas["export"].Required)
	assert.ElementsMatch(t, []string{"space_id", "skill_id"}, got.Data.Skills.RequestSchemas["update"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["update"].Optional, "files")
	assert.ElementsMatch(t, []string{"space_id", "skill_id", "scope"}, got.Data.Skills.RequestSchemas["publish"].Required)
	assert.ElementsMatch(t, []string{"space_id"}, got.Data.Skills.RequestSchemas["list"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["list"].Optional, "page_size")
	assert.Contains(t, got.Data.Skills.RequestSchemas["list"].Optional, "keyword")
	assert.ElementsMatch(t, []string{"space_id"}, got.Data.Skills.RequestSchemas["marketplace.list"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["marketplace.list"].Optional, "scope")
	assert.ElementsMatch(t, []string{"space_id", "skill_id"}, got.Data.Skills.RequestSchemas["marketplace.install"].Required)
	assert.ElementsMatch(t, []string{"space_id", "skill_id"}, got.Data.Skills.RequestSchemas["assets.list"].Required)
	assert.ElementsMatch(t, []string{"space_id", "skill_id", "path"}, got.Data.Skills.RequestSchemas["assets.get"].Required)
	assert.ElementsMatch(t, []string{"space_id", "skill_id", "path", "content"}, got.Data.Skills.RequestSchemas["assets.upsert"].Required)
	assert.Contains(t, got.Data.Skills.RequestSchemas["assets.upsert"].Optional, "mime")
	assert.ElementsMatch(t, []string{"space_id", "skill_id", "path"}, got.Data.Skills.RequestSchemas["assets.delete"].Required)
	assert.Equal(t, "assets/", got.Data.Skills.Assets.Root)
	assert.Equal(t, "GET /api/super-agent/skills/assets/list", got.Data.Skills.Assets.ListRoute)
	assert.Equal(t, "GET /api/super-agent/skills/assets/get", got.Data.Skills.Assets.GetRoute)
	assert.Equal(t, "POST /api/super-agent/skills/assets/upsert", got.Data.Skills.Assets.UpsertRoute)
	assert.Equal(t, "POST /api/super-agent/skills/assets/delete", got.Data.Skills.Assets.DeleteRoute)
	assert.Contains(t, got.Data.Skills.Assets.AllowedMIMEs, "image/png")
	assert.Equal(t, "utf8-or-data-url", got.Data.Skills.Assets.ContentEncoding)
	assert.Equal(t, "skill_manage", got.Data.Skills.AgentTool.Name)
	assert.ElementsMatch(t, []string{"create", "list", "read", "diff", "write_file", "edit", "remove_file", "delete"}, got.Data.Skills.AgentTool.Actions)
	assert.Equal(t, "/outputs", got.Data.Harness.DeliverableRoot)
	assert.Equal(t, "/workspace/.plan.json", got.Data.Harness.PlanPath)
	assert.Equal(t, "/workspace/.agent/sessions/{conversation_id}/plan.json", got.Data.Harness.SessionPlanPathTemplate)
	assert.Equal(t, "/workspace/.agent/tooloutputs", got.Data.Harness.ToolOutputRoot)
	assert.Equal(t, "POST /api/super-agent/harness/state", got.Data.Harness.StateRoute)
	assert.ElementsMatch(t, []string{"plan", "tool_outputs", "runtime_skills", "context"}, got.Data.Harness.StateFields)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.state"].Optional, "conversation_id")
	assert.Equal(t, "POST /api/super-agent/harness/plan", got.Data.Harness.PlanUpdateRoute)
	assert.Equal(t, "POST /api/super-agent/workspace/read", got.Data.Harness.PlanRoute)
	assert.Equal(t, "POST /api/super-agent/harness/tool-outputs", got.Data.Harness.ToolOutputsRoute)
	assert.Contains(t, got.Data.Harness.ToolOutputFields, "tool_call_id")
	assert.Contains(t, got.Data.Harness.ToolOutputFields, "arguments")
	assert.Contains(t, got.Data.Harness.ToolOutputFields, "result")
	assert.Equal(t, "POST /api/super-agent/harness/cleanup", got.Data.Harness.CleanupRoute)
	assert.Equal(t, "POST /api/super-agent/harness/snapshot", got.Data.Harness.SnapshotRoute)
	assert.Contains(t, got.Data.Harness.SnapshotFields, "approval_decisions")
	assert.Equal(t, "POST /api/super-agent/harness/resume", got.Data.Harness.ResumeRoute)
	assert.Contains(t, got.Data.Harness.ResumeFields, "version")
	assert.Contains(t, got.Data.Harness.ResumeFields, "summary_path")
	assert.Contains(t, got.Data.Harness.ResumeFields, "plan_path")
	assert.Contains(t, got.Data.Harness.ResumeFields, "prompt")
	assert.Contains(t, got.Data.Harness.ResumeFields, "components")
	assert.Equal(t, "/skills", got.Data.Harness.SkillRuntimeRoot)
	assert.Equal(t, "SKILL.md", got.Data.Harness.SkillEntryFile)
	assert.ElementsMatch(t, []string{"SKILL.md", "scripts/", "references/", "templates/", "assets/"}, got.Data.Harness.SkillFileRoots)
	assert.Contains(t, got.Data.ExternalAPI.EntryRoutes, "harness.tool_outputs")
	assert.Equal(t, "POST /api/super-agent/harness/tool-outputs", got.Data.ExternalAPI.EntryRoutes["harness.tool_outputs"])
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.tool_outputs"].Optional, "include_content")
	assert.Contains(t, got.Data.ExternalAPI.EntryRoutes, "harness.cleanup")
	assert.Equal(t, "POST /api/super-agent/harness/cleanup", got.Data.ExternalAPI.EntryRoutes["harness.cleanup"])
	assert.Empty(t, got.Data.ExternalAPI.RequestSchemas["harness.cleanup"].Required)
	assert.ElementsMatch(t, [][]string{{"paths"}, {"keep_latest"}}, got.Data.ExternalAPI.RequestSchemas["harness.cleanup"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.cleanup"].Optional, "dry_run")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.cleanup"].Optional, "keep_latest")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.cleanup"].Optional, "prefix")
	assert.Contains(t, got.Data.ExternalAPI.EntryRoutes, "harness.snapshot")
	assert.Equal(t, "POST /api/super-agent/harness/snapshot", got.Data.ExternalAPI.EntryRoutes["harness.snapshot"])
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_trace")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_artifacts")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_messages")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "message_limit")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_runs")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "run_limit")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_workspace")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "workspace_path")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "workspace_recursive")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_context")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.snapshot"].Optional, "include_tool_output_content")
	assert.Contains(t, got.Data.ExternalAPI.EntryRoutes, "harness.resume")
	assert.Equal(t, "POST /api/super-agent/harness/resume", got.Data.ExternalAPI.EntryRoutes["harness.resume"])
	assert.ElementsMatch(t, [][]string{{"conversation_id"}}, got.Data.ExternalAPI.RequestSchemas["harness.resume"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.resume"].Optional, "agent_id")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.resume"].Optional, "bot_id")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.resume"].Optional, "message_limit")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.resume"].Optional, "artifact_limit")
	assert.Equal(t, "POST /api/super-agent/harness/resume", got.Data.Routes["harness.resume"])
	assert.Contains(t, got.Data.ExternalAPI.EntryRoutes, "messages.list")
	assert.Equal(t, "POST /api/super-agent/messages/list", got.Data.ExternalAPI.EntryRoutes["messages.list"])
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["messages.list"].Required)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["messages.list"].Optional, "order_by")
	harnessTools := map[string]struct {
		Category  string
		Available string
		Mutates   bool
	}{}
	for _, tool := range got.Data.Harness.Tools {
		harnessTools[tool.Name] = struct {
			Category  string
			Available string
			Mutates   bool
		}{Category: tool.Category, Available: tool.Available, Mutates: tool.Mutates}
	}
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"sandbox", "super_agent", true}, harnessTools["write_file"])
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"sandbox", "super_agent", true}, harnessTools["apply_patch"])
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"sandbox", "super_agent", false}, harnessTools["grep"])
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"delegation", "super_agent", true}, harnessTools["deep_task"])
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"skills", "when_agent_has_bound_skills", false}, harnessTools["read_skill"])
	assert.Equal(t, struct {
		Category  string
		Available string
		Mutates   bool
	}{"web", "super_agent", false}, harnessTools["web_search"])
	assert.Equal(t, "/api/super-agent", got.Data.ExternalAPI.BasePath)
	assert.Equal(t, "super-agent.app-server.v1", got.Data.ExternalAPI.ProtocolVersion)
	assert.Equal(t, "bearer", got.Data.ExternalAPI.Auth["type"])
	assert.Equal(t, "Authorization", got.Data.ExternalAPI.Auth["header"])
	assert.Equal(t, "Bearer", got.Data.ExternalAPI.Auth["scheme"])
	assert.Equal(t, "GET /api/super-agent/openapi.json", got.Data.ExternalAPI.SchemaRoute)
	assert.True(t, got.Data.ExternalAPI.SessionAuthSupported)
	assert.ElementsMatch(t, []string{"json", "sse"}, got.Data.ExternalAPI.Transports)
	assert.ElementsMatch(t, []string{"agent_id", "bot_id"}, got.Data.ExternalAPI.IdentifierFields)
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "sandbox")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "workspace")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "skills")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "products")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "runtime_config")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "harness")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "artifacts")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "approvals")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "sessions")
	assert.Contains(t, got.Data.ExternalAPI.Capabilities, "traces")
	assert.Equal(t, "POST /api/super-agent/runs/create", got.Data.ExternalAPI.EntryRoutes["create"])
	assert.Equal(t, "POST /api/super-agent/runs/stream", got.Data.ExternalAPI.EntryRoutes["stream"])
	assert.Equal(t, "POST /api/super-agent/runs/reply", got.Data.ExternalAPI.EntryRoutes["reply"])
	assert.Equal(t, "POST /api/super-agent/runs/get", got.Data.ExternalAPI.EntryRoutes["get"])
	assert.Equal(t, "POST /api/super-agent/runs/list", got.Data.ExternalAPI.EntryRoutes["list"])
	assert.Equal(t, "POST /api/super-agent/runs/cancel", got.Data.ExternalAPI.EntryRoutes["cancel"])
	assert.Equal(t, "POST /api/super-agent/traces/get", got.Data.ExternalAPI.EntryRoutes["traces.get"])
	assert.Equal(t, "POST /api/super-agent/sessions/create", got.Data.ExternalAPI.EntryRoutes["sessions.create"])
	assert.Equal(t, "POST /api/super-agent/sessions/get", got.Data.ExternalAPI.EntryRoutes["sessions.get"])
	assert.Equal(t, "POST /api/super-agent/sessions/list", got.Data.ExternalAPI.EntryRoutes["sessions.list"])
	assert.Equal(t, "POST /api/super-agent/sessions/rename", got.Data.ExternalAPI.EntryRoutes["sessions.rename"])
	assert.Equal(t, "POST /api/super-agent/sessions/delete", got.Data.ExternalAPI.EntryRoutes["sessions.delete"])
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.ExternalAPI.RequestSchemas["create"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["create"].Optional, "additional_messages")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["create"].Optional, "client_id")
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["reply"].Required)
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.ExternalAPI.RequestSchemas["reply"].RequiredOneOf)
	assert.ElementsMatch(t, []string{"run_id"}, got.Data.ExternalAPI.RequestSchemas["get"].Required)
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["list"].Required)
	assert.ElementsMatch(t, []string{"run_id"}, got.Data.ExternalAPI.RequestSchemas["cancel"].Required)
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["traces.get"].Required)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["traces.get"].Optional, "run_id")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["traces.get"].Optional, "limit")
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.ExternalAPI.RequestSchemas["sessions.list"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["sessions.list"].Optional, "page_size")
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, got.Data.ExternalAPI.RequestSchemas["sessions.create"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["sessions.create"].Optional, "title")
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["sessions.get"].Required)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["sessions.get"].Optional, "include_snapshot")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["sessions.get"].Optional, "include_context")
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["sessions.get"].Optional, "include_tool_output_content")
	assert.ElementsMatch(t, []string{"conversation_id", "title"}, got.Data.ExternalAPI.RequestSchemas["sessions.rename"].Required)
	assert.ElementsMatch(t, []string{"conversation_id"}, got.Data.ExternalAPI.RequestSchemas["sessions.delete"].Required)
	assert.Empty(t, got.Data.ExternalAPI.RequestSchemas["harness.plan"].Required)
	assert.ElementsMatch(t, [][]string{{"plan", "items"}}, got.Data.ExternalAPI.RequestSchemas["harness.plan"].RequiredOneOf)
	assert.Contains(t, got.Data.ExternalAPI.RequestSchemas["harness.plan"].Optional, "conversation_id")
	assert.Empty(t, got.Data.ExternalAPI.RequestSchemas["workspace.move"].Required)
	assert.ElementsMatch(t, [][]string{{"path", "from_path"}, {"target_path", "to_path"}}, got.Data.ExternalAPI.RequestSchemas["workspace.move"].RequiredOneOf)
	assert.Equal(t, "super_agent_app_server", got.Data.ExternalAPI.ClientMetadata["app_server_flag"])
	assert.Equal(t, "super_agent_capabilities", got.Data.ExternalAPI.ClientMetadata["capabilities_param"])
	assert.Equal(t, "sessions,messages,sandbox,workspace,skills,products,runtime_config,harness,artifacts,approvals,traces", got.Data.ExternalAPI.ClientMetadata["capabilities_value"])
	assert.Equal(t, "3.1.0", got.Data.OpenAPI.Version)
	assert.Equal(t, "Super Agent App Server API", got.Data.OpenAPI.Title)
	assert.Equal(t, "GET /api/super-agent/openapi.json", got.Data.OpenAPI.SchemaRoute)
	openAPIOperations := map[string]struct {
		Method        string
		Path          string
		Transport     string
		Mutates       bool
		RequestSchema string
	}{}
	for _, operation := range got.Data.OpenAPI.Operations {
		openAPIOperations[operation.OperationID] = struct {
			Method        string
			Path          string
			Transport     string
			Mutates       bool
			RequestSchema string
		}{
			Method:        operation.Method,
			Path:          operation.Path,
			Transport:     operation.Transport,
			Mutates:       operation.Mutates,
			RequestSchema: operation.RequestSchema,
		}
		expectedRoute := operation.Method + " " + operation.Path
		assert.Equalf(
			t,
			expectedRoute,
			got.Data.ExternalAPI.EntryRoutes[operation.OperationID],
			"external_api.entry_routes should expose OpenAPI operation %s",
			operation.OperationID,
		)
		assert.Containsf(
			t,
			got.Data.ExternalAPI.RequestSchemas,
			operation.OperationID,
			"external_api.request_schemas should expose OpenAPI operation %s",
			operation.OperationID,
		)
	}
	assert.Equal(t, struct {
		Method        string
		Path          string
		Transport     string
		Mutates       bool
		RequestSchema string
	}{"POST", "/api/super-agent/runs/create", "json", true, "RunsCreateRequest"}, openAPIOperations["runs.create"])
	assert.Equal(t, "sse", openAPIOperations["runs.stream"].Transport)
	assert.Equal(t, "POST", openAPIOperations["sessions.create"].Method)
	assert.Equal(t, "SessionsGetRequest", openAPIOperations["sessions.get"].RequestSchema)
	assert.Equal(t, "SessionsDeleteRequest", openAPIOperations["sessions.delete"].RequestSchema)
	assert.Equal(t, "RuntimeConfigUpdateRequest", openAPIOperations["runtime_config.update"].RequestSchema)
	assert.Equal(t, "MessagesListRequest", openAPIOperations["messages.list"].RequestSchema)
	assert.Equal(t, "HarnessSnapshotRequest", openAPIOperations["harness.snapshot"].RequestSchema)
	assert.Equal(t, "WorkspacePatchRequest", openAPIOperations["workspace.patch"].RequestSchema)
	assert.Equal(t, "SkillsImportRequest", openAPIOperations["skills.import"].RequestSchema)
	assert.Equal(t, "ProductsInstallRequest", openAPIOperations["products.install"].RequestSchema)
	assert.Equal(t, "MarketplaceProductsListRequest", openAPIOperations["marketplace.products.list"].RequestSchema)
	assert.Equal(t, int8(1), got.Data.SkillPublishScopes["private"])
	assert.Equal(t, int8(2), got.Data.SkillPublishScopes["space"])
	assert.Equal(t, int8(3), got.Data.SkillPublishScopes["global"])
	assert.Contains(t, got.Data.Routes, "runs.create")
	assert.Equal(t, "GET /api/super-agent/openapi.json", got.Data.Routes["openapi"])
	assert.Equal(t, "POST /api/super-agent/runs/create", got.Data.Routes["runs.create"])
	assert.Equal(t, "POST /api/super-agent/runs/get", got.Data.Routes["runs.get"])
	assert.Equal(t, "POST /api/super-agent/runs/list", got.Data.Routes["runs.list"])
	assert.Equal(t, "POST /api/super-agent/runs/reply", got.Data.Routes["runs.reply"])
	assert.Equal(t, "POST /api/super-agent/runs/cancel", got.Data.Routes["runs.cancel"])
	assert.Equal(t, "POST /api/super-agent/sessions/create", got.Data.Routes["sessions.create"])
	assert.Equal(t, "POST /api/super-agent/sessions/get", got.Data.Routes["sessions.get"])
	assert.Equal(t, "POST /api/super-agent/sessions/list", got.Data.Routes["sessions.list"])
	assert.Equal(t, "POST /api/super-agent/sessions/rename", got.Data.Routes["sessions.rename"])
	assert.Equal(t, "POST /api/super-agent/sessions/delete", got.Data.Routes["sessions.delete"])
	assert.Equal(t, "POST /api/super-agent/runtime-config/get", got.Data.Routes["runtime_config.get"])
	assert.Equal(t, "POST /api/super-agent/runtime-config/update", got.Data.Routes["runtime_config.update"])
	assert.Equal(t, "POST /api/super-agent/runtime-config/delete", got.Data.Routes["runtime_config.delete"])
	assert.Equal(t, "POST /api/super-agent/messages/list", got.Data.Routes["messages.list"])
	assert.Equal(t, "POST /api/super-agent/harness/state", got.Data.Routes["harness.state"])
	assert.Equal(t, "POST /api/super-agent/harness/plan", got.Data.Routes["harness.plan"])
	assert.Equal(t, "POST /api/super-agent/harness/tool-outputs", got.Data.Routes["harness.tool_outputs"])
	assert.Equal(t, "POST /api/super-agent/harness/cleanup", got.Data.Routes["harness.cleanup"])
	assert.Equal(t, "POST /api/super-agent/harness/snapshot", got.Data.Routes["harness.snapshot"])
	assert.Equal(t, "POST /api/super-agent/traces/get", got.Data.Routes["traces.get"])
	assert.Equal(t, "POST /api/super-agent/approvals/list", got.Data.Routes["approvals.list"])
	assert.Equal(t, "POST /api/super-agent/approvals/resolve", got.Data.Routes["approvals.resolve"])
	assert.Equal(t, "POST /api/super-agent/workspace/move", got.Data.Routes["workspace.move"])
	assert.Equal(t, "POST /api/super-agent/workspace/write", got.Data.Routes["workspace.write"])
	assert.Equal(t, "POST /api/super-agent/workspace/mkdir", got.Data.Routes["workspace.mkdir"])
	assert.Equal(t, "POST /api/super-agent/workspace/stat", got.Data.Routes["workspace.stat"])
	assert.Equal(t, "POST /api/super-agent/workspace/grep", got.Data.Routes["workspace.grep"])
	assert.Equal(t, "POST /api/super-agent/workspace/glob", got.Data.Routes["workspace.glob"])
	assert.Equal(t, "POST /api/super-agent/workspace/edit", got.Data.Routes["workspace.edit"])
	assert.Equal(t, "POST /api/super-agent/workspace/patch", got.Data.Routes["workspace.patch"])
	assert.Equal(t, "POST /api/super-agent/sandbox/exec", got.Data.Routes["sandbox.exec"])
	assert.Equal(t, "POST /api/super-agent/artifacts/list", got.Data.Routes["artifacts.list"])
	assert.Equal(t, "POST /api/super-agent/artifacts/download", got.Data.Routes["artifacts.download"])
	assert.Equal(t, "POST /api/super-agent/artifacts/delete", got.Data.Routes["artifacts.delete"])
	assert.Equal(t, "POST /api/super-agent/artifacts/move", got.Data.Routes["artifacts.move"])
	assert.Equal(t, "POST /api/super-agent/skills/create", got.Data.Routes["skills.create"])
	assert.Equal(t, "GET /api/super-agent/skills/get", got.Data.Routes["skills.get"])
	assert.Equal(t, "POST /api/super-agent/skills/update", got.Data.Routes["skills.update"])
	assert.Equal(t, "POST /api/super-agent/skills/delete", got.Data.Routes["skills.delete"])
	assert.Equal(t, "POST /api/super-agent/skills/publish", got.Data.Routes["skills.publish"])
	assert.Equal(t, "GET /api/super-agent/skills/list", got.Data.Routes["skills.list"])
	assert.Equal(t, "GET /api/super-agent/skills/assets/list", got.Data.Routes["skills.assets.list"])
	assert.Equal(t, "GET /api/super-agent/skills/assets/get", got.Data.Routes["skills.assets.get"])
	assert.Equal(t, "POST /api/super-agent/skills/assets/upsert", got.Data.Routes["skills.assets.upsert"])
	assert.Equal(t, "POST /api/super-agent/skills/assets/delete", got.Data.Routes["skills.assets.delete"])
	assert.Equal(t, "POST /api/super-agent/skills/validate-package", got.Data.Routes["skills.validate_package"])
	assert.Equal(t, "POST /api/super-agent/skills/import", got.Data.Routes["skills.import"])
	assert.Equal(t, "POST /api/super-agent/skills/import-runtime", got.Data.Routes["skills.import_runtime"])
	assert.Equal(t, "POST /api/super-agent/skills/export", got.Data.Routes["skills.export"])
	assert.Equal(t, "GET /api/super-agent/marketplace/list", got.Data.Routes["skills.marketplace.list"])
	assert.Equal(t, "GET /api/super-agent/marketplace/get", got.Data.Routes["skills.marketplace.get"])
	assert.Equal(t, "POST /api/super-agent/marketplace/install", got.Data.Routes["skills.marketplace.install"])
	assert.Equal(t, "POST /api/super-agent/products/list", got.Data.Routes["products.list"])
	assert.Equal(t, "POST /api/super-agent/products/get", got.Data.Routes["products.get"])
	assert.Equal(t, "POST /api/super-agent/products/install", got.Data.Routes["products.install"])
	assert.Equal(t, "POST /api/super-agent/products/upgrade", got.Data.Routes["products.upgrade"])
	assert.Equal(t, "POST /api/super-agent/products/uninstall", got.Data.Routes["products.uninstall"])
	assert.Equal(t, "POST /api/super-agent/marketplace/products/list", got.Data.Routes["marketplace.products.list"])
	assert.Equal(t, "POST /api/super-agent/marketplace/products/get", got.Data.Routes["marketplace.products.get"])
	assert.Equal(t, "POST /api/super-agent/marketplace/products/install", got.Data.Routes["marketplace.products.install"])

	var skillsRootFound bool
	for _, root := range got.Data.WorkspaceRoots {
		if root.Path == "/skills" {
			skillsRootFound = true
			assert.Equal(t, "技能", root.Label)
			assert.True(t, root.Readonly)
		}
	}
	assert.True(t, skillsRootFound)
}

func TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/super-agent/openapi.json",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Security []map[string][]string `json:"security"`
		Paths    map[string]map[string]struct {
			OperationID string   `json:"operationId"`
			Tags        []string `json:"tags"`
			XTransport  string   `json:"x-transport"`
			XMutates    bool     `json:"x-mutates"`
			RequestBody *struct {
				Required bool `json:"required"`
				Content  map[string]struct {
					Schema struct {
						Ref string `json:"$ref"`
					} `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
			Parameters []struct {
				Name     string `json:"name"`
				In       string `json:"in"`
				Required bool   `json:"required"`
			} `json:"parameters"`
		} `json:"paths"`
		Components struct {
			SecuritySchemes map[string]struct {
				Type         string `json:"type"`
				Scheme       string `json:"scheme"`
				BearerFormat string `json:"bearerFormat"`
			} `json:"securitySchemes"`
			Schemas map[string]struct {
				Type           string                 `json:"type"`
				Required       []string               `json:"required"`
				Properties     map[string]interface{} `json:"properties"`
				XRequiredOneOf [][]string             `json:"x-required-one-of"`
				XOptional      []string               `json:"x-optional-fields"`
			} `json:"schemas"`
		} `json:"components"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)

	assert.Equal(t, "3.1.0", got.OpenAPI)
	assert.Equal(t, "Super Agent App Server API", got.Info.Title)
	assert.Equal(t, "super-agent.app-server.v1", got.Info.Version)
	require.NotEmpty(t, got.Security)
	assert.Contains(t, got.Security[0], "BearerAuth")
	assert.Equal(t, "http", got.Components.SecuritySchemes["BearerAuth"].Type)
	assert.Equal(t, "bearer", got.Components.SecuritySchemes["BearerAuth"].Scheme)
	assert.Equal(t, "Bearer", got.Components.SecuritySchemes["BearerAuth"].BearerFormat)

	createOperation := got.Paths["/api/super-agent/runs/create"]["post"]
	assert.Equal(t, "runs.create", createOperation.OperationID)
	assert.ElementsMatch(t, []string{"runs"}, createOperation.Tags)
	assert.Equal(t, "json", createOperation.XTransport)
	assert.True(t, createOperation.XMutates)
	require.NotNil(t, createOperation.RequestBody)
	assert.True(t, createOperation.RequestBody.Required)
	assert.Equal(t, "#/components/schemas/RunsCreateRequest", createOperation.RequestBody.Content["application/json"].Schema.Ref)

	streamOperation := got.Paths["/api/super-agent/runs/stream"]["post"]
	assert.Equal(t, "runs.stream", streamOperation.OperationID)
	assert.Equal(t, "sse", streamOperation.XTransport)

	sessionCreateOperation := got.Paths["/api/super-agent/sessions/create"]["post"]
	assert.Equal(t, "sessions.create", sessionCreateOperation.OperationID)
	assert.Equal(t, "#/components/schemas/SessionsCreateRequest", sessionCreateOperation.RequestBody.Content["application/json"].Schema.Ref)

	sessionGetOperation := got.Paths["/api/super-agent/sessions/get"]["post"]
	assert.Equal(t, "sessions.get", sessionGetOperation.OperationID)
	assert.Equal(t, "#/components/schemas/SessionsGetRequest", sessionGetOperation.RequestBody.Content["application/json"].Schema.Ref)

	sessionDeleteOperation := got.Paths["/api/super-agent/sessions/delete"]["post"]
	assert.Equal(t, "sessions.delete", sessionDeleteOperation.OperationID)
	assert.Equal(t, "#/components/schemas/SessionsDeleteRequest", sessionDeleteOperation.RequestBody.Content["application/json"].Schema.Ref)

	workspacePatchOperation := got.Paths["/api/super-agent/workspace/patch"]["post"]
	assert.Equal(t, "workspace.patch", workspacePatchOperation.OperationID)
	assert.Equal(t, "#/components/schemas/WorkspacePatchRequest", workspacePatchOperation.RequestBody.Content["application/json"].Schema.Ref)

	skillImportOperation := got.Paths["/api/super-agent/skills/import"]["post"]
	assert.Equal(t, "skills.import", skillImportOperation.OperationID)
	assert.Equal(t, "#/components/schemas/SkillsImportRequest", skillImportOperation.RequestBody.Content["application/json"].Schema.Ref)

	skillListOperation := got.Paths["/api/super-agent/skills/list"]["get"]
	assert.Equal(t, "skills.list", skillListOperation.OperationID)
	require.NotEmpty(t, skillListOperation.Parameters)
	assert.Equal(t, "space_id", skillListOperation.Parameters[0].Name)
	assert.Equal(t, "query", skillListOperation.Parameters[0].In)
	assert.True(t, skillListOperation.Parameters[0].Required)

	runsCreateSchema := got.Components.Schemas["RunsCreateRequest"]
	assert.Equal(t, "object", runsCreateSchema.Type)
	assert.Contains(t, runsCreateSchema.Properties, "agent_id")
	assert.Contains(t, runsCreateSchema.Properties, "bot_id")
	assert.Contains(t, runsCreateSchema.Properties, "additional_messages")
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, runsCreateSchema.XRequiredOneOf)
	assert.Contains(t, got.Components.Schemas, "SessionsCreateRequest")
	assert.Contains(t, got.Components.Schemas, "SessionsGetRequest")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "include_runs")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "run_limit")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "include_workspace")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "workspace_path")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "workspace_recursive")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "include_context")
	assert.Contains(t, got.Components.Schemas["SessionsGetRequest"].XOptional, "include_tool_output_content")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "include_runs")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "run_limit")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "include_workspace")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "workspace_path")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "workspace_recursive")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "include_context")
	assert.Contains(t, got.Components.Schemas["HarnessSnapshotRequest"].XOptional, "include_tool_output_content")
	assert.Contains(t, got.Components.Schemas, "SessionsDeleteRequest")
	assert.Contains(t, got.Components.Schemas, "WorkspacePatchRequest")
	assert.Contains(t, got.Components.Schemas, "SkillsImportRequest")
}

func TestSuperAgentApprovalListRouteReturnsEmptyPendingQueue(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"conversation_id":"123","agent_id":"456"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/approvals/list",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ConversationID string `json:"conversation_id"`
			Approvals      []struct {
				ApprovalID string `json:"approval_id"`
			} `json:"approvals"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "success", got.Msg)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Empty(t, got.Data.Approvals)
}

func TestSuperAgentApprovalListRouteIncludesRequiredActionDetails(t *testing.T) {
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{
		record: &agentrunEntity.RunRecordMeta{
			ID:             765,
			ConversationID: 123,
			AgentID:        456,
			Status:         agentrunEntity.RunStatusRequiredAction,
			CreatedAt:      1000,
			UpdatedAt:      2000,
		},
	}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:             999,
				RunID:          765,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeAnswer,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "approval needed",
				Ext: map[string]string{
					"tool_calls_ids": "tool-call-1",
				},
				RequiredAction: &messageModel.RequiredAction{
					Type: "submit_tool_outputs",
					SubmitToolOutputs: &messageModel.SubmitToolOutputs{
						ToolCalls: []*messageModel.InterruptPlugin{
							{ID: "tool-call-1", Type: "function"},
						},
					},
				},
				CreatedAt: 1500,
				UpdatedAt: 1600,
			},
		},
	}
	t.Cleanup(func() {
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
	})

	h := server.Default()
	Register(h)

	body := `{"conversation_id":"123","agent_id":"456"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/approvals/list",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Approvals []struct {
				ApprovalID     string `json:"approval_id"`
				RunID          string `json:"run_id"`
				MessageID      string `json:"message_id"`
				Status         string `json:"status"`
				RequiredAction struct {
					Type              string `json:"type"`
					SubmitToolOutputs struct {
						ToolCalls []struct {
							ID   string `json:"id"`
							Type string `json:"type"`
						} `json:"tool_calls"`
					} `json:"submit_tool_outputs"`
				} `json:"required_action"`
				ToolCallIDs []string `json:"tool_call_ids"`
			} `json:"approvals"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Len(t, got.Data.Approvals, 1)
	assert.Equal(t, "run:765", got.Data.Approvals[0].ApprovalID)
	assert.Equal(t, "765", got.Data.Approvals[0].RunID)
	assert.Equal(t, "999", got.Data.Approvals[0].MessageID)
	assert.Equal(t, "pending", got.Data.Approvals[0].Status)
	assert.Equal(t, "submit_tool_outputs", got.Data.Approvals[0].RequiredAction.Type)
	assert.Len(t, got.Data.Approvals[0].RequiredAction.SubmitToolOutputs.ToolCalls, 1)
	assert.Equal(t, "tool-call-1", got.Data.Approvals[0].RequiredAction.SubmitToolOutputs.ToolCalls[0].ID)
	assert.Equal(t, "function", got.Data.Approvals[0].RequiredAction.SubmitToolOutputs.ToolCalls[0].Type)
	assert.ElementsMatch(t, []string{"tool-call-1"}, got.Data.Approvals[0].ToolCallIDs)
}

func TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   456,
			CreatorID: 42,
		},
	}
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{
		record: &agentrunEntity.RunRecordMeta{
			ID:             765,
			ConversationID: 123,
			AgentID:        456,
			Status:         agentrunEntity.RunStatusRequiredAction,
			CreatedAt:      1000,
			UpdatedAt:      2000,
		},
	}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:             999,
				RunID:          765,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeAnswer,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "approval needed",
				Ext: map[string]string{
					"tool_calls_ids": "tool-call-1",
				},
				RequiredAction: &messageModel.RequiredAction{
					Type: "submit_tool_outputs",
					SubmitToolOutputs: &messageModel.SubmitToolOutputs{
						ToolCalls: []*messageModel.InterruptPlugin{
							{ID: "tool-call-1", Type: "function"},
						},
					},
				},
				CreatedAt: 1500,
				UpdatedAt: 1600,
			},
		},
	}
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	commands := []string{}
	crosssandbox.SetDefaultSVC(fakeSuperAgentApprovalDecisionListSandboxManager{commands: &commands})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"conversation_id":"123","agent_id":"456","trace_limit":20,"include_workspace":true,"workspace_path":"/workspace","workspace_recursive":true,"include_harness":false,"include_tool_outputs":false,"include_artifacts":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Components     []string `json:"components"`
			ConversationID string   `json:"conversation_id"`
			Messages       *struct {
				Messages []struct {
					MessageID       string `json:"message_id"`
					Role            string `json:"role"`
					Type            string `json:"type"`
					Content         string `json:"content"`
					HasModelContent bool   `json:"model_content,omitempty"`
				} `json:"messages"`
				OrderBy    string `json:"order_by"`
				NextCursor string `json:"next_cursor"`
			} `json:"messages"`
			Runs *struct {
				Runs []struct {
					RunID  string `json:"run_id"`
					Status string `json:"status"`
					Active bool   `json:"active"`
				} `json:"runs"`
			} `json:"runs"`
			Workspace *struct {
				Path  string `json:"path"`
				Files []struct {
					Name  string `json:"name"`
					Path  string `json:"path"`
					IsDir bool   `json:"is_dir"`
					Size  int64  `json:"size"`
				} `json:"files"`
			} `json:"workspace"`
			Trace   *superagenttrace.Data `json:"trace"`
			Context *struct {
				SummaryExists bool `json:"summary_exists"`
				Summary       *struct {
					SummaryPath string `json:"summary_path"`
				} `json:"summary"`
			} `json:"context"`
			Approvals []struct {
				ApprovalID  string   `json:"approval_id"`
				RunID       string   `json:"run_id"`
				MessageID   string   `json:"message_id"`
				ToolCallIDs []string `json:"tool_call_ids"`
			} `json:"approvals"`
			ApprovalDecisions []struct {
				ApprovalID   string `json:"approval_id"`
				RunID        string `json:"run_id"`
				Decision     string `json:"decision"`
				Status       string `json:"status"`
				Source       string `json:"source"`
				DecisionPath string `json:"decision_path"`
			} `json:"approval_decisions"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Contains(t, got.Data.Components, "messages")
	assert.Contains(t, got.Data.Components, "runs")
	assert.Contains(t, got.Data.Components, "workspace")
	assert.Contains(t, got.Data.Components, "trace")
	assert.Contains(t, got.Data.Components, "approvals")
	require.NotNil(t, got.Data.Messages)
	require.Len(t, got.Data.Messages.Messages, 1)
	assert.Equal(t, "999", got.Data.Messages.Messages[0].MessageID)
	assert.Equal(t, "assistant", got.Data.Messages.Messages[0].Role)
	assert.Equal(t, "answer", got.Data.Messages.Messages[0].Type)
	assert.Equal(t, "approval needed", got.Data.Messages.Messages[0].Content)
	assert.Equal(t, "ASC", got.Data.Messages.OrderBy)
	assert.Equal(t, "999", got.Data.Messages.NextCursor)
	require.NotNil(t, got.Data.Runs)
	require.Len(t, got.Data.Runs.Runs, 1)
	assert.Equal(t, "765", got.Data.Runs.Runs[0].RunID)
	assert.Equal(t, string(agentrunEntity.RunStatusRequiredAction), got.Data.Runs.Runs[0].Status)
	assert.False(t, got.Data.Runs.Runs[0].Active)
	require.NotNil(t, got.Data.Workspace)
	assert.Equal(t, "/workspace", got.Data.Workspace.Path)
	require.Len(t, got.Data.Workspace.Files, 2)
	assert.Equal(t, "/workspace/src", got.Data.Workspace.Files[0].Path)
	assert.True(t, got.Data.Workspace.Files[0].IsDir)
	assert.Equal(t, "/workspace/src/main.go", got.Data.Workspace.Files[1].Path)
	assert.False(t, got.Data.Workspace.Files[1].IsDir)
	require.NotNil(t, got.Data.Trace)
	require.Len(t, got.Data.Trace.Runs, 1)
	assert.Equal(t, "765", got.Data.Trace.Runs[0].RunID)
	contextEvent := findSuperAgentTraceEvent(got.Data.Trace.Events, "context.compacted")
	require.NotNil(t, contextEvent)
	assert.Equal(t, 1, countSuperAgentTraceEvents(got.Data.Trace.Events, "context.compacted"))
	assert.Equal(t, "context", contextEvent.Kind)
	assert.Equal(t, "compacted", contextEvent.Status)
	assert.Equal(t, "766", contextEvent.RunID)
	assert.Equal(t, "902", contextEvent.MessageID)
	assert.Equal(t, "history_bytes_exceeded", contextEvent.Metadata["trigger"])
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", contextEvent.Metadata["summary_path"])
	assert.Equal(t, "21", contextEvent.Metadata["original_messages"])
	assert.Equal(t, "13", contextEvent.Metadata["compacted_messages"])
	assert.Equal(t, "8", contextEvent.Metadata["retained_messages"])
	assert.Equal(t, "131072", contextEvent.Metadata["original_bytes"])
	assert.Equal(t, "163840", contextEvent.Metadata["max_bytes"])
	require.NotNil(t, got.Data.Context)
	assert.True(t, got.Data.Context.SummaryExists)
	require.NotNil(t, got.Data.Context.Summary)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.Context.Summary.SummaryPath)
	require.Len(t, got.Data.Approvals, 1)
	assert.Equal(t, "run:765", got.Data.Approvals[0].ApprovalID)
	assert.Equal(t, "765", got.Data.Approvals[0].RunID)
	assert.Equal(t, "999", got.Data.Approvals[0].MessageID)
	assert.ElementsMatch(t, []string{"tool-call-1"}, got.Data.Approvals[0].ToolCallIDs)
	require.Lenf(t, got.Data.ApprovalDecisions, 1, "commands=%v body=%s", commands, string(w.Result().Body()))
	assert.Equal(t, "run:765", got.Data.ApprovalDecisions[0].ApprovalID)
	assert.Equal(t, "765", got.Data.ApprovalDecisions[0].RunID)
	assert.Equal(t, "approve", got.Data.ApprovalDecisions[0].Decision)
	assert.Equal(t, "approved", got.Data.ApprovalDecisions[0].Status)
	assert.Equal(t, "app_server", got.Data.ApprovalDecisions[0].Source)
	assert.Equal(t, "/workspace/.agent/approvals/run-765.json", got.Data.ApprovalDecisions[0].DecisionPath)
}

func TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentToolOutputSandboxManager{})
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","include_messages":false,"include_runs":false,"include_workspace":false,"include_harness":false,"include_tool_outputs":true,"include_tool_output_content":true,"include_artifacts":false,"include_trace":false,"include_approvals":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Components  []string `json:"components"`
			ToolOutputs *struct {
				Root  string `json:"root"`
				Path  string `json:"path"`
				Files []struct {
					Name  string `json:"name"`
					Path  string `json:"path"`
					IsDir bool   `json:"is_dir"`
				} `json:"files"`
				Entries []struct {
					Path             string `json:"path"`
					Name             string `json:"name"`
					Tool             string `json:"tool"`
					Status           string `json:"status"`
					ArgumentsPreview string `json:"arguments_preview"`
					ResultPreview    string `json:"result_preview"`
					Summary          string `json:"summary"`
				} `json:"entries"`
				Contents map[string]struct {
					Path      string `json:"path"`
					Content   string `json:"content"`
					IsBinary  bool   `json:"is_binary"`
					TotalSize int64  `json:"total_size"`
				} `json:"contents"`
			} `json:"tool_outputs"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Contains(t, got.Data.Components, "tool_outputs")
	require.NotNil(t, got.Data.ToolOutputs)
	assert.Equal(t, "/workspace/.agent/tooloutputs", got.Data.ToolOutputs.Root)
	require.Len(t, got.Data.ToolOutputs.Files, 1)
	assert.Equal(t, "/workspace/.agent/tooloutputs/tool-call-search.json", got.Data.ToolOutputs.Files[0].Path)
	require.Len(t, got.Data.ToolOutputs.Entries, 1)
	assert.Equal(t, "/workspace/.agent/tooloutputs/tool-call-search.json", got.Data.ToolOutputs.Entries[0].Path)
	assert.Equal(t, "tool-call-search.json", got.Data.ToolOutputs.Entries[0].Name)
	assert.Equal(t, "Search", got.Data.ToolOutputs.Entries[0].Tool)
	assert.Equal(t, "completed", got.Data.ToolOutputs.Entries[0].Status)
	assert.JSONEq(t, `{"query":"GLM-5.2"}`, got.Data.ToolOutputs.Entries[0].ArgumentsPreview)
	assert.JSONEq(t, `{"count":8}`, got.Data.ToolOutputs.Entries[0].ResultPreview)
	assert.Contains(t, got.Data.ToolOutputs.Entries[0].Summary, "Search")
	assert.Contains(t, got.Data.ToolOutputs.Entries[0].Summary, "query")
	assert.Contains(t, got.Data.ToolOutputs.Entries[0].Summary, "count")
	require.Contains(t, got.Data.ToolOutputs.Contents, "/workspace/.agent/tooloutputs/tool-call-search.json")
	content := got.Data.ToolOutputs.Contents["/workspace/.agent/tooloutputs/tool-call-search.json"]
	assert.Equal(t, "/workspace/.agent/tooloutputs/tool-call-search.json", content.Path)
	assert.False(t, content.IsBinary)
	assert.Contains(t, content.Content, `"tool":"Search"`)
	assert.Contains(t, content.Content, `"query":"GLM-5.2"`)
	assert.Contains(t, content.Content, `"count":8`)
	assert.Equal(t, int64(len(content.Content)), content.TotalSize)
}

func TestSuperAgentHarnessSnapshotRouteIncludesContextState(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentContextSandboxManager{})
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","include_context":true,"include_messages":false,"include_runs":false,"include_workspace":false,"include_harness":false,"include_tool_outputs":false,"include_artifacts":false,"include_trace":false,"include_approvals":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Components []string `json:"components"`
			Context    *struct {
				Strategy             string   `json:"strategy"`
				SummaryPath          string   `json:"summary_path"`
				SummaryExists        bool     `json:"summary_exists"`
				ToolOutputRoot       string   `json:"tool_output_root"`
				ToolOutputOffload    bool     `json:"tool_output_offload"`
				RecentMessagesPolicy string   `json:"recent_messages_policy"`
				Components           []string `json:"components"`
				Summary              *struct {
					Version           string   `json:"version"`
					Summary           string   `json:"summary"`
					UpdatedAt         int64    `json:"updated_at"`
					MessageID         string   `json:"message_id"`
					RunID             string   `json:"run_id"`
					Trigger           string   `json:"trigger"`
					OriginalMessages  int      `json:"original_messages"`
					CompactedMessages int      `json:"compacted_messages"`
					RetainedMessages  int      `json:"retained_messages"`
					OriginalBytes     int      `json:"original_bytes"`
					MaxBytes          int      `json:"max_bytes"`
					SummaryPath       string   `json:"summary_path"`
					KeyFiles          []string `json:"key_files"`
					Artifacts         []string `json:"artifacts"`
					NextActions       []string `json:"next_actions"`
				} `json:"summary"`
			} `json:"context"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Contains(t, got.Data.Components, "context")
	require.NotNil(t, got.Data.Context)
	assert.Equal(t, "tool-output-offload+session-summary", got.Data.Context.Strategy)
	assert.Equal(t, "/workspace/.agent/context-summary.json", got.Data.Context.SummaryPath)
	assert.True(t, got.Data.Context.SummaryExists)
	assert.Equal(t, "/workspace/.agent/tooloutputs", got.Data.Context.ToolOutputRoot)
	assert.True(t, got.Data.Context.ToolOutputOffload)
	assert.Equal(t, "message_limit+recent_tail", got.Data.Context.RecentMessagesPolicy)
	assert.ElementsMatch(t, []string{"summary", "tool_outputs", "recent_messages"}, got.Data.Context.Components)
	require.NotNil(t, got.Data.Context.Summary)
	assert.Equal(t, "v1", got.Data.Context.Summary.Version)
	assert.Contains(t, got.Data.Context.Summary.Summary, "Codex-like super agent")
	assert.Equal(t, int64(1781900300), got.Data.Context.Summary.UpdatedAt)
	assert.Equal(t, "901", got.Data.Context.Summary.MessageID)
	assert.Equal(t, "765", got.Data.Context.Summary.RunID)
	assert.Equal(t, "history_bytes_exceeded", got.Data.Context.Summary.Trigger)
	assert.Equal(t, 42, got.Data.Context.Summary.OriginalMessages)
	assert.Equal(t, 26, got.Data.Context.Summary.CompactedMessages)
	assert.Equal(t, 16, got.Data.Context.Summary.RetainedMessages)
	assert.Equal(t, 262144, got.Data.Context.Summary.OriginalBytes)
	assert.Equal(t, 163840, got.Data.Context.Summary.MaxBytes)
	assert.Equal(t, "/workspace/.agent/context-summary.json", got.Data.Context.Summary.SummaryPath)
	assert.ElementsMatch(t, []string{"/workspace/main.go"}, got.Data.Context.Summary.KeyFiles)
	assert.ElementsMatch(t, []string{"/outputs/report.html"}, got.Data.Context.Summary.Artifacts)
	assert.ElementsMatch(t, []string{"continue context compaction"}, got.Data.Context.Summary.NextActions)
}

func TestSuperAgentHarnessSnapshotRouteUsesConversationScopedContextState(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentSessionContextSandboxManager{})
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","conversation_id":"123","include_context":true,"include_messages":false,"include_runs":false,"include_workspace":false,"include_harness":false,"include_tool_outputs":false,"include_artifacts":false,"include_trace":false,"include_approvals":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Context *struct {
				SummaryPath string `json:"summary_path"`
				Summary     *struct {
					Summary     string `json:"summary"`
					SummaryPath string `json:"summary_path"`
				} `json:"summary"`
			} `json:"context"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	require.NotNil(t, got.Data.Context)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.Context.SummaryPath)
	require.NotNil(t, got.Data.Context.Summary)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.Context.Summary.SummaryPath)
	assert.Contains(t, got.Data.Context.Summary.Summary, "Session 123")
}

func TestSuperAgentHarnessSnapshotRouteIncludesResumeHandoff(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   456,
			CreatorID: 42,
		},
	}
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:             1001,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.User,
				MessageType:    crossMessage.MessageTypeQuestion,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "continue the harness",
				CreatedAt:      1781900510,
				UpdatedAt:      1781900510,
			},
			{
				ID:             1002,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeAnswer,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "working on it",
				CreatedAt:      1781900520,
				UpdatedAt:      1781900520,
			},
		},
	}
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentSessionContextSandboxManager{})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","conversation_id":"123","include_messages":true,"message_limit":3,"include_runs":false,"include_workspace":false,"include_harness":true,"include_context":true,"include_tool_outputs":true,"include_artifacts":false,"include_trace":false,"include_approvals":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Components []string `json:"components"`
			Resume     *struct {
				Version            string   `json:"version"`
				ConversationID     string   `json:"conversation_id"`
				AgentID            string   `json:"agent_id"`
				Summary            string   `json:"summary"`
				SummaryPath        string   `json:"summary_path"`
				SummaryExists      bool     `json:"summary_exists"`
				PlanPath           string   `json:"plan_path"`
				PlanExists         bool     `json:"plan_exists"`
				PlanStatus         string   `json:"plan_status"`
				PlanItemCount      int      `json:"plan_item_count"`
				RecentMessageCount int      `json:"recent_message_count"`
				MessageIDs         []string `json:"message_ids"`
				ToolOutputRoot     string   `json:"tool_output_root"`
				ToolOutputCount    int      `json:"tool_output_count"`
				ToolOutputPaths    []string `json:"tool_output_paths"`
				ArtifactPaths      []string `json:"artifact_paths"`
				Prompt             string   `json:"prompt"`
			} `json:"resume"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Contains(t, got.Data.Components, "resume")
	require.NotNil(t, got.Data.Resume)
	assert.Equal(t, "v1", got.Data.Resume.Version)
	assert.Equal(t, "123", got.Data.Resume.ConversationID)
	assert.Equal(t, "456", got.Data.Resume.AgentID)
	assert.True(t, got.Data.Resume.SummaryExists)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.Resume.SummaryPath)
	assert.Contains(t, got.Data.Resume.Summary, "Session 123 compacted context")
	assert.Equal(t, "/workspace/.agent/sessions/123/plan.json", got.Data.Resume.PlanPath)
	assert.True(t, got.Data.Resume.PlanExists)
	assert.Equal(t, "in_progress", got.Data.Resume.PlanStatus)
	assert.Equal(t, 1, got.Data.Resume.PlanItemCount)
	assert.Equal(t, 2, got.Data.Resume.RecentMessageCount)
	assert.ElementsMatch(t, []string{"1001", "1002"}, got.Data.Resume.MessageIDs)
	assert.Equal(t, "/workspace/.agent/tooloutputs", got.Data.Resume.ToolOutputRoot)
	assert.Equal(t, 1, got.Data.Resume.ToolOutputCount)
	assert.Contains(t, got.Data.Resume.ToolOutputPaths, "/workspace/.agent/tooloutputs/tool-call-search.json")
	assert.Contains(t, got.Data.Resume.ArtifactPaths, "/outputs/session.html")
	assert.Contains(t, got.Data.Resume.Prompt, "Session 123 compacted context")
	assert.Contains(t, got.Data.Resume.Prompt, "/workspace/.agent/sessions/123/plan.json")
	assert.Contains(t, got.Data.Resume.Prompt, "/workspace/.agent/tooloutputs/tool-call-search.json")
	assert.Contains(t, got.Data.Resume.Prompt, "/outputs/session.html")
}

func TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   456,
			CreatorID: 42,
		},
	}
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:             1001,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.User,
				MessageType:    crossMessage.MessageTypeQuestion,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "continue the harness",
				CreatedAt:      1781900510,
				UpdatedAt:      1781900510,
			},
			{
				ID:             1002,
				ConversationID: 123,
				AgentID:        456,
				Role:           schema.Assistant,
				MessageType:    crossMessage.MessageTypeAnswer,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "working on it",
				CreatedAt:      1781900520,
				UpdatedAt:      1781900520,
			},
		},
	}
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentSessionContextSandboxManager{})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","conversation_id":"123","message_limit":3,"include_artifacts":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/resume",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	bodyBytes := w.Result().Body()
	var got struct {
		Code int `json:"code"`
		Data struct {
			Version            string   `json:"version"`
			ConversationID     string   `json:"conversation_id"`
			AgentID            string   `json:"agent_id"`
			Summary            string   `json:"summary"`
			SummaryPath        string   `json:"summary_path"`
			SummaryExists      bool     `json:"summary_exists"`
			PlanPath           string   `json:"plan_path"`
			PlanExists         bool     `json:"plan_exists"`
			PlanStatus         string   `json:"plan_status"`
			PlanItemCount      int      `json:"plan_item_count"`
			RecentMessageCount int      `json:"recent_message_count"`
			MessageIDs         []string `json:"message_ids"`
			ToolOutputRoot     string   `json:"tool_output_root"`
			ToolOutputCount    int      `json:"tool_output_count"`
			ToolOutputPaths    []string `json:"tool_output_paths"`
			ArtifactPaths      []string `json:"artifact_paths"`
			Components         []string `json:"components"`
			Prompt             string   `json:"prompt"`
		} `json:"data"`
	}
	err := json.Unmarshal(bodyBytes, &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "v1", got.Data.Version)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Equal(t, "456", got.Data.AgentID)
	assert.True(t, got.Data.SummaryExists)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.SummaryPath)
	assert.Contains(t, got.Data.Summary, "Session 123 compacted context")
	assert.Equal(t, "/workspace/.agent/sessions/123/plan.json", got.Data.PlanPath)
	assert.True(t, got.Data.PlanExists)
	assert.Equal(t, "in_progress", got.Data.PlanStatus)
	assert.Equal(t, 1, got.Data.PlanItemCount)
	assert.Equal(t, 2, got.Data.RecentMessageCount)
	assert.ElementsMatch(t, []string{"1001", "1002"}, got.Data.MessageIDs)
	assert.Equal(t, "/workspace/.agent/tooloutputs", got.Data.ToolOutputRoot)
	assert.Equal(t, 1, got.Data.ToolOutputCount)
	assert.Contains(t, got.Data.ToolOutputPaths, "/workspace/.agent/tooloutputs/tool-call-search.json")
	assert.Contains(t, got.Data.ArtifactPaths, "/outputs/session.html")
	assert.Contains(t, got.Data.Components, "messages")
	assert.Contains(t, got.Data.Components, "harness")
	assert.Contains(t, got.Data.Components, "context")
	assert.Contains(t, got.Data.Components, "tool_outputs")
	assert.NotContains(t, string(bodyBytes), `"messages":`)
	assert.NotContains(t, string(bodyBytes), `"harness":`)
	assert.NotContains(t, string(bodyBytes), `"context":`)
	assert.NotContains(t, string(bodyBytes), `"tool_outputs":`)
	assert.Contains(t, got.Data.Prompt, "Session 123 compacted context")
	assert.Contains(t, got.Data.Prompt, "/workspace/.agent/sessions/123/plan.json")
	assert.Contains(t, got.Data.Prompt, "/workspace/.agent/tooloutputs/tool-call-search.json")
}

func TestSuperAgentHarnessSnapshotRouteDoesNotFallbackGlobalContextWhenConversationScopedMissing(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentContextSandboxManager{})
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","conversation_id":"123","include_context":true,"include_messages":false,"include_runs":false,"include_workspace":false,"include_harness":false,"include_tool_outputs":false,"include_artifacts":false,"include_trace":false,"include_approvals":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Context *struct {
				SummaryPath   string `json:"summary_path"`
				SummaryExists bool   `json:"summary_exists"`
				Summary       *struct {
					SummaryPath string `json:"summary_path"`
				} `json:"summary"`
			} `json:"context"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	require.NotNil(t, got.Data.Context)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.Context.SummaryPath)
	assert.False(t, got.Data.Context.SummaryExists)
	assert.Nil(t, got.Data.Context.Summary)
}

func TestSuperAgentHarnessContextClearRouteDeletesConversationScopedSummaryOnly(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	commands := []string{}
	crosssandbox.SetDefaultSVC(fakeSuperAgentContextClearSandboxManager{commands: &commands})
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"agent_id":"456","conversation_id":"123"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/context/clear",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			ConversationID      string `json:"conversation_id"`
			SummaryPath         string `json:"summary_path"`
			SummaryExistsBefore bool   `json:"summary_exists_before"`
			Cleared             bool   `json:"cleared"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Equal(t, "/workspace/.agent/sessions/123/context-summary.json", got.Data.SummaryPath)
	assert.True(t, got.Data.SummaryExistsBefore)
	assert.True(t, got.Data.Cleared)
	assert.Contains(t, commands, "rm -f '/workspace/.agent/sessions/123/context-summary.json'")
	joined := strings.Join(commands, "\n")
	assert.NotContains(t, joined, "rm -f '/workspace/.agent/context-summary.json'")
}

func TestSuperAgentManifestRouteIncludesHarnessContextClearOperation(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/super-agent/manifest", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Code int `json:"code"`
		Data struct {
			Harness struct {
				ContextClearRoute string `json:"context_clear_route"`
				ContextPolicy     struct {
					Strategy                   string `json:"strategy"`
					SummaryVersion             string `json:"summary_version"`
					Trigger                    string `json:"trigger"`
					MaxBytes                   int    `json:"max_bytes"`
					RecentMessages             int    `json:"recent_messages"`
					SummaryMaxRunes            int    `json:"summary_max_runes"`
					SummaryPath                string `json:"summary_path"`
					SessionSummaryPathTemplate string `json:"session_summary_path_template"`
					ClearRoute                 string `json:"clear_route"`
				} `json:"context_policy"`
			} `json:"harness"`
			ExternalAPI struct {
				EntryRoutes    map[string]string `json:"entry_routes"`
				RequestSchemas map[string]struct {
					Required      []string   `json:"required"`
					RequiredOneOf [][]string `json:"required_one_of"`
					Optional      []string   `json:"optional"`
				} `json:"request_schemas"`
			} `json:"external_api"`
			OpenAPI struct {
				Operations []struct {
					OperationID   string `json:"operation_id"`
					Path          string `json:"path"`
					Method        string `json:"method"`
					Category      string `json:"category"`
					Mutates       bool   `json:"mutates"`
					RequestSchema string `json:"request_schema"`
				} `json:"operations"`
			} `json:"openapi"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "POST /api/super-agent/harness/context/clear", got.Data.Harness.ContextClearRoute)
	assert.Equal(t, "tool-output-offload+session-summary", got.Data.Harness.ContextPolicy.Strategy)
	assert.Equal(t, "v1", got.Data.Harness.ContextPolicy.SummaryVersion)
	assert.Equal(t, "history_bytes_exceeded", got.Data.Harness.ContextPolicy.Trigger)
	assert.Equal(t, 160*1024, got.Data.Harness.ContextPolicy.MaxBytes)
	assert.Equal(t, 16, got.Data.Harness.ContextPolicy.RecentMessages)
	assert.Equal(t, 1600, got.Data.Harness.ContextPolicy.SummaryMaxRunes)
	assert.Equal(t, "/workspace/.agent/context-summary.json", got.Data.Harness.ContextPolicy.SummaryPath)
	assert.Equal(t, "/workspace/.agent/sessions/{conversation_id}/context-summary.json", got.Data.Harness.ContextPolicy.SessionSummaryPathTemplate)
	assert.Equal(t, "POST /api/super-agent/harness/context/clear", got.Data.Harness.ContextPolicy.ClearRoute)
	assert.Equal(t, "POST /api/super-agent/harness/context/clear", got.Data.ExternalAPI.EntryRoutes["harness.context_clear"])
	schema, ok := got.Data.ExternalAPI.RequestSchemas["harness.context_clear"]
	require.True(t, ok)
	assert.ElementsMatch(t, [][]string{{"agent_id", "bot_id"}}, schema.RequiredOneOf)
	assert.Contains(t, schema.Optional, "conversation_id")
	found := false
	for _, operation := range got.Data.OpenAPI.Operations {
		if operation.OperationID == "harness.context_clear" {
			found = true
			assert.Equal(t, "/api/super-agent/harness/context/clear", operation.Path)
			assert.Equal(t, http.MethodPost, operation.Method)
			assert.Equal(t, "harness", operation.Category)
			assert.True(t, operation.Mutates)
			assert.Equal(t, "HarnessContextClearRequest", operation.RequestSchema)
		}
	}
	assert.True(t, found)
}

func TestSuperAgentApprovalResolveRouteRejectsMissingApprovalID(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"decision":"approve"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/approvals/resolve",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "approval_id")
}

func TestSuperAgentApprovalResolveRoutePersistsHarnessDecision(t *testing.T) {
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   456,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	fakeSandbox := &fakeSuperAgentApprovalDecisionSandboxManager{}
	crosssandbox.SetDefaultSVC(fakeSandbox)
	t.Cleanup(func() {
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"approval_id":"run:765","decision":"approve","note":"looks good","conversation_id":"123","agent_id":"456","bot_id":"456"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/approvals/resolve",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			ApprovalID   string `json:"approval_id"`
			RunID        string `json:"run_id"`
			Decision     string `json:"decision"`
			Status       string `json:"status"`
			DecisionPath string `json:"decision_path"`
			Persisted    bool   `json:"persisted"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "run:765", got.Data.ApprovalID)
	assert.Equal(t, "765", got.Data.RunID)
	assert.Equal(t, "approve", got.Data.Decision)
	assert.Equal(t, "approved", got.Data.Status)
	assert.True(t, got.Data.Persisted)
	assert.Equal(t, "/workspace/.agent/approvals/run-765.json", got.Data.DecisionPath)

	rawDecision, ok := fakeSandbox.writes["/workspace/.agent/approvals/run-765.json"]
	require.True(t, ok)
	var decision struct {
		Version        string `json:"version"`
		ApprovalID     string `json:"approval_id"`
		RunID          string `json:"run_id"`
		ConversationID string `json:"conversation_id"`
		AgentID        string `json:"agent_id"`
		BotID          string `json:"bot_id"`
		Decision       string `json:"decision"`
		Status         string `json:"status"`
		Note           string `json:"note"`
		Source         string `json:"source"`
	}
	err = json.Unmarshal(rawDecision, &decision)
	require.NoError(t, err)
	assert.Equal(t, "v1", decision.Version)
	assert.Equal(t, "run:765", decision.ApprovalID)
	assert.Equal(t, "765", decision.RunID)
	assert.Equal(t, "123", decision.ConversationID)
	assert.Equal(t, "456", decision.AgentID)
	assert.Equal(t, "456", decision.BotID)
	assert.Equal(t, "approve", decision.Decision)
	assert.Equal(t, "approved", decision.Status)
	assert.Equal(t, "looks good", decision.Note)
	assert.Equal(t, "app_server", decision.Source)
}

func TestSuperAgentSessionCreateRouteCreatesRenamableSession(t *testing.T) {
	fakeConversationSVC := &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:          321,
			SectionID:   654,
			AgentID:     789,
			ConnectorID: 10000010,
			CreatorID:   42,
			Scene:       common.Scene_Playground,
			Ext:         `{"super_agent_session_title":"竞品 Codex 调研"}`,
			CreatedAt:   1000,
			UpdatedAt:   2000,
		},
	}
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	conversation.ConversationSVC.ConversationDomainSVC = fakeConversationSVC
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
	})

	h := server.Default()
	Register(h)

	body := `{"agent_id":"789","title":"竞品 Codex 调研","user_id":"42"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/create",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Session struct {
				SessionID      string       `json:"session_id"`
				ConversationID string       `json:"conversation_id"`
				SectionID      string       `json:"section_id"`
				AgentID        string       `json:"agent_id"`
				Scene          common.Scene `json:"scene"`
				Title          string       `json:"title"`
				Renamable      bool         `json:"renamable"`
			} `json:"session"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "321", got.Data.Session.SessionID)
	assert.Equal(t, "321", got.Data.Session.ConversationID)
	assert.Equal(t, "654", got.Data.Session.SectionID)
	assert.Equal(t, "789", got.Data.Session.AgentID)
	assert.Equal(t, common.Scene_Playground, got.Data.Session.Scene)
	assert.Equal(t, "竞品 Codex 调研", got.Data.Session.Title)
	assert.True(t, got.Data.Session.Renamable)
	require.NotNil(t, fakeConversationSVC.createdMeta)
	assert.Equal(t, int64(789), fakeConversationSVC.createdMeta.AgentID)
	assert.Equal(t, int64(42), fakeConversationSVC.createdMeta.UserID)
	assert.Equal(t, int64(10000010), fakeConversationSVC.createdMeta.ConnectorID)
	assert.Equal(t, common.Scene_Playground, fakeConversationSVC.createdMeta.Scene)
	assert.Contains(t, fakeConversationSVC.createdMeta.Ext, `"super_agent_session_title":"竞品 Codex 调研"`)
}

func TestSuperAgentSessionListRouteReturnsRenamableSessions(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		list: []*convEntity.Conversation{
			{
				ID:          123,
				SectionID:   456,
				AgentID:     789,
				ConnectorID: 10000010,
				CreatorID:   42,
				Scene:       common.Scene_Playground,
				Ext:         `{"super_agent_session_title":"Deep research"}`,
				CreatedAt:   1000,
				UpdatedAt:   2000,
			},
		},
		hasMore: true,
	}
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
	})

	h := server.Default()
	Register(h)

	body := `{"agent_id":"789","user_id":"42","page":1,"page_size":20}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/list",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			HasMore  bool `json:"has_more"`
			Sessions []struct {
				SessionID string `json:"session_id"`
				Title     string `json:"title"`
				Renamable bool   `json:"renamable"`
			} `json:"sessions"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.True(t, got.Data.HasMore)
	assert.Len(t, got.Data.Sessions, 1)
	assert.Equal(t, "123", got.Data.Sessions[0].SessionID)
	assert.Equal(t, "Deep research", got.Data.Sessions[0].Title)
	assert.True(t, got.Data.Sessions[0].Renamable)
}

func TestSuperAgentSessionGetRouteReturnsSessionAndOptionalSnapshot(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalRunSVC := conversation.ConversationSVC.AgentRunDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:          123,
			SectionID:   456,
			AgentID:     789,
			ConnectorID: 10000010,
			CreatorID:   42,
			Scene:       common.Scene_Playground,
			Ext:         `{"super_agent_session_title":"Harness restore"}`,
			CreatedAt:   1000,
			UpdatedAt:   2000,
		},
	}
	conversation.ConversationSVC.AgentRunDomainSVC = &fakeSuperAgentRunDomainSVC{
		record: &agentrunEntity.RunRecordMeta{
			ID:             765,
			ConversationID: 123,
			AgentID:        789,
			Status:         agentrunEntity.RunStatusCompleted,
			CreatedAt:      1000,
			UpdatedAt:      2000,
			CompletedAt:    2100,
		},
	}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:             901,
				RunID:          765,
				ConversationID: 123,
				AgentID:        789,
				Role:           schema.User,
				MessageType:    crossMessage.MessageTypeQuestion,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "restore this session",
				Status:         crossMessage.MessageStatusAvailable,
				CreatedAt:      1500,
				UpdatedAt:      1600,
			},
		},
	}
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   789,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	crosssandbox.SetDefaultSVC(fakeSuperAgentWorkspaceSandboxManager{})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.AgentRunDomainSVC = originalRunSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"conversation_id":"123","include_snapshot":true,"message_limit":10,"include_workspace":true,"workspace_path":"/workspace","workspace_recursive":true,"include_trace":false,"include_approvals":false,"include_harness":false,"include_tool_outputs":false,"include_artifacts":false}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/get",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Session struct {
				SessionID      string       `json:"session_id"`
				ConversationID string       `json:"conversation_id"`
				SectionID      string       `json:"section_id"`
				AgentID        string       `json:"agent_id"`
				Scene          common.Scene `json:"scene"`
				Title          string       `json:"title"`
				Renamable      bool         `json:"renamable"`
			} `json:"session"`
			Snapshot *struct {
				Components []string `json:"components"`
				Messages   *struct {
					Messages []struct {
						MessageID string `json:"message_id"`
						Role      string `json:"role"`
						Type      string `json:"type"`
						Content   string `json:"content"`
					} `json:"messages"`
				} `json:"messages"`
				Runs *struct {
					Runs []struct {
						RunID  string `json:"run_id"`
						Status string `json:"status"`
						Active bool   `json:"active"`
					} `json:"runs"`
				} `json:"runs"`
				Workspace *struct {
					Path  string `json:"path"`
					Files []struct {
						Path  string `json:"path"`
						IsDir bool   `json:"is_dir"`
					} `json:"files"`
				} `json:"workspace"`
			} `json:"snapshot"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.Session.SessionID)
	assert.Equal(t, "123", got.Data.Session.ConversationID)
	assert.Equal(t, "456", got.Data.Session.SectionID)
	assert.Equal(t, "789", got.Data.Session.AgentID)
	assert.Equal(t, common.Scene_Playground, got.Data.Session.Scene)
	assert.Equal(t, "Harness restore", got.Data.Session.Title)
	assert.True(t, got.Data.Session.Renamable)
	require.NotNil(t, got.Data.Snapshot)
	assert.Contains(t, got.Data.Snapshot.Components, "messages")
	assert.Contains(t, got.Data.Snapshot.Components, "runs")
	require.NotNil(t, got.Data.Snapshot.Messages)
	require.Len(t, got.Data.Snapshot.Messages.Messages, 1)
	assert.Equal(t, "901", got.Data.Snapshot.Messages.Messages[0].MessageID)
	assert.Equal(t, "user", got.Data.Snapshot.Messages.Messages[0].Role)
	assert.Equal(t, "question", got.Data.Snapshot.Messages.Messages[0].Type)
	assert.Equal(t, "restore this session", got.Data.Snapshot.Messages.Messages[0].Content)
	require.NotNil(t, got.Data.Snapshot.Runs)
	require.Len(t, got.Data.Snapshot.Runs.Runs, 1)
	assert.Equal(t, "765", got.Data.Snapshot.Runs.Runs[0].RunID)
	assert.Equal(t, string(agentrunEntity.RunStatusCompleted), got.Data.Snapshot.Runs.Runs[0].Status)
	assert.False(t, got.Data.Snapshot.Runs.Runs[0].Active)
	assert.Contains(t, got.Data.Snapshot.Components, "workspace")
	require.NotNil(t, got.Data.Snapshot.Workspace)
	assert.Equal(t, "/workspace", got.Data.Snapshot.Workspace.Path)
	require.Len(t, got.Data.Snapshot.Workspace.Files, 2)
	assert.Equal(t, "/workspace/src/main.go", got.Data.Snapshot.Workspace.Files[1].Path)
	assert.False(t, got.Data.Snapshot.Workspace.Files[1].IsDir)
}

func TestSuperAgentSessionRenameRoutePersistsTitleInConversationExt(t *testing.T) {
	fakeConversationSVC := &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:          123,
			SectionID:   456,
			AgentID:     789,
			ConnectorID: 10000010,
			CreatorID:   42,
			Scene:       common.Scene_Playground,
			Ext:         `{"custom_variables":{"topic":"ai"}}`,
			CreatedAt:   1000,
			UpdatedAt:   2000,
		},
	}
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	conversation.ConversationSVC.ConversationDomainSVC = fakeConversationSVC
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
	})

	h := server.Default()
	Register(h)

	body := `{"conversation_id":"123","title":"竞品 Codex 对比","user_id":"42"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/rename",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			Session struct {
				SessionID string `json:"session_id"`
				Title     string `json:"title"`
			} `json:"session"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.Session.SessionID)
	assert.Equal(t, "竞品 Codex 对比", got.Data.Session.Title)
	assert.Contains(t, fakeConversationSVC.updatedExt, `"super_agent_session_title":"竞品 Codex 对比"`)
	assert.Contains(t, fakeConversationSVC.updatedExt, `"custom_variables"`)
}

func TestSuperAgentSessionDeleteRouteDeletesOwnedSession(t *testing.T) {
	fakeConversationSVC := &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   789,
			CreatorID: 42,
			Scene:     common.Scene_Playground,
		},
	}
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalSingleAgentSVC := application.SingleAgentSVC
	originalSandboxSVC := crosssandbox.DefaultSVC()
	conversation.ConversationSVC.ConversationDomainSVC = fakeConversationSVC
	application.SingleAgentSVC = &application.SingleAgentApplicationService{
		DomainSVC: &fakeSuperAgentSingleAgentDomainSVC{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   789,
					CreatorID: 42,
					SpaceID:   1,
				},
			},
		},
	}
	commands := []string{}
	crosssandbox.SetDefaultSVC(fakeSuperAgentContextClearSandboxManager{commands: &commands})
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		application.SingleAgentSVC = originalSingleAgentSVC
		crosssandbox.SetDefaultSVC(originalSandboxSVC)
	})

	h := server.Default()
	Register(h)

	body := `{"conversation_id":"123","user_id":"42"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/sessions/delete",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			SessionID      string `json:"session_id"`
			ConversationID string `json:"conversation_id"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	assert.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.SessionID)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Equal(t, int64(123), fakeConversationSVC.deletedID)
	assert.Contains(t, commands, "rm -f '/workspace/.agent/sessions/123/context-summary.json'")
	assert.Contains(t, commands, "rm -rf '/workspace/.agent/sessions/123/plan.json'")
	assert.Contains(t, commands, "rm -rf '/workspace/.agent/tooloutputs/sessions/123'")
	joined := strings.Join(commands, "\n")
	assert.NotContains(t, joined, "rm -f '/workspace/.agent/context-summary.json'")
	assert.NotContains(t, joined, "rm -f '/workspace/.plan.json'")
}

func TestSuperAgentMessageListRouteReturnsSessionMessages(t *testing.T) {
	originalConversationSVC := conversation.ConversationSVC.ConversationDomainSVC
	originalMessageSVC := conversation.ConversationSVC.MessageDomainSVC
	conversation.ConversationSVC.ConversationDomainSVC = &fakeSuperAgentConversationDomainSVC{
		conversation: &convEntity.Conversation{
			ID:        123,
			AgentID:   456,
			CreatorID: 42,
		},
	}
	conversation.ConversationSVC.MessageDomainSVC = &fakeSuperAgentMessageDomainSVC{
		messages: []*messageEntity.Message{
			{
				ID:               102,
				RunID:            765,
				ConversationID:   123,
				AgentID:          456,
				SectionID:        9,
				Role:             schema.Assistant,
				MessageType:      crossMessage.MessageTypeAnswer,
				ContentType:      crossMessage.ContentTypeText,
				Content:          "assistant answer",
				ReasoningContent: "reasoning",
				Status:           crossMessage.MessageStatusAvailable,
				Ext: map[string]string{
					"tool_calls_ids": "tool-call-1",
				},
				CreatedAt: 2000,
				UpdatedAt: 2100,
			},
			{
				ID:             101,
				RunID:          764,
				ConversationID: 123,
				AgentID:        456,
				SectionID:      9,
				Role:           schema.User,
				MessageType:    crossMessage.MessageTypeQuestion,
				ContentType:    crossMessage.ContentTypeText,
				Content:        "user question",
				Status:         crossMessage.MessageStatusAvailable,
				CreatedAt:      1000,
				UpdatedAt:      1100,
			},
		},
	}
	t.Cleanup(func() {
		conversation.ConversationSVC.ConversationDomainSVC = originalConversationSVC
		conversation.ConversationSVC.MessageDomainSVC = originalMessageSVC
	})

	h := server.Default()
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		ctx = ctxcache.Init(ctx)
		ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 42})
		c.Next(ctx)
	})
	Register(h)

	body := `{"conversation_id":"123","limit":10,"order_by":"ASC"}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/messages/list",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Code int `json:"code"`
		Data struct {
			ConversationID string `json:"conversation_id"`
			AgentID        string `json:"agent_id"`
			Messages       []struct {
				MessageID        string            `json:"message_id"`
				ConversationID   string            `json:"conversation_id"`
				RunID            string            `json:"run_id"`
				AgentID          string            `json:"agent_id"`
				SectionID        string            `json:"section_id"`
				Role             string            `json:"role"`
				Type             string            `json:"type"`
				Content          string            `json:"content"`
				ContentType      string            `json:"content_type"`
				ReasoningContent string            `json:"reasoning_content"`
				Metadata         map[string]string `json:"metadata"`
				CreatedAt        int64             `json:"created_at"`
				UpdatedAt        int64             `json:"updated_at"`
			} `json:"messages"`
			HasMore    bool   `json:"has_more"`
			PrevCursor string `json:"prev_cursor"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Result().Body(), &got)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "123", got.Data.ConversationID)
	assert.Equal(t, "456", got.Data.AgentID)
	require.Len(t, got.Data.Messages, 2)
	assert.Equal(t, "101", got.Data.Messages[0].MessageID)
	assert.Equal(t, "user", got.Data.Messages[0].Role)
	assert.Equal(t, "question", got.Data.Messages[0].Type)
	assert.Equal(t, "user question", got.Data.Messages[0].Content)
	assert.Equal(t, "102", got.Data.Messages[1].MessageID)
	assert.Equal(t, "assistant", got.Data.Messages[1].Role)
	assert.Equal(t, "answer", got.Data.Messages[1].Type)
	assert.Equal(t, "assistant answer", got.Data.Messages[1].Content)
	assert.Equal(t, "reasoning", got.Data.Messages[1].ReasoningContent)
	assert.Equal(t, "tool-call-1", got.Data.Messages[1].Metadata["tool_calls_ids"])
	assert.Equal(t, "101", got.Data.PrevCursor)
	assert.Equal(t, "102", got.Data.NextCursor)
}

func TestSuperAgentMarketplaceGetRouteIsRegistered(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/super-agent/marketplace/get",
		nil,
	)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestSuperAgentHarnessRoutesRejectMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	for _, path := range []string{
		"/api/super-agent/harness/state",
		"/api/super-agent/harness/plan",
		"/api/super-agent/harness/tool-outputs",
		"/api/super-agent/harness/cleanup",
		"/api/super-agent/harness/snapshot",
		"/api/super-agent/sessions/get",
		"/api/super-agent/messages/list",
		"/api/super-agent/runtime-config/get",
		"/api/super-agent/runtime-config/update",
		"/api/super-agent/runtime-config/delete",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestSuperAgentHarnessSnapshotRouteRejectsMissingSnapshotKey(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"trace_limit":20}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/super-agent/harness/snapshot",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Result().Body()), "conversation_id or agent_id is required")
}

func TestSuperAgentSkillRoutesRejectMalformedJSON(t *testing.T) {
	h := server.Default()
	Register(h)

	for _, path := range []string{
		"/api/super-agent/skills/create",
		"/api/super-agent/skills/update",
		"/api/super-agent/skills/delete",
		"/api/super-agent/skills/publish",
		"/api/super-agent/skills/assets/upsert",
		"/api/super-agent/skills/assets/delete",
		"/api/super-agent/skills/validate-package",
		"/api/super-agent/skills/import",
		"/api/super-agent/skills/import-runtime",
		"/api/super-agent/skills/export",
		"/api/super-agent/marketplace/install",
		"/api/super-agent/products/list",
		"/api/super-agent/products/get",
		"/api/super-agent/products/install",
		"/api/super-agent/products/upgrade",
		"/api/super-agent/products/uninstall",
		"/api/super-agent/marketplace/products/list",
		"/api/super-agent/marketplace/products/get",
		"/api/super-agent/marketplace/products/install",
	} {
		t.Run(path, func(t *testing.T) {
			w := ut.PerformRequest(
				h.Engine,
				http.MethodPost,
				path,
				&ut.Body{Body: bytes.NewBufferString(`{`), Len: len(`{`)},
				ut.Header{Key: "Content-Type", Value: "application/json"},
			)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func findSuperAgentTraceEvent(events []superagenttrace.Event, eventName string) *superagenttrace.Event {
	for i := range events {
		if events[i].Event == eventName {
			return &events[i]
		}
	}
	return nil
}

func countSuperAgentTraceEvents(events []superagenttrace.Event, eventName string) int {
	count := 0
	for _, event := range events {
		if event.Event == eventName {
			count++
		}
	}
	return count
}
