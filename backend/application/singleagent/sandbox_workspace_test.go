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

package singleagent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	agentservice "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/service"
	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox"
	sbx "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type fakeSandboxManager struct{}

func (fakeSandboxManager) Exec(context.Context, string, string, int) (*sbx.ExecResponse, error) {
	return &sbx.ExecResponse{}, nil
}

func (fakeSandboxManager) ReadFile(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (fakeSandboxManager) WriteFile(context.Context, string, string, []byte) error {
	return nil
}

func (fakeSandboxManager) ListFiles(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (fakeSandboxManager) EditFile(context.Context, string, string, string, string, bool) (int, error) {
	return 0, nil
}

func (fakeSandboxManager) Grep(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (fakeSandboxManager) Glob(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeSandboxManager) SyncSkill(context.Context, string, string, map[string][]byte) error {
	return nil
}

func (fakeSandboxManager) CheckpointTo(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeSandboxManager) RestoreFrom(context.Context, string, string) error {
	return nil
}

func (fakeSandboxManager) EnsureSandboxWithTemplate(context.Context, string, string, bool) error {
	return nil
}

type fakeHarnessSandboxManager struct {
	fakeSandboxManager
}

func (fakeHarnessSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "/workspace/.agent/tooloutputs") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "call-1.json",
				"is_dir": false,
				"size":   128,
				"mtime":  int64(1781800000),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	if strings.Contains(cmd, "/skills") {
		out, _ := json.Marshal(map[string]any{
			"ok": true,
			"skills": []map[string]any{
				{
					"name":        "report-kit",
					"path":        "/skills/report-kit",
					"entry_path":  "/skills/report-kit/SKILL.md",
					"standard":    true,
					"description": "Build reports",
					"version":     "0.1.0",
					"category":    "productivity",
					"file_paths":  []string{"SKILL.md", "assets/logo.png", "scripts/run.sh"},
					"asset_paths": []string{"assets/logo.png"},
					"image_paths": []string{"assets/logo.png"},
				},
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return &sbx.ExecResponse{}, nil
}

func (fakeHarnessSandboxManager) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	if path == "/workspace/.plan.json" {
		return []byte(`{"steps":[{"step":"write code","status":"in_progress"}]}`), nil
	}
	if path == "/workspace/.agent/tooloutputs/call-1.json" {
		return []byte(`{"call_id":"call-1","tool":"run_bash","args":{"cmd":"ls"},"result":{"exit_code":0}}`), nil
	}
	return nil, assert.AnError
}

type fakeHarnessSessionToolOutputSandboxManager struct {
	fakeHarnessSandboxManager
}

func (fakeHarnessSessionToolOutputSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "/workspace/.agent/tooloutputs/sessions/456") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "call-session.json",
				"path":   "/workspace/.agent/tooloutputs/sessions/456/call-session.json",
				"is_dir": false,
				"size":   256,
				"mtime":  int64(1781800100),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return fakeHarnessSandboxManager{}.Exec(context.Background(), "", cmd, 0)
}

func (fakeHarnessSessionToolOutputSandboxManager) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	if path == "/workspace/.agent/tooloutputs/sessions/456/call-session.json" {
		return []byte(`{"call_id":"call-session","conversation_id":"456","tool":"grep","args":{"pattern":"TODO"},"result":{"matches":2}}`), nil
	}
	return fakeHarnessSandboxManager{}.ReadFile(context.Background(), "", path)
}

type fakeHarnessSessionPlanSandboxManager struct {
	fakeHarnessSandboxManager
}

func (fakeHarnessSessionPlanSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "/workspace/.agent/sessions/456/plan.json") {
		return &sbx.ExecResponse{Stdout: `{"exists":true,"name":"plan.json","path":"/workspace/.agent/sessions/456/plan.json","is_dir":false,"size":58,"mtime":1781800200}`}, nil
	}
	return fakeHarnessSandboxManager{}.Exec(context.Background(), "", cmd, 0)
}

func (fakeHarnessSessionPlanSandboxManager) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	if path == "/workspace/.agent/sessions/456/plan.json" {
		return []byte(`[
  {"content":"会话级计划","status":"in_progress"}
]`), nil
	}
	if path == "/workspace/.plan.json" {
		return []byte(`[
  {"content":"全局计划不应该污染会话","status":"completed"}
]`), nil
	}
	return fakeHarnessSandboxManager{}.ReadFile(context.Background(), "", path)
}

type fakeHarnessCleanupSandboxManager struct {
	fakeSandboxManager
	files  map[string][]byte
	mtimes map[string]int64
}

func (m *fakeHarnessCleanupSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.HasPrefix(cmd, "rm -f -- '") && strings.HasSuffix(cmd, "'") {
		delete(m.files, strings.TrimSuffix(strings.TrimPrefix(cmd, "rm -f -- '"), "'"))
	}
	if strings.HasPrefix(cmd, "python3 -c ") && strings.Contains(cmd, "/workspace/.agent/tooloutputs") {
		paths := make([]string, 0, len(m.files))
		for path := range m.files {
			if strings.HasPrefix(path, "/workspace/.agent/tooloutputs/") {
				paths = append(paths, path)
			}
		}
		sort.Strings(paths)
		out := make([]map[string]any, 0, len(paths))
		for _, path := range paths {
			out = append(out, map[string]any{
				"name":   strings.TrimPrefix(path, "/workspace/.agent/tooloutputs/"),
				"path":   path,
				"is_dir": false,
				"size":   len(m.files[path]),
				"mtime":  m.mtimes[path],
			})
		}
		blob, _ := json.Marshal(out)
		return &sbx.ExecResponse{Stdout: string(blob), ExitCode: 0}, nil
	}
	return &sbx.ExecResponse{ExitCode: 0}, nil
}

type fakeRecordingSandboxManager struct {
	fakeSandboxManager
	execs []string
}

func (f *fakeRecordingSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	f.execs = append(f.execs, cmd)
	return &sbx.ExecResponse{}, nil
}

type fakeArtifactListSandboxManager struct {
	fakeSandboxManager
}

func (fakeArtifactListSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "hashlib,mimetypes") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "final-report.html",
				"path":   "/outputs/final-report.html",
				"size":   int64(2048),
				"mtime":  int64(1781900200),
				"mime":   "text/html",
				"sha256": strings.Repeat("a", 64),
			},
			{
				"name":   "chart.png",
				"path":   "/outputs/chart.png",
				"size":   int64(4096),
				"mtime":  int64(1781900201),
				"mime":   "image/png",
				"sha256": strings.Repeat("b", 64),
			},
			{
				"name":   "metrics.csv",
				"path":   "/outputs/metrics.csv",
				"size":   int64(512),
				"mtime":  int64(1781900202),
				"mime":   "text/csv",
				"sha256": strings.Repeat("c", 64),
			},
			{
				"name":   "notes.txt",
				"path":   "/outputs/notes.txt",
				"size":   int64(128),
				"mtime":  int64(1781900203),
				"mime":   "text/plain",
				"sha256": strings.Repeat("d", 64),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	return &sbx.ExecResponse{}, nil
}

