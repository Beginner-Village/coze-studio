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

// 超级智能体「沙箱空间管理」应用服务:列目录 / 读文件 / 上传 / 删除。
// 沙箱 key 必须与 agent 运行时(agentflow.sandboxKeyFor)一致 —— 都用
// agentsandbox.SandboxKeyFor(connectorID, agentID, userID),其中 connectorID 默认取
// 调试预览连接器 CozeConnectorID,这样管理界面看到的就是 agent 调试时实际用的沙箱。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// 允许管理的沙箱根目录(契约目录)。
var sandboxAllowedRoots = []string{"/workspace", "/uploads", "/outputs"}
var sandboxSuperAgentReadableRoots = []string{"/workspace", "/uploads", "/outputs", "/skills"}

const sandboxReadMaxBytes = 25 * 1024 * 1024 // 单文件读取上限 25MB(覆盖绝大多数生成的 PPT/文档)

const (
	superAgentHarnessPlanPath               = "/workspace/.plan.json"
	superAgentHarnessContextPath            = "/workspace/.agent/context-summary.json"
	superAgentHarnessToolOutputRoot         = "/workspace/.agent/tooloutputs"
	superAgentHarnessContextStrategy        = "tool-output-offload+session-summary"
	superAgentHarnessSummaryVersion         = "v1"
	superAgentHarnessCompactTrigger         = "history_bytes_exceeded"
	superAgentHarnessSummaryTemplate        = "/workspace/.agent/sessions/{conversation_id}/context-summary.json"
	superAgentHarnessContextClearRoute      = "POST /api/super-agent/harness/context/clear"
	superAgentHarnessDefaultCompactMaxBytes = 160 * 1024
	superAgentHarnessDefaultRecentMessages  = 16
	superAgentHarnessSummaryMaxRunes        = 1600
	superAgentSearchMaxOutputBytes          = 64 * 1024
	superAgentGlobMaxMatches                = int32(500)
	superAgentExecDefaultTimeoutSec         = 60
	superAgentExecMaxTimeoutSec             = 300
)

// purgeAgentSandboxes 删除某智能体名下「所有用户/所有连接器」的沙箱容器及宿主机数据目录。
// 在删除智能体时调用,避免容器与持久化目录成为孤儿。失败仅告警,不阻断智能体删除。
func purgeAgentSandboxes(ctx context.Context, agentID int64) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return
	}
	pm, ok := svc.(interface {
		PurgeAgent(ctx context.Context, agentID int64) error
	})
	if !ok {
		return
	}
	if err := pm.PurgeAgent(ctx, agentID); err != nil {
		logs.CtxWarnf(ctx, "[purgeAgentSandboxes] purge agent %d sandboxes failed: %v", agentID, err)
	}
}

// resolveSandboxKey 校验访问权限并派生沙箱 key(与运行时一致)。
func (s *SingleAgentApplicationService) resolveSandboxKey(ctx context.Context, agentID int64, connectorIDStr *string) (crosssandbox.Manager, string, error) {
	if _, err := s.ValidateAgentDraftAccess(ctx, agentID); err != nil {
		return nil, "", err
	}
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return nil, "", errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "sandbox service is not available"))
	}
	connectorID := consts.CozeConnectorID
	if connectorIDStr != nil && *connectorIDStr != "" {
		if id, err := conv.StrToInt64(*connectorIDStr); err == nil && id != 0 {
			connectorID = id
		}
	}
	userKey, err := sandboxWorkspaceUserKey(ctx)
	if err != nil {
		return nil, "", err
	}
	key := agentsandbox.SandboxKeyFor(connectorID, agentID, userKey)
	return svc, key, nil
}

func sandboxWorkspaceUserKey(ctx context.Context) (string, error) {
	if uid := ctxutil.GetUIDFromCtx(ctx); uid != nil {
		return strconv.FormatInt(*uid, 10), nil
	}
	apiAuth := ctxutil.GetApiAuthFromCtx(ctx)
	if apiAuth != nil && apiAuth.UserID != 0 {
		return fmt.Sprintf("api-user-%d", apiAuth.UserID), nil
	}
	return "", errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "session or api auth required"))
}

// sanitizeSandboxPathWithRoots 把入参规整为绝对路径并限制在指定根目录内,防目录穿越。
func sanitizeSandboxPathWithRoots(p string, roots []string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = roots[0]
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("path must not contain '..'")
	}
	for _, root := range roots {
		if p == root || strings.HasPrefix(p, root+"/") {
			return p, nil
		}
	}
	return "", fmt.Errorf("path must be under %s", strings.Join(roots, ", "))
}

// sanitizeSandboxPath 把入参规整为绝对路径并限制在可写沙箱根目录内。
func sanitizeSandboxPath(p string) (string, error) {
	return sanitizeSandboxPathWithRoots(p, sandboxAllowedRoots)
}

// sanitizeSuperAgentReadablePath 允许超级体 App Server 读取 /skills,但写入/删除仍走可写根目录策略。
func sanitizeSuperAgentReadablePath(p string) (string, error) {
	return sanitizeSandboxPathWithRoots(p, sandboxSuperAgentReadableRoots)
}

func sanitizeSuperAgentHarnessToolOutputPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return superAgentHarnessToolOutputRoot, nil
	}
	if !strings.HasPrefix(p, "/") {
		switch {
		case p == "workspace/.agent/tooloutputs" || strings.HasPrefix(p, "workspace/.agent/tooloutputs/"):
			p = "/" + p
		case p == ".agent/tooloutputs" || strings.HasPrefix(p, ".agent/tooloutputs/"):
			p = "/workspace/" + p
		default:
			p = strings.TrimRight(superAgentHarnessToolOutputRoot, "/") + "/" + p
		}
	}
	return sanitizeSandboxPathWithRoots(p, []string{superAgentHarnessToolOutputRoot})
}

func superAgentHarnessToolOutputDefaultPath(conversationID int64) string {
	if conversationID > 0 {
		return fmt.Sprintf("%s/sessions/%d", superAgentHarnessToolOutputRoot, conversationID)
	}
	return superAgentHarnessToolOutputRoot
}

func superAgentHarnessToolOutputRequestedPath(path string, conversationID int64) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	return superAgentHarnessToolOutputDefaultPath(conversationID)
}

func superAgentHarnessToolOutputListPath(requestPath string, conversationID int64, outputPath string) bool {
	if outputPath == superAgentHarnessToolOutputRoot {
		return true
	}
	return strings.TrimSpace(requestPath) == "" && conversationID > 0
}

type sandboxPathSanitizer func(string) (string, error)

type SuperAgentHarnessToolOutputsRequest struct {
	SpaceID        int64   `json:"space_id,string,omitempty"`
	AgentID        int64   `json:"agent_id,string,omitempty"`
	BotID          int64   `json:"bot_id,string,omitempty"`
	ConversationID int64   `json:"conversation_id,string,omitempty"`
	ConnectorID    *string `json:"connector_id,omitempty"`
	Path           string  `json:"path,omitempty"`
	Recursive      bool    `json:"recursive,omitempty"`
	IncludeContent bool    `json:"include_content,omitempty"`
}

type SuperAgentHarnessToolOutputsResponse struct {
	Code int64                             `json:"code"`
	Msg  string                            `json:"msg"`
	Data *SuperAgentHarnessToolOutputsData `json:"data"`
}

type SuperAgentHarnessToolOutputsData struct {
	Root     string                                        `json:"root"`
	Path     string                                        `json:"path"`
	Files    []*developer_api.SandboxFileInfo              `json:"files"`
	Entries  []*SuperAgentHarnessToolOutputEntry           `json:"entries,omitempty"`
	Content  *developer_api.ReadSandboxFileData            `json:"content,omitempty"`
	Contents map[string]*developer_api.ReadSandboxFileData `json:"contents,omitempty"`
}

