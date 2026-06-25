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
	"encoding/json"
	"fmt"
	"path"
	"strings"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	superAgentArtifactRoot          = "/outputs"
	superAgentArtifactMaxListItems  = int32(200)
	superAgentArtifactDownloadRoute = "POST /api/super-agent/artifacts/download"
	superAgentArtifactDeleteRoute   = "POST /api/super-agent/artifacts/delete"
	superAgentArtifactMoveRoute     = "POST /api/super-agent/artifacts/move"
)

type SuperAgentArtifactListRequest struct {
	BotID       int64   `json:"bot_id,string,omitempty"`
	AgentID     int64   `json:"agent_id,string,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
	Path        string  `json:"path,omitempty"`
	Limit       int32   `json:"limit,omitempty"`
}

type SuperAgentArtifactDownloadRequest struct {
	BotID       int64   `json:"bot_id,string,omitempty"`
	AgentID     int64   `json:"agent_id,string,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
	Path        string  `json:"path,omitempty"`
	ArtifactID  string  `json:"artifact_id,omitempty"`
}

type SuperAgentArtifactDeleteRequest = SuperAgentArtifactDownloadRequest

type SuperAgentArtifactMoveRequest struct {
	BotID       int64   `json:"bot_id,string,omitempty"`
	AgentID     int64   `json:"agent_id,string,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
	Path        string  `json:"path,omitempty"`
	ArtifactID  string  `json:"artifact_id,omitempty"`
	TargetPath  string  `json:"target_path,omitempty"`
}

type SuperAgentArtifactDeleteResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type SuperAgentArtifactMoveResponse struct {
	Code int64                       `json:"code"`
	Msg  string                      `json:"msg"`
	Data *SuperAgentArtifactMoveData `json:"data"`
}

type SuperAgentArtifactMoveData struct {
	ArtifactID string `json:"artifact_id"`
	Path       string `json:"path"`
	FromPath   string `json:"from_path"`
}

type SuperAgentArtifactListResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data *SuperAgentArtifactListData `json:"data"`
}

type SuperAgentArtifactListData struct {
	Root      string                   `json:"root"`
	Path      string                   `json:"path"`
	Artifacts []SuperAgentArtifactMeta `json:"artifacts"`
}

type SuperAgentArtifactMeta struct {
	ArtifactID    string `json:"artifact_id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Kind          string `json:"kind"`
	PreviewType   string `json:"preview_type"`
	Summary       string `json:"summary"`
	Size          int64  `json:"size"`
	Mtime         int64  `json:"mtime"`
	Mime          string `json:"mime"`
	SHA256        string `json:"sha256"`
	Previewable   bool   `json:"previewable"`
	Downloadable  bool   `json:"downloadable"`
	DownloadRoute string `json:"download_route"`
}

type SuperAgentArtifactDownloadData struct {
	Path    string
	Content []byte
	Size    int64
}

func sanitizeSuperAgentArtifactPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return superAgentArtifactRoot, nil
	}
	if !strings.HasPrefix(p, "/") {
		if p == "outputs" || strings.HasPrefix(p, "outputs/") {
			p = "/" + p
		} else {
			p = superAgentArtifactRoot + "/" + p
		}
	}
	return sanitizeSandboxPathWithRoots(p, []string{superAgentArtifactRoot})
}

