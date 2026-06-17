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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// 允许管理的沙箱根目录(契约目录)。
var sandboxAllowedRoots = []string{"/workspace", "/uploads", "/outputs"}

const sandboxReadMaxBytes = 2 * 1024 * 1024 // 单文件读取上限 2MB

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
	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		return nil, "", errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "session required"))
	}
	key := agentsandbox.SandboxKeyFor(connectorID, agentID, strconv.FormatInt(*uid, 10))
	return svc, key, nil
}

// sanitizeSandboxPath 把入参规整为绝对路径并限制在允许的根目录内,防目录穿越。
func sanitizeSandboxPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "/workspace"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("path must not contain '..'")
	}
	for _, root := range sandboxAllowedRoots {
		if p == root || strings.HasPrefix(p, root+"/") {
			return p, nil
		}
	}
	return "", fmt.Errorf("path must be under /workspace, /uploads or /outputs")
}

// ListSandboxFiles 列出指定目录下的文件/子目录。
func (s *SingleAgentApplicationService) ListSandboxFiles(ctx context.Context, req *developer_api.ListSandboxFilesRequest) (*developer_api.ListSandboxFilesResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	// 确保契约目录存在(沙箱可能是冷启动)。
	_, _ = svc.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs", 20)

	// 用 python3 输出 JSON,避免解析 ls 的脆弱性(slim 镜像自带 python3)。
	py := "import os,json,sys\n" +
		"p=sys.argv[1]\n" +
		"out=[]\n" +
		"for n in sorted(os.listdir(p)):\n" +
		"  fp=os.path.join(p,n)\n" +
		"  try: out.append({'name':n,'is_dir':os.path.isdir(fp),'size':(0 if os.path.isdir(fp) else os.path.getsize(fp))})\n" +
		"  except OSError: pass\n" +
		"print(json.dumps(out))"
	cmd := fmt.Sprintf("python3 -c %s %s", shellQuote(py), shellQuote(path))
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("list failed: %v", err)))
	}
	files := make([]*developer_api.SandboxFileInfo, 0)
	stdout := strings.TrimSpace(res.Stdout)
	if stdout != "" {
		var raw []struct {
			Name  string `json:"name"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size"`
		}
		if jErr := json.Unmarshal([]byte(stdout), &raw); jErr == nil {
			for _, r := range raw {
				fp := strings.TrimRight(path, "/") + "/" + r.Name
				files = append(files, &developer_api.SandboxFileInfo{
					Name:  r.Name,
					Path:  fp,
					IsDir: r.IsDir,
					Size:  r.Size,
				})
			}
		}
	}
	return &developer_api.ListSandboxFilesResponse{
		Code: 0,
		Data: &developer_api.ListSandboxFilesData{Path: path, Files: files},
	}, nil
}

// ReadSandboxFile 读取单个文件内容(文本直出,二进制 base64)。
func (s *SingleAgentApplicationService) ReadSandboxFile(ctx context.Context, req *developer_api.ReadSandboxFileRequest) (*developer_api.ReadSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("read failed: %v", err)))
	}
	if len(b) > sandboxReadMaxBytes {
		b = b[:sandboxReadMaxBytes]
	}
	data := &developer_api.ReadSandboxFileData{Path: path, Size: int64(len(b))}
	if utf8.Valid(b) {
		data.Content = string(b)
		data.IsBinary = false
	} else {
		data.Content = base64.StdEncoding.EncodeToString(b)
		data.IsBinary = true
	}
	return &developer_api.ReadSandboxFileResponse{Code: 0, Data: data}, nil
}

// UploadSandboxFile 写入/上传文件(默认进 /uploads)。
func (s *SingleAgentApplicationService) UploadSandboxFile(ctx context.Context, req *developer_api.UploadSandboxFileRequest) (*developer_api.UploadSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSandboxPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	var content []byte
	if req.IsBase64 {
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "invalid base64 content"))
		}
	} else {
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
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
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

// shellQuote 把字符串安全包成单引号 shell 参数。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