type fakeStatSandboxManager struct {
	fakeSandboxManager
}

func (fakeStatSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	switch {
	case strings.Contains(cmd, "/skills/pdf/SKILL.md"):
		return &sbx.ExecResponse{Stdout: `{"exists":true,"name":"SKILL.md","path":"/skills/pdf/SKILL.md","is_dir":false,"size":512,"mtime":1781800100}`}, nil
	case strings.Contains(cmd, "/workspace/missing.md"):
		return &sbx.ExecResponse{Stdout: `{"exists":false,"name":"missing.md","path":"/workspace/missing.md","is_dir":false,"size":0,"mtime":0}`}, nil
	default:
		return &sbx.ExecResponse{Stdout: `{"exists":false}`}, nil
	}
}

type fakeListSandboxManager struct {
	fakeSandboxManager
	listCmd string
}

func (f *fakeListSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	f.listCmd = cmd
	if strings.Contains(cmd, "os.walk") {
		out, _ := json.Marshal([]map[string]any{
			{
				"name":   "reports",
				"path":   "/workspace/reports",
				"is_dir": true,
				"size":   0,
				"mtime":  int64(1781900000),
			},
			{
				"name":   "final.md",
				"path":   "/workspace/reports/final.md",
				"is_dir": false,
				"size":   12,
				"mtime":  int64(1781900001),
			},
		})
		return &sbx.ExecResponse{Stdout: string(out)}, nil
	}
	out, _ := json.Marshal([]map[string]any{
		{
			"name":   "reports",
			"is_dir": true,
			"size":   0,
			"mtime":  int64(1781900000),
		},
	})
	return &sbx.ExecResponse{Stdout: string(out)}, nil
}

type fakeReadSandboxManager struct {
	fakeSandboxManager
	content []byte
}

func (f *fakeReadSandboxManager) ReadFile(_ context.Context, _ string, _ string) ([]byte, error) {
	return append([]byte(nil), f.content...), nil
}

type fakeSearchSandboxManager struct {
	fakeSandboxManager
	grepPattern string
	grepPath    string
	grepCmd     string
	globCmd     string
}

func (f *fakeSearchSandboxManager) Grep(_ context.Context, _ string, pattern string, path string) (string, error) {
	f.grepPattern = pattern
	f.grepPath = path
	return "/skills/pdf/SKILL.md:1:# PDF skill\n", nil
}

func (f *fakeSearchSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	if strings.Contains(cmd, "--no-heading") || strings.Contains(cmd, "grep") && strings.Contains(cmd, "-RIn") {
		f.grepCmd = cmd
		return &sbx.ExecResponse{Stdout: "/skills/pdf/SKILL.md:1:# PDF skill\n"}, nil
	}
	f.globCmd = cmd
	return &sbx.ExecResponse{Stdout: "/skills/pdf/SKILL.md\n/skills/docx/SKILL.md\n"}, nil
}

type fakeEditSandboxManager struct {
	fakeSandboxManager
	path       string
	oldString  string
	newString  string
	replaceAll bool
}

func (f *fakeEditSandboxManager) EditFile(_ context.Context, _ string, path, oldStr, newStr string, replaceAll bool) (int, error) {
	f.path = path
	f.oldString = oldStr
	f.newString = newStr
	f.replaceAll = replaceAll
	return 2, nil
}

type fakeExecSandboxManager struct {
	fakeSandboxManager
	cmd        string
	timeoutSec int
}

func (f *fakeExecSandboxManager) Exec(_ context.Context, _ string, cmd string, timeoutSec int) (*sbx.ExecResponse, error) {
	f.cmd = cmd
	f.timeoutSec = timeoutSec
	return &sbx.ExecResponse{
		Stdout:   "ok\n",
		Stderr:   "warn\n",
		ExitCode: 7,
	}, nil
}

type fakePlanSandboxManager struct {
	fakeSandboxManager
	path    string
	content string
}

func (f *fakePlanSandboxManager) WriteFile(_ context.Context, _ string, path string, content []byte) error {
	f.path = path
	f.content = string(content)
	return nil
}

type fakeUploadSandboxManager struct {
	fakeSandboxManager
	path    string
	content []byte
}

func (f *fakeUploadSandboxManager) WriteFile(_ context.Context, _ string, path string, content []byte) error {
	f.path = path
	f.content = append([]byte(nil), content...)
	return nil
}

type fakePatchSandboxManager struct {
	fakeSandboxManager
	files  map[string]string
	execs  []string
	writes []string
}

func (f *fakePatchSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	f.execs = append(f.execs, cmd)
	if strings.HasPrefix(cmd, "rm -rf ") {
		target := strings.TrimPrefix(cmd, "rm -rf ")
		target = strings.Trim(target, "'")
		delete(f.files, target)
	}
	if strings.Contains(cmd, " && mv ") {
		parts := strings.Split(cmd, "'")
		if len(parts) >= 6 {
			fromPath := parts[3]
			toPath := parts[5]
			f.files[toPath] = f.files[fromPath]
			delete(f.files, fromPath)
		}
	}
	return &sbx.ExecResponse{}, nil
}

func (f *fakePatchSandboxManager) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	content, ok := f.files[path]
	if !ok {
		return nil, assert.AnError
	}
	return []byte(content), nil
}

func (f *fakePatchSandboxManager) WriteFile(_ context.Context, _ string, path string, content []byte) error {
	f.files[path] = string(content)
	f.writes = append(f.writes, path)
	return nil
}

type fakeSingleAgentDomain struct {
	agentservice.SingleAgent
	draft *agententity.SingleAgent
}

