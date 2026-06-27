# Virtual Employee (agent_app) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an owner publish a configured super-agent as a reusable "virtual employee" (`agent_app` product) whose skills+dependencies are frozen into a sandbox checkpoint template at publish time; other users recruit it from the space marketplace and each gets an independent sandbox instance restored from that template, with skills read-only and artifacts isolated per user.

**Architecture:** Add `agent_app` as a real AIProduct type. Publish = freeze the agent's identity (model/prompt/capabilities/MCP/skill-set) into `ai_product.feature.agent_snapshot`, then an async job spins a throwaway sandbox, injects skills, installs declared dependencies, and `Checkpoint`s `/workspace` to `templates/agent_app/{product}/{version}.tgz`. Recruit = materialize a read-only shadow `single_agent_draft` (marked `source_product_id`). Run = the shadow agent's sandbox cold-starts by restoring the template instead of blank, mounts `/skills` read-only, and offloads artifacts under a `s{space}/u{user}` key prefix. Reclaim reuses the existing Reaper untouched.

**Tech Stack:** Go + Hertz + GORM (domain/application/handler layering), existing `backend/pkg/agentsandbox` Docker sandbox, existing `backend/domain/aiproduct` + `backend/application/aiproduct`, existing super-agent agentflow, TypeScript api-schema + React super-mode UI, Playwright on `10.10.10.226:8896`.

## Global Constraints