type SuperAgentHarnessToolOutputEntry struct {
	Path             string `json:"path"`
	Name             string `json:"name"`
	ToolCallID       string `json:"tool_call_id,omitempty"`
	Tool             string `json:"tool,omitempty"`
	Status           string `json:"status,omitempty"`
	Arguments        any    `json:"arguments,omitempty"`
	Result           any    `json:"result,omitempty"`
	Error            any    `json:"error,omitempty"`
	ArgumentsPreview string `json:"arguments_preview,omitempty"`
	ResultPreview    string `json:"result_preview,omitempty"`
	Summary          string `json:"summary,omitempty"`
	Size             int64  `json:"size,omitempty"`
	Mtime            int64  `json:"mtime,omitempty"`
}

type SuperAgentHarnessCleanupRequest struct {
	SpaceID     int64    `json:"space_id,string,omitempty"`
	AgentID     int64    `json:"agent_id,string,omitempty"`
	BotID       int64    `json:"bot_id,string,omitempty"`
	ConnectorID *string  `json:"connector_id,omitempty"`
	Paths       []string `json:"paths,omitempty"`
	KeepLatest  int32    `json:"keep_latest,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	DryRun      bool     `json:"dry_run,omitempty"`
}

type SuperAgentHarnessCleanupResponse struct {
	Code int64                         `json:"code"`
	Msg  string                        `json:"msg"`
	Data *SuperAgentHarnessCleanupData `json:"data"`
}

type SuperAgentHarnessCleanupData struct {
	Root    string   `json:"root"`
	Prefix  string   `json:"prefix,omitempty"`
	DryRun  bool     `json:"dry_run"`
	Matched int32    `json:"matched"`
	Deleted int32    `json:"deleted"`
	Paths   []string `json:"paths"`
}

func sandboxRequestAgentID(botID, agentID int64) int64 {
	if agentID != 0 {
		return agentID
	}
	return botID
}

func (s *SingleAgentApplicationService) listSandboxFilesWithSanitizer(ctx context.Context, req *developer_api.ListSandboxFilesRequest, sanitize sandboxPathSanitizer, ensureDirs string) (*developer_api.ListSandboxFilesResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitize(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	// 确保契约目录存在(沙箱可能是冷启动)。
	_, _ = svc.Exec(ctx, key, ensureDirs, 20)

	return s.listSandboxFilesByPath(ctx, svc, key, path, req.Recursive)
}

func (s *SingleAgentApplicationService) listSandboxFilesByPath(ctx context.Context, svc crosssandbox.Manager, key string, path string, recursive bool) (*developer_api.ListSandboxFilesResponse, error) {
	// 用 python3 输出 JSON,避免解析 ls 的脆弱性(slim 镜像自带 python3)。
	py := sandboxListFilesScript(recursive)
	cmd := fmt.Sprintf("python3 -c %s %s", shellQuote(py), shellQuote(path))
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("list failed: %v", err)))
	}
	files := make([]*developer_api.SandboxFileInfo, 0)
	stdout := strings.TrimSpace(res.Stdout)
	if stdout != "" {
		var raw []struct {
			Path  string `json:"path"`
			Name  string `json:"name"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size"`
			Mtime int64  `json:"mtime"`
		}
		if jErr := json.Unmarshal([]byte(stdout), &raw); jErr == nil {
			for _, r := range raw {
				fp := strings.TrimRight(path, "/") + "/" + r.Name
				if strings.TrimSpace(r.Path) != "" {
					fp = r.Path
				}
				files = append(files, &developer_api.SandboxFileInfo{
					Name:  r.Name,
					Path:  fp,
					IsDir: r.IsDir,
					Size:  r.Size,
					Mtime: r.Mtime,
				})
			}
		}
	}
	return &developer_api.ListSandboxFilesResponse{
		Code: 0,
		Data: &developer_api.ListSandboxFilesData{Path: path, Files: files},
	}, nil
}

func sandboxListFilesScript(recursive bool) string {
	if recursive {
		return "import os,json,sys\n" +
			"p=sys.argv[1]\n" +
			"out=[]\n" +
			"for root,dirs,files in os.walk(p):\n" +
			"  dirs[:]=sorted(dirs)\n" +
			"  for n in dirs:\n" +
			"    fp=os.path.join(root,n)\n" +
			"    try:\n" +
			"      st=os.stat(fp)\n" +
			"      out.append({'name':n,'path':fp,'is_dir':True,'size':0,'mtime':int(st.st_mtime)})\n" +
			"    except OSError: pass\n" +
			"  for n in sorted(files):\n" +
			"    fp=os.path.join(root,n)\n" +
			"    try:\n" +
			"      st=os.stat(fp)\n" +
			"      out.append({'name':n,'path':fp,'is_dir':False,'size':st.st_size,'mtime':int(st.st_mtime)})\n" +
			"    except OSError: pass\n" +
			"print(json.dumps(out))"
	}
	return "import os,json,sys\n" +
		"p=sys.argv[1]\n" +
		"out=[]\n" +
		"for n in sorted(os.listdir(p)):\n" +
		"  fp=os.path.join(p,n)\n" +
		"  try:\n" +
		"    st=os.stat(fp)\n" +
		"    out.append({'name':n,'is_dir':os.path.isdir(fp),'size':(0 if os.path.isdir(fp) else st.st_size),'mtime':int(st.st_mtime)})\n" +
		"  except OSError: pass\n" +
		"print(json.dumps(out))"
}

// ListSandboxFiles 列出指定目录下的文件/子目录。
func (s *SingleAgentApplicationService) ListSandboxFiles(ctx context.Context, req *developer_api.ListSandboxFilesRequest) (*developer_api.ListSandboxFilesResponse, error) {
	return s.listSandboxFilesWithSanitizer(ctx, req, sanitizeSandboxPath, "mkdir -p /workspace /uploads /outputs")
}

// ReadSandboxFile 读取单个文件内容(文本直出,二进制 base64)。
func (s *SingleAgentApplicationService) ReadSandboxFile(ctx context.Context, req *developer_api.ReadSandboxFileRequest) (*developer_api.ReadSandboxFileResponse, error) {
	return s.readSandboxFileWithSanitizer(ctx, req, sanitizeSandboxPath)
}

func (s *SingleAgentApplicationService) readSandboxFileWithSanitizer(ctx context.Context, req *developer_api.ReadSandboxFileRequest, sanitize sandboxPathSanitizer) (*developer_api.ReadSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitize(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	return s.readSandboxFileByPath(ctx, svc, key, path)
}

func (s *SingleAgentApplicationService) readSandboxFileByPath(ctx context.Context, svc crosssandbox.Manager, key string, path string) (*developer_api.ReadSandboxFileResponse, error) {
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("read failed: %v", err)))
	}
	data := sandboxReadFileDataFromBytes(path, b)
	return &developer_api.ReadSandboxFileResponse{Code: 0, Data: data}, nil
}

func sandboxReadFileDataFromBytes(path string, b []byte) *developer_api.ReadSandboxFileData {
	total := int64(len(b))
	isText := utf8.Valid(b)
	data := &developer_api.ReadSandboxFileData{Path: path, TotalSize: total}
	if total > sandboxReadMaxBytes {
		data.IsTruncated = true
		if isText {
			// 文本可截断预览
			b = b[:sandboxReadMaxBytes]
			data.Content = string(b)
			data.IsBinary = false
			data.Size = int64(len(b))
		} else {
			// 二进制超限时不回传 Content,前端提示下载。
			data.IsBinary = true
			data.Size = 0
		}
		return data
	}
	data.Size = total
	if isText {
		data.Content = string(b)
		data.IsBinary = false
	} else {
		data.Content = base64.StdEncoding.EncodeToString(b)
		data.IsBinary = true
	}
	return data
}

// ListSuperAgentWorkspaceFiles 列出超级体 App Server 可见工作区,包含只读 /skills。
func (s *SingleAgentApplicationService) ListSuperAgentWorkspaceFiles(ctx context.Context, req *developer_api.ListSandboxFilesRequest) (*developer_api.ListSandboxFilesResponse, error) {
	return s.listSandboxFilesWithSanitizer(ctx, req, sanitizeSuperAgentReadablePath, "mkdir -p /workspace /uploads /outputs /skills")
}

