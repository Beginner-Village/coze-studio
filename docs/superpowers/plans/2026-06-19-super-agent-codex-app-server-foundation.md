# Super Agent Codex App Server Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first isolated foundation for the Codex-like super agent: a `/api/super-agent/*` workspace service, persisted standard skill folders, and front-end polish verified against the deployed 8896 environment.

**Architecture:** Keep all new behavior behind the existing super-agent/sandbox boundary. Reuse the current sandbox manager and skill domain service, but expose a separate super-agent App Server route surface and extend skill snapshots so standard folder skills are first-class data. Frontend changes stay inside super-mode and chat rendering paths that were exercised by Playwright MCP.

**Tech Stack:** Go/Hertz/GORM backend, existing sandbox manager, TypeScript/React frontend, Vitest, Go tests, Playwright MCP for deployed smoke tests.

---

## Current Evidence

- Playwright MCP target: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`.
- Current super-mode shows `自主规划`, `独立沙箱`, `技能 & MCP`, `长期记忆`, plus workspace roots `工作区`, `产出物`, `上传区`.
- `/workspace` currently lists `.agent/`, `node_modules/`, `create_excel.py`, `glm5_model_comparison.md`, `generate_glm52_doc.js`, `primes.txt`, `primes.py`, `package-lock.json`, `package.json`, `generate_glm_news.js`, `hello.txt`.
- Skill management page has 4 skills (`pptx`, `pdf`, `docx`, `xlsx`) but cards render full `SKILL.md` content directly.
- Skill create page is still name/description/full-instruction only; no standard folder editor, file tree, scripts, references, assets, version, visibility, or publish controls.
- Global store has `项目商店`, `智能体`, `外部应用`, `插件商店`; there is no skill marketplace entry.
- Read-only chat smoke test succeeded, but filenames such as `create_excel.py`, `doc.js`, `hello.txt` were auto-linked as external URLs. Markdown rendering must treat sandbox filenames/paths safely.
- Prior Playwright MCP console evidence showed `updateRespondingInImmer: cannot find related function call , expect index -1`; current rerun did not reproduce, so cover it with a regression test instead of broad state rewrites.

## File Structure

### Backend Super-Agent App Server

- Modify `backend/application/singleagent/sandbox_workspace.go`
  - Add reusable path-root policies.
  - Add super-agent list/read methods that can inspect `/skills`.
  - Keep upload/delete limited to mutable roots `/workspace`, `/uploads`, `/outputs`.
- Create `backend/api/handler/coze/super_agent_workspace_service.go`
  - Bind requests and call super-agent workspace methods.
  - Keep response types identical to existing sandbox workspace models.
- Modify `backend/api/router/coze/api.go`
  - Register `/api/super-agent/workspace/list`.
  - Register `/api/super-agent/workspace/read`.
  - Register `/api/super-agent/workspace/upload`.
  - Register `/api/super-agent/workspace/delete`.

### Backend Standard Skill Folders

- Modify `backend/domain/skill/entity/skill.go`
  - Add `Files map[string]string` to `SkillVersion`.
- Modify `backend/domain/skill/internal/dal/skill_version_dao.go`
  - Persist `files`.
  - Hash canonical files content with prompt.
- Modify `backend/domain/skill/internal/dal/skill_dao.go`
  - Save `files` during update when provided.
  - Snapshot `files` into `skill_version`.
- Modify `backend/application/skill/skill_application.go`
  - Let update accept `files`.
  - Parse `SKILL.md` frontmatter on update the same way create does.
- Modify `backend/api/handler/skill/skill_service.go`
  - Accept `files` in update.
  - Return `files` and `version` in skill responses.
- Modify `docs/ynet-database-sql/99-skill-version.sql`
  - Add `files JSON NULL`.
- Modify `backend/domain/skill/internal/dal/skill_dao_test.go`
  - Assert update snapshots preserve standard skill files.

### Frontend Super-Mode And Chat Rendering

- Modify `frontend/packages/arch/idl/src/auto-generated/developer_api/index.ts`
  - Add `SuperAgentListWorkspaceFiles`, `SuperAgentReadWorkspaceFile`, `SuperAgentUploadWorkspaceFile`, `SuperAgentDeleteWorkspaceFile`.
- Modify `frontend/packages/arch/idl/src/auto-generated/developer_api/namespaces/developer_api.ts`
  - Reuse or alias the existing sandbox workspace request/response interfaces.
- Modify `frontend/packages/agent-ide/entry/src/modes/super-mode/sandbox-workspace.tsx`
  - Call super-agent workspace APIs.
  - Add read-only `/skills` root.
  - Hide delete/upload affordances when root starts with `/skills`.
  - Render Markdown preview inside an accessible article wrapper.
- Modify `frontend/packages/common/chat-area/chat-area/src/store/waiting.ts`
  - For orphan `tool_response` indices, do not log an error or mutate unrelated response state.
  - Remove noisy debug logs from this hot path.
- Modify `frontend/packages/common/chat-area/chat-area/__tests__/waiting.test.ts`
  - Add regression for `tool_response` index `-1`.
- Locate the Markdown renderer for chat output before editing
  - If auto-linking is controlled by `LazyCozeMdBox`, add a super-mode/chat answer option that disables bare URL linkification for filename-like tokens.
  - If the behavior is inside a shared renderer, add a tiny sanitizer for sandbox filename lists and test it in that package.

## Task 1: Add Super-Agent Workspace App Server Routes

**Files:**
- Modify: `backend/application/singleagent/sandbox_workspace.go`
- Create: `backend/api/handler/coze/super_agent_workspace_service.go`
- Modify: `backend/api/router/coze/api.go`

- [ ] **Step 1: Write the root-policy helper in `sandbox_workspace.go`**

Add the helper near `sandboxAllowedRoots`:

```go
var sandboxAllowedRoots = []string{"/workspace", "/uploads", "/outputs"}
var sandboxSuperAgentReadableRoots = []string{"/workspace", "/uploads", "/outputs", "/skills"}

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