- Go module root: `backend/`. Run backend tests with `SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test <pkg>`.
- Reuse, do not fork: AIProduct framework (`backend/domain/aiproduct`, `backend/application/aiproduct`), sandbox manager + Reaper (`backend/pkg/agentsandbox`). Do NOT change the sandbox key algorithm or the Reaper.
- `agent_app` product type constant already exists: `entity.AIProductTypeAgentApp = "agent_app"` (`backend/domain/aiproduct/entity/product.go:27`). Do not redefine it.
- Skills are read-only in instances: `/skills` must be unwritable via BOTH a read-only mount AND a tool-layer guard. A virtual-employee instance is any agent with `source_product_id != 0`.
- Do not weaken existing skill/super-agent behavior. Existing `(connector, agent, user)` sandboxes and standard_skill products must keep working.
- Secrets (model/MCP credentials) are stored as `*_ref` references only — never freeze plaintext into snapshots, never log them.
- Commits: Conventional Commits, end body with `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. Commit only the files a task touches (the working tree has unrelated dirty files — never `git add .`).
- MVP scope: space-visibility only (global+review is out of scope for this plan); freeze only model/prompt/capabilities/MCP/skill_set (knowledge/workflow/database/plugin are out of scope).

---

## File Structure

**Create:**
- `docs/ynet-database-sql/105-agent-app-shadow-columns.sql` — adds `source_product_id`/`source_product_version` to `single_agent_draft`.
- `backend/application/aiproduct/agent_app_snapshot.go` — freeze a `single_agent_draft` into `agent_snapshot` map; pin published skill versions.
- `backend/application/aiproduct/agent_app_snapshot_test.go`
- `backend/application/aiproduct/agent_app_template_builder.go` — async template build job (inject skills → install deps → checkpoint).
- `backend/application/aiproduct/agent_app_template_builder_test.go`
- `backend/application/aiproduct/agent_app_application.go` — PublishAgentApp + RecruitAgentApp (materialize shadow draft) + GetBuildStatus orchestration.
- `backend/application/aiproduct/agent_app_application_test.go`
- `backend/api/handler/coze/super_agent_agent_app_service.go` — handlers under `/api/super-agent/agent-app/*`.
- `backend/api/handler/coze/super_agent_agent_app_service_test.go`
- `frontend/apps/coze-studio/src/pages/space-skill/agent-app-api.ts` — frontend API adapter for publish/recruit/build-status.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/publish-virtual-employee-modal.tsx` — publish modal + build-status polling.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/__tests__/publish-virtual-employee-modal.test.tsx`

**Modify:**
- `backend/domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go` — add the two shadow columns to the GORM model.
- `backend/api/model/crossdomain/singleagent/single_agent.go` — add `SourceProductID`/`SourceProductVersion` to the `SingleAgent` entity.
- `backend/pkg/agentsandbox/contract/sandbox.go` + `backend/pkg/agentsandbox/manager.go` + `backend/pkg/agentsandbox/docker/command.go` — add a cold-start "restore-from-template + read-only `/skills`" option (additive, default off).
- `backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox.go` — tool-layer guard rejecting writes under `/skills` for instances.
- `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go` — when `source_product_id != 0`, drive instance cold-start from the template + freeze capabilities.
- `backend/application/singleagent/single_agent.go` — reject config writes when `source_product_id != 0` (read-only shadow).
- `backend/application/singleagent/sandbox_artifact.go` — artifact object-key prefix `artifacts/s{space}/u{user}/...` + owner check.
- `backend/api/router/coze/api.go` — register `/api/super-agent/agent-app/*` routes.
- `backend/application/application.go` — wire the agent_app application service.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/super-config-area.tsx` — add "发布为虚拟员工" entry.
- `frontend/apps/coze-studio/src/pages/space-skill/SkillPageView.tsx` — render `agent_app` cards in space marketplace with a 招聘 button.

---

## Phase A — Data model & identity snapshot

### Task 1: Shadow columns on `single_agent_draft`

**Files:**
- Create: `docs/ynet-database-sql/105-agent-app-shadow-columns.sql`
- Modify: `backend/domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go`
- Modify: `backend/api/model/crossdomain/singleagent/single_agent.go`

**Interfaces:**
- Produces: GORM model field `SourceProductID int64` (column `source_product_id`), `SourceProductVersion string` (column `source_product_version`); entity fields `SingleAgent.SourceProductID int64`, `SingleAgent.SourceProductVersion string`.

- [ ] **Step 1: Write the migration SQL**

Create `docs/ynet-database-sql/105-agent-app-shadow-columns.sql`:

```sql
ALTER TABLE `single_agent_draft`
  ADD COLUMN `source_product_id` bigint(20) NOT NULL DEFAULT 0
    COMMENT 'agent_app product id this draft was materialized from; 0 = normal agent';
ALTER TABLE `single_agent_draft`
  ADD COLUMN `source_product_version` varchar(64) NOT NULL DEFAULT ''
    COMMENT 'pinned agent_app product version for this shadow instance';
```

- [ ] **Step 2: Verify SQL references both columns**

Run: `rg -n "source_product_id|source_product_version" docs/ynet-database-sql/105-agent-app-shadow-columns.sql`
Expected: both column names appear.

- [ ] **Step 3: Add columns to the GORM model**

In `single_agent_draft.gen.go`, find the struct (`backend/domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go:29-77`) and add two fields next to `AgentType` (read the file first to match the exact `gorm:"column:..."` tag style used by neighbors):

```go
	SourceProductID      int64  `gorm:"column:source_product_id;default:0" json:"source_product_id"`
	SourceProductVersion string `gorm:"column:source_product_version;default:''" json:"source_product_version"`
```

- [ ] **Step 4: Add fields to the crossdomain entity**

In `backend/api/model/crossdomain/singleagent/single_agent.go`, add to the `SingleAgent` struct (near `AgentType`):

```go
	SourceProductID      int64  `json:"source_product_id,omitempty"`
	SourceProductVersion string `json:"source_product_version,omitempty"`
```

- [ ] **Step 5: Confirm the package still compiles**

Run: `cd backend && go build ./domain/agent/singleagent/... ./api/model/crossdomain/singleagent/...`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add docs/ynet-database-sql/105-agent-app-shadow-columns.sql \
  backend/domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go \
  backend/api/model/crossdomain/singleagent/single_agent.go
git commit -m "feat(agent-app): add shadow-instance columns to single_agent_draft

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 2: Identity snapshot (`agent_snapshot` builder)

**Files:**
- Create: `backend/application/aiproduct/agent_app_snapshot.go`
- Test: `backend/application/aiproduct/agent_app_snapshot_test.go`

**Interfaces:**
- Consumes: a minimal agent view — define a local input struct in this file so the task is self-contained:
  ```go
  type AgentSnapshotInput struct {
      ModelID      string
      ModelParams  map[string]any
      Prompt       string
      Capabilities map[string]any            // SuperAgentToolConfig as a plain map
      MCPServers   []map[string]any
      Skills       []SnapshotSkill           // each skill + its published version
  }
  type SnapshotSkill struct {
      SkillID      int64
      Name         string
      Version      string
      PackageHash  string
  }
  ```
- Produces: `func BuildAgentSnapshot(in AgentSnapshotInput) map[string]any` returning the `agent_snapshot` shape from the spec §5.1.

- [ ] **Step 1: Write the failing test**

Create `backend/application/aiproduct/agent_app_snapshot_test.go`:

```go
package aiproduct

import (
	"testing"
)

func TestBuildAgentSnapshotFreezesSkillVersions(t *testing.T) {
	in := AgentSnapshotInput{
		ModelID:      "1",
		ModelParams:  map[string]any{"max_tokens": 8192},
		Prompt:       "you are a helper",
		Capabilities: map[string]any{"sandbox": true, "skill_manage": false},
		MCPServers:   []map[string]any{{"name": "docs", "type": "streamable_http"}},
		Skills: []SnapshotSkill{
			{SkillID: 765, Name: "pdf-tools", Version: "4", PackageHash: "sha256:abc"},
		},
	}

	snap := BuildAgentSnapshot(in)

	skills, ok := snap["skill_set"].([]map[string]any)
	if !ok || len(skills) != 1 {
		t.Fatalf("skill_set wrong: %#v", snap["skill_set"])
	}
	if skills[0]["skill_version"] != "4" || skills[0]["package_hash"] != "sha256:abc" {
		t.Fatalf("skill not pinned: %#v", skills[0])
	}
	if snap["prompt"] != "you are a helper" {
		t.Fatalf("prompt missing: %#v", snap["prompt"])
	}
	if caps := snap["capabilities"].(map[string]any); caps["skill_manage"] != false {
		t.Fatalf("capabilities not frozen: %#v", caps)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run TestBuildAgentSnapshotFreezesSkillVersions -v`
Expected: FAIL (`undefined: BuildAgentSnapshot`).

- [ ] **Step 3: Implement the builder**

Create `backend/application/aiproduct/agent_app_snapshot.go`:

```go
package aiproduct

type SnapshotSkill struct {
	SkillID     int64
	Name        string
	Version     string
	PackageHash string
}

type AgentSnapshotInput struct {
	ModelID      string
	ModelParams  map[string]any
	Prompt       string
	Capabilities map[string]any
	MCPServers   []map[string]any
	Skills       []SnapshotSkill
}

// BuildAgentSnapshot freezes a super-agent's identity into the agent_snapshot
// map stored in ai_product.feature. Skill versions are pinned so a published
// virtual employee never drifts when the source skills are later edited.
func BuildAgentSnapshot(in AgentSnapshotInput) map[string]any {
	skillSet := make([]map[string]any, 0, len(in.Skills))
	for _, s := range in.Skills {
		skillSet = append(skillSet, map[string]any{
			"skill_id":      s.SkillID,
			"name":          s.Name,
			"skill_version": s.Version,
			"package_hash":  s.PackageHash,
		})
	}
	return map[string]any{
		"model":        map[string]any{"model_id": in.ModelID, "params": in.ModelParams},
		"prompt":       in.Prompt,
		"capabilities": in.Capabilities,
		"mcp_servers":  in.MCPServers,
		"skill_set":    skillSet,
	}
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run TestBuildAgentSnapshotFreezesSkillVersions -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/application/aiproduct/agent_app_snapshot.go backend/application/aiproduct/agent_app_snapshot_test.go
git commit -m "feat(agent-app): freeze agent identity into agent_snapshot

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase B — Template builder

### Task 3: Template build job (inject skills → install deps → checkpoint)

**Files:**
- Create: `backend/application/aiproduct/agent_app_template_builder.go`
- Test: `backend/application/aiproduct/agent_app_template_builder_test.go`

**Interfaces:**
- Consumes: a narrow sandbox port so the builder is unit-testable with a fake (the real impl is `crossdomain/contract/sandbox`). Define in this file:
  ```go
  type TemplateSandbox interface {
      EnsureSandbox(ctx context.Context, key string) error
      SyncSkill(ctx context.Context, key, skillName string, files map[string]string) error
      Exec(ctx context.Context, key, cmd string, timeoutSec int) (stdout, stderr string, exit int, err error)
      Checkpoint(ctx context.Context, key, objectKey string) (contentHash string, err error)
      Destroy(ctx context.Context, key string) error
  }
  ```
  (When wiring for real in Task 5, adapt `crosssandbox.DefaultSVC()` — `backend/crossdomain/contract/sandbox/sandbox.go` — to this interface; read that file to confirm method names before adapting.)
- Produces: `func BuildTemplate(ctx context.Context, sb TemplateSandbox, req BuildTemplateRequest) (BuildTemplateResult, error)` where
  ```go
  type BuildTemplateRequest struct {
      BuildKey   string
      Skills     []BuildSkill        // name -> files (already loaded) + pip/npm deps
      ObjectKey  string              // templates/agent_app/{product}/{version}.tgz
  }
  type BuildSkill struct {
      Name     string
      Files    map[string]string
      PipDeps  []string
      NpmDeps  []string
  }
  type BuildTemplateResult struct {
      Status      string  // "ready" | "failed"
      ContentHash string
      Detail      string  // failure reason when status=failed
  }
  ```

- [ ] **Step 1: Write the failing test (happy path + failure path)**

Create `backend/application/aiproduct/agent_app_template_builder_test.go`:

```go
package aiproduct

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeSandbox struct {
	synced     []string
	execCmds   []string
	checkpoint string
	failOnCmd  string
	destroyed  bool
}

func (f *fakeSandbox) EnsureSandbox(_ context.Context, _ string) error { return nil }
func (f *fakeSandbox) SyncSkill(_ context.Context, _, name string, _ map[string]string) error {
	f.synced = append(f.synced, name)
	return nil
}
func (f *fakeSandbox) Exec(_ context.Context, _, cmd string, _ int) (string, string, int, error) {
	f.execCmds = append(f.execCmds, cmd)
	if f.failOnCmd != "" && strings.Contains(cmd, f.failOnCmd) {
		return "", "boom", 1, nil
	}
	return "ok", "", 0, nil
}
func (f *fakeSandbox) Checkpoint(_ context.Context, _, objectKey string) (string, error) {
	f.checkpoint = objectKey
	return "sha256:deadbeef", nil
}
func (f *fakeSandbox) Destroy(_ context.Context, _ string) error { f.destroyed = true; return nil }

func TestBuildTemplateInjectsSkillsInstallsDepsCheckpoints(t *testing.T) {
	fb := &fakeSandbox{}
	res, err := BuildTemplate(context.Background(), fb, BuildTemplateRequest{
		BuildKey:  "build-1",
		ObjectKey: "templates/agent_app/100/1.tgz",
		Skills: []BuildSkill{
			{Name: "pdf-tools", Files: map[string]string{"SKILL.md": "x"}, PipDeps: []string{"pypdf"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != "ready" || res.ContentHash != "sha256:deadbeef" {
		t.Fatalf("bad result: %#v", res)
	}
	if len(fb.synced) != 1 || fb.synced[0] != "pdf-tools" {
		t.Fatalf("skill not injected: %#v", fb.synced)
	}
	if fb.checkpoint != "templates/agent_app/100/1.tgz" {
		t.Fatalf("checkpoint key wrong: %s", fb.checkpoint)
	}
	if !fb.destroyed {
		t.Fatalf("build sandbox not destroyed")
	}
	joined := strings.Join(fb.execCmds, " | ")
	if !strings.Contains(joined, "pip install") || !strings.Contains(joined, "pypdf") {
		t.Fatalf("pip deps not installed: %s", joined)
	}
}

func TestBuildTemplateFailsWhenDependencyInstallFails(t *testing.T) {
	fb := &fakeSandbox{failOnCmd: "pip install"}
	res, err := BuildTemplate(context.Background(), fb, BuildTemplateRequest{
		BuildKey:  "build-2",
		ObjectKey: "templates/agent_app/100/2.tgz",
		Skills:    []BuildSkill{{Name: "x", Files: map[string]string{"SKILL.md": "x"}, PipDeps: []string{"bad"}}},
	})
	if err != nil && !errors.Is(err, nil) {
		// builder returns failure via result, not error
	}
	if res.Status != "failed" || !strings.Contains(res.Detail, "boom") {
		t.Fatalf("expected failed result with detail, got %#v", res)
	}
	if !fb.destroyed {
		t.Fatalf("build sandbox must be destroyed even on failure")
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run TestBuildTemplate -v`
Expected: FAIL (`undefined: BuildTemplate`).

- [ ] **Step 3: Implement the builder**

Create `backend/application/aiproduct/agent_app_template_builder.go`:

```go
package aiproduct

import (
	"context"
	"fmt"
	"strings"
)

type TemplateSandbox interface {
	EnsureSandbox(ctx context.Context, key string) error
	SyncSkill(ctx context.Context, key, skillName string, files map[string]string) error
	Exec(ctx context.Context, key, cmd string, timeoutSec int) (stdout, stderr string, exit int, err error)
	Checkpoint(ctx context.Context, key, objectKey string) (contentHash string, err error)
	Destroy(ctx context.Context, key string) error
}

type BuildSkill struct {
	Name    string
	Files   map[string]string
	PipDeps []string
	NpmDeps []string
}

type BuildTemplateRequest struct {
	BuildKey  string
	ObjectKey string
	Skills    []BuildSkill
}

type BuildTemplateResult struct {
	Status      string
	ContentHash string
	Detail      string
}

const buildExecTimeoutSec = 600

// BuildTemplate spins a throwaway sandbox, injects each skill's pinned files,
// installs declared pip/npm dependencies, then checkpoints /workspace to the
// template object key. The build sandbox is always destroyed.
func BuildTemplate(ctx context.Context, sb TemplateSandbox, req BuildTemplateRequest) (BuildTemplateResult, error) {
	defer func() { _ = sb.Destroy(ctx, req.BuildKey) }()

	if err := sb.EnsureSandbox(ctx, req.BuildKey); err != nil {
		return BuildTemplateResult{Status: "failed", Detail: "ensure: " + err.Error()}, nil
	}

	for _, sk := range req.Skills {
		if err := sb.SyncSkill(ctx, req.BuildKey, sk.Name, sk.Files); err != nil {
			return BuildTemplateResult{Status: "failed", Detail: "sync " + sk.Name + ": " + err.Error()}, nil
		}
		if len(sk.PipDeps) > 0 {
			cmd := "pip install " + strings.Join(sk.PipDeps, " ")
			if _, stderr, exit, err := sb.Exec(ctx, req.BuildKey, cmd, buildExecTimeoutSec); err != nil || exit != 0 {
				return BuildTemplateResult{Status: "failed", Detail: fmt.Sprintf("pip(%s): exit=%d %s %v", sk.Name, exit, stderr, err)}, nil
			}
		}
		if len(sk.NpmDeps) > 0 {
			cmd := "npm i -g " + strings.Join(sk.NpmDeps, " ")
			if _, stderr, exit, err := sb.Exec(ctx, req.BuildKey, cmd, buildExecTimeoutSec); err != nil || exit != 0 {
				return BuildTemplateResult{Status: "failed", Detail: fmt.Sprintf("npm(%s): exit=%d %s %v", sk.Name, exit, stderr, err)}, nil
			}
		}
	}

	hash, err := sb.Checkpoint(ctx, req.BuildKey, req.ObjectKey)
	if err != nil {
		return BuildTemplateResult{Status: "failed", Detail: "checkpoint: " + err.Error()}, nil
	}
	return BuildTemplateResult{Status: "ready", ContentHash: hash}, nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run TestBuildTemplate -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add backend/application/aiproduct/agent_app_template_builder.go backend/application/aiproduct/agent_app_template_builder_test.go
git commit -m "feat(agent-app): template builder (inject skills, install deps, checkpoint)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase C — Publish & recruit application + handlers

### Task 4: PublishAgentApp + RecruitAgentApp application

**Files:**
- Create: `backend/application/aiproduct/agent_app_application.go`
- Test: `backend/application/aiproduct/agent_app_application_test.go`
- Modify: `backend/application/application.go`

**Interfaces:**
- Consumes: `BuildAgentSnapshot` (Task 2), `BuildTemplate` (Task 3), the existing aiproduct domain `Service` (`backend/domain/aiproduct/service/service.go` — `SyncProduct`, `Install`), and an agent-read port + shadow-create port defined locally:
  ```go
  type AgentReader interface {
      GetDraft(ctx context.Context, agentID int64) (*AgentDraftView, error)       // model/prompt/caps/mcp/skills
  }
  type ShadowAgentWriter interface {
      CreateShadowDraft(ctx context.Context, spaceID, userID, productID int64, version string, snapshot map[string]any) (int64, error) // returns new shadow agentID
  }
  ```
  (Wire `AgentReader`/`ShadowAgentWriter` to `backend/application/singleagent` in Task 5; read `backend/domain/agent/singleagent/service/single_agent.go` for the real create/get signatures before adapting.)
- Produces: `PublishAgentApp(ctx, req PublishReq) (productID int64, version string, err error)` and `RecruitAgentApp(ctx, productID, spaceID, userID int64) (shadowAgentID int64, err error)`.

- [ ] **Step 1: Write the failing test (publish freezes + triggers build; recruit materializes read-only shadow)**

Create `backend/application/aiproduct/agent_app_application_test.go` with two tests using fakes for `AgentReader`, `ShadowAgentWriter`, the domain `Service`, and `TemplateSandbox`:

```go
package aiproduct

import (
	"context"
	"testing"
)

func TestPublishAgentAppFreezesSnapshotAndBuildsTemplate(t *testing.T) {
	app := newTestAgentAppApp(t) // helper wires fakes; build returns ready
	pid, ver, err := app.PublishAgentApp(context.Background(), PublishReq{
		AgentID: 7, SpaceID: 1, UserID: 9, Name: "PDF 助手", Version: "1",
	})
	if err != nil || pid == 0 || ver != "1" {
		t.Fatalf("publish failed: pid=%d ver=%s err=%v", pid, ver, err)
	}
	if got := app.fakeSvc.lastSyncedType; got != "agent_app" {
		t.Fatalf("product type not agent_app: %s", got)
	}
	if app.fakeSandbox.checkpoint == "" {
		t.Fatalf("template not built")
	}
}

func TestRecruitMaterializesReadOnlyShadow(t *testing.T) {
	app := newTestAgentAppApp(t)
	shadowID, err := app.RecruitAgentApp(context.Background(), 100, 1, 9)
	if err != nil || shadowID == 0 {
		t.Fatalf("recruit failed: id=%d err=%v", shadowID, err)
	}
	if app.fakeShadow.lastProductID != 100 {
		t.Fatalf("shadow not linked to product: %d", app.fakeShadow.lastProductID)
	}
}
```

(Write `newTestAgentAppApp` + the four fakes inline in the test file; each fake records its last call. `fakeSvc.lastSyncedType` captures the product type passed to `SyncProduct`; `fakeShadow.lastProductID` captures the productID passed to `CreateShadowDraft`.)

- [ ] **Step 2: Run the tests, verify they fail**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run "TestPublishAgentApp|TestRecruit" -v`
Expected: FAIL (`undefined: PublishReq` / undefined app type).

- [ ] **Step 3: Implement the application**

Create `backend/application/aiproduct/agent_app_application.go`. It holds the domain `Service`, an `AgentReader`, a `ShadowAgentWriter`, a `TemplateSandbox`, and an object-key prefix. `PublishAgentApp`: read draft → `BuildAgentSnapshot` → `SyncProduct(type=agent_app, status=building, feature=agent_snapshot)` → run `BuildTemplate` (synchronously in a goroutine in real wiring; in the test the fake returns ready) → update version `feature_snapshot.template.build_status`. `RecruitAgentApp`: `Service.Install(...)` then `ShadowAgentWriter.CreateShadowDraft(... productID, version, snapshot)`. Include the `PublishReq` struct:

```go
type PublishReq struct {
	AgentID int64
	SpaceID int64
	UserID  int64
	Name    string
	Version string
}
```

Keep all credential refs untouched (copy `*_ref` strings verbatim from the draft into the snapshot; never resolve to plaintext).

- [ ] **Step 4: Run the tests, verify they pass**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct -run "TestPublishAgentApp|TestRecruit" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/application/aiproduct/agent_app_application.go backend/application/aiproduct/agent_app_application_test.go
git commit -m "feat(agent-app): publish + recruit application orchestration

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 5: Wire real ports (agent read, shadow create, sandbox adapter) + register service

**Files:**
- Modify: `backend/application/aiproduct/agent_app_application.go`
- Modify: `backend/application/application.go`
- Modify: `backend/application/singleagent/single_agent.go` (add a `CreateShadowDraft` + `GetDraftView` method, or a thin adapter)

**Interfaces:**
- Consumes: `crosssandbox.DefaultSVC()` (`backend/crossdomain/contract/sandbox/sandbox.go`), singleagent service (`backend/domain/agent/singleagent/service/single_agent.go`), aiproduct service (already constructed in `application.go`).
- Produces: a package-level `AgentAppSVC` constructed in `application.go`, consumed by the handler in Task 6.

- [ ] **Step 1: Add a sandbox adapter** that satisfies `TemplateSandbox` by delegating to `crosssandbox.DefaultSVC()`. Read `backend/crossdomain/contract/sandbox/sandbox.go` to confirm `Exec`/`SyncSkill`/`Checkpoint` signatures, then write `backend/application/aiproduct/sandbox_adapter.go` mapping them 1:1 (Checkpoint that returns `(string, error)`; if the real one returns only `error`, compute hash separately or return "").

- [ ] **Step 2: Add `GetDraftView` + `CreateShadowDraft`** to `backend/application/singleagent/single_agent.go` (or a new `shadow_agent.go` in that package). `CreateShadowDraft` inserts a `single_agent_draft` row with `agent_type="super"`, fields filled from `snapshot`, `source_product_id=productID`, `source_product_version=version`; returns new agentID.

- [ ] **Step 3: Construct `AgentAppSVC`** in `application.go` (read `backend/application/application.go:332-335` for how sandbox + other SVCs are injected; follow that pattern), passing the aiproduct service, the singleagent adapter, and the sandbox adapter.

- [ ] **Step 4: Build the whole backend**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 5: Run the aiproduct + singleagent suites**

Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/aiproduct ./application/singleagent`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/application/aiproduct/ backend/application/application.go backend/application/singleagent/
git commit -m "feat(agent-app): wire real agent/shadow/sandbox ports + register service

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 6: HTTP handlers + routes

**Files:**
- Create: `backend/api/handler/coze/super_agent_agent_app_service.go`
- Test: `backend/api/handler/coze/super_agent_agent_app_service_test.go`
- Modify: `backend/api/router/coze/api.go`

**Interfaces:**
- Consumes: `AgentAppSVC` (Task 5).
- Produces routes: `POST /api/super-agent/agent-app/publish`, `POST /api/super-agent/agent-app/build-status`, `POST /api/super-agent/agent-app/recruit`.

- [ ] **Step 1: Write the failing handler test** asserting caller is resolved from context (not request body) and that publish returns `product_id`+`version`. Follow the existing handler-test pattern in `backend/api/handler/coze/super_agent_product_service_test.go` (read it first).

- [ ] **Step 2: Run it, verify it fails.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./api/handler/coze -run AgentApp -v`
Expected: FAIL.

- [ ] **Step 3: Implement handlers** (`CreateAgentApp`/`AgentAppBuildStatus`/`RecruitAgentApp`), resolving caller via the same context helper the neighboring super-agent handlers use; reject body `user_id` for permission.

- [ ] **Step 4: Register routes** in `backend/api/router/coze/api.go` beside the existing `/api/super-agent/...` group.

- [ ] **Step 5: Run handler tests, verify pass.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./api/handler/coze -run AgentApp -v`
Expected: PASS.

- [ ] **Step 6: Commit.**
```bash
git add backend/api/handler/coze/super_agent_agent_app_service.go backend/api/handler/coze/super_agent_agent_app_service_test.go backend/api/router/coze/api.go
git commit -m "feat(agent-app): publish/build-status/recruit HTTP handlers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase D — Instance runtime: read-only skills, template restore, artifact isolation

### Task 7: Tool-layer `/skills` write guard for instances

**Files:**
- Modify: `backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox.go`
- Test: `backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox_test.go`

**Interfaces:**
- Consumes: nothing new; reads a per-instance flag `readonlySkills bool` threaded onto the sandbox tool structs (set when `source_product_id != 0` — wired in Task 9).
- Produces: `func pathIsUnderSkills(p string) bool` and a guard that returns an error string for write tools when `readonlySkills && pathIsUnderSkills(target)`.

- [ ] **Step 1: Write the failing test**

Add to `node_tool_sandbox_test.go`:

```go
func TestSkillsPathGuardBlocksInstanceWrites(t *testing.T) {
	if !pathIsUnderSkills("/skills/pdf/SKILL.md") {
		t.Fatal("should detect /skills path")
	}
	if pathIsUnderSkills("/workspace/out.txt") {
		t.Fatal("should not flag /workspace path")
	}
	if err := guardSkillWrite(true, "/skills/pdf/x.py"); err == nil {
		t.Fatal("write under /skills must be rejected for instances")
	}
	if err := guardSkillWrite(false, "/skills/pdf/x.py"); err != nil {
		t.Fatal("non-instance writes must be allowed")
	}
}
```

- [ ] **Step 2: Run it, verify it fails.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./domain/agent/singleagent/internal/agentflow -run TestSkillsPathGuard -v`
Expected: FAIL.

- [ ] **Step 3: Implement the guard helpers** in `node_tool_sandbox.go`:

```go
func pathIsUnderSkills(p string) bool {
	cleaned := strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "./")
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned == "/skills" || strings.HasPrefix(cleaned, "/skills/")
}

func guardSkillWrite(readonlySkills bool, target string) error {
	if readonlySkills && pathIsUnderSkills(target) {
		return fmt.Errorf("permission denied: /skills is read-only for a virtual employee instance")
	}
	return nil
}
```

Then call `guardSkillWrite(t.readonlySkills, target)` at the top of `write_file`, `edit_file`, and (best-effort, parse the cwd/redirect target) `run_bash` write paths; return the error string to the model instead of executing.

- [ ] **Step 4: Run it, verify pass.** (same command) Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox.go backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox_test.go
git commit -m "feat(agent-app): tool-layer /skills read-only guard for instances

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 8: Sandbox cold-start from template + read-only `/skills` mount

**Files:**
- Modify: `backend/pkg/agentsandbox/contract/sandbox.go` (add optional `TemplateObjectKey string` + `ReadonlySkills bool` to the create/ensure request type)
- Modify: `backend/pkg/agentsandbox/manager.go` (cold-start: if `TemplateObjectKey != ""` and no existing instance checkpoint, restore from template)
- Modify: `backend/pkg/agentsandbox/docker/command.go` (when `ReadonlySkills`, append `-v {skillsHostDir}:/skills:ro`)
- Test: `backend/pkg/agentsandbox/manager_test.go` (or a new `template_coldstart_test.go`)

**Interfaces:**
- Consumes: existing `restore`/`Checkpoint` (`manager.go:293-342`).
- Produces: cold-start precedence — instance checkpoint first, else template, else blank.

- [ ] **Step 1: Write the failing test** with a fake blob store proving: (a) when instance checkpoint exists, template is NOT used; (b) when only template exists, template tgz is restored; (c) when neither, layout is created blank. Read `manager.go:84-138` and `293-342` first to mirror the existing restore call shape.

- [ ] **Step 2: Run it, verify it fails.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./pkg/agentsandbox -run Template -v`
Expected: FAIL.

- [ ] **Step 3: Implement** the additive request fields + precedence in `coldStart`, and the `:ro` mount in `command.go` (guard behind `ReadonlySkills` so existing callers are unaffected).

- [ ] **Step 4: Run focused + full package tests, verify pass.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./pkg/agentsandbox/...`
Expected: PASS (new + existing).

- [ ] **Step 5: Commit.**
```bash
git add backend/pkg/agentsandbox/
git commit -m "feat(agent-app): sandbox cold-start from template + read-only /skills mount

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 9: Drive instance build from agentflow + freeze capabilities + read-only config

**Files:**
- Modify: `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go`
- Modify: `backend/application/singleagent/single_agent.go` (reject config writes when `source_product_id != 0`)
- Test: `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder_test.go`
- Test: `backend/application/singleagent/single_agent_test.go`

**Interfaces:**
- Consumes: `SingleAgent.SourceProductID` (Task 1), template object key from the product (`templates/agent_app/{product}/{version}.tgz`), the sandbox request fields (Task 8), `readonlySkills` flag (Task 7).
- Produces: instances pass `TemplateObjectKey` + `ReadonlySkills=true` into sandbox cold-start; their `SuperAgentToolConfig` is taken from the frozen snapshot (the consumer cannot widen it); shadow drafts reject config updates.

- [ ] **Step 1: Write failing tests:** (a) builder, given a config with `SourceProductID!=0`, sets `ReadonlySkills=true` and the template key on the sandbox request and sources capabilities from the snapshot; (b) `single_agent.go` update/config-write returns an error for a draft with `SourceProductID!=0`.

- [ ] **Step 2: Run, verify fail.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./domain/agent/singleagent/internal/agentflow ./application/singleagent -run "Instance|ReadOnly|Shadow" -v`
Expected: FAIL.

- [ ] **Step 3: Implement** in `agent_flow_builder.go` (read `agent_flow_builder.go:259-297` for where super-agent skills/sandbox tools are wired; branch on `SourceProductID`) and the write-reject in `single_agent.go`.

- [ ] **Step 4: Run, verify pass.** (same command) Expected: PASS.

- [ ] **Step 5: Run super-agent regression** to prove normal agents unaffected:
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./domain/agent/singleagent/... ./application/singleagent ./domain/conversation/agentrun/service`
Expected: PASS.

- [ ] **Step 6: Commit.**
```bash
git add backend/domain/agent/singleagent/ backend/application/singleagent/single_agent.go
git commit -m "feat(agent-app): instances restore template, freeze capabilities, read-only config

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 10: Artifact object-key isolation per user/space

**Files:**
- Modify: `backend/application/singleagent/sandbox_artifact.go`
- Test: `backend/application/singleagent/sandbox_artifact_test.go`

**Interfaces:**
- Consumes: caller `spaceID`/`userID` from context (already resolved in `resolveSandboxKey`, `sandbox_artifact.go:89-110`).
- Produces: `func artifactObjectKey(spaceID, userID, productID, conversationID int64, name string) string` → `artifacts/s{space}/u{user}/p{product}/c{conv}/{name}`; download/list reject when caller != owner.

- [ ] **Step 1: Write the failing test**

```go
func TestArtifactObjectKeyIsUserSpaceScoped(t *testing.T) {
	got := artifactObjectKey(1, 9, 100, 50, "report.pdf")
	want := "artifacts/s1/u9/p100/c50/report.pdf"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
```

- [ ] **Step 2: Run, verify fail.**
Run: `cd backend && SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false go test ./application/singleagent -run TestArtifactObjectKey -v`
Expected: FAIL.

- [ ] **Step 3: Implement** `artifactObjectKey` and use it where artifacts are offloaded; add an owner check (`callerUserID == keyUserID`) in download/list.

- [ ] **Step 4: Run, verify pass.** Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
git add backend/application/singleagent/sandbox_artifact.go backend/application/singleagent/sandbox_artifact_test.go
git commit -m "feat(agent-app): isolate artifact object keys per user/space

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase E — Frontend

### Task 11: Publish-as-virtual-employee modal

**Files:**
- Create: `frontend/apps/coze-studio/src/pages/space-skill/agent-app-api.ts`
- Create: `frontend/packages/agent-ide/entry/src/modes/super-mode/publish-virtual-employee-modal.tsx`
- Test: `frontend/packages/agent-ide/entry/src/modes/super-mode/__tests__/publish-virtual-employee-modal.test.tsx`
- Modify: `frontend/packages/agent-ide/entry/src/modes/super-mode/super-config-area.tsx`

**Interfaces:**
- Consumes: routes from Task 6.
- Produces: `agentAppApi.publish({bot_id, space_id, name, version})`, `agentAppApi.buildStatus({product_id})`; a modal that calls publish then polls build-status until `ready|failed`.

- [ ] **Step 1: Write the failing render test** asserting the modal shows a name/version field, calls `publish` on submit, and renders "构建中"→"已就绪" as `buildStatus` resolves (mock `agentAppApi`). Follow the existing test pattern in `frontend/apps/coze-studio/src/pages/skill-marketplace/__tests__/index.test.tsx`.

- [ ] **Step 2: Run, verify fail.**
Run: `cd frontend && pnpm --filter @coze-studio/agent-ide-entry test -- publish-virtual-employee`
Expected: FAIL.

- [ ] **Step 3: Implement** `agent-app-api.ts` (using the same axios/api-schema pattern as `skill-marketplace/index.tsx`) and the modal; add a "发布为虚拟员工" button to `super-config-area.tsx`.

- [ ] **Step 4: Run, verify pass.** Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
git add frontend/apps/coze-studio/src/pages/space-skill/agent-app-api.ts frontend/packages/agent-ide/entry/src/modes/super-mode/publish-virtual-employee-modal.tsx frontend/packages/agent-ide/entry/src/modes/super-mode/__tests__/publish-virtual-employee-modal.test.tsx frontend/packages/agent-ide/entry/src/modes/super-mode/super-config-area.tsx
git commit -m "feat(agent-app): publish-as-virtual-employee modal + api adapter

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 12: Marketplace virtual-employee card + recruit

**Files:**
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/SkillPageView.tsx`
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/agent-app-api.ts`
- Test: `frontend/apps/coze-studio/src/pages/space-skill/__tests__/index.test.tsx`

**Interfaces:**
- Consumes: marketplace product list filtered to `type=agent_app`; `agentAppApi.recruit({product_id, space_id})` (returns `shadow_agent_id`).
- Produces: an `agent_app` card with a 招聘 button; on success navigates to the shadow agent's super-mode chat (`/space/{space}/bot/{shadow_agent_id}/...`).

- [ ] **Step 1: Write the failing test** asserting an `agent_app` product renders a 招聘 button and clicking it calls `recruit` then navigates with the returned `shadow_agent_id`.

- [ ] **Step 2: Run, verify fail.**
Run: `cd frontend && pnpm --filter app test -- space-skill`
Expected: FAIL.

- [ ] **Step 3: Implement** the card branch (when `type==='agent_app'` show 招聘 instead of 安装/编辑) + `recruit` adapter + navigation.

- [ ] **Step 4: Run, verify pass.** Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
git add frontend/apps/coze-studio/src/pages/space-skill/SkillPageView.tsx frontend/apps/coze-studio/src/pages/space-skill/agent-app-api.ts frontend/apps/coze-studio/src/pages/space-skill/__tests__/index.test.tsx
git commit -m "feat(agent-app): marketplace virtual-employee card + recruit

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase F — Integration verification on 226

### Task 13: End-to-end Playwright record

**Files:**
- Create: `docs/super-agent-virtual-employee-e2e-record.md`

- [ ] **Step 1: Apply SQL on 226** (`105-agent-app-shadow-columns.sql`), build+deploy backend binary and frontend dist (follow the deploy steps already used in this repo's super-agent deploys), restart `coze-super`, health check `curl -sS -I http://127.0.0.1:8896/`.

- [ ] **Step 2: Publish flow** — as owner, open a super-agent with a skill, click 发布为虚拟员工; confirm `/api/super-agent/agent-app/publish` 200 and build-status reaches `ready`; confirm `templates/agent_app/{product}/{ver}.tgz` exists in MinIO.

- [ ] **Step 3: Recruit + run** — as a second user, recruit from the space marketplace; open the shadow agent chat; send a task; confirm a sandbox instance cold-started from the template (no dependency install in logs); confirm `/skills` write is rejected; confirm an artifact lands under `artifacts/s{space}/u{user}/...`.

- [ ] **Step 4: Reclaim** — leave the instance idle; confirm Reaper pauses then checkpoints it; re-open and confirm fast restore.

- [ ] **Step 5: Record** results in `docs/super-agent-virtual-employee-e2e-record.md` and commit.

---

## Verification Commands

Backend focused:
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false \
  go test ./application/aiproduct/... ./application/singleagent ./api/handler/coze ./pkg/agentsandbox/... ./domain/agent/singleagent/...
```

Backend regression:
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent MOCKEY_CHECK_GCFLAGS=false \
  go test ./domain/conversation/agentrun/service ./api/router/coze
```

Frontend:
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
pnpm --filter @coze-studio/agent-ide-entry test -- publish-virtual-employee
pnpm --filter app test -- space-skill
```

## Acceptance Criteria (mirrors spec §14)

- A super-agent publishes to an `agent_app` product; async build produces a template checkpoint (`build_status=ready`).
- A second user recruits from the space marketplace; a read-only shadow `single_agent_draft` is materialized.
- The shadow agent's session cold-starts by restoring the template (no dependency install at run time).
- Writes under `/skills` are rejected (read-only mount + tool guard); `/workspace` and `/outputs` are writable.
- Artifacts land under `artifacts/s{space}/u{user}/...` and are not visible cross-user.
- Idle instance is reaped + checkpointed; re-use restores quickly.
- Existing skill products and normal/super-agent sessions still pass their tests.