func (s *SingleAgentApplicationService) GetSuperAgentHarnessState(ctx context.Context, req *developer_api.SuperAgentHarnessStateRequest) (*developer_api.SuperAgentHarnessStateResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}

	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills /workspace/.agent/tooloutputs", 20)

	planPath := superAgentHarnessPlanPathForConversation(req.ConversationID)
	plan := &developer_api.SuperAgentHarnessPlanState{
		Path:   planPath,
		Exists: false,
	}
	if b, readErr := svc.ReadFile(ctx, key, planPath); readErr == nil {
		fileData := sandboxReadFileDataFromBytes(planPath, b)
		plan.Exists = true
		plan.Content = fileData.Content
		plan.IsBinary = fileData.IsBinary
		plan.Size = fileData.Size
		plan.TotalSize = fileData.TotalSize
		plan.IsTruncated = fileData.IsTruncated
		if fileInfo := sandboxStatFileInfo(ctx, svc, key, planPath); fileInfo != nil {
			plan.Mtime = fileInfo.Mtime
			plan.Size = fileInfo.Size
		}
	}

	toolOutputPath := superAgentHarnessToolOutputDefaultPath(req.ConversationID)
	toolOutputs := &developer_api.SuperAgentHarnessToolOutputsState{
		Root:  superAgentHarnessToolOutputRoot,
		Path:  toolOutputPath,
		Files: []*developer_api.SandboxFileInfo{},
	}
	if listResp, listErr := s.listSandboxFilesByPath(ctx, svc, key, toolOutputPath, false); listErr == nil && listResp.Data != nil {
		toolOutputs.Files = listResp.Data.Files
	}

	runtimeSkills := s.listSuperAgentHarnessRuntimeSkills(ctx, svc, key)
	contextState := s.superAgentHarnessContextState(ctx, svc, key, req.ConversationID)

	return &developer_api.SuperAgentHarnessStateResponse{
		Code: 0,
		Data: &developer_api.SuperAgentHarnessStateData{
			Plan:          plan,
			ToolOutputs:   toolOutputs,
			RuntimeSkills: runtimeSkills,
			Context:       contextState,
		},
	}, nil
}