func sanitizeSandboxPath(p string) (string, error) {
	return sanitizeSandboxPathWithRoots(p, sandboxAllowedRoots)
}

func sanitizeSuperAgentReadablePath(p string) (string, error) {
	return sanitizeSandboxPathWithRoots(p, sandboxSuperAgentReadableRoots)
}
```

- [ ] **Step 2: Refactor list/read internals to accept a sanitizer**

Introduce private helpers and keep the existing public methods unchanged:

```go
type sandboxPathSanitizer func(string) (string, error)

func (s *SingleAgentApplicationService) listSandboxFilesWithSanitizer(ctx context.Context, req *developer_api.ListSandboxFilesRequest, sanitize sandboxPathSanitizer, ensureDirs string) (*developer_api.ListSandboxFilesResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitize(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	_, _ = svc.Exec(ctx, key, ensureDirs, 20)
	return s.listSandboxFilesByPath(ctx, svc, key, path)
}
```

Move the existing Python `os.listdir` block into:

```go
func (s *SingleAgentApplicationService) listSandboxFilesByPath(ctx context.Context, svc crosssandbox.Manager, key string, path string) (*developer_api.ListSandboxFilesResponse, error) {
	py := "import os,json,sys\n" +
		"p=sys.argv[1]\n" +
		"out=[]\n" +
		"for n in sorted(os.listdir(p)):\n" +
		"  fp=os.path.join(p,n)\n" +
		"  try:\n" +
		"    st=os.stat(fp)\n" +
		"    out.append({'name':n,'is_dir':os.path.isdir(fp),'size':(0 if os.path.isdir(fp) else st.st_size),'mtime':int(st.st_mtime)})\n" +
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
			Mtime int64  `json:"mtime"`
		}
		if jErr := json.Unmarshal([]byte(stdout), &raw); jErr == nil {
			for _, r := range raw {
				fp := strings.TrimRight(path, "/") + "/" + r.Name
				files = append(files, &developer_api.SandboxFileInfo{
					Name: r.Name,
					Path: fp,
					IsDir: r.IsDir,
					Size: r.Size,
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
```

Add the read helper:

```go
func (s *SingleAgentApplicationService) readSandboxFileWithSanitizer(ctx context.Context, req *developer_api.ReadSandboxFileRequest, sanitize sandboxPathSanitizer) (*developer_api.ReadSandboxFileResponse, error) {
	svc, key, err := s.resolveSandboxKey(ctx, req.BotID, req.ConnectorID)
	if err != nil {
		return nil, err
	}
	path, err := sanitize(req.Path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", err.Error()))
	}
	return s.readSandboxFileByPath(ctx, svc, key, path)
}
```

Move the existing `ReadSandboxFile` body after `svc.ReadFile` into:

```go
func (s *SingleAgentApplicationService) readSandboxFileByPath(ctx context.Context, svc crosssandbox.Manager, key string, path string) (*developer_api.ReadSandboxFileResponse, error) {
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil {
		return nil, errorx.New(errno.ErrAgentInvalidParamCode, errorx.KV("msg", fmt.Sprintf("read failed: %v", err)))
	}
	total := int64(len(b))
	isText := utf8.Valid(b)
	data := &developer_api.ReadSandboxFileData{Path: path, TotalSize: total}
	if total > sandboxReadMaxBytes {
		data.IsTruncated = true
		if isText {
			b = b[:sandboxReadMaxBytes]
			data.Content = string(b)
			data.IsBinary = false
			data.Size = int64(len(b))
		} else {
			data.IsBinary = true
			data.Size = 0
		}
		return &developer_api.ReadSandboxFileResponse{Code: 0, Data: data}, nil
	}
	data.Size = total
	if isText {
		data.Content = string(b)
		data.IsBinary = false
	} else {
		data.Content = base64.StdEncoding.EncodeToString(b)
		data.IsBinary = true
	}
	return &developer_api.ReadSandboxFileResponse{Code: 0, Data: data}, nil
}
```

- [ ] **Step 3: Add super-agent public service methods**

Append these methods after `ReadSandboxFile`:

```go
func (s *SingleAgentApplicationService) ListSuperAgentWorkspaceFiles(ctx context.Context, req *developer_api.ListSandboxFilesRequest) (*developer_api.ListSandboxFilesResponse, error) {
	return s.listSandboxFilesWithSanitizer(ctx, req, sanitizeSuperAgentReadablePath, "mkdir -p /workspace /uploads /outputs /skills")
}

func (s *SingleAgentApplicationService) ReadSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.ReadSandboxFileRequest) (*developer_api.ReadSandboxFileResponse, error) {
	return s.readSandboxFileWithSanitizer(ctx, req, sanitizeSuperAgentReadablePath)
}

func (s *SingleAgentApplicationService) UploadSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.UploadSandboxFileRequest) (*developer_api.UploadSandboxFileResponse, error) {
	return s.UploadSandboxFile(ctx, req)
}

func (s *SingleAgentApplicationService) DeleteSuperAgentWorkspaceFile(ctx context.Context, req *developer_api.DeleteSandboxFileRequest) (*developer_api.DeleteSandboxFileResponse, error) {
	return s.DeleteSandboxFile(ctx, req)
}
```

- [ ] **Step 4: Add the handler file**

Create `backend/api/handler/coze/super_agent_workspace_service.go`:

```go
package coze

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
)

func SuperAgentListWorkspaceFiles(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ListSandboxFilesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ListSuperAgentWorkspaceFiles(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

func SuperAgentReadWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ReadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ReadSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

func SuperAgentUploadWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.UploadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.UploadSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

func SuperAgentDeleteWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.DeleteSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.DeleteSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}
```

- [ ] **Step 5: Register the routes**

In `backend/api/router/coze/api.go`, inside the existing `_api := root.Group("/api", _apiMw()...)` block, add:

```go
{
	_superAgent := _api.Group("/super-agent", _draftbotMw()...)
	{
		_workspace := _superAgent.Group("/workspace")
		_workspace.POST("/list", coze.SuperAgentListWorkspaceFiles)
		_workspace.POST("/read", coze.SuperAgentReadWorkspaceFile)
		_workspace.POST("/upload", coze.SuperAgentUploadWorkspaceFile)
		_workspace.POST("/delete", coze.SuperAgentDeleteWorkspaceFile)
	}
}
```

- [ ] **Step 6: Run backend compile tests**

Run:

```bash
go test ./backend/application/singleagent ./backend/api/handler/coze
```

Expected: packages compile and tests pass.

## Task 2: Persist Standard Skill Files Across Update And Version Snapshots

**Files:**
- Modify: `backend/domain/skill/entity/skill.go`
- Modify: `backend/domain/skill/internal/dal/skill_version_dao.go`
- Modify: `backend/domain/skill/internal/dal/skill_dao.go`
- Modify: `backend/application/skill/skill_application.go`
- Modify: `backend/api/handler/skill/skill_service.go`
- Modify: `docs/ynet-database-sql/99-skill-version.sql`
- Modify: `backend/domain/skill/internal/dal/skill_dao_test.go`

- [ ] **Step 1: Extend the version entity**

In `backend/domain/skill/entity/skill.go`, add this field to `SkillVersion`:

```go
Files map[string]string
```

- [ ] **Step 2: Extend the skill version table SQL**

In `docs/ynet-database-sql/99-skill-version.sql`, add `files` after `prompt`:

```sql
  `files` json NULL COMMENT 'standard skill folder files snapshot',
```

- [ ] **Step 3: Persist files in version PO**

In `backend/domain/skill/internal/dal/skill_version_dao.go`, add `encoding/json` and change the hash helper:

```go
func contentHash(prompt string, files map[string]string) string {
	payload := struct {
		Prompt string            `json:"prompt"`
		Files  map[string]string `json:"files,omitempty"`
	}{
		Prompt: prompt,
		Files:  files,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
```

Add `Files *string` to `skillVersionPO`:

```go
Files *string `gorm:"column:files"`
```

In `createVersionFromSkill`, decode current skill files and hash them:

```go
files := map[string]string(nil)
if po.Files != nil && *po.Files != "" {
	_ = json.Unmarshal([]byte(*po.Files), &files)
}
v := &entity.SkillVersion{
	SkillID:     po.SkillID,
	Version:     po.Version,
	Name:        po.Name,
	Prompt:      prompt,
	Files:       files,
	IconURI:     po.IconURI,
	ContentHash: contentHash(prompt, files),
	CreatedAt:   now,
}
```

In `skillVersionDo2po`, marshal files when non-nil:

```go
if v.Files != nil {
	if b, err := json.Marshal(v.Files); err == nil {
		s := string(b)
		po.Files = &s
	}
}
```

In `skillVersionPo2do`, unmarshal files:

```go
if po.Files != nil && *po.Files != "" {
	files := map[string]string{}
	if err := json.Unmarshal([]byte(*po.Files), &files); err == nil {
		v.Files = files
	}
}
```

- [ ] **Step 4: Save files during skill update**

In `backend/domain/skill/internal/dal/skill_dao.go`, inside `Update`, add:

```go
if skill.Files != nil {
	if b, err := json.Marshal(skill.Files); err == nil {
		updates["files"] = string(b)
	}
}
```

In `do2po`, change the files branch to preserve an explicitly empty map:

```go
if do.Files != nil {
	if b, err := json.Marshal(do.Files); err == nil {
		s := string(b)
		po.Files = &s
	}
}
```

- [ ] **Step 5: Let application update parse `SKILL.md`**

Change the signature in `backend/application/skill/skill_application.go`:

```go
func (s *SkillApplicationService) UpdateSkill(ctx context.Context, skillID, spaceID int64, name, description, prompt, iconURI string, files map[string]string) (*entity.Skill, error) {
```

Before constructing `skill`, add:

```go
if md, ok := files["SKILL.md"]; ok && md != "" {
	fmName, fmDesc, body := parseSkillFrontmatter(md)
	if name == "" {
		name = fmName
	}
	if description == "" {
		description = fmDesc
	}
	if prompt == "" {
		prompt = body
	}
}
```

And set files on the domain entity:

```go
Files: files,
```

- [ ] **Step 6: Let API update accept and return files/version**

In `backend/api/handler/skill/skill_service.go`, add `Files` to `updateSkillRequest`:

```go
Files map[string]string `json:"files"`
```

Add `Files` and `Version` to `skillInfoResponse`:

```go
Files   map[string]string `json:"files,omitempty"`
Version int64             `json:"version"`
```

Return them in `entityToResponse`:

```go
Files:   s.Files,
Version: s.Version,
```

Update the application call:

```go
skill, err := skillApp.SkillApplicationSVC.UpdateSkill(ctx, req.SkillID, req.SpaceID, req.Name, req.Description, req.Prompt, req.IconURI, req.Files)
```

- [ ] **Step 7: Add DAO regression assertions**

In `backend/domain/skill/internal/dal/skill_dao_test.go`, change the update assertions to:

```go
filesV2 := map[string]string{
	"SKILL.md":              "---\nname: demo\ndescription: Demo skill\n---\n# Demo\nUse v2.",
	"references/guide.md":   "# Guide\n",
	"scripts/run.sh":        "echo ok\n",
	"assets/example.txt":    "asset\n",
}
err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v2 prompt", Files: filesV2})
assert.NoError(t, err)

got, err = dao.Get(ctx, skillID)
assert.NoError(t, err)
assert.Equal(t, int64(2), got.Version)
assert.Equal(t, "v2 prompt", got.Prompt)
assert.Equal(t, filesV2, got.Files)

v2, err := dao.GetVersion(ctx, skillID, 2)
assert.NoError(t, err)
assert.NotNil(t, v2)
assert.Equal(t, int64(2), v2.Version)
assert.Equal(t, "v2 prompt", v2.Prompt)
assert.Equal(t, filesV2, v2.Files)
assert.Equal(t, contentHash("v2 prompt", filesV2), v2.ContentHash)
```

For the second update:

```go
err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v3 prompt", Files: map[string]string{"SKILL.md": "# V3\n"}})
assert.NoError(t, err)
latest, err = dao.GetLatestVersion(ctx, skillID)
assert.NoError(t, err)
assert.Equal(t, int64(3), latest.Version)
assert.Equal(t, map[string]string{"SKILL.md": "# V3\n"}, latest.Files)
```

- [ ] **Step 8: Run skill tests**

Run:

```bash
go test ./backend/domain/skill/... ./backend/application/skill
```

Expected: tests pass.

## Task 3: Wire Frontend Super-Mode To The New App Server Surface

**Files:**
- Modify: `frontend/packages/arch/idl/src/auto-generated/developer_api/index.ts`
- Modify: `frontend/packages/arch/idl/src/auto-generated/developer_api/namespaces/developer_api.ts`
- Modify: `frontend/packages/agent-ide/entry/src/modes/super-mode/sandbox-workspace.tsx`

- [ ] **Step 1: Add generated-style API methods**

In `frontend/packages/arch/idl/src/auto-generated/developer_api/index.ts`, add:

```ts
  /** POST /api/super-agent/workspace/list Super agent app-server workspace: list files */
  SuperAgentListWorkspaceFiles(
    req: developer_api.ListSandboxFilesRequest,
    options?: T,
  ): Promise<developer_api.ListSandboxFilesResponse> {
    const _req = req;
    const url = this.genBaseURL('/api/super-agent/workspace/list');
    const method = 'POST';
    const data = {
      space_id: _req['space_id'],
      bot_id: _req['bot_id'],
      path: _req['path'],
      connector_id: _req['connector_id'],
    };
    return this.request({ url, method, data }, options);
  }
```

Repeat the same generated shape for:

```ts
SuperAgentReadWorkspaceFile(req: developer_api.ReadSandboxFileRequest, options?: T): Promise<developer_api.ReadSandboxFileResponse>
SuperAgentUploadWorkspaceFile(req: developer_api.UploadSandboxFileRequest, options?: T): Promise<developer_api.UploadSandboxFileResponse>
SuperAgentDeleteWorkspaceFile(req: developer_api.DeleteSandboxFileRequest, options?: T): Promise<developer_api.DeleteSandboxFileResponse>
```

Use URLs:

```ts
'/api/super-agent/workspace/read'
'/api/super-agent/workspace/upload'
'/api/super-agent/workspace/delete'
```

- [ ] **Step 2: Add `/skills` to super-mode roots**

In `sandbox-workspace.tsx`, change `ROOTS`:

```ts
const ROOTS = [
  { key: '/workspace', label: '工作区', readonly: false },
  { key: '/outputs', label: '产出物', readonly: false },
  { key: '/uploads', label: '上传区', readonly: false },
  { key: '/skills', label: '技能', readonly: true },
];
```

Add:

```ts
const isReadonlyRoot = (path: string) =>
  ROOTS.find(root => path === root.key || path.startsWith(`${root.key}/`))
    ?.readonly ?? false;
```

- [ ] **Step 3: Call super-agent workspace APIs**

Replace the existing calls:

```ts
DeveloperApi.ListSandboxFiles
DeveloperApi.ReadSandboxFile
DeveloperApi.UploadSandboxFile
DeveloperApi.DeleteSandboxFile
```

with:

```ts
DeveloperApi.SuperAgentListWorkspaceFiles
DeveloperApi.SuperAgentReadWorkspaceFile
DeveloperApi.SuperAgentUploadWorkspaceFile
DeveloperApi.SuperAgentDeleteWorkspaceFile
```

- [ ] **Step 4: Hide destructive controls for `/skills`**

Extend `TreeNode` props:

```ts
readonly?: boolean;
```

Pass it through recursive render:

```tsx
readonly={readonly}
```

Change delete rendering:

```tsx
{!readonly ? (
  <span
    className="opacity-0 group-hover:opacity-100 shrink-0 coz-fg-dim hover:coz-fg-hglt-red"
    title="删除"
    onClick={e => {
      e.stopPropagation();
      onDelete(file);
    }}
  >
    <IcTrash size={14} />
  </span>
) : null}
```

At the upload buttons and drag/drop, guard with:

```tsx
{!isReadonlyRoot(root) ? (
  <span className="shrink-0 coz-fg-secondary hover:coz-fg-primary cursor-pointer p-[4px]" title="上传" onClick={() => fileInputRef.current?.click()}>
    <IcUpload size={16} />
  </span>
) : null}
```

And:

```ts
if (isReadonlyRoot(root)) {
  return;
}
```

inside `onDrop`.

- [ ] **Step 5: Make Markdown previews accessible**

Wrap Markdown preview in `article`:

```tsx
<article className="px-[16px] py-[12px]" aria-label={`${name} Markdown preview`}>
  <LazyCozeMdBox
    markDown={content}
    autoFixSyntax={{ autoFixEnding: false }}
  />
</article>
```

- [ ] **Step 6: Run frontend typecheck for touched package**

Run:

```bash
node common/scripts/install-run-rush.js lint:type --to @coze-studio/bot-detail
```

If this repo uses a different package name for `frontend/packages/agent-ide/entry`, find it with:

```bash
node common/scripts/install-run-rush.js list --full-path | rg "frontend/packages/agent-ide/entry"
```

Expected: typecheck passes for the package containing super-mode.

## Task 4: Fix Chat Tool-Response Regression And Filename Auto-Linking

**Files:**
- Modify: `frontend/packages/common/chat-area/chat-area/src/store/waiting.ts`
- Modify: `frontend/packages/common/chat-area/chat-area/__tests__/waiting.test.ts`
- Locate and modify the chat Markdown renderer package after searching `LazyCozeMdBox` usage.

- [ ] **Step 1: Write the orphan tool-response regression test**

Add to `waiting.test.ts`:

```ts
  it('ignores orphan tool response without console error', () => {
    const { updateResponding } = useWaitingStore.getState();
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    const respondingMessage = {
      ...llmMessage,
      type: 'tool_response',
      index: -1,
    };

    // @ts-expect-error -- test
    updateResponding(respondingMessage);

    const { responding } = useWaitingStore.getState();
    expect(responding).toBeNull();
    expect(errorSpy).not.toHaveBeenCalledWith(
      expect.stringContaining('cannot find related function call'),
    );

    errorSpy.mockRestore();
  });
```

- [ ] **Step 2: Run the test and confirm it fails before implementation**

Run:

```bash
cd frontend/packages/common/chat-area/chat-area
npm run test -- __tests__/waiting.test.ts
```

Expected before implementation: the new assertion fails because `console.error` is called.

- [ ] **Step 3: Implement the minimal state fix**

In `handleNormalPluginMessage`, replace the missing function-call branch:

```ts
  if (functionCallIndex < 0) {
    return;
  }
```

Remove the `console.error` for this branch only. Keep the `typeof curIndex !== 'number'` branch intact.

- [ ] **Step 4: Remove noisy debug logs in `updateRespondingInImmer`**

Delete these debug-only blocks:

```ts
console.log('[ChatFlow Debug] updateRespondingInImmer called:', { ... });
```

and:

```ts
if (message.type === 'verbose') {
  console.log('[ChatFlow Debug] Verbose message content check:', { ... });
}
```

Keep the actual state logic unchanged.

- [ ] **Step 5: Fix filename auto-linking**

Search:

```bash
rg "LazyCozeMdBox|markDown=|autoFixSyntax|linkify|autolink" frontend/packages/common frontend/packages/agent-ide -n
```

If chat answer output uses a shared Markdown component with a linkify option, set that option to `false` for assistant answer text in super-mode chat.

If no option exists, add a local helper in the renderer package:

```ts
const protectSandboxFilenames = (text: string) =>
  text.replace(
    /(^|[\s,，、])([A-Za-z0-9_-]+\.(?:py|js|ts|tsx|jsx|json|md|txt|csv|xlsx|docx|pptx|pdf|sh|go|yaml|yml|toml|lock))(?![A-Za-z0-9_/.-])/g,
    '$1`$2`',
  );
```

Apply it only to model answer Markdown before rendering. Do not apply it to explicit Markdown links such as `[name](https://example.com)`.

- [ ] **Step 6: Add a renderer test for filenames**

In the renderer package test file, assert that:

```ts
create_excel.py
generate_glm52_doc.js
hello.txt
package-lock.json
```

render as text/code, not as `href="https://excel.py"` or similar external links.

- [ ] **Step 7: Run chat-area tests**

Run:

```bash
cd frontend/packages/common/chat-area/chat-area
npm run test -- __tests__/waiting.test.ts
npm run lint:type
```

Expected: tests and typecheck pass.

## Task 5: Playwright MCP Acceptance On 8896

**Files:**
- No repo files should be edited by this task.
- Do not write credentials into any file or final answer.

- [ ] **Step 1: Open deployed super-agent**

Use Playwright MCP:

```text
browser_navigate("http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange")
browser_resize({ width: 1440, height: 1000 })
browser_snapshot({ depth: 7 })
```

Expected:
- Page title contains `演示超级体`.
- Header contains `单 Agent （自主规划模式）`.
- Ability strip contains `自主规划`, `独立沙箱`, `技能 & MCP`, `长期记忆`.

- [ ] **Step 2: Verify workspace roots**

Click each root:

```text
工作区
产出物
上传区
技能
```

Expected:
- `工作区` lists sandbox files.
- `产出物` lists generated artifacts.
- `技能` lists `/skills/<skill-name>/SKILL.md` or skill directories when skills are bound.
- Delete/upload controls are hidden or disabled for `技能`.

- [ ] **Step 3: Verify read-only chat smoke**

Send:

```text
请只读取并列出 /workspace 下的文件名，不要创建、删除或修改任何文件；回答只输出文件名列表。
```

Expected:
- A tool call is shown.
- The assistant returns filenames.
- Filenames are not converted to external links such as `https://excel.py`.

- [ ] **Step 4: Check console**

Use:

```text
browser_console_messages({ level: "warning", all: false })
```

Expected:
- No `updateRespondingInImmer: cannot find related function call` error.
- No new frontend error during the read-only smoke.

- [ ] **Step 5: Verify skill management current state**

Open:

```text
http://10.10.10.226:8896/space/7652614054615187456/skills
```

Expected after this foundation:
- Skill cards show concise metadata and version.
- Details or edit page can expose standard `files`.
- Full `SKILL.md` content is not dumped into the card grid.

## Verification Commands

Run backend tests:

```bash
go test ./backend/domain/skill/... ./backend/application/skill ./backend/application/singleagent ./backend/api/handler/coze
```

Run frontend tests/typecheck:

```bash
cd frontend/packages/common/chat-area/chat-area
npm run test -- __tests__/waiting.test.ts
npm run lint:type
```

Run super-mode package typecheck after identifying package name:

```bash
node common/scripts/install-run-rush.js list --full-path | rg "frontend/packages/agent-ide/entry"
node common/scripts/install-run-rush.js lint:type --to <agent-ide-entry-package-name>
```

Run Playwright MCP acceptance steps in Task 5.

## Scope Gaps For Later Plans

- Full skill publishing workflow: personal draft, space publish, global publish review, official badge, install/fork/update semantics.
- Global skill marketplace UI and backend listing/favorite/install APIs.
- Long-running harness job lifecycle: task queue, resumable sessions, branch/patch artifacts, external App Server auth tokens, and webhook callbacks.
- Asset library beyond sandbox files: ownership, retention, provenance, artifact indexing, thumbnails, and sharing.
- Subagent orchestration UI and server-side durable task scheduling.