func (f *fakeSingleAgentDomain) GetSingleAgentDraft(context.Context, int64) (*agententity.SingleAgent, error) {
	return f.draft, nil
}

func sandboxFilePaths(files []*developer_api.SandboxFileInfo) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file != nil {
			paths = append(paths, file.Path)
		}
	}
	return paths
}

func TestResolveSandboxKeyUsesOpenAPIAuthUser(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	_, key, err := svc.resolveSandboxKey(ctx, agentID, nil)

	assert.NoError(t, err)
	assert.Equal(t, agentsandbox.SandboxKeyFor(consts.CozeConnectorID, agentID, "api-user-77"), key)
}

func TestSuperAgentWorkspaceListAcceptsAgentIDAlias(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ListSuperAgentWorkspaceFiles(ctx, &developer_api.ListSandboxFilesRequest{
		AgentID: agentID,
		Path:    "/workspace",
	})

	assert.NoError(t, err)
	assert.Equal(t, "/workspace", resp.Data.Path)
}

func TestSuperAgentWorkspaceListHonorsManifestRecursive(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeListSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	var req developer_api.ListSandboxFilesRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"path": "/workspace",
		"recursive": true
	}`), &req))

	resp, err := svc.ListSuperAgentWorkspaceFiles(ctx, &req)

	require.NoError(t, err)
	assert.Equal(t, "/workspace", resp.Data.Path)
	assert.Contains(t, fakeSandbox.listCmd, "os.walk")
	assert.ElementsMatch(t, []string{"/workspace/reports", "/workspace/reports/final.md"}, sandboxFilePaths(resp.Data.Files))
}

func TestUploadSuperAgentWorkspaceFileAcceptsManifestEncodingBase64(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeUploadSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	binaryContent := []byte{0, 1, 2, 3, 255}
	var req developer_api.UploadSandboxFileRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"path": "/workspace/binary.bin",
		"content": "`+base64.StdEncoding.EncodeToString(binaryContent)+`",
		"encoding": "base64"
	}`), &req))

	resp, err := svc.UploadSuperAgentWorkspaceFile(ctx, &req)

	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Code)
	assert.Equal(t, "/workspace/binary.bin", resp.Data.Path)
	assert.Equal(t, "/workspace/binary.bin", fakeSandbox.path)
	assert.Equal(t, binaryContent, fakeSandbox.content)
}

func TestGetSuperAgentHarnessStateReturnsPlanAndToolOutputs(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeHarnessSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.GetSuperAgentHarnessState(ctx, &developer_api.SuperAgentHarnessStateRequest{
		AgentID: agentID,
	})

	assert.NoError(t, err)
	assert.Equal(t, "/workspace/.plan.json", resp.Data.Plan.Path)
	assert.True(t, resp.Data.Plan.Exists)
	assert.False(t, resp.Data.Plan.IsBinary)
	assert.Contains(t, resp.Data.Plan.Content, "write code")
	assert.Equal(t, "/workspace/.agent/tooloutputs", resp.Data.ToolOutputs.Root)
	assert.Len(t, resp.Data.ToolOutputs.Files, 1)
	assert.Equal(t, "/workspace/.agent/tooloutputs/call-1.json", resp.Data.ToolOutputs.Files[0].Path)
	assert.Equal(t, "/skills", resp.Data.RuntimeSkills.Root)
	require.Len(t, resp.Data.RuntimeSkills.Skills, 1)
	assert.Equal(t, "report-kit", resp.Data.RuntimeSkills.Skills[0].Name)
	assert.Equal(t, "/skills/report-kit/SKILL.md", resp.Data.RuntimeSkills.Skills[0].EntryPath)
	assert.True(t, resp.Data.RuntimeSkills.Skills[0].Standard)
	assert.Equal(t, "Build reports", resp.Data.RuntimeSkills.Skills[0].Description)
	assert.Equal(t, "0.1.0", resp.Data.RuntimeSkills.Skills[0].Version)
	assert.ElementsMatch(t, []string{"SKILL.md", "assets/logo.png", "scripts/run.sh"}, resp.Data.RuntimeSkills.Skills[0].FilePaths)
	assert.ElementsMatch(t, []string{"assets/logo.png"}, resp.Data.RuntimeSkills.Skills[0].ImagePaths)
	rawContext, err := json.Marshal(resp.Data.Context)
	require.NoError(t, err)
	var contextState struct {
		Policy struct {
			Strategy                   string `json:"strategy"`
			SummaryVersion             string `json:"summary_version"`
			Trigger                    string `json:"trigger"`
			MaxBytes                   int    `json:"max_bytes"`
			RecentMessages             int    `json:"recent_messages"`
			SummaryMaxRunes            int    `json:"summary_max_runes"`
			SummaryPath                string `json:"summary_path"`
			SessionSummaryPathTemplate string `json:"session_summary_path_template"`
			ClearRoute                 string `json:"clear_route"`
		} `json:"policy"`
	}
	require.NoError(t, json.Unmarshal(rawContext, &contextState))
	assert.Equal(t, "tool-output-offload+session-summary", contextState.Policy.Strategy)
	assert.Equal(t, "v1", contextState.Policy.SummaryVersion)
	assert.Equal(t, "history_bytes_exceeded", contextState.Policy.Trigger)
	assert.Equal(t, 160*1024, contextState.Policy.MaxBytes)
	assert.Equal(t, 16, contextState.Policy.RecentMessages)
	assert.Equal(t, 1600, contextState.Policy.SummaryMaxRunes)
	assert.Equal(t, "/workspace/.agent/context-summary.json", contextState.Policy.SummaryPath)
	assert.Equal(t, "/workspace/.agent/sessions/{conversation_id}/context-summary.json", contextState.Policy.SessionSummaryPathTemplate)
	assert.Equal(t, "POST /api/super-agent/harness/context/clear", contextState.Policy.ClearRoute)
}