func sandboxStatFileInfo(ctx context.Context, svc crosssandbox.Manager, key, p string) *developer_api.SandboxFileInfo {
	if svc == nil || strings.TrimSpace(p) == "" {
		return nil
	}
	py := "import os,json,sys\n" +
		"p=sys.argv[1]\n" +
		"exists=os.path.exists(p)\n" +
		"name=os.path.basename(p.rstrip('/')) or p\n" +
		"out={'exists':exists,'name':name,'path':p,'is_dir':False,'size':0,'mtime':0}\n" +
		"if exists:\n" +
		"  st=os.stat(p)\n" +
		"  out.update({'is_dir':os.path.isdir(p),'size':(0 if os.path.isdir(p) else st.st_size),'mtime':int(st.st_mtime)})\n" +
		"print(json.dumps(out, ensure_ascii=False))"
	res, err := svc.Exec(ctx, key, fmt.Sprintf("python3 -c %s %s", shellQuote(py), shellQuote(p)), 20)
	if err != nil {
		return nil
	}
	var raw struct {
		Exists bool   `json:"exists"`
		Name   string `json:"name"`
		Path   string `json:"path"`
		IsDir  bool   `json:"is_dir"`
		Size   int64  `json:"size"`
		Mtime  int64  `json:"mtime"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &raw); err != nil || !raw.Exists {
		return nil
	}
	return &developer_api.SandboxFileInfo{
		Name:  raw.Name,
		Path:  raw.Path,
		IsDir: raw.IsDir,
		Size:  raw.Size,
		Mtime: raw.Mtime,
	}
}

func (s *SingleAgentApplicationService) GetSuperAgentHarnessContext(ctx context.Context, req *developer_api.SuperAgentHarnessStateRequest) (*developer_api.SuperAgentHarnessContextState, error) {
	if req == nil {
		req = &developer_api.SuperAgentHarnessStateRequest{}
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace/.agent/tooloutputs", 20)
	return s.superAgentHarnessContextState(ctx, svc, key, req.ConversationID), nil
}

func (s *SingleAgentApplicationService) ClearSuperAgentHarnessContext(ctx context.Context, req *developer_api.SuperAgentHarnessContextClearRequest) (*developer_api.SuperAgentHarnessContextClearResponse, error) {
	if req == nil {
		req = &developer_api.SuperAgentHarnessContextClearRequest{}
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	summaryPath := superAgentHarnessContextPathForConversation(req.ConversationID)
	summaryExistsBefore := false
	if _, readErr := svc.ReadFile(ctx, key, summaryPath); readErr == nil {
		summaryExistsBefore = true
	}
	cleared := false
	if !req.DryRun {
		if _, err := svc.Exec(ctx, key, fmt.Sprintf("rm -f %s", shellQuote(summaryPath)), 20); err != nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("clear context failed: %v", err)))
		}
		cleared = true
	}
	data := &developer_api.SuperAgentHarnessContextClearData{
		SummaryPath:         summaryPath,
		SummaryExistsBefore: summaryExistsBefore,
		Cleared:             cleared,
		DryRun:              req.DryRun,
	}
	if req.ConversationID > 0 {
		data.ConversationID = strconv.FormatInt(req.ConversationID, 10)
	}
	return &developer_api.SuperAgentHarnessContextClearResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	}, nil
}

func (s *SingleAgentApplicationService) superAgentHarnessContextState(ctx context.Context, svc crosssandbox.Manager, key string, conversationID int64) *developer_api.SuperAgentHarnessContextState {
	summaryPath := superAgentHarnessContextPathForConversation(conversationID)
	state := &developer_api.SuperAgentHarnessContextState{
		Strategy:             superAgentHarnessContextStrategy,
		SummaryPath:          summaryPath,
		SummaryExists:        false,
		Policy:               superAgentHarnessContextPolicy(),
		ToolOutputRoot:       superAgentHarnessToolOutputRoot,
		ToolOutputOffload:    true,
		RecentMessagesPolicy: "message_limit+recent_tail",
		Components:           []string{"summary", "tool_outputs", "recent_messages"},
	}
	if svc == nil {
		return state
	}
	b, err := svc.ReadFile(ctx, key, summaryPath)
	if err != nil || strings.TrimSpace(string(b)) == "" {
		return state
	}
	state.SummaryExists = true
	summary := &developer_api.SuperAgentHarnessContextSummary{}
	if err := json.Unmarshal(b, summary); err == nil {
		state.Summary = summary
	}
	return state
}

func superAgentHarnessContextPolicy() *developer_api.SuperAgentHarnessContextPolicy {
	return &developer_api.SuperAgentHarnessContextPolicy{
		Strategy:                   superAgentHarnessContextStrategy,
		SummaryVersion:             superAgentHarnessSummaryVersion,
		Trigger:                    superAgentHarnessCompactTrigger,
		MaxBytes:                   superAgentHarnessEnvInt("AGENT_CONTEXT_COMPACT_MAX_BYTES", superAgentHarnessDefaultCompactMaxBytes),
		RecentMessages:             superAgentHarnessEnvInt("AGENT_CONTEXT_COMPACT_RECENT_MESSAGES", superAgentHarnessDefaultRecentMessages),
		SummaryMaxRunes:            superAgentHarnessSummaryMaxRunes,
		SummaryPath:                superAgentHarnessContextPath,
		SessionSummaryPathTemplate: superAgentHarnessSummaryTemplate,
		ClearRoute:                 superAgentHarnessContextClearRoute,
	}
}

func superAgentHarnessEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func superAgentHarnessContextPathForConversation(conversationID int64) string {
	if conversationID > 0 {
		return fmt.Sprintf("/workspace/.agent/sessions/%d/context-summary.json", conversationID)
	}
	return superAgentHarnessContextPath
}

func superAgentHarnessPlanPathForConversation(conversationID int64) string {
	if conversationID > 0 {
		return fmt.Sprintf("/workspace/.agent/sessions/%d/plan.json", conversationID)
	}
	return superAgentHarnessPlanPath
}

func (s *SingleAgentApplicationService) ListSuperAgentHarnessToolOutputs(ctx context.Context, req *SuperAgentHarnessToolOutputsRequest) (*SuperAgentHarnessToolOutputsResponse, error) {
	if req == nil {
		req = &SuperAgentHarnessToolOutputsRequest{}
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	outputPath, err := sanitizeSuperAgentHarnessToolOutputPath(superAgentHarnessToolOutputRequestedPath(req.Path, req.ConversationID))
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace/.agent/tooloutputs", 20)
	if strings.TrimSpace(req.Path) == "" && req.ConversationID > 0 {
		_, _ = svc.Exec(ctx, key, fmt.Sprintf("mkdir -p %s", shellQuote(outputPath)), 20)
	}

	data := &SuperAgentHarnessToolOutputsData{
		Root:  superAgentHarnessToolOutputRoot,
		Path:  outputPath,
		Files: []*developer_api.SandboxFileInfo{},
	}
	listOutputPath := superAgentHarnessToolOutputListPath(req.Path, req.ConversationID, outputPath)
	if listOutputPath {
		listResp, err := s.listSandboxFilesByPath(ctx, svc, key, outputPath, req.Recursive)
		if err != nil {
			return nil, err
		}
		if listResp.Data != nil {
			data.Files = listResp.Data.Files
		}
	}
	if req.IncludeContent && !listOutputPath {
		readResp, err := s.readSandboxFileByPath(ctx, svc, key, outputPath)
		if err != nil {
			return nil, err
		}
		data.Content = readResp.Data
		if readResp.Data != nil {
			data.Contents = map[string]*developer_api.ReadSandboxFileData{
				outputPath: readResp.Data,
			}
		}
		data.Entries = append(data.Entries, superAgentHarnessToolOutputEntryFromContent(outputPath, nil, readResp.Data))
	}
	if listOutputPath {
		if req.IncludeContent {
			data.Contents = map[string]*developer_api.ReadSandboxFileData{}
		}
		for _, file := range data.Files {
			if file == nil || file.IsDir || strings.TrimSpace(file.Path) == "" {
				continue
			}
			readResp, err := s.readSandboxFileByPath(ctx, svc, key, file.Path)
			if err != nil {
				if req.IncludeContent {
					return nil, err
				}
				data.Entries = append(data.Entries, superAgentHarnessToolOutputEntryFromContent(file.Path, file, nil))
				continue
			}
			if req.IncludeContent && readResp.Data != nil {
				data.Contents[file.Path] = readResp.Data
			}
			data.Entries = append(data.Entries, superAgentHarnessToolOutputEntryFromContent(file.Path, file, readResp.Data))
		}
	}

	return &SuperAgentHarnessToolOutputsResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	}, nil
}

func (s *SingleAgentApplicationService) CleanupSuperAgentHarnessToolOutputs(ctx context.Context, req *SuperAgentHarnessCleanupRequest) (*SuperAgentHarnessCleanupResponse, error) {
	if req == nil {
		req = &SuperAgentHarnessCleanupRequest{}
	}
	if req.KeepLatest < 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "keep_latest must be greater than or equal to 0"))
	}
	if len(req.Paths) == 0 && req.KeepLatest <= 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "paths or keep_latest is required"))
	}
	prefix, err := sanitizeSuperAgentHarnessToolOutputPrefix(req.Prefix)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace/.agent/tooloutputs", 20)

	paths := make([]string, 0, len(req.Paths))
	for _, rawPath := range req.Paths {
		cleanPath, err := sanitizeSuperAgentHarnessToolOutputPath(rawPath)
		if err != nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
		}
		if cleanPath == superAgentHarnessToolOutputRoot {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "path must point to a tool output file"))
		}
		paths = append(paths, cleanPath)
	}
	if req.KeepLatest > 0 {
		listResp, err := s.listSandboxFilesByPath(ctx, svc, key, superAgentHarnessToolOutputRoot, false)
		if err != nil {
			return nil, err
		}
		files := []*developer_api.SandboxFileInfo{}
		if listResp.Data != nil {
			files = make([]*developer_api.SandboxFileInfo, 0, len(listResp.Data.Files))
			for _, file := range listResp.Data.Files {
				if file != nil && !file.IsDir && strings.TrimSpace(file.Path) != "" &&
					(prefix == "" || strings.HasPrefix(file.Path, prefix)) {
					files = append(files, file)
				}
			}
		}
		sort.SliceStable(files, func(i, j int) bool {
			if files[i].Mtime == files[j].Mtime {
				return files[i].Path > files[j].Path
			}
			return files[i].Mtime > files[j].Mtime
		})
		if int(req.KeepLatest) < len(files) {
			candidates := files[req.KeepLatest:]
			sort.SliceStable(candidates, func(i, j int) bool {
				if candidates[i].Mtime == candidates[j].Mtime {
					return candidates[i].Path < candidates[j].Path
				}
				return candidates[i].Mtime < candidates[j].Mtime
			})
			for _, file := range candidates {
				cleanPath, err := sanitizeSuperAgentHarnessToolOutputPath(file.Path)
				if err != nil {
					return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
				}
				paths = append(paths, cleanPath)
			}
		}
	}
	paths = uniqueSuperAgentHarnessCleanupPaths(paths)

	data := &SuperAgentHarnessCleanupData{
		Root:    superAgentHarnessToolOutputRoot,
		Prefix:  prefix,
		DryRun:  req.DryRun,
		Matched: int32(len(paths)),
		Paths:   paths,
	}
	if req.DryRun {
		return &SuperAgentHarnessCleanupResponse{Code: 0, Msg: "success", Data: data}, nil
	}
	for _, cleanPath := range paths {
		if _, err := svc.Exec(ctx, key, "rm -f -- "+shellQuote(cleanPath), 20); err != nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("cleanup failed: %v", err)))
		}
		data.Deleted++
	}
	return &SuperAgentHarnessCleanupResponse{Code: 0, Msg: "success", Data: data}, nil
}

func sanitizeSuperAgentHarnessToolOutputPrefix(raw string) (string, error) {
	prefix := strings.TrimSpace(raw)
	if prefix == "" {
		return "", nil
	}
	if strings.Contains(prefix, "\x00") {
		return "", fmt.Errorf("prefix contains invalid character")
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = superAgentHarnessToolOutputRoot + "/" + strings.TrimLeft(prefix, "/")
	}
	cleanPrefix := path.Clean(prefix)
	if strings.HasSuffix(prefix, "/") && cleanPrefix != "/" {
		cleanPrefix += "/"
	}
	if cleanPrefix == superAgentHarnessToolOutputRoot || !strings.HasPrefix(cleanPrefix, superAgentHarnessToolOutputRoot+"/") {
		return "", fmt.Errorf("prefix must be under %s", superAgentHarnessToolOutputRoot)
	}
	return cleanPrefix, nil
}

func uniqueSuperAgentHarnessCleanupPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func superAgentHarnessToolOutputEntryFromContent(outputPath string, file *developer_api.SandboxFileInfo, content *developer_api.ReadSandboxFileData) *SuperAgentHarnessToolOutputEntry {
	name := path.Base(outputPath)
	size := int64(0)
	mtime := int64(0)
	if file != nil {
		if strings.TrimSpace(file.Name) != "" {
			name = file.Name
		}
		size = file.Size
		mtime = file.Mtime
	}
	entry := &SuperAgentHarnessToolOutputEntry{
		Path:   outputPath,
		Name:   name,
		Status: "completed",
		Size:   size,
		Mtime:  mtime,
	}
	if content == nil || strings.TrimSpace(content.Content) == "" {
		entry.Summary = name
		return entry
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(content.Content), &raw); err != nil {
		entry.ResultPreview = superAgentHarnessPreviewString(content.Content)
		entry.Summary = superAgentHarnessPreviewString(name + " " + entry.ResultPreview)
		return entry
	}
	entry.ToolCallID = superAgentHarnessFirstString(raw, "tool_call_id", "call_id", "id")
	entry.Tool = superAgentHarnessFirstString(raw, "tool", "tool_name", "name", "action", "operation")
	if entry.Tool == "" {
		entry.Tool = strings.TrimSuffix(name, path.Ext(name))
	}
	entry.Status = superAgentHarnessFirstString(raw, "status", "state")
	if entry.Status == "" {
		if raw["error"] != nil {
			entry.Status = "failed"
		} else {
			entry.Status = "completed"
		}
	}
	entry.Arguments = superAgentHarnessFirstValue(raw, "args", "arguments", "params", "input")
	entry.Result = superAgentHarnessFirstValue(raw, "result", "output", "outputs", "return")
	entry.Error = superAgentHarnessFirstValue(raw, "error")
	entry.ArgumentsPreview = superAgentHarnessPreviewJSONField(raw, "args", "arguments", "params", "input")
	entry.ResultPreview = superAgentHarnessPreviewJSONField(raw, "result", "output", "outputs", "return", "error")
	entry.Summary = superAgentHarnessBuildToolOutputSummary(entry)
	return entry
}

func superAgentHarnessFirstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func superAgentHarnessFirstValue(raw map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := raw[key]
		if ok && value != nil {
			return value
		}
	}
	return nil
}

func superAgentHarnessPreviewJSONField(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		return superAgentHarnessPreviewValue(value)
	}
	return ""
}

func superAgentHarnessPreviewValue(value any) string {
	if s, ok := value.(string); ok {
		return superAgentHarnessPreviewString(s)
	}
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return superAgentHarnessPreviewString(string(blob))
}

func superAgentHarnessBuildToolOutputSummary(entry *SuperAgentHarnessToolOutputEntry) string {
	parts := make([]string, 0, 3)
	if entry.Tool != "" {
		parts = append(parts, entry.Tool)
	}
	if entry.ArgumentsPreview != "" {
		parts = append(parts, "args="+entry.ArgumentsPreview)
	}
	if entry.ResultPreview != "" {
		parts = append(parts, "result="+entry.ResultPreview)
	}
	if len(parts) == 0 {
		return entry.Name
	}
	return superAgentHarnessPreviewString(strings.Join(parts, " "))
}

func superAgentHarnessPreviewString(s string) string {
	const maxPreview = 500
	s = strings.TrimSpace(strings.ToValidUTF8(s, ""))
	if len(s) <= maxPreview {
		return s
	}
	return strings.TrimSpace(s[:maxPreview]) + "..."
}

type superAgentHarnessRuntimeSkillsPayload struct {
	OK     bool                                                  `json:"ok"`
	Error  string                                                `json:"error"`
	Skills []*developer_api.SuperAgentHarnessRuntimeSkillSummary `json:"skills"`
}

func (s *SingleAgentApplicationService) listSuperAgentHarnessRuntimeSkills(ctx context.Context, svc crosssandbox.Manager, key string) *developer_api.SuperAgentHarnessRuntimeSkillsState {
	state := &developer_api.SuperAgentHarnessRuntimeSkillsState{
		Root:   "/skills",
		Skills: []*developer_api.SuperAgentHarnessRuntimeSkillSummary{},
	}
	py := `import json,os
root="/skills"
allowed=("scripts/","references/","templates/","assets/")
images=(".png",".jpg",".jpeg",".webp",".gif",".svg")
def valid(rel):
  return rel=="SKILL.md" or any(rel.startswith(p) and len(rel)>len(p) for p in allowed)
def metadata(md):
  out={}
  lines=md.splitlines()
  if len(lines) >= 2 and lines[0].strip() == "---":
    for line in lines[1:]:
      if line.strip() == "---":
        break
      if ":" not in line or line.startswith(" "):
        continue
      k,v=line.split(":",1)
      k=k.strip()
      if k in ("description","version","category"):
        out[k]=v.strip().strip("'\"")
  return out
skills=[]
if os.path.isdir(root):
  for name in sorted(os.listdir(root)):
    if name.startswith("."):
      continue
    base=os.path.join(root,name)
    if not os.path.isdir(base):
      continue
    file_paths=[]
    for dirpath, dirnames, filenames in os.walk(base):
      dirnames[:]=sorted([d for d in dirnames if not d.startswith(".") and d not in ("__pycache__",".git")])
      for filename in sorted(filenames):
        if filename.startswith(".") or filename in (".skillhash",".DS_Store"):
          continue
        rel=os.path.relpath(os.path.join(dirpath,filename),base).replace(os.sep,"/")
        if "/." in rel or "\\" in rel or ".." in rel.split("/"):
          continue
        file_paths.append(rel)
    file_paths=sorted(file_paths)
    asset_paths=[p for p in file_paths if p.startswith("assets/")]
    image_paths=[p for p in asset_paths if p.lower().endswith(images)]
    md=""
    entry=os.path.join(base,"SKILL.md")
    if os.path.isfile(entry):
      try:
        with open(entry,"r",encoding="utf-8",errors="replace") as f:
          md=f.read(65536)
      except OSError:
        md=""
    meta=metadata(md)
    skills.append({
      "name":name,
      "path":base,
      "entry_path":base+"/SKILL.md",
      "standard":("SKILL.md" in file_paths and all(valid(p) for p in file_paths)),
      "description":meta.get("description",""),
      "version":meta.get("version",""),
      "category":meta.get("category",""),
      "file_paths":file_paths,
      "asset_paths":asset_paths,
      "image_paths":image_paths,
    })
print(json.dumps({"ok":True,"skills":skills}))`
	cmd := fmt.Sprintf("python3 -c %s", shellQuote(py))
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		logs.CtxWarnf(ctx, "[listSuperAgentHarnessRuntimeSkills] list failed: %v", err)
		return state
	}
	if res.ExitCode != 0 {
		logs.CtxWarnf(ctx, "[listSuperAgentHarnessRuntimeSkills] list exit %d: %s", res.ExitCode, res.Stderr)
		return state
	}
	var payload superAgentHarnessRuntimeSkillsPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &payload); err != nil {
		logs.CtxWarnf(ctx, "[listSuperAgentHarnessRuntimeSkills] parse failed: %v", err)
		return state
	}
	if !payload.OK {
		logs.CtxWarnf(ctx, "[listSuperAgentHarnessRuntimeSkills] payload error: %s", payload.Error)
		return state
	}
	if payload.Skills != nil {
		state.Skills = payload.Skills
	}
	return state
}

func (s *SingleAgentApplicationService) UpdateSuperAgentHarnessPlan(ctx context.Context, req *developer_api.SuperAgentHarnessPlanUpdateRequest) (*developer_api.SuperAgentHarnessPlanUpdateResponse, error) {
	plan := req.Plan
	if len(plan) == 0 {
		plan = req.Items
	}
	if len(plan) == 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "plan must contain at least one step"))
	}
	for _, step := range plan {
		if step == nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "plan step is required"))
		}
		step.Content = strings.TrimSpace(step.Content)
		if step.Content == "" {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "plan step content is required"))
		}
		switch strings.TrimSpace(step.Status) {
		case "pending", "in_progress", "completed", "done":
			step.Status = strings.TrimSpace(step.Status)
		default:
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid plan status: %s", step.Status)))
		}
	}

	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	planPath := superAgentHarnessPlanPathForConversation(req.ConversationID)
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills /workspace/.agent/tooloutputs", 20)
	if req.ConversationID > 0 {
		_, _ = svc.Exec(ctx, key, fmt.Sprintf("mkdir -p %s", shellQuote(fmt.Sprintf("/workspace/.agent/sessions/%d", req.ConversationID))), 20)
	}
	blob, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("marshal plan failed: %v", err)))
	}
	if err := svc.WriteFile(ctx, key, planPath, blob); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("update plan failed: %v", err)))
	}
	return &developer_api.SuperAgentHarnessPlanUpdateResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.SuperAgentHarnessPlanUpdateData{
			Path:    planPath,
			Steps:   int32(len(plan)),
			Content: string(blob),
		},
	}, nil
}

func (s *SingleAgentApplicationService) RunSuperAgentSandboxCommand(ctx context.Context, req *developer_api.ExecSandboxCommandRequest) (*developer_api.ExecSandboxCommandResponse, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "command is required"))
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	workdir, err := sanitizeSandboxPath(sandboxExecWorkdir(req))
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = superAgentExecDefaultTimeoutSec
	}
	if timeoutSec > superAgentExecMaxTimeoutSec {
		timeoutSec = superAgentExecMaxTimeoutSec
	}

	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills", 20)
	res, err := svc.Exec(ctx, key, fmt.Sprintf("cd %s && %s", shellQuote(workdir), command), timeoutSec)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("exec failed: %v", err)))
	}
	stdout, stdoutTruncated := truncateSuperAgentSearchOutput(res.Stdout)
	stderr, stderrTruncated := truncateSuperAgentSearchOutput(res.Stderr)
	return &developer_api.ExecSandboxCommandResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.ExecSandboxCommandData{
			ExitCode:    int32(res.ExitCode),
			Stdout:      stdout,
			Stderr:      stderr,
			IsTruncated: stdoutTruncated || stderrTruncated,
		},
	}, nil
}

func sandboxExecWorkdir(req *developer_api.ExecSandboxCommandRequest) string {
	if strings.TrimSpace(req.WorkDir) != "" {
		return req.WorkDir
	}
	return req.WorkDirAlias
}

// ReadSuperAgentWorkspaceFile 读取超级体 App Server 可见文件,包含只读 /skills。
func (s *SingleAgentApplicationService) ReadSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.ReadSandboxFileRequest) (*developer_api.ReadSandboxFileResponse, error) {
	return s.readSandboxFileWithSanitizer(ctx, req, sanitizeSuperAgentReadablePath)
}

// DownloadSuperAgentWorkspaceFile 下载超级体 App Server 可见文件,包含只读 /skills。
func (s *SingleAgentApplicationService) DownloadSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.DownloadSandboxFileRequest) (*developer_api.DownloadSandboxFileData, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSuperAgentReadablePath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("download failed: %v", err)))
	}
	return &developer_api.DownloadSandboxFileData{
		Path:    path,
		Content: b,
		Size:    int64(len(b)),
	}, nil
}

// UploadSuperAgentWorkspaceFile 上传超级体文件,仍限制在 /workspace、/uploads、/outputs。
func (s *SingleAgentApplicationService) UploadSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.UploadSandboxFileRequest) (*developer_api.UploadSandboxFileResponse, error) {
	return s.UploadSandboxFile(ctx, req)
}

// DeleteSuperAgentWorkspaceFile 删除超级体文件,仍限制在 /workspace、/uploads、/outputs。
func (s *SingleAgentApplicationService) DeleteSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.DeleteSandboxFileRequest) (*developer_api.DeleteSandboxFileResponse, error) {
	return s.DeleteSandboxFile(ctx, req)
}

// MoveSuperAgentWorkspaceFile 移动或重命名超级体可写工作区文件。
func (s *SingleAgentApplicationService) MoveSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.MoveSandboxFileRequest) (*developer_api.MoveSandboxFileResponse, error) {
	return s.MoveSandboxFile(ctx, req)
}

// CreateSuperAgentWorkspaceDirectory 创建超级体可写工作区目录。
func (s *SingleAgentApplicationService) CreateSuperAgentWorkspaceDirectory(ctx context.Context, req *developer_api.CreateSandboxDirectoryRequest) (*developer_api.CreateSandboxDirectoryResponse, error) {
	return s.CreateSandboxDirectory(ctx, req)
}

// StatSuperAgentWorkspacePath 查询超级体 App Server 可见路径元信息,包含只读 /skills。
func (s *SingleAgentApplicationService) StatSuperAgentWorkspacePath(ctx context.Context, req *developer_api.StatSandboxFileRequest) (*developer_api.StatSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	statPath, err := sanitizeSuperAgentReadablePath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills", 20)
	py := "import os,json,sys\n" +
		"p=sys.argv[1]\n" +
		"exists=os.path.exists(p)\n" +
		"name=os.path.basename(p.rstrip('/')) or p\n" +
		"out={'exists':exists,'name':name,'path':p,'is_dir':False,'size':0,'mtime':0}\n" +
		"if exists:\n" +
		"  st=os.stat(p)\n" +
		"  out.update({'is_dir':os.path.isdir(p),'size':(0 if os.path.isdir(p) else st.st_size),'mtime':int(st.st_mtime)})\n" +
		"print(json.dumps(out, ensure_ascii=False))"
	cmd := fmt.Sprintf("python3 -c %s %s", shellQuote(py), shellQuote(statPath))
	res, err := svc.Exec(ctx, key, cmd, 20)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("stat failed: %v", err)))
	}
	var raw struct {
		Exists bool   `json:"exists"`
		Name   string `json:"name"`
		Path   string `json:"path"`
		IsDir  bool   `json:"is_dir"`
		Size   int64  `json:"size"`
		Mtime  int64  `json:"mtime"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &raw); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("parse stat failed: %v", err)))
	}
	data := &developer_api.StatSandboxFileData{
		Path:   statPath,
		Exists: raw.Exists,
	}
	if raw.Exists {
		data.File = &developer_api.SandboxFileInfo{
			Name:  raw.Name,
			Path:  raw.Path,
			IsDir: raw.IsDir,
			Size:  raw.Size,
			Mtime: raw.Mtime,
		}
	}
	return &developer_api.StatSandboxFileResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	}, nil
}

