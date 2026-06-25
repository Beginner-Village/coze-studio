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
	"fmt"
	pathutil "path"
	"regexp"
	"strings"
	"unicode/utf8"

	skillapp "github.com/ynet-dev/ynet-studio/backend/application/skill"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	skillentity "github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	superAgentRuntimeSkillMaxFiles     = 200
	superAgentRuntimeSkillMaxBytes     = 25 * 1024 * 1024
	superAgentRuntimeSkillMaxFileBytes = 5 * 1024 * 1024
)

var superAgentRuntimeSkillNameRe = regexp.MustCompile(`^[a-z0-9-]+$`)

type SuperAgentRuntimeSkillImportRequest struct {
	AgentID      int64
	BotID        int64
	ConnectorID  *string
	Name         string
	SkillID      int64
	IconURI      string
	PublishScope int8
}

func (s *SingleAgentApplicationService) ImportSuperAgentRuntimeSkill(ctx context.Context, req *SuperAgentRuntimeSkillImportRequest) (*skillentity.Skill, error) {
	if req == nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "request is required"))
	}
	agentID := sandboxRequestAgentID(req.BotID, req.AgentID)
	if agentID <= 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "agent_id is required"))
	}
	name, err := sanitizeSuperAgentRuntimeSkillName(req.Name)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	draft, err := s.ValidateAgentDraftAccess(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if draft == nil || draft.SpaceID <= 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "agent space is required"))
	}
	sandboxSvc, key, err := s.resolveSandboxKey(ctx, agentID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	files, err := readSuperAgentRuntimeSkillFiles(ctx, sandboxSvc, key, name)
	if err != nil {
		return nil, err
	}

	skillSvc := skillapp.SkillApplicationSVC
	if skillSvc == nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill service is not available"))
	}

	iconURI := req.IconURI
	var saved *skillentity.Skill
	if req.SkillID > 0 {
		if iconURI == "" {
			if existing, getErr := skillSvc.GetSkill(ctx, req.SkillID, draft.SpaceID); getErr != nil {
				return nil, getErr
			} else if existing != nil {
				iconURI = existing.IconURI
			}
		}
		saved, err = skillSvc.UpdateSkill(ctx, req.SkillID, draft.SpaceID, "", "", "", iconURI, files)
	} else {
		saved, err = skillSvc.CreateSkill(ctx, draft.SpaceID, "", "", "", iconURI, files)
	}
	if err != nil {
		return nil, err
	}
	if req.PublishScope > 0 {
		saved, err = skillSvc.PublishSkill(ctx, saved.SkillID, draft.SpaceID, req.PublishScope)
		if err != nil {
			return nil, err
		}
	}
	return saved, nil
}

func sanitizeSuperAgentRuntimeSkillName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if !superAgentRuntimeSkillNameRe.MatchString(name) {
		return "", fmt.Errorf("name must match [a-z0-9-]+")
	}
	return name, nil
}

type superAgentRuntimeSkillFilesPayload struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Files []struct {
		Path          string `json:"path"`
		ContentBase64 string `json:"content_base64"`
	} `json:"files"`
}

func readSuperAgentRuntimeSkillFiles(ctx context.Context, svc crosssandbox.Manager, key, name string) (map[string]string, error) {
	base := "/skills/" + name
	py := `import base64,json,os,sys
root=sys.argv[1]
max_files=int(sys.argv[2])
max_file=int(sys.argv[3])
max_total=int(sys.argv[4])
ignored={".DS_Store",".skillhash"}
def fail(msg):
  print(json.dumps({"ok":False,"error":msg}))
  sys.exit(0)
# 标准技能包允许任意相对路径布局(顶层 README.md/requirements.txt 等),
# 不限制目录;路径穿越由下方 "/." / "\\" / ".." 检查拦截。
def valid(rel):
  return True
if not os.path.isdir(root):
  fail("skill folder not found")
files=[]
total=0
for dirpath, dirnames, filenames in os.walk(root):
  dirnames[:]=sorted([d for d in dirnames if d not in ("__pycache__",".git") and not d.startswith(".")])
  for filename in sorted(filenames):
    if filename in ignored or filename.startswith("."):
      continue
    path=os.path.join(dirpath,filename)
    rel=os.path.relpath(path,root).replace(os.sep,"/")
    if "/." in rel or "\\" in rel or ".." in rel.split("/"):
      fail("invalid skill file path: "+rel)
    if not valid(rel):
      fail("invalid skill file path: "+rel)
    size=os.path.getsize(path)
    if size > max_file:
      fail("skill file is too large: "+rel)
    total += size
    if total > max_total:
      fail("skill package is too large")
    with open(path,"rb") as f:
      raw=f.read(max_file+1)
    if len(raw) > max_file:
      fail("skill file is too large: "+rel)
    files.append({"path":rel,"content_base64":base64.b64encode(raw).decode("ascii")})
    if len(files) > max_files:
      fail("skill package has too many files")
if not any(f["path"]=="SKILL.md" for f in files):
  fail("SKILL.md is required")
print(json.dumps({"ok":True,"files":files}))`
	cmd := fmt.Sprintf(
		"python3 -c %s %s %d %d %d",
		shellQuote(py),
		shellQuote(base),
		superAgentRuntimeSkillMaxFiles,
		superAgentRuntimeSkillMaxFileBytes,
		superAgentRuntimeSkillMaxBytes,
	)
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("read skill folder failed: %v", err)))
	}
	if res.ExitCode != 0 {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("read skill folder failed: %s", strings.TrimSpace(res.Stderr))))
	}
	var payload superAgentRuntimeSkillFilesPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &payload); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "invalid skill folder snapshot"))
	}
	if !payload.OK {
		msg := strings.TrimSpace(payload.Error)
		if msg == "" {
			msg = "invalid skill folder"
		}
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", msg))
	}
	files := make(map[string]string, len(payload.Files))
	for _, file := range payload.Files {
		relPath := strings.TrimSpace(file.Path)
		content, err := base64.StdEncoding.DecodeString(file.ContentBase64)
		if err != nil {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file content: %s", relPath)))
		}
		encoded, err := encodeSuperAgentRuntimeSkillFile(relPath, content)
		if err != nil {
			return nil, err
		}
		files[relPath] = encoded
	}
	return files, nil
}

func encodeSuperAgentRuntimeSkillFile(relPath string, content []byte) (string, error) {
	if strings.HasPrefix(relPath, "assets/") && !utf8.Valid(content) {
		return "data:" + superAgentRuntimeSkillAssetMIME(relPath) + ";base64," + base64.StdEncoding.EncodeToString(content), nil
	}
	if !utf8.Valid(content) {
		return "", errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("skill file must be UTF-8 text: %s", relPath)))
	}
	return string(content), nil
}

func superAgentRuntimeSkillAssetMIME(relPath string) string {
	switch strings.ToLower(pathutil.Ext(relPath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