func (s *SingleAgentApplicationService) ListSuperAgentArtifacts(ctx context.Context, req *SuperAgentArtifactListRequest) (*SuperAgentArtifactListResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitizeSuperAgentArtifactPath(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	limit := req.Limit
	if limit <= 0 || limit > superAgentArtifactMaxListItems {
		limit = superAgentArtifactMaxListItems
	}
	_, _ = svc.Exec(ctx, key, "mkdir -p /outputs", 20)
	artifacts, err := s.listSuperAgentArtifactsByPath(ctx, svc, key, path, limit)
	if err != nil {
		return nil, err
	}
	return &SuperAgentArtifactListResponse{
		Code: 0,
		Msg:  "success",
		Data: &SuperAgentArtifactListData{
			Root:      superAgentArtifactRoot,
			Path:      path,
			Artifacts: artifacts,
		},
	}, nil
}

func (s *SingleAgentApplicationService) listSuperAgentArtifactsByPath(ctx context.Context, svc crosssandbox.Manager, key string, path string, limit int32) ([]SuperAgentArtifactMeta, error) {
	py := "import os,json,sys,hashlib,mimetypes\n" +
		"root=sys.argv[1]\n" +
		"limit=int(sys.argv[2])\n" +
		"paths=[]\n" +
		"if os.path.isfile(root):\n" +
		"  paths=[root]\n" +
		"elif os.path.isdir(root):\n" +
		"  stop=False\n" +
		"  for dirpath, dirnames, filenames in os.walk(root):\n" +
		"    dirnames.sort(); filenames.sort()\n" +
		"    for n in filenames:\n" +
		"      paths.append(os.path.join(dirpath,n))\n" +
		"      if len(paths) >= limit:\n" +
		"        stop=True\n" +
		"        break\n" +
		"    if stop:\n" +
		"      break\n" +
		"out=[]\n" +
		"for fp in paths[:limit]:\n" +
		"  try:\n" +
		"    st=os.stat(fp)\n" +
		"    h=hashlib.sha256()\n" +
		"    with open(fp,'rb') as f:\n" +
		"      for chunk in iter(lambda:f.read(1024*1024), b''):\n" +
		"        h.update(chunk)\n" +
		"    mime=mimetypes.guess_type(fp)[0] or 'application/octet-stream'\n" +
		"    out.append({'name':os.path.basename(fp),'path':fp,'size':st.st_size,'mtime':int(st.st_mtime),'mime':mime,'sha256':h.hexdigest()})\n" +
		"  except OSError:\n" +
		"    pass\n" +
		"print(json.dumps(out, ensure_ascii=False))"
	cmd := fmt.Sprintf("python3 -c %s %s %d", shellQuote(py), shellQuote(path), limit)
	res, err := svc.Exec(ctx, key, cmd, 30)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("list artifacts failed: %v", err)))
	}
	artifacts, err := parseSuperAgentArtifactList(res.Stdout)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("parse artifacts failed: %v", err)))
	}
	return artifacts, nil
}

func (s *SingleAgentApplicationService) DownloadSuperAgentArtifact(ctx context.Context, req *SuperAgentArtifactDownloadRequest) (*SuperAgentArtifactDownloadData, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path := req.Path
	if strings.TrimSpace(path) == "" {
		path = req.ArtifactID
	}
	path, err = sanitizeSuperAgentArtifactPath(path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("download failed: %v", err)))
	}
	return &SuperAgentArtifactDownloadData{Path: path, Content: b, Size: int64(len(b))}, nil
}

func (s *SingleAgentApplicationService) DeleteSuperAgentArtifact(ctx context.Context, req *SuperAgentArtifactDeleteRequest) (*SuperAgentArtifactDeleteResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path := req.Path
	if strings.TrimSpace(path) == "" {
		path = req.ArtifactID
	}
	path, err = sanitizeSuperAgentArtifactPath(path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	if path == superAgentArtifactRoot {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "cannot delete root directory"))
	}
	if _, err := svc.Exec(ctx, key, fmt.Sprintf("rm -rf %s", shellQuote(path)), 20); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("delete artifact failed: %v", err)))
	}
	return &SuperAgentArtifactDeleteResponse{Code: 0, Msg: "success"}, nil
}

func (s *SingleAgentApplicationService) MoveSuperAgentArtifact(ctx context.Context, req *SuperAgentArtifactMoveRequest) (*SuperAgentArtifactMoveResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, sandboxRequestAgentID(req.BotID, req.AgentID), req.ConnectorID)
	if err != nil {
		return nil, err
	}
	fromPath := req.Path
	if strings.TrimSpace(fromPath) == "" {
		fromPath = req.ArtifactID
	}
	fromPath, err = sanitizeSuperAgentArtifactPath(fromPath)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	targetPath, err := sanitizeSuperAgentArtifactPath(req.TargetPath)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	if fromPath == superAgentArtifactRoot {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "cannot move root directory"))
	}
	if targetPath == superAgentArtifactRoot {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", "target_path must include a file name"))
	}
	targetDir := path.Dir(targetPath)
	cmd := fmt.Sprintf("mkdir -p %s && mv %s %s", shellQuote(targetDir), shellQuote(fromPath), shellQuote(targetPath))
	if _, err := svc.Exec(ctx, key, cmd, 20); err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("move artifact failed: %v", err)))
	}
	return &SuperAgentArtifactMoveResponse{
		Code: 0,
		Msg:  "success",
		Data: &SuperAgentArtifactMoveData{
			ArtifactID: targetPath,
			Path:       targetPath,
			FromPath:   fromPath,
		},
	}, nil
}