// GrepSuperAgentWorkspace 在超级体 App Server 可见根目录内搜索文件内容。
func (s *SingleAgentApplicationService) GrepSuperAgentWorkspace(ctx context.Context, req *developer_api.GrepSandboxFilesRequest) (*developer_api.GrepSandboxFilesResponse, error) {
	if strings.TrimSpace(req.Pattern) == "" {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "pattern is required"))
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	searchPath, err := sanitizeSuperAgentReadablePath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills", 20)
	var out string
	if superAgentGrepHasOptions(req) {
		out, err = s.grepSuperAgentWorkspaceWithOptions(ctx, svc, key, req, searchPath)
	} else {
		out, err = svc.Grep(ctx, key, req.Pattern, searchPath)
	}
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("grep failed: %v", err)))
	}
	output, truncated := truncateSuperAgentSearchOutput(out)
	return &developer_api.GrepSandboxFilesResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.GrepSandboxFilesData{
			Path:        searchPath,
			Pattern:     req.Pattern,
			Output:      output,
			IsTruncated: truncated,
		},
	}, nil
}

func superAgentGrepHasOptions(req *developer_api.GrepSandboxFilesRequest) bool {
	return req.CaseSensitive != nil || len(req.Include) > 0 || len(req.Exclude) > 0
}