func TestGetSuperAgentHarnessStateDefaultsToolOutputsToConversationScope(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeHarnessSessionToolOutputSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.GetSuperAgentHarnessState(ctx, &developer_api.SuperAgentHarnessStateRequest{
		AgentID:        agentID,
		ConversationID: 456,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Data.ToolOutputs)
	assert.Equal(t, "/workspace/.agent/tooloutputs", resp.Data.ToolOutputs.Root)
	rawToolOutputs, err := json.Marshal(resp.Data.ToolOutputs)
	require.NoError(t, err)
	var toolOutputs struct {
		Path string `json:"path"`
	}
	require.NoError(t, json.Unmarshal(rawToolOutputs, &toolOutputs))
	assert.Equal(t, "/workspace/.agent/tooloutputs/sessions/456", toolOutputs.Path)
	require.Len(t, resp.Data.ToolOutputs.Files, 1)
	assert.Equal(t, "/workspace/.agent/tooloutputs/sessions/456/call-session.json", resp.Data.ToolOutputs.Files[0].Path)
	assert.Equal(t, "call-session.json", resp.Data.ToolOutputs.Files[0].Name)
}

func TestGetSuperAgentHarnessStateDefaultsPlanToConversationScope(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeHarnessSessionPlanSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.GetSuperAgentHarnessState(ctx, &developer_api.SuperAgentHarnessStateRequest{
		AgentID:        agentID,
		ConversationID: 456,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Data.Plan)
	assert.Equal(t, "/workspace/.agent/sessions/456/plan.json", resp.Data.Plan.Path)
	assert.True(t, resp.Data.Plan.Exists)
	assert.Contains(t, resp.Data.Plan.Content, "会话级计划")
	assert.NotContains(t, resp.Data.Plan.Content, "全局计划不应该污染会话")
	rawPlan, err := json.Marshal(resp.Data.Plan)
	require.NoError(t, err)
	var planState struct {
		Mtime int64 `json:"mtime"`
	}
	require.NoError(t, json.Unmarshal(rawPlan, &planState))
	assert.Equal(t, int64(1781800200), planState.Mtime)
}

func TestListSuperAgentHarnessToolOutputsListsAndReadsDedicatedRoot(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeHarnessSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ListSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessToolOutputsRequest{
		AgentID: agentID,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "/workspace/.agent/tooloutputs", resp.Data.Root)
	assert.Equal(t, "/workspace/.agent/tooloutputs", resp.Data.Path)
	require.Len(t, resp.Data.Files, 1)
	assert.Equal(t, "/workspace/.agent/tooloutputs/call-1.json", resp.Data.Files[0].Path)
	assert.Nil(t, resp.Data.Content)

	readResp, err := svc.ListSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessToolOutputsRequest{
		AgentID:        agentID,
		Path:           "call-1.json",
		IncludeContent: true,
	})

	require.NoError(t, err)
	require.NotNil(t, readResp.Data.Content)
	assert.Equal(t, "/workspace/.agent/tooloutputs/call-1.json", readResp.Data.Path)
	assert.Contains(t, readResp.Data.Content.Content, `"tool":"run_bash"`)
}

func TestListSuperAgentHarnessToolOutputsReturnsStructuredEntries(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeHarnessSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ListSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessToolOutputsRequest{
		AgentID: agentID,
	})

	require.NoError(t, err)
	blob, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var got struct {
		Entries []struct {
			Path             string `json:"path"`
			Name             string `json:"name"`
			ToolCallID       string `json:"tool_call_id"`
			Tool             string `json:"tool"`
			Status           string `json:"status"`
			Arguments        any    `json:"arguments"`
			Result           any    `json:"result"`
			ArgumentsPreview string `json:"arguments_preview"`
			ResultPreview    string `json:"result_preview"`
			Summary          string `json:"summary"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(blob, &got))
	require.Len(t, got.Entries, 1)
	entry := got.Entries[0]
	assert.Equal(t, "/workspace/.agent/tooloutputs/call-1.json", entry.Path)
	assert.Equal(t, "call-1.json", entry.Name)
	assert.Equal(t, "call-1", entry.ToolCallID)
	assert.Equal(t, "run_bash", entry.Tool)
	assert.Equal(t, "completed", entry.Status)
	assert.Equal(t, map[string]any{"cmd": "ls"}, entry.Arguments)
	assert.Equal(t, map[string]any{"exit_code": float64(0)}, entry.Result)
	assert.JSONEq(t, `{"cmd":"ls"}`, entry.ArgumentsPreview)
	assert.JSONEq(t, `{"exit_code":0}`, entry.ResultPreview)
	assert.Contains(t, entry.Summary, "run_bash")
	assert.Contains(t, entry.Summary, "cmd")
	assert.Contains(t, entry.Summary, "exit_code")
}

func TestListSuperAgentHarnessToolOutputsRejectsPathOutsideRoot(t *testing.T) {
	_, err := sanitizeSuperAgentHarnessToolOutputPath("/workspace/.plan.json")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace/.agent/tooloutputs")
}

func TestCleanupSuperAgentHarnessToolOutputsDryRunAndApply(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fm := &fakeHarnessCleanupSandboxManager{files: map[string][]byte{
		"/workspace/.agent/tooloutputs/old.json": []byte(`{"tool":"run_bash"}`),
		"/workspace/.agent/tooloutputs/new.json": []byte(`{"tool":"grep"}`),
		"/workspace/keep.txt":                    []byte("keep"),
	}}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	dryRunResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID: agentID,
		Paths:   []string{"old.json"},
		DryRun:  true,
	})

	require.NoError(t, err)
	require.NotNil(t, dryRunResp.Data)
	assert.True(t, dryRunResp.Data.DryRun)
	assert.Equal(t, "/workspace/.agent/tooloutputs", dryRunResp.Data.Root)
	assert.Equal(t, int32(1), dryRunResp.Data.Matched)
	assert.Equal(t, int32(0), dryRunResp.Data.Deleted)
	assert.Equal(t, []string{"/workspace/.agent/tooloutputs/old.json"}, dryRunResp.Data.Paths)
	assert.Contains(t, string(fm.files["/workspace/.agent/tooloutputs/old.json"]), "run_bash")

	applyResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID: agentID,
		Paths:   []string{"old.json", "new.json"},
	})

	require.NoError(t, err)
	require.NotNil(t, applyResp.Data)
	assert.False(t, applyResp.Data.DryRun)
	assert.Equal(t, int32(2), applyResp.Data.Matched)
	assert.Equal(t, int32(2), applyResp.Data.Deleted)
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/old.json")
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/new.json")
	assert.Contains(t, string(fm.files["/workspace/keep.txt"]), "keep")
}

func TestCleanupSuperAgentHarnessToolOutputsKeepLatest(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fm := &fakeHarnessCleanupSandboxManager{
		files: map[string][]byte{
			"/workspace/.agent/tooloutputs/old.json": []byte(`{"tool":"run_bash"}`),
			"/workspace/.agent/tooloutputs/mid.json": []byte(`{"tool":"grep"}`),
			"/workspace/.agent/tooloutputs/new.json": []byte(`{"tool":"glob"}`),
			"/workspace/keep.txt":                    []byte("keep"),
		},
		mtimes: map[string]int64{
			"/workspace/.agent/tooloutputs/old.json": 10,
			"/workspace/.agent/tooloutputs/mid.json": 20,
			"/workspace/.agent/tooloutputs/new.json": 30,
			"/workspace/keep.txt":                    5,
		},
	}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	dryRunResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID:    agentID,
		KeepLatest: 1,
		DryRun:     true,
	})

	require.NoError(t, err)
	require.NotNil(t, dryRunResp.Data)
	assert.Equal(t, int32(2), dryRunResp.Data.Matched)
	assert.Equal(t, int32(0), dryRunResp.Data.Deleted)
	assert.Equal(t, []string{
		"/workspace/.agent/tooloutputs/old.json",
		"/workspace/.agent/tooloutputs/mid.json",
	}, dryRunResp.Data.Paths)
	assert.Contains(t, fm.files, "/workspace/.agent/tooloutputs/old.json")

	applyResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID:    agentID,
		KeepLatest: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, applyResp.Data)
	assert.Equal(t, int32(2), applyResp.Data.Matched)
	assert.Equal(t, int32(2), applyResp.Data.Deleted)
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/old.json")
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/mid.json")
	assert.Contains(t, fm.files, "/workspace/.agent/tooloutputs/new.json")
	assert.Contains(t, fm.files, "/workspace/keep.txt")
}

func TestCleanupSuperAgentHarnessToolOutputsKeepLatestWithPrefix(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fm := &fakeHarnessCleanupSandboxManager{
		files: map[string][]byte{
			"/workspace/.agent/tooloutputs/run-a-old.json": []byte(`{"tool":"run_bash"}`),
			"/workspace/.agent/tooloutputs/run-a-mid.json": []byte(`{"tool":"grep"}`),
			"/workspace/.agent/tooloutputs/run-a-new.json": []byte(`{"tool":"glob"}`),
			"/workspace/.agent/tooloutputs/legacy.txt":     []byte(`legacy output`),
		},
		mtimes: map[string]int64{
			"/workspace/.agent/tooloutputs/legacy.txt":     5,
			"/workspace/.agent/tooloutputs/run-a-old.json": 10,
			"/workspace/.agent/tooloutputs/run-a-mid.json": 20,
			"/workspace/.agent/tooloutputs/run-a-new.json": 30,
		},
	}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	dryRunResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID:    agentID,
		KeepLatest: 1,
		Prefix:     "run-a-",
		DryRun:     true,
	})

	require.NoError(t, err)
	require.NotNil(t, dryRunResp.Data)
	assert.Equal(t, int32(2), dryRunResp.Data.Matched)
	assert.Equal(t, []string{
		"/workspace/.agent/tooloutputs/run-a-old.json",
		"/workspace/.agent/tooloutputs/run-a-mid.json",
	}, dryRunResp.Data.Paths)
	assert.Contains(t, fm.files, "/workspace/.agent/tooloutputs/legacy.txt")

	applyResp, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID:    agentID,
		KeepLatest: 1,
		Prefix:     "run-a-",
	})

	require.NoError(t, err)
	require.NotNil(t, applyResp.Data)
	assert.Equal(t, int32(2), applyResp.Data.Deleted)
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/run-a-old.json")
	assert.NotContains(t, fm.files, "/workspace/.agent/tooloutputs/run-a-mid.json")
	assert.Contains(t, fm.files, "/workspace/.agent/tooloutputs/run-a-new.json")
	assert.Contains(t, fm.files, "/workspace/.agent/tooloutputs/legacy.txt")
}

func TestCleanupSuperAgentHarnessToolOutputsRejectsPathOutsideRoot(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(&fakeHarnessCleanupSandboxManager{files: map[string][]byte{}})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   123,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	_, err := svc.CleanupSuperAgentHarnessToolOutputs(ctx, &SuperAgentHarnessCleanupRequest{
		AgentID: 123,
		Paths:   []string{"/workspace/keep.txt"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace/.agent/tooloutputs")
}

func TestSuperAgentArtifactPathIsRestrictedToOutputs(t *testing.T) {
	got, err := sanitizeSuperAgentArtifactPath("reports/final.html")

	assert.NoError(t, err)
	assert.Equal(t, "/outputs/reports/final.html", got)

	_, err = sanitizeSuperAgentArtifactPath("/workspace/final.html")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/outputs")
}

func TestListSuperAgentArtifactsReturnsPreviewMetadata(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeArtifactListSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ListSuperAgentArtifacts(ctx, &SuperAgentArtifactListRequest{
		AgentID: agentID,
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	blob, err := json.Marshal(resp.Data.Artifacts)
	require.NoError(t, err)
	var artifacts []struct {
		ArtifactID    string `json:"artifact_id"`
		Name          string `json:"name"`
		Path          string `json:"path"`
		Kind          string `json:"kind"`
		PreviewType   string `json:"preview_type"`
		Summary       string `json:"summary"`
		Previewable   bool   `json:"previewable"`
		Downloadable  bool   `json:"downloadable"`
		DownloadRoute string `json:"download_route"`
	}
	require.NoError(t, json.Unmarshal(blob, &artifacts))
	require.Len(t, artifacts, 4)
	byName := map[string]struct {
		ArtifactID    string `json:"artifact_id"`
		Name          string `json:"name"`
		Path          string `json:"path"`
		Kind          string `json:"kind"`
		PreviewType   string `json:"preview_type"`
		Summary       string `json:"summary"`
		Previewable   bool   `json:"previewable"`
		Downloadable  bool   `json:"downloadable"`
		DownloadRoute string `json:"download_route"`
	}{}
	for _, artifact := range artifacts {
		byName[artifact.Name] = artifact
		assert.Equal(t, artifact.Path, artifact.ArtifactID)
		assert.True(t, artifact.Downloadable)
		assert.Equal(t, "POST /api/super-agent/artifacts/download", artifact.DownloadRoute)
		assert.NotEmpty(t, artifact.Kind)
		assert.NotEmpty(t, artifact.PreviewType)
		assert.NotEmpty(t, artifact.Summary)
	}
	assert.Equal(t, "report", byName["final-report.html"].Kind)
	assert.Equal(t, "html", byName["final-report.html"].PreviewType)
	assert.True(t, byName["final-report.html"].Previewable)
	assert.Contains(t, byName["final-report.html"].Summary, "HTML report")

	assert.Equal(t, "image", byName["chart.png"].Kind)
	assert.Equal(t, "image", byName["chart.png"].PreviewType)
	assert.True(t, byName["chart.png"].Previewable)
	assert.Contains(t, byName["chart.png"].Summary, "PNG image")

	assert.Equal(t, "table", byName["metrics.csv"].Kind)
	assert.Equal(t, "csv", byName["metrics.csv"].PreviewType)
	assert.True(t, byName["metrics.csv"].Previewable)
	assert.Contains(t, byName["metrics.csv"].Summary, "CSV table")

	assert.Equal(t, "document", byName["notes.txt"].Kind)
	assert.Equal(t, "text", byName["notes.txt"].PreviewType)
	assert.True(t, byName["notes.txt"].Previewable)
	assert.Contains(t, byName["notes.txt"].Summary, "Text document")
}

func TestDeleteSuperAgentArtifactRemovesOnlyOutputsPath(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeRecordingSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.DeleteSuperAgentArtifact(ctx, &SuperAgentArtifactDeleteRequest{
		AgentID:    agentID,
		ArtifactID: "/outputs/reports/final.html",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), resp.Code)
	assert.Contains(t, fakeSandbox.execs, "rm -rf '/outputs/reports/final.html'")

	_, err = svc.DeleteSuperAgentArtifact(ctx, &SuperAgentArtifactDeleteRequest{
		AgentID:    agentID,
		ArtifactID: "/workspace/final.html",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/outputs")
}

func TestMoveSuperAgentArtifactRenamesOnlyOutputsPath(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeRecordingSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.MoveSuperAgentArtifact(ctx, &SuperAgentArtifactMoveRequest{
		AgentID:    agentID,
		ArtifactID: "/outputs/tmp/report.html",
		TargetPath: "/outputs/final/report.html",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), resp.Code)
	assert.Equal(t, "/outputs/tmp/report.html", resp.Data.FromPath)
	assert.Equal(t, "/outputs/final/report.html", resp.Data.Path)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/outputs/final' && mv '/outputs/tmp/report.html' '/outputs/final/report.html'")

	_, err = svc.MoveSuperAgentArtifact(ctx, &SuperAgentArtifactMoveRequest{
		AgentID:    agentID,
		ArtifactID: "/outputs/tmp/report.html",
		TargetPath: "/workspace/report.html",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/outputs")

	_, err = svc.MoveSuperAgentArtifact(ctx, &SuperAgentArtifactMoveRequest{
		AgentID:    agentID,
		ArtifactID: "/outputs",
		TargetPath: "/outputs/final/report.html",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot move root directory")
}

func TestMoveSuperAgentWorkspaceFileRenamesOnlyWritableRoots(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeRecordingSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.MoveSuperAgentWorkspaceFile(ctx, &developer_api.MoveSandboxFileRequest{
		AgentID:    agentID,
		Path:       "/workspace/draft.txt",
		TargetPath: "/uploads/final.txt",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), resp.Code)
	assert.Equal(t, "/workspace/draft.txt", resp.Data.FromPath)
	assert.Equal(t, "/uploads/final.txt", resp.Data.Path)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/uploads' && mv '/workspace/draft.txt' '/uploads/final.txt'")

	var aliasReq developer_api.MoveSandboxFileRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"from_path": "/workspace/manifest-draft.txt",
		"to_path": "/outputs/manifest-final.txt"
	}`), &aliasReq))

	resp, err = svc.MoveSuperAgentWorkspaceFile(ctx, &aliasReq)

	require.NoError(t, err)
	assert.Equal(t, "/workspace/manifest-draft.txt", resp.Data.FromPath)
	assert.Equal(t, "/outputs/manifest-final.txt", resp.Data.Path)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/outputs' && mv '/workspace/manifest-draft.txt' '/outputs/manifest-final.txt'")

	_, err = svc.MoveSuperAgentWorkspaceFile(ctx, &developer_api.MoveSandboxFileRequest{
		AgentID:    agentID,
		Path:       "/workspace/draft.txt",
		TargetPath: "/skills/draft.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")

	_, err = svc.MoveSuperAgentWorkspaceFile(ctx, &developer_api.MoveSandboxFileRequest{
		AgentID:    agentID,
		Path:       "/workspace",
		TargetPath: "/workspace/renamed",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot move root directory")
}

func TestCreateSuperAgentWorkspaceDirectoryCreatesOnlyWritableRoots(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeRecordingSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.CreateSuperAgentWorkspaceDirectory(ctx, &developer_api.CreateSandboxDirectoryRequest{
		AgentID: agentID,
		Path:    "/workspace/reports/2026",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), resp.Code)
	assert.Equal(t, "/workspace/reports/2026", resp.Data.Path)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/workspace/reports/2026'")

	_, err = svc.CreateSuperAgentWorkspaceDirectory(ctx, &developer_api.CreateSandboxDirectoryRequest{
		AgentID: agentID,
		Path:    "/skills/generated",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")

	_, err = svc.CreateSuperAgentWorkspaceDirectory(ctx, &developer_api.CreateSandboxDirectoryRequest{
		AgentID: agentID,
		Path:    "/workspace",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path must include a directory name")
}

func TestStatSuperAgentWorkspacePathReadsMetadataFromReadableRoots(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeStatSandboxManager{})
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.StatSuperAgentWorkspacePath(ctx, &developer_api.StatSandboxFileRequest{
		AgentID: agentID,
		Path:    "/skills/pdf/SKILL.md",
	})

	assert.NoError(t, err)
	assert.True(t, resp.Data.Exists)
	assert.Equal(t, "/skills/pdf/SKILL.md", resp.Data.File.Path)
	assert.Equal(t, "SKILL.md", resp.Data.File.Name)
	assert.False(t, resp.Data.File.IsDir)
	assert.Equal(t, int64(512), resp.Data.File.Size)
	assert.Equal(t, int64(1781800100), resp.Data.File.Mtime)

	missing, err := svc.StatSuperAgentWorkspacePath(ctx, &developer_api.StatSandboxFileRequest{
		AgentID: agentID,
		Path:    "/workspace/missing.md",
	})
	assert.NoError(t, err)
	assert.False(t, missing.Data.Exists)
	assert.Equal(t, "/workspace/missing.md", missing.Data.Path)
	assert.Nil(t, missing.Data.File)

	_, err = svc.StatSuperAgentWorkspacePath(ctx, &developer_api.StatSandboxFileRequest{
		AgentID: agentID,
		Path:    "/etc/passwd",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs, /skills")
}

func TestSearchSuperAgentWorkspaceUsesReadableRoots(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeSearchSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	grepResp, err := svc.GrepSuperAgentWorkspace(ctx, &developer_api.GrepSandboxFilesRequest{
		AgentID: agentID,
		Path:    "/skills",
		Pattern: "PDF",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/skills", grepResp.Data.Path)
	assert.Equal(t, "PDF", grepResp.Data.Pattern)
	assert.Contains(t, grepResp.Data.Output, "/skills/pdf/SKILL.md:1:# PDF skill")
	assert.False(t, grepResp.Data.IsTruncated)
	assert.Equal(t, "/skills", fakeSandbox.grepPath)
	assert.Equal(t, "PDF", fakeSandbox.grepPattern)

	globResp, err := svc.GlobSuperAgentWorkspace(ctx, &developer_api.GlobSandboxFilesRequest{
		AgentID: agentID,
		Path:    "/skills",
		Pattern: "SKILL.md",
		Limit:   10,
	})
	assert.NoError(t, err)
	assert.Equal(t, "/skills", globResp.Data.Path)
	assert.Equal(t, "SKILL.md", globResp.Data.Pattern)
	assert.ElementsMatch(t, []string{"/skills/pdf/SKILL.md", "/skills/docx/SKILL.md"}, globResp.Data.Matches)
	assert.Contains(t, fakeSandbox.globCmd, "find '/skills'")
	assert.Contains(t, fakeSandbox.globCmd, "-name 'SKILL.md'")

	_, err = svc.GrepSuperAgentWorkspace(ctx, &developer_api.GrepSandboxFilesRequest{
		AgentID: agentID,
		Path:    "/etc",
		Pattern: "secret",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs, /skills")
}

func TestGrepSuperAgentWorkspaceHonorsManifestFilters(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeSearchSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	var req developer_api.GrepSandboxFilesRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"path": "/skills",
		"pattern": "pdf",
		"case_sensitive": false,
		"include": ["*.md"],
		"exclude": ["draft*"]
	}`), &req))

	resp, err := svc.GrepSuperAgentWorkspace(ctx, &req)

	require.NoError(t, err)
	assert.Equal(t, "/skills", resp.Data.Path)
	assert.Equal(t, "pdf", resp.Data.Pattern)
	assert.Contains(t, resp.Data.Output, "/skills/pdf/SKILL.md:1:# PDF skill")
	require.NotEmpty(t, fakeSandbox.grepCmd)
	assert.Contains(t, fakeSandbox.grepCmd, "'rg' '-n' '--no-heading' '-i'")
	assert.Contains(t, fakeSandbox.grepCmd, "'-g' '*.md'")
	assert.Contains(t, fakeSandbox.grepCmd, "'-g' '!draft*'")
	assert.Contains(t, fakeSandbox.grepCmd, "'--' 'pdf' '/skills'")
}

func TestEditSuperAgentWorkspaceFileUsesWritableRoots(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeEditSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.EditSuperAgentWorkspaceFile(ctx, &developer_api.EditSandboxFileRequest{
		AgentID:    agentID,
		Path:       "/workspace/app/main.ts",
		OldString:  "oldCall()",
		NewString:  "newCall()",
		ReplaceAll: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, "/workspace/app/main.ts", resp.Data.Path)
	assert.Equal(t, int32(2), resp.Data.Replacements)
	assert.Equal(t, "/workspace/app/main.ts", fakeSandbox.path)
	assert.Equal(t, "oldCall()", fakeSandbox.oldString)
	assert.Equal(t, "newCall()", fakeSandbox.newString)
	assert.True(t, fakeSandbox.replaceAll)

	_, err = svc.EditSuperAgentWorkspaceFile(ctx, &developer_api.EditSandboxFileRequest{
		AgentID:   agentID,
		Path:      "/skills/pdf/SKILL.md",
		OldString: "old",
		NewString: "new",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")
}

func TestApplySuperAgentWorkspacePatchMovesFiles(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakePatchSandboxManager{
		files: map[string]string{
			"/workspace/draft.txt": "draft\n",
		},
	}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ApplySuperAgentWorkspacePatch(ctx, &developer_api.ApplySandboxPatchRequest{
		AgentID: agentID,
		WorkDir: "/workspace",
		Patch: `*** Begin Patch
*** Update File: draft.txt
*** Move to: final/report.txt
*** End Patch
`,
	})

	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, int32(1), resp.Data.ChangedFiles)
	assert.ElementsMatch(t, []string{"/workspace/final/report.txt"}, resp.Data.Paths)
	assert.Equal(t, "draft\n", fakeSandbox.files["/workspace/final/report.txt"])
	_, stillOld := fakeSandbox.files["/workspace/draft.txt"]
	assert.False(t, stillOld)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/workspace/final' && mv '/workspace/draft.txt' '/workspace/final/report.txt'")

	resp, err = svc.ApplySuperAgentWorkspacePatch(ctx, &developer_api.ApplySandboxPatchRequest{
		AgentID:      agentID,
		WorkDirAlias: "/outputs",
		Patch: `*** Begin Patch
*** Add File: reports/alias.txt
+alias
*** End Patch
`,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Data.ChangedFiles)
	assert.ElementsMatch(t, []string{"/outputs/reports/alias.txt"}, resp.Data.Paths)
	assert.Equal(t, "alias\n", fakeSandbox.files["/outputs/reports/alias.txt"])

	_, err = svc.ApplySuperAgentWorkspacePatch(ctx, &developer_api.ApplySandboxPatchRequest{
		AgentID: agentID,
		WorkDir: "/workspace",
		Patch: `*** Begin Patch
*** Update File: final/report.txt
*** Move to: /skills/report.txt
*** End Patch
`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")
}

func TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeExecSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.RunSuperAgentSandboxCommand(ctx, &developer_api.ExecSandboxCommandRequest{
		AgentID:    agentID,
		Command:    "npm test",
		WorkDir:    "/workspace/project",
		TimeoutSec: 12,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(7), resp.Data.ExitCode)
	assert.Equal(t, "ok\n", resp.Data.Stdout)
	assert.Equal(t, "warn\n", resp.Data.Stderr)
	assert.False(t, resp.Data.IsTruncated)
	assert.Equal(t, "cd '/workspace/project' && npm test", fakeSandbox.cmd)
	assert.Equal(t, 12, fakeSandbox.timeoutSec)

	resp, err = svc.RunSuperAgentSandboxCommand(ctx, &developer_api.ExecSandboxCommandRequest{
		AgentID:      agentID,
		Command:      "pwd",
		WorkDirAlias: "/outputs",
	})
	assert.NoError(t, err)
	assert.Equal(t, "cd '/outputs' && pwd", fakeSandbox.cmd)

	_, err = svc.RunSuperAgentSandboxCommand(ctx, &developer_api.ExecSandboxCommandRequest{
		AgentID: agentID,
		Command: "npm test",
		WorkDir: "/etc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")

	_, err = svc.RunSuperAgentSandboxCommand(ctx, &developer_api.ExecSandboxCommandRequest{
		AgentID: agentID,
		Command: "python scripts/run.py",
		WorkDir: "/skills/pdf",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")

	_, err = svc.RunSuperAgentSandboxCommand(ctx, &developer_api.ExecSandboxCommandRequest{
		AgentID: agentID,
		Command: " ",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

func TestUpdateSuperAgentHarnessPlanWritesToolCompatiblePlan(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakePlanSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.UpdateSuperAgentHarnessPlan(ctx, &developer_api.SuperAgentHarnessPlanUpdateRequest{
		AgentID: agentID,
		Plan: []*developer_api.SuperAgentHarnessPlanStep{
			{Content: "梳理需求", Status: "completed"},
			{Content: "实现接口", Status: "in_progress"},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "/workspace/.plan.json", resp.Data.Path)
	assert.Equal(t, int32(2), resp.Data.Steps)
	assert.Equal(t, "/workspace/.plan.json", fakeSandbox.path)
	assert.JSONEq(t, `[
  {"content":"梳理需求","status":"completed"},
  {"content":"实现接口","status":"in_progress"}
]`, fakeSandbox.content)

	var itemsReq developer_api.SuperAgentHarnessPlanUpdateRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"items": [
			{"content": "按 manifest 字段更新计划", "status": "completed"}
		]
	}`), &itemsReq))

	resp, err = svc.UpdateSuperAgentHarnessPlan(ctx, &itemsReq)

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Data.Steps)
	assert.JSONEq(t, `[
  {"content":"按 manifest 字段更新计划","status":"completed"}
]`, fakeSandbox.content)

	_, err = svc.UpdateSuperAgentHarnessPlan(ctx, &developer_api.SuperAgentHarnessPlanUpdateRequest{
		AgentID: agentID,
		Plan:    []*developer_api.SuperAgentHarnessPlanStep{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan must contain at least one step")

	_, err = svc.UpdateSuperAgentHarnessPlan(ctx, &developer_api.SuperAgentHarnessPlanUpdateRequest{
		AgentID: agentID,
		Plan: []*developer_api.SuperAgentHarnessPlanStep{
			{Content: "坏状态", Status: "blocked"},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plan status")
}

func TestUpdateSuperAgentHarnessPlanWritesConversationScopedPlan(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakePlanSandboxManager{}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	var req developer_api.SuperAgentHarnessPlanUpdateRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"agent_id": "123",
		"conversation_id": "456",
		"items": [
			{"content": "隔离会话计划", "status": "in_progress"}
		]
	}`), &req))

	resp, err := svc.UpdateSuperAgentHarnessPlan(ctx, &req)

	require.NoError(t, err)
	assert.Equal(t, "/workspace/.agent/sessions/456/plan.json", resp.Data.Path)
	assert.Equal(t, "/workspace/.agent/sessions/456/plan.json", fakeSandbox.path)
	assert.JSONEq(t, `[
  {"content":"隔离会话计划","status":"in_progress"}
]`, fakeSandbox.content)
}

func TestApplySuperAgentWorkspacePatchAppliesCodexPatch(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakePatchSandboxManager{
		files: map[string]string{
			"/workspace/app.ts": "console.log(\"old\")\n",
			"/outputs/old.txt":  "obsolete\n",
		},
	}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   1,
				},
			},
		},
	}

	resp, err := svc.ApplySuperAgentWorkspacePatch(ctx, &developer_api.ApplySandboxPatchRequest{
		AgentID: agentID,
		WorkDir: "/workspace",
		Patch: `*** Begin Patch
*** Add File: reports/summary.md
+# Summary
+
+done
*** Update File: app.ts
@@
-console.log("old")
+console.log("new")
*** Delete File: /outputs/old.txt
*** End Patch
`,
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(3), resp.Data.ChangedFiles)
	assert.ElementsMatch(t, []string{"/workspace/reports/summary.md", "/workspace/app.ts", "/outputs/old.txt"}, resp.Data.Paths)
	assert.Equal(t, "# Summary\n\ndone\n", fakeSandbox.files["/workspace/reports/summary.md"])
	assert.Equal(t, "console.log(\"new\")\n", fakeSandbox.files["/workspace/app.ts"])
	_, deleted := fakeSandbox.files["/outputs/old.txt"]
	assert.False(t, deleted)
	assert.Contains(t, fakeSandbox.execs, "mkdir -p '/workspace/reports'")
	assert.Contains(t, fakeSandbox.execs, "rm -rf '/outputs/old.txt'")

	_, err = svc.ApplySuperAgentWorkspacePatch(ctx, &developer_api.ApplySandboxPatchRequest{
		AgentID: agentID,
		WorkDir: "/workspace",
		Patch: `*** Begin Patch
*** Add File: /skills/pwn.md
+nope
*** End Patch
`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/workspace, /uploads, /outputs")
}