func superAgentArtifactPreviewable(mime string) bool {
	if strings.HasPrefix(mime, "text/") || strings.HasPrefix(mime, "image/") {
		return true
	}
	switch mime {
	case "application/json", "application/pdf", "text/html", "text/markdown", "text/csv":
		return true
	default:
		return false
	}
}

func enrichSuperAgentArtifactMeta(artifact *SuperAgentArtifactMeta) {
	if artifact == nil {
		return
	}
	artifact.Kind, artifact.PreviewType = superAgentArtifactKindAndPreviewType(artifact.Name, artifact.Path, artifact.Mime)
	artifact.Summary = superAgentArtifactSummary(*artifact)
}

func superAgentArtifactKindAndPreviewType(name, artifactPath, mime string) (string, string) {
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	ext := strings.ToLower(path.Ext(name))
	if ext == "" {
		ext = strings.ToLower(path.Ext(artifactPath))
	}
	switch {
	case mime == "text/html" || ext == ".html" || ext == ".htm":
		return "report", "html"
	case strings.HasPrefix(mime, "image/"):
		return "image", "image"
	case mime == "text/csv" || ext == ".csv":
		return "table", "csv"
	case mime == "application/pdf" || ext == ".pdf":
		return "document", "pdf"
	case mime == "application/json" || ext == ".json":
		return "data", "json"
	case mime == "text/markdown" || ext == ".md" || ext == ".markdown":
		return "document", "markdown"
	case strings.HasPrefix(mime, "text/"):
		return "document", "text"
	case ext == ".zip" || ext == ".tar" || ext == ".gz" || ext == ".tgz":
		return "archive", "none"
	default:
		return "file", "none"
	}
}

func superAgentArtifactSummary(artifact SuperAgentArtifactMeta) string {
	label := "File"
	switch artifact.PreviewType {
	case "html":
		label = "HTML report"
	case "image":
		imageType := strings.TrimPrefix(strings.ToLower(strings.Split(artifact.Mime, ";")[0]), "image/")
		if imageType == "" || imageType == artifact.Mime {
			imageType = strings.TrimPrefix(strings.ToLower(path.Ext(artifact.Name)), ".")
		}
		if imageType == "" {
			label = "Image"
		} else {
			label = strings.ToUpper(imageType) + " image"
		}
	case "csv":
		label = "CSV table"
	case "pdf":
		label = "PDF document"
	case "json":
		label = "JSON data"
	case "markdown":
		label = "Markdown document"
	case "text":
		label = "Text document"
	case "none":
		switch artifact.Kind {
		case "archive":
			label = "Archive"
		default:
			label = "File"
		}
	}
	return fmt.Sprintf("%s, %s", label, superAgentArtifactFormatBytes(artifact.Size))
}

func superAgentArtifactFormatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

func parseSuperAgentArtifactList(stdout string) ([]SuperAgentArtifactMeta, error) {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return []SuperAgentArtifactMeta{}, nil
	}
	var artifacts []SuperAgentArtifactMeta
	if err := json.Unmarshal([]byte(stdout), &artifacts); err != nil {
		return nil, err
	}
	for i := range artifacts {
		artifacts[i].ArtifactID = artifacts[i].Path
		artifacts[i].Previewable = superAgentArtifactPreviewable(artifacts[i].Mime)
		artifacts[i].Downloadable = true
		artifacts[i].DownloadRoute = superAgentArtifactDownloadRoute
		enrichSuperAgentArtifactMeta(&artifacts[i])
	}
	return artifacts, nil
}