func (s *SingleAgentApplicationService) grepSuperAgentWorkspaceWithOptions(ctx context.Context, svc crosssandbox.Manager, key string, req *developer_api.GrepSandboxFilesRequest, searchPath string) (string, error) {
	cmd := buildSuperAgentGrepCommand(req, searchPath)
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Stdout) == "" {
		return "(no matches)", nil
	}
	return res.Stdout, nil
}

func buildSuperAgentGrepCommand(req *developer_api.GrepSandboxFilesRequest, searchPath string) string {
	rgArgs := []string{"rg", "-n", "--no-heading"}
	grepArgs := []string{"grep", "-RIn"}
	if req.CaseSensitive != nil && !*req.CaseSensitive {
		rgArgs = append(rgArgs, "-i")
		grepArgs = append(grepArgs, "-i")
	}
	for _, include := range cleanSuperAgentGrepGlobs(req.Include) {
		rgArgs = append(rgArgs, "-g", include)
		grepArgs = append(grepArgs, "--include", include)
	}
	for _, exclude := range cleanSuperAgentGrepGlobs(req.Exclude) {
		rgArgs = append(rgArgs, "-g", "!"+exclude)
		grepArgs = append(grepArgs, "--exclude", exclude)
	}
	rgArgs = append(rgArgs, "--", req.Pattern, searchPath)
	grepArgs = append(grepArgs, "--", req.Pattern, searchPath)
	return fmt.Sprintf("if command -v rg >/dev/null 2>&1; then %s; else %s; fi",
		shellJoin(rgArgs),
		shellJoin(grepArgs),
	)
}

func cleanSuperAgentGrepGlobs(globs []string) []string {
	cleaned := make([]string, 0, len(globs))
	for _, glob := range globs {
		glob = strings.TrimSpace(glob)
		if glob != "" {
			cleaned = append(cleaned, glob)
		}
	}
	return cleaned
}

// GlobSuperAgentWorkspace 在超级体 App Server 可见根目录内按文件名模式查找文件。
func (s *SingleAgentApplicationService) GlobSuperAgentWorkspace(ctx context.Context, req *developer_api.GlobSandboxFilesRequest) (*developer_api.GlobSandboxFilesResponse, error) {
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "pattern is required"))
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	searchPath, err := sanitizeSuperAgentReadablePath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	limit := req.Limit
	if limit <= 0 || limit > superAgentGlobMaxMatches {
		limit = superAgentGlobMaxMatches
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs /skills", 20)
	cmd := fmt.Sprintf("find %s -type f -name %s 2>/dev/null | sort | head -n %d", shellQuote(searchPath), shellQuote(pattern), limit+1)
	res, err := svc.Exec(ctx, key, cmd, 20)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("glob failed: %v", err)))
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	matches := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			matches = append(matches, line)
		}
	}
	truncated := len(matches) > int(limit)
	if truncated {
		matches = matches[:int(limit)]
	}
	return &developer_api.GlobSandboxFilesResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.GlobSandboxFilesData{
			Path:        searchPath,
			Pattern:     pattern,
			Matches:     matches,
			IsTruncated: truncated,
		},
	}, nil
}

func truncateSuperAgentSearchOutput(out string) (string, bool) {
	if len(out) <= superAgentSearchMaxOutputBytes {
		return out, false
	}
	return strings.ToValidUTF8(out[:superAgentSearchMaxOutputBytes], "") + "\n...[truncated]...", true
}

// EditSuperAgentWorkspaceFile 对超级体可写工作区文件执行精确字符串替换。
func (s *SingleAgentApplicationService) EditSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.EditSandboxFileRequest) (*developer_api.EditSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	editPath, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	n, err := svc.EditFile(ctx, key, editPath, req.OldString, req.NewString, req.ReplaceAll)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("edit failed: %v", err)))
	}
	return &developer_api.EditSandboxFileResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.EditSandboxFileData{
			Path:         editPath,
			Replacements: int32(n),
		},
	}, nil
}

type workspacePatchOp struct {
	kind       string
	path       string
	targetPath string
	content    string
	hunks      []workspacePatchHunk
}

type workspacePatchHunk struct {
	oldText string
	newText string
}

func (s *SingleAgentApplicationService) ApplySuperAgentWorkspacePatch(ctx context.Context, req *developer_api.ApplySandboxPatchRequest) (*developer_api.ApplySandboxPatchResponse, error) {
	if strings.TrimSpace(req.Patch) == "" {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "patch is required"))
	}
	workdir := strings.TrimSpace(sandboxPatchWorkdir(req))
	if workdir == "" {
		workdir = "/workspace"
	}
	workdir, err := sanitizeSandboxPath(workdir)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	ops, err := parseSuperAgentWorkspacePatch(req.Patch, workdir)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs", 20)

	paths := make([]string, 0, len(ops))
	for _, op := range ops {
		switch op.kind {
		case "add":
			if err := ensureSandboxPatchFilePath(op.path); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
			}
			_, _ = svc.Exec(ctx, key, fmt.Sprintf("mkdir -p %s", shellQuote(path.Dir(op.path))), 20)
			if err := svc.WriteFile(ctx, key, op.path, []byte(op.content)); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch failed: %v", err)))
			}
		case "update":
			if err := ensureSandboxPatchFilePath(op.path); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
			}
			if op.targetPath != "" {
				if err := ensureSandboxPatchFilePath(op.targetPath); err != nil {
					return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
				}
			}
			if op.targetPath != "" && len(op.hunks) == 0 {
				cmd := fmt.Sprintf("mkdir -p %s && mv %s %s", shellQuote(path.Dir(op.targetPath)), shellQuote(op.path), shellQuote(op.targetPath))
				if _, err := svc.Exec(ctx, key, cmd, 20); err != nil {
					return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch move failed: %v", err)))
				}
				break
			}
			b, err := svc.ReadFile(ctx, key, op.path)
			if err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch read failed: %v", err)))
			}
			content, err := applySuperAgentWorkspacePatchHunks(string(b), op.hunks)
			if err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
			}
			writePath := op.path
			if op.targetPath != "" {
				writePath = op.targetPath
				_, _ = svc.Exec(ctx, key, fmt.Sprintf("mkdir -p %s", shellQuote(path.Dir(writePath))), 20)
			}
			if err := svc.WriteFile(ctx, key, writePath, []byte(content)); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch failed: %v", err)))
			}
			if op.targetPath != "" && op.targetPath != op.path {
				if _, err := svc.Exec(ctx, key, fmt.Sprintf("rm -rf %s", shellQuote(op.path)), 20); err != nil {
					return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch delete failed: %v", err)))
				}
			}
		case "delete":
			if err := ensureSandboxPatchFilePath(op.path); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
			}
			if _, err := svc.Exec(ctx, key, fmt.Sprintf("rm -rf %s", shellQuote(op.path)), 20); err != nil {
				return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("apply patch delete failed: %v", err)))
			}
		default:
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("unsupported patch operation: %s", op.kind)))
		}
		changedPath := op.path
		if op.targetPath != "" {
			changedPath = op.targetPath
		}
		paths = append(paths, changedPath)
	}
	return &developer_api.ApplySandboxPatchResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.ApplySandboxPatchData{
			ChangedFiles: int32(len(paths)),
			Paths:        paths,
		},
	}, nil
}

func sandboxPatchWorkdir(req *developer_api.ApplySandboxPatchRequest) string {
	if strings.TrimSpace(req.WorkDir) != "" {
		return req.WorkDir
	}
	return req.WorkDirAlias
}

func parseSuperAgentWorkspacePatch(raw string, workdir string) ([]workspacePatchOp, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "*** Begin Patch" {
		return nil, fmt.Errorf("patch must start with *** Begin Patch")
	}
	ops := make([]workspacePatchOp, 0)
	for i := 1; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		if line == "*** End Patch" {
			if len(ops) == 0 {
				return nil, fmt.Errorf("patch must contain at least one operation")
			}
			return ops, nil
		}
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			target, err := resolveSuperAgentPatchPath(strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: ")), workdir)
			if err != nil {
				return nil, err
			}
			i++
			addLines := make([]string, 0)
			for i < len(lines) && !isSuperAgentPatchOperationMarker(lines[i]) {
				if !strings.HasPrefix(lines[i], "+") {
					return nil, fmt.Errorf("add file lines must start with +")
				}
				addLines = append(addLines, strings.TrimPrefix(lines[i], "+"))
				i++
			}
			content := strings.Join(addLines, "\n")
			if len(addLines) > 0 {
				content += "\n"
			}
			ops = append(ops, workspacePatchOp{kind: "add", path: target, content: content})
		case strings.HasPrefix(line, "*** Delete File: "):
			target, err := resolveSuperAgentPatchPath(strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: ")), workdir)
			if err != nil {
				return nil, err
			}
			ops = append(ops, workspacePatchOp{kind: "delete", path: target})
			i++
		case strings.HasPrefix(line, "*** Update File: "):
			target, err := resolveSuperAgentPatchPath(strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: ")), workdir)
			if err != nil {
				return nil, err
			}
			i++
			moveTarget := ""
			if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "*** Move to: ") {
				moveTarget, err = resolveSuperAgentPatchPath(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "*** Move to: ")), workdir)
				if err != nil {
					return nil, err
				}
				i++
			}
			hunks := make([]workspacePatchHunk, 0)
			for i < len(lines) && !isSuperAgentPatchOperationMarker(lines[i]) {
				if strings.TrimSpace(lines[i]) == "*** End of File" {
					i++
					continue
				}
				if !strings.HasPrefix(strings.TrimSpace(lines[i]), "@@") {
					return nil, fmt.Errorf("update file requires @@ hunk")
				}
				i++
				oldLines := make([]string, 0)
				newLines := make([]string, 0)
				for i < len(lines) && !isSuperAgentPatchOperationMarker(lines[i]) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "@@") {
					if strings.TrimSpace(lines[i]) == "*** End of File" {
						i++
						break
					}
					if lines[i] == "" {
						return nil, fmt.Errorf("patch hunk lines must start with space, +, or -")
					}
					body := lines[i][1:]
					switch lines[i][0] {
					case ' ':
						oldLines = append(oldLines, body)
						newLines = append(newLines, body)
					case '-':
						oldLines = append(oldLines, body)
					case '+':
						newLines = append(newLines, body)
					default:
						return nil, fmt.Errorf("patch hunk lines must start with space, +, or -")
					}
					i++
				}
				hunks = append(hunks, workspacePatchHunk{
					oldText: joinSuperAgentPatchLines(oldLines),
					newText: joinSuperAgentPatchLines(newLines),
				})
			}
			if len(hunks) == 0 && moveTarget == "" {
				return nil, fmt.Errorf("update file requires at least one hunk")
			}
			ops = append(ops, workspacePatchOp{kind: "update", path: target, targetPath: moveTarget, hunks: hunks})
		default:
			return nil, fmt.Errorf("unsupported patch marker: %s", line)
		}
	}
	return nil, fmt.Errorf("patch must end with *** End Patch")
}

func isSuperAgentPatchOperationMarker(line string) bool {
	line = strings.TrimSpace(line)
	return line == "*** End Patch" ||
		strings.HasPrefix(line, "*** Add File: ") ||
		strings.HasPrefix(line, "*** Delete File: ") ||
		strings.HasPrefix(line, "*** Update File: ")
}

func resolveSuperAgentPatchPath(p string, workdir string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("patch path is required")
	}
	if strings.HasPrefix(p, "/") {
		return sanitizeSandboxPath(p)
	}
	return sanitizeSandboxPath(path.Clean(path.Join(workdir, p)))
}

func ensureSandboxPatchFilePath(p string) error {
	for _, root := range sandboxAllowedRoots {
		if p == root {
			return fmt.Errorf("patch path must include a file name")
		}
	}
	return nil
}

func joinSuperAgentPatchLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func applySuperAgentWorkspacePatchHunks(content string, hunks []workspacePatchHunk) (string, error) {
	for _, hunk := range hunks {
		if hunk.oldText == "" {
			return "", fmt.Errorf("patch update hunk must remove or match existing text")
		}
		if strings.Contains(content, hunk.oldText) {
			content = strings.Replace(content, hunk.oldText, hunk.newText, 1)
			continue
		}
		oldNoTrailingNewline := strings.TrimSuffix(hunk.oldText, "\n")
		if oldNoTrailingNewline != hunk.oldText && strings.Contains(content, oldNoTrailingNewline) {
			content = strings.Replace(content, oldNoTrailingNewline, strings.TrimSuffix(hunk.newText, "\n"), 1)
			continue
		}
		return "", fmt.Errorf("patch hunk did not match target file")
	}
	return content, nil
}

// UploadSandboxFile 写入/上传文件(默认进 /uploads)。
func (s *SingleAgentApplicationService) UploadSandboxFile(ctx context.Context, req *developer_api.UploadSandboxFileRequest) (*developer_api.UploadSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	var content []byte
	encoding := strings.ToLower(strings.TrimSpace(req.Encoding))
	if req.IsBase64 || encoding == "base64" {
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "invalid base64 content"))
		}
	} else {
		switch encoding {
		case "", "utf-8", "utf8", "text", "plain":
		default:
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("unsupported encoding: %s", req.Encoding)))
		}
		content = []byte(req.Content)
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs", 20)
	if err := svc.WriteFile(ctx, key, path, content); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("write failed: %v", err)))
	}
	return &developer_api.UploadSandboxFileResponse{Code: 0, Data: &developer_api.UploadSandboxFileData{Path: path}}, nil
}

// DeleteSandboxFile 删除文件或目录。
func (s *SingleAgentApplicationService) DeleteSandboxFile(ctx context.Context, req *developer_api.DeleteSandboxFileRequest) (*developer_api.DeleteSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	// 不允许删除根契约目录本身。
	for _, root := range sandboxAllowedRoots {
		if path == root {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "cannot delete root directory"))
		}
	}
	if _, err := svc.Exec(ctx, key, fmt.Sprintf("rm -rf %s", shellQuote(path)), 20); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("delete failed: %v", err)))
	}
	return &developer_api.DeleteSandboxFileResponse{Code: 0}, nil
}

// MoveSandboxFile 移动或重命名文件/目录。
func (s *SingleAgentApplicationService) MoveSandboxFile(ctx context.Context, req *developer_api.MoveSandboxFileRequest) (*developer_api.MoveSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	fromPathValue := req.Path
	if strings.TrimSpace(fromPathValue) == "" {
		fromPathValue = req.FromPath
	}
	targetPathValue := req.TargetPath
	if strings.TrimSpace(targetPathValue) == "" {
		targetPathValue = req.ToPath
	}
	fromPath, err := sanitizeSandboxPath(fromPathValue)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	targetPath, err := sanitizeSandboxPath(targetPathValue)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	for _, root := range sandboxAllowedRoots {
		if fromPath == root {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "cannot move root directory"))
		}
	}
	for _, root := range sandboxAllowedRoots {
		if targetPath == root {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "target_path must include a file name"))
		}
	}
	targetDir := path.Dir(targetPath)
	cmd := fmt.Sprintf("mkdir -p %s && mv %s %s", shellQuote(targetDir), shellQuote(fromPath), shellQuote(targetPath))
	if _, err := svc.Exec(ctx, key, cmd, 20); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("move failed: %v", err)))
	}
	return &developer_api.MoveSandboxFileResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.MoveSandboxFileData{
			Path:     targetPath,
			FromPath: fromPath,
		},
	}, nil
}

// CreateSandboxDirectory 创建目录。
func (s *SingleAgentApplicationService) CreateSandboxDirectory(ctx context.Context, req *developer_api.CreateSandboxDirectoryRequest) (*developer_api.CreateSandboxDirectoryResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	dirPath, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	for _, root := range sandboxAllowedRoots {
		if dirPath == root {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "path must include a directory name"))
		}
	}
	if _, err := svc.Exec(ctx, key, fmt.Sprintf("mkdir -p %s", shellQuote(dirPath)), 20); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("create directory failed: %v", err)))
	}
	return &developer_api.CreateSandboxDirectoryResponse{
		Code: 0,
		Msg:  "success",
		Data: &developer_api.CreateSandboxDirectoryData{Path: dirPath},
	}, nil
}

// shellQuote 把字符串安全包成单引号 shell 参数。
func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
