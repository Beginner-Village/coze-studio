# AIProduct Runtime Marketplace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the second phase productization layer for the super-agent platform: model products, MCP products, standard skill products, space/global assets, session runtime configuration, permissions, audit logs, versioning, installation, uninstallation, and upgrade strategy.

**Architecture:** Add a lightweight `AIProduct` domain beside the existing `skill`, `singleagent`, and `conversation` domains. The first working vertical slice wraps existing standard skills as products, then reuses the same product/install/runtime resolver contracts for model and MCP products. Session creation stores runtime product IDs, the runtime resolver expands them into model/MCP/skill assets, and harness/sandbox code consumes the resolved snapshot.

**Tech Stack:** Go, Hertz, GORM, existing Coze Studio domain/application/handler layering, existing skill package implementation, existing super-agent sessions/harness APIs, TypeScript API schema, React super-mode configuration UI, Playwright MCP verification on `10.10.10.226:8896`.

---

## Current Baseline

The first phase has already stabilized these foundations:

- Super-agent App Server routes: manifest, OpenAPI, sessions, runs, messages, traces, approvals, artifacts, workspace, harness state/snapshot/resume.
- Standard skill package basics: `SKILL.md` folder skills, ZIP validation/import/export, assets, publish/review, space/global marketplace, runtime import.
- Session management UI: create, list, switch, rename, delete.
- Harness context compaction exists in `backend/application/singleagent/sandbox_workspace.go` and `backend/domain/conversation/agentrun/service/agent_run_impl.go`.

The second phase should not replace those implementations. It should wrap them with a company-level product layer.

## Scope Decision

Recommended scope for the next implementation goal:

1. Build `AIProduct` foundation and make standard skills the first product type.
2. Add installation/version/audit data and APIs.
3. Add session runtime config that references product IDs.
4. Add a runtime resolver that returns a resolved model/MCP/skill snapshot.
5. Connect the resolver to harness state/snapshot and skill injection.
6. Add frontend product pickers and marketplace data source changes after backend contracts are stable.

Model and MCP product types should be included in the schema now, but their runtime resolution can initially support metadata-only records plus one integration test path. The first production-quality vertical slice is standard skill products.

## Product Model

### Product Types

Use these enum values in Go and frontend DTOs:

```go
type AIProductType string

const (
	AIProductTypeModel         AIProductType = "model"
	AIProductTypeMCPServer     AIProductType = "mcp_server"
	AIProductTypeStandardSkill AIProductType = "standard_skill"
	AIProductTypeAgentApp      AIProductType = "agent_app"
)
```

### Visibility And Status

```go
type AIProductVisibility string

const (
	AIProductVisibilityPrivate AIProductVisibility = "private"
	AIProductVisibilitySpace   AIProductVisibility = "space"
	AIProductVisibilityGlobal  AIProductVisibility = "global"
)

type AIProductStatus string

const (
	AIProductStatusDraft      AIProductStatus = "draft"
	AIProductStatusReviewing  AIProductStatus = "reviewing"
	AIProductStatusPublished  AIProductStatus = "published"
	AIProductStatusDeprecated AIProductStatus = "deprecated"
	AIProductStatusArchived   AIProductStatus = "archived"
)

type AIProductInstallationStatus string

const (
	AIProductInstallationActive    AIProductInstallationStatus = "active"
	AIProductInstallationDisabled  AIProductInstallationStatus = "disabled"
	AIProductInstallationUninstalled AIProductInstallationStatus = "uninstalled"
)
```

### Feature JSON Shapes

`ai_product.feature` carries product-type-specific metadata. Keep it display-safe and do not store plaintext credentials.

Standard skill:

```json
{
  "skill_id": "765...",
  "skill_version": "4",
  "package_hash": "sha256:...",
  "file_count": 12,
  "asset_count": 4,
  "entry_file": "SKILL.md",
  "categories": ["document", "office"],
  "tags": ["pdf", "excel"],
  "capabilities": ["read_files", "write_files", "execute_scripts"]
}
```

Model:

```json
{
  "provider": "openai-compatible",
  "model_id": "glm-5.2",
  "base_url_ref": "space_secret:model_base_url",
  "credential_ref": "space_secret:model_api_key",
  "context_window": 200000,
  "capabilities": ["text", "tool_call", "reasoning"]
}
```

MCP server:

```json
{
  "server_name": "company-docs",
  "transport": "streamable_http",
  "endpoint_ref": "space_secret:mcp_company_docs_url",
  "headers_policy": "space_secret:mcp_company_docs_headers",
  "tools_schema_hash": "sha256:...",
  "capabilities": ["search", "fetch"]
}
```

## New Files And Responsibilities

Create:

- `docs/ynet-database-sql/104-ai-product-foundation.sql`: schema for products, versions, installations, audit logs, and session runtime config.
- `backend/domain/aiproduct/entity/product.go`: product, version, installation, audit, runtime config entity types and constants.
- `backend/domain/aiproduct/repository/repository.go`: repository interfaces.
- `backend/domain/aiproduct/internal/dal/product_dao.go`: GORM models and query implementation for product/version/installation/audit.
- `backend/domain/aiproduct/service/service.go`: domain service interface.
- `backend/domain/aiproduct/service/service_impl.go`: domain rules for product visibility, install, uninstall, upgrade, and audit.
- `backend/domain/aiproduct/service/service_test.go`: domain service unit tests.
- `backend/application/aiproduct/product_application.go`: application orchestration, caller permission checks, skill adapter integration.
- `backend/application/aiproduct/runtime_resolver.go`: resolves session runtime config into concrete model/MCP/skill assets.
- `backend/application/aiproduct/product_application_test.go`: application tests for skill sync, installation, and resolver behavior.
- `backend/api/handler/coze/super_agent_product_service.go`: product and marketplace handlers under `/api/super-agent/products` and `/api/super-agent/marketplace/products`.
- `backend/api/handler/coze/super_agent_session_runtime_service.go`: session runtime get/update handlers if runtime config is updated after session creation.
- `frontend/packages/arch/api-schema/src/idl/aiproduct/aiproduct.ts`: frontend request/response types.
- `frontend/packages/arch/api-schema/__tests__/aiproduct.test.ts`: schema export tests.
- `frontend/apps/coze-studio/src/pages/skill-marketplace/api-products.ts`: marketplace API adapter.
- `frontend/apps/coze-studio/src/pages/skill-marketplace/__tests__/ai-product-marketplace.test.tsx`: marketplace rendering contract tests.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/runtime-config.ts`: shared frontend runtime config types/helpers.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/__tests__/runtime-config.test.ts`: runtime config tests.

Modify:

- `backend/api/router/coze/api.go`: register product, marketplace product, and session runtime routes beside current super-agent routes.
- `backend/application/application.go`: initialize `aiproduct` application service.
- `backend/application/skill/skill_application.go`: call skill-to-product sync after create/update/publish/review/install flows.
- `backend/api/handler/skill/skill_service.go`: include product ID in skill marketplace responses where safe.
- `backend/api/handler/coze/super_agent_session_service.go`: accept and return `runtime_config` on session create/get/list.
- `backend/application/singleagent/sandbox_skill.go`: inject installed/resolved skill products into sandbox skill directories.
- `backend/application/singleagent/sandbox_workspace.go`: include `runtime_assets` in harness state/snapshot/resume payloads.
- `frontend/apps/coze-studio/src/pages/skill-marketplace/index.tsx`: render marketplace from product API.
- `frontend/apps/coze-studio/src/pages/space-skill/index.tsx`: show installed product status and upgrade prompts.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/super-config-area.tsx`: model/MCP/skill product selectors.
- `frontend/packages/agent-ide/entry/src/modes/super-mode/super-session-sidebar.tsx`: display runtime badges and keep session create payload wired to runtime config.

## Database Contract

### Task 1: Add AIProduct Tables

**Files:**

- Create: `docs/ynet-database-sql/104-ai-product-foundation.sql`
- Test: `backend/domain/aiproduct/service/service_test.go`

- [ ] **Step 1: Write schema migration**

Use these tables:

```sql
CREATE TABLE IF NOT EXISTS `ai_product` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint NOT NULL,
  `space_id` bigint NOT NULL DEFAULT 0,
  `creator_id` bigint NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `description` text,
  `type` varchar(64) NOT NULL,
  `status` varchar(64) NOT NULL,
  `visibility` varchar(64) NOT NULL,
  `icon_uri` varchar(1024) NOT NULL DEFAULT '',
  `cover_uri` varchar(1024) NOT NULL DEFAULT '',
  `document` mediumtext,
  `feature` json,
  `source_ref_type` varchar(64) NOT NULL DEFAULT '',
  `source_ref_id` bigint NOT NULL DEFAULT 0,
  `latest_version` varchar(64) NOT NULL DEFAULT '',
  `published_version` varchar(64) NOT NULL DEFAULT '',
  `official` tinyint(1) NOT NULL DEFAULT 0,
  `featured` tinyint(1) NOT NULL DEFAULT 0,
  `install_count` bigint NOT NULL DEFAULT 0,
  `download_count` bigint NOT NULL DEFAULT 0,
  `created_at` bigint NOT NULL DEFAULT 0,
  `updated_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_ai_product_product_id` (`product_id`),
  KEY `idx_ai_product_source` (`source_ref_type`, `source_ref_id`),
  KEY `idx_ai_product_market` (`type`, `visibility`, `status`, `updated_at`),
  KEY `idx_ai_product_space` (`space_id`, `type`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ai_product_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint NOT NULL,
  `version` varchar(64) NOT NULL,
  `source_version` varchar(64) NOT NULL DEFAULT '',
  `status` varchar(64) NOT NULL,
  `review_status` varchar(64) NOT NULL DEFAULT '',
  `review_note` text,
  `reviewer_id` bigint NOT NULL DEFAULT 0,
  `content_hash` varchar(128) NOT NULL DEFAULT '',
  `feature_snapshot` json,
  `published_at` bigint NOT NULL DEFAULT 0,
  `created_at` bigint NOT NULL DEFAULT 0,
  `updated_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_ai_product_version` (`product_id`, `version`),
  KEY `idx_ai_product_version_status` (`product_id`, `status`, `published_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ai_product_installation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `installation_id` bigint NOT NULL,
  `product_id` bigint NOT NULL,
  `product_version` varchar(64) NOT NULL,
  `target_space_id` bigint NOT NULL,
  `target_user_id` bigint NOT NULL DEFAULT 0,
  `installed_by` bigint NOT NULL DEFAULT 0,
  `status` varchar(64) NOT NULL,
  `install_mode` varchar(64) NOT NULL DEFAULT 'space',
  `runtime_config` json,
  `created_at` bigint NOT NULL DEFAULT 0,
  `updated_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_ai_product_installation` (`product_id`, `target_space_id`, `target_user_id`),
  KEY `idx_ai_product_installation_space` (`target_space_id`, `status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ai_product_audit_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `audit_id` bigint NOT NULL,
  `product_id` bigint NOT NULL DEFAULT 0,
  `installation_id` bigint NOT NULL DEFAULT 0,
  `space_id` bigint NOT NULL DEFAULT 0,
  `user_id` bigint NOT NULL DEFAULT 0,
  `action` varchar(64) NOT NULL,
  `target_type` varchar(64) NOT NULL DEFAULT '',
  `target_id` varchar(128) NOT NULL DEFAULT '',
  `detail` json,
  `created_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_ai_product_audit_id` (`audit_id`),
  KEY `idx_ai_product_audit_product` (`product_id`, `created_at`),
  KEY `idx_ai_product_audit_space` (`space_id`, `created_at`),
  KEY `idx_ai_product_audit_user` (`user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `super_agent_session_runtime_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` bigint NOT NULL,
  `agent_id` bigint NOT NULL DEFAULT 0,
  `space_id` bigint NOT NULL DEFAULT 0,
  `model_product_id` bigint NOT NULL DEFAULT 0,
  `mcp_product_ids` json,
  `skill_product_ids` json,
  `tool_policy` json,
  `context_policy` json,
  `resolved_snapshot` json,
  `created_by` bigint NOT NULL DEFAULT 0,
  `created_at` bigint NOT NULL DEFAULT 0,
  `updated_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_super_agent_session_runtime` (`conversation_id`),
  KEY `idx_super_agent_session_runtime_agent` (`agent_id`, `space_id`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- [ ] **Step 2: Run backend SQL syntax check**

Run:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
rg -n "ai_product|super_agent_session_runtime_config" docs/ynet-database-sql/104-ai-product-foundation.sql
```

Expected: every table name appears.

## Backend Domain And Application

### Task 2: Add AIProduct Entities And Repository

**Files:**

- Create: `backend/domain/aiproduct/entity/product.go`
- Create: `backend/domain/aiproduct/repository/repository.go`
- Create: `backend/domain/aiproduct/internal/dal/product_dao.go`
- Test: `backend/domain/aiproduct/service/service_test.go`

- [ ] **Step 1: Define entities**

Core structs:

```go
package entity

type Product struct {
	ID               int64
	ProductID        int64
	SpaceID          int64
	CreatorID        int64
	Name             string
	Description      string
	Type             AIProductType
	Status           AIProductStatus
	Visibility       AIProductVisibility
	IconURI          string
	CoverURI         string
	Document         string
	Feature          map[string]any
	SourceRefType    string
	SourceRefID      int64
	LatestVersion    string
	PublishedVersion string
	Official         bool
	Featured         bool
	InstallCount     int64
	DownloadCount    int64
	CreatedAt        int64
	UpdatedAt        int64
}

type ProductVersion struct {
	ID              int64
	ProductID       int64
	Version         string
	SourceVersion   string
	Status          AIProductStatus
	ReviewStatus    string
	ReviewNote      string
	ReviewerID      int64
	ContentHash     string
	FeatureSnapshot map[string]any
	PublishedAt     int64
	CreatedAt       int64
	UpdatedAt       int64
}

type ProductInstallation struct {
	ID             int64
	InstallationID int64
	ProductID      int64
	ProductVersion string
	TargetSpaceID  int64
	TargetUserID   int64
	InstalledBy    int64
	Status         AIProductInstallationStatus
	InstallMode    string
	RuntimeConfig   map[string]any
	CreatedAt      int64
	UpdatedAt      int64
}
```

- [ ] **Step 2: Define repository methods**

Repository interface:

```go
type Repository interface {
	UpsertProduct(ctx context.Context, product *entity.Product) error
	GetProduct(ctx context.Context, productID int64) (*entity.Product, error)
	GetProductBySource(ctx context.Context, sourceType string, sourceID int64) (*entity.Product, error)
	ListProducts(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error)
	UpsertVersion(ctx context.Context, version *entity.ProductVersion) error
	ListVersions(ctx context.Context, productID int64) ([]*entity.ProductVersion, error)
	InstallProduct(ctx context.Context, installation *entity.ProductInstallation) error
	UpdateInstallation(ctx context.Context, installation *entity.ProductInstallation) error
	GetInstallation(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error)
	ListInstallations(ctx context.Context, req *entity.ListInstallationsRequest) ([]*entity.ProductInstallation, error)
	CreateAudit(ctx context.Context, audit *entity.AuditLog) error
	ListAudits(ctx context.Context, req *entity.ListAuditsRequest) ([]*entity.AuditLog, error)
}
```

- [ ] **Step 3: Run focused domain tests**

Run:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./domain/aiproduct/...
```

Expected: tests compile and pass after Tasks 2-4 are implemented.

### Task 3: Add Product Service Rules

**Files:**

- Create: `backend/domain/aiproduct/service/service.go`
- Create: `backend/domain/aiproduct/service/service_impl.go`
- Test: `backend/domain/aiproduct/service/service_test.go`

- [ ] **Step 1: Add failing tests**

Required test cases:

```go
func TestServiceInstallRejectsUnpublishedGlobalProduct(t *testing.T) {}
func TestServiceInstallPinsPublishedVersion(t *testing.T) {}
func TestServiceUpgradeMovesInstallationToLatestPublishedVersion(t *testing.T) {}
func TestServiceUninstallMarksInstallationInactive(t *testing.T) {}
func TestServiceListMarketplaceHidesPrivateAndReviewingProducts(t *testing.T) {}
func TestServiceAuditRecordsInstallUpgradeUninstall(t *testing.T) {}
```

- [ ] **Step 2: Implement service methods**

Service interface:

```go
type Service interface {
	SyncProduct(ctx context.Context, product *entity.Product, version *entity.ProductVersion) (*entity.Product, error)
	ListMarketplace(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error)
	GetVisibleProduct(ctx context.Context, productID, spaceID, userID int64) (*entity.Product, error)
	Install(ctx context.Context, productID, spaceID, userID int64, version string) (*entity.ProductInstallation, error)
	Uninstall(ctx context.Context, productID, spaceID, userID int64) error
	Upgrade(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error)
	ListInstalled(ctx context.Context, spaceID, userID int64, productType entity.AIProductType) ([]*entity.ProductInstallation, error)
	RecordAudit(ctx context.Context, audit *entity.AuditLog) error
}
```

Rules:

- `Install` accepts `published` products with `visibility = global` or `visibility = space` for the same space.
- `Install` pins `product.published_version` unless caller provides an older published version.
- `Upgrade` moves an active installation to `product.published_version`.
- `Uninstall` changes installation status to `uninstalled`; it does not delete rows.
- `ListMarketplace` returns only `published` products visible to the caller.
- All install/upgrade/uninstall/review/session-bind actions create audit rows.

### Task 4: Add Skill Product Adapter

**Files:**

- Create: `backend/application/aiproduct/product_application.go`
- Modify: `backend/application/skill/skill_application.go`
- Test: `backend/application/aiproduct/product_application_test.go`
- Test: `backend/application/skill/skill_application_test.go`

- [ ] **Step 1: Add adapter contract**

```go
type SkillProductAdapter interface {
	SyncSkillProduct(ctx context.Context, skillID int64) (*aiproductentity.Product, error)
}
```

`SyncSkillProduct` maps:

- `skill.SkillID` -> `ai_product.source_ref_id`
- `skill.PublishScope` -> `ai_product.visibility`
- `skill.ReviewStatus` + `skill.PublishScope` -> `ai_product.status`
- `skill.PublishedVersion` -> `ai_product.published_version`
- `skill.LatestVersion` or current version -> `ai_product.latest_version`
- `skill.StandardFiles` summary -> `ai_product.feature`

- [ ] **Step 2: Trigger sync from skill flows**

Call the adapter after these skill application methods complete successfully:

- `CreateSkill`
- `UpdateSkill`
- `PublishSkill`
- `ReviewSkill`
- `ImportSkillPackage`
- `InstallMarketplaceSkill`

The call must not weaken existing skill validation. If product sync fails, return the error for publish/review/install; for create/update/import, return the error only after the skill write transaction is preserved by the existing repository behavior.

- [ ] **Step 3: Test publish/review sync**

Test expected states:

- Space published skill becomes `type=standard_skill`, `visibility=space`, `status=published`.
- Global pending review skill becomes `visibility=global`, `status=reviewing`.
- Global approved skill becomes `visibility=global`, `status=published`.
- Rejected global skill does not appear in product marketplace.

## Product APIs

### Task 5: Add Product And Marketplace Handlers

**Files:**

- Create: `backend/api/handler/coze/super_agent_product_service.go`
- Modify: `backend/api/router/coze/api.go`
- Create: `frontend/packages/arch/api-schema/src/idl/aiproduct/aiproduct.ts`
- Test: `backend/api/handler/coze/super_agent_product_service_test.go`
- Test: `frontend/packages/arch/api-schema/__tests__/aiproduct.test.ts`

- [ ] **Step 1: Add backend handlers**

Routes:

```text
POST /api/super-agent/products/list
POST /api/super-agent/products/get
POST /api/super-agent/products/install
POST /api/super-agent/products/uninstall
POST /api/super-agent/products/upgrade
POST /api/super-agent/products/versions/list
POST /api/super-agent/products/audits/list
POST /api/super-agent/marketplace/products/list
POST /api/super-agent/marketplace/products/get
POST /api/super-agent/marketplace/products/install
```

Response item shape:

```json
{
  "product_id": "765...",
  "type": "standard_skill",
  "name": "PDF 工具箱",
  "description": "处理 PDF 的标准技能包",
  "visibility": "global",
  "status": "published",
  "published_version": "4",
  "latest_version": "4",
  "installed": true,
  "installed_version": "3",
  "upgrade_available": true,
  "official": false,
  "featured": false,
  "install_count": 12,
  "download_count": 8,
  "feature": {}
}
```

- [ ] **Step 2: Add schema tests**

Verify frontend exports:

```ts
expect(aiProductAPI.ListMarketplaceProducts).toBeDefined();
expect(aiProductAPI.InstallProduct).toBeDefined();
expect(aiProductAPI.UpgradeProduct).toBeDefined();
```

- [ ] **Step 3: Run API tests**

Run:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/handler/coze ./domain/aiproduct/... ./application/aiproduct
```

Expected: all product handler/domain/application tests pass.

## Session Runtime Configuration

### Task 6: Persist Runtime Config On Session Create/Get

**Files:**

- Modify: `backend/api/handler/coze/super_agent_session_service.go`
- Create: `backend/api/handler/coze/super_agent_session_runtime_service.go`
- Create: `backend/application/aiproduct/runtime_resolver.go`
- Test: `backend/api/handler/coze/super_agent_session_service_test.go`
- Test: `backend/application/aiproduct/product_application_test.go`

- [ ] **Step 1: Add runtime request/response types**

```go
type SuperAgentSessionRuntimeConfig struct {
	ModelProductID string   `json:"model_product_id,omitempty"`
	MCPProductIDs  []string `json:"mcp_product_ids,omitempty"`
	SkillProductIDs []string `json:"skill_product_ids,omitempty"`
	ToolPolicy     *crossagent.SuperAgentToolConfig `json:"tool_policy,omitempty"`
	ContextPolicy  *SuperAgentContextPolicy `json:"context_policy,omitempty"`
}

type SuperAgentContextPolicy struct {
	Strategy           string `json:"strategy,omitempty"`
	MaxInputTokens     int64  `json:"max_input_tokens,omitempty"`
	KeepRecentTurns    int32  `json:"keep_recent_turns,omitempty"`
	SummaryProductID   string `json:"summary_product_id,omitempty"`
	AutoCompactEnabled bool   `json:"auto_compact_enabled,omitempty"`
}
```

Add `RuntimeConfig *SuperAgentSessionRuntimeConfig` to `superAgentCreateSessionRequest` and `superAgentSessionItem`.

- [ ] **Step 2: Store runtime config**

Persist runtime config in `super_agent_session_runtime_config` keyed by `conversation_id`. Keep the existing conversation `Ext` title behavior unchanged.

- [ ] **Step 3: Add resolver output**

```go
type ResolvedRuntime struct {
	Model  *ResolvedModelProduct  `json:"model,omitempty"`
	MCPs   []*ResolvedMCPProduct   `json:"mcps,omitempty"`
	Skills []*ResolvedSkillProduct `json:"skills,omitempty"`
	Assets []*ResolvedRuntimeAsset `json:"assets,omitempty"`
	ToolPolicy map[string]any      `json:"tool_policy,omitempty"`
	ContextPolicy map[string]any   `json:"context_policy,omitempty"`
	ResolvedAt int64               `json:"resolved_at"`
}
```

Resolver rules:

- Reject private products not owned by the caller.
- Reject products not installed into the session space.
- Allow global products only after install or explicit install-on-bind policy.
- Resolve skill products to skill ID, skill version, package hash, file count, asset count, and sandbox injection path.
- Resolve model and MCP products to metadata and credential references, not plaintext secrets.

- [ ] **Step 4: Add update route**

`POST /api/super-agent/sessions/runtime/update` accepts `conversation_id`, `space_id`, `agent_id`, and `runtime_config`. It re-runs resolver, persists the resolved snapshot, and writes audit action `session_runtime_update`.

## Harness Integration

### Task 7: Expose Runtime Assets In Harness State And Snapshot

**Files:**

- Modify: `backend/application/singleagent/sandbox_workspace.go`
- Modify: `backend/application/singleagent/sandbox_skill.go`
- Test: `backend/application/singleagent/sandbox_workspace_test.go`
- Test: `backend/application/singleagent/sandbox_skill_test.go`

- [ ] **Step 1: Add runtime assets payload**

Harness state/snapshot should include:

```json
{
  "runtime_assets": {
    "model": { "product_id": "1", "name": "GLM-5.2", "model_id": "glm-5.2" },
    "mcp_servers": [
      { "product_id": "2", "server_name": "company-docs", "transport": "streamable_http" }
    ],
    "skills": [
      {
        "product_id": "3",
        "skill_id": "765...",
        "version": "4",
        "inject_path": "/workspace/.codex/skills/pdf"
      }
    ]
  }
}
```

- [ ] **Step 2: Inject resolved skills**

For every resolved skill product, copy the pinned skill version into the sandbox path:

```text
/workspace/.codex/skills/<skill-name>/SKILL.md
/workspace/.codex/skills/<skill-name>/scripts/*
/workspace/.codex/skills/<skill-name>/references/*
/workspace/.codex/skills/<skill-name>/templates/*
/workspace/.codex/skills/<skill-name>/assets/*
```

The injection must use the pinned installation version, not whatever draft version happens to be latest.

- [ ] **Step 3: Verify context compaction still works**

Run:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./application/singleagent ./domain/conversation/agentrun/service
```

Expected: existing context compaction tests still pass and new runtime asset tests pass.

## Permissions And Audit

### Task 8: Enforce Permissions

**Files:**

- Modify: `backend/application/aiproduct/product_application.go`
- Modify: `backend/api/handler/coze/super_agent_product_service.go`
- Test: `backend/application/aiproduct/product_application_test.go`

- [ ] **Step 1: Use logged-in caller**

All product handlers must resolve caller from context, matching the current super-agent config hardening. Do not trust request body `user_id` for ownership or permission decisions.

- [ ] **Step 2: Define permission matrix**

Rules:

- Product create/update/delete: creator or space manager.
- Global publish/review: platform reviewer or space/admin permission currently used by skill review.
- Install/uninstall/upgrade: target space manager or product owner for private installs.
- Session bind: user can only bind installed products in the current space, unless the product is private and owned by that user.
- Audit list: product owner, space manager, or platform reviewer.

- [ ] **Step 3: Add negative tests**

Required tests:

```go
func TestInstallRejectsCallerWithoutSpacePermission(t *testing.T) {}
func TestRuntimeResolverRejectsUninstalledGlobalProduct(t *testing.T) {}
func TestRuntimeResolverRejectsOtherUserPrivateProduct(t *testing.T) {}
func TestAuditListRejectsUnprivilegedCaller(t *testing.T) {}
```

### Task 9: Add Audit Events

**Files:**

- Modify: `backend/domain/aiproduct/service/service_impl.go`
- Modify: `backend/application/aiproduct/product_application.go`
- Test: `backend/domain/aiproduct/service/service_test.go`

- [ ] **Step 1: Emit audit events**

Actions:

```text
product_create
product_update
product_publish
product_review_approve
product_review_reject
product_install
product_uninstall
product_upgrade
session_runtime_bind
session_runtime_update
runtime_resolve
runtime_skill_inject
```

- [ ] **Step 2: Include stable details**

Audit `detail` JSON must include:

```json
{
  "product_type": "standard_skill",
  "from_version": "3",
  "to_version": "4",
  "visibility": "global",
  "conversation_id": "765...",
  "agent_id": "765..."
}
```

Do not write request headers, raw credentials, or large package file content into audit logs.

## Frontend Productization

### Task 10: Switch Skill Store To Product Data Source

**Files:**

- Create: `frontend/apps/coze-studio/src/pages/skill-marketplace/api-products.ts`
- Modify: `frontend/apps/coze-studio/src/pages/skill-marketplace/index.tsx`
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/index.tsx`
- Test: `frontend/apps/coze-studio/src/pages/skill-marketplace/__tests__/ai-product-marketplace.test.tsx`

- [ ] **Step 1: Keep visual scope from the current design**

The skill store should remain a skill-only marketplace:

- no project store submenu
- no intelligent-agent/external-app/plugin entries
- hero/header from the existing skill store redesign
- skill cards using product DTOs
- install/upgrade/uninstall actions visible per installed state

- [ ] **Step 2: Product card state mapping**

Card states:

```ts
type ProductCardState =
  | 'not_installed'
  | 'installed_current'
  | 'upgrade_available'
  | 'uninstalled'
  | 'reviewing'
  | 'deprecated';
```

- [ ] **Step 3: Run frontend tests**

Run:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
pnpm --filter @coze-studio/api-schema test -- aiproduct
pnpm --filter app test -- skill-marketplace
```

Expected: product schema and marketplace render tests pass.

### Task 11: Add Runtime Product Pickers To Super-Agent Config

**Files:**

- Create: `frontend/packages/agent-ide/entry/src/modes/super-mode/runtime-config.ts`
- Modify: `frontend/packages/agent-ide/entry/src/modes/super-mode/super-config-area.tsx`
- Modify: `frontend/packages/agent-ide/entry/src/modes/super-mode/super-session-sidebar.tsx`
- Test: `frontend/packages/agent-ide/entry/src/modes/super-mode/__tests__/runtime-config.test.ts`

- [ ] **Step 1: Add runtime config helper**

```ts
export interface SuperAgentSessionRuntimeConfig {
  model_product_id?: string;
  mcp_product_ids?: string[];
  skill_product_ids?: string[];
  tool_policy?: Record<string, unknown>;
  context_policy?: {
    strategy?: string;
    max_input_tokens?: number;
    keep_recent_turns?: number;
    summary_product_id?: string;
    auto_compact_enabled?: boolean;
  };
}
```

- [ ] **Step 2: Config modal sections**

Add sections:

- 模型产品: one selected model product.
- MCP 产品: multiple installed MCP products.
- 标准技能产品: multiple installed skill products.
- 会话上下文: auto compact enabled, keep recent turns, max input tokens.
- 工具策略: reuse current `SuperAgentToolConfig`.

- [ ] **Step 3: Session create payload**

When creating a session, include:

```json
{
  "space_id": "765...",
  "agent_id": "765...",
  "title": "新会话",
  "runtime_config": {
    "model_product_id": "1",
    "mcp_product_ids": ["2"],
    "skill_product_ids": ["3", "4"],
    "context_policy": {
      "strategy": "tool-output-offload+session-summary",
      "auto_compact_enabled": true
    }
  }
}
```

## Model And MCP Product Strategy

### Task 12: Add Metadata-Only Model/MCP Products

**Files:**

- Modify: `backend/application/aiproduct/product_application.go`
- Modify: `backend/application/aiproduct/runtime_resolver.go`
- Test: `backend/application/aiproduct/product_application_test.go`

- [ ] **Step 1: Model product create/update**

Support creating product records with `type=model` and validated feature fields:

- `provider`
- `model_id`
- `base_url_ref`
- `credential_ref`
- `context_window`
- `capabilities`

Do not expose plaintext API keys in API responses.

- [ ] **Step 2: MCP product create/update**

Support creating product records with `type=mcp_server` and validated feature fields:

- `server_name`
- `transport`
- `endpoint_ref`
- `headers_policy`
- `tools_schema_hash`
- `capabilities`

Do not connect to remote MCP servers during product list calls. Runtime resolver may validate transport/endpoint shape only.

- [ ] **Step 3: Resolver returns metadata**

Resolved model/MCP snapshots include only safe runtime metadata and secret references. Actual credential material is loaded by the runtime component that needs it.

## Installation, Upgrade, Uninstall Strategy

### Task 13: Lock Versions And Surface Upgrades

**Files:**

- Modify: `backend/domain/aiproduct/service/service_impl.go`
- Modify: `backend/api/handler/coze/super_agent_product_service.go`
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/index.tsx`
- Test: `backend/domain/aiproduct/service/service_test.go`

- [ ] **Step 1: Install pins version**

When installing without an explicit version, installation gets `product.published_version`.

- [ ] **Step 2: Upgrade compares versions**

`upgrade_available = installation.product_version != product.published_version`.

- [ ] **Step 3: Uninstall preserves history**

Uninstall changes status to `uninstalled` and keeps `product_version` and `runtime_config`.

- [ ] **Step 4: Session runtime remains stable**

Existing sessions continue to use their `resolved_snapshot`. A product upgrade does not mutate running sessions until the user updates the session runtime config.

## Remote Testing On 226

### Task 14: Verify With Local Frontend And Remote Backend

**Files:**

- Create or update: `docs/super-agent-8896-playwright-test-record.md`

- [ ] **Step 1: Check remote disk before upload**

Run:

```bash
ssh dev@10.10.10.226 'df -h / /data 2>/dev/null || df -h /'
```

Expected: do not upload large backups when `/` free space is below 8 GiB.

- [ ] **Step 2: Start local frontend against remote backend**

Use the existing project command if available in package scripts. Record the actual command and URL in `docs/super-agent-8896-playwright-test-record.md`.

- [ ] **Step 3: Playwright MCP checks**

Verify:

- `/explore/project/latest` shows skill-only marketplace.
- Product list loads from `/api/super-agent/marketplace/products/list`.
- Skill product detail shows versions and install state.
- Install creates an active installation.
- Upgrade appears after a newer published version exists.
- Super-agent config modal can select model/MCP/skill product IDs.
- Session create sends `runtime_config`.
- Session get returns `runtime_config` and `resolved_runtime`.
- Harness state returns `runtime_assets`.
- A resolved skill product appears under `.codex/skills/<name>` in workspace list.

## Verification Commands

Backend focused:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./domain/aiproduct/... ./application/aiproduct ./application/skill ./application/singleagent ./api/handler/coze ./api/handler/skill
```

Backend super-agent regression:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/router/coze ./domain/conversation/agentrun/service ./pkg/agentsandbox/...
```

Frontend typecheck:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
pnpm --filter @coze-studio/api-schema tsc --noEmit
pnpm --filter @coze-studio/bot-api tsc --noEmit
pnpm --filter @coze-studio/agent-ide-entry tsc --noEmit
pnpm --filter app tsc --noEmit
```

Frontend focused tests:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
pnpm --filter @coze-studio/api-schema test -- aiproduct
pnpm --filter @coze-studio/agent-ide-entry test -- runtime-config
pnpm --filter app test -- skill-marketplace
```

## Acceptance Criteria

The second phase is complete when:

- Standard skills have corresponding `AIProduct` rows and product versions.
- Global/space skill marketplace can be driven by product APIs.
- Space installation supports install, uninstall, upgrade, installed version, and upgrade availability.
- Session create/get/list include `runtime_config`.
- Runtime resolver rejects unauthorized or uninstalled products.
- Harness state/snapshot includes `runtime_assets`.
- Skill product injection uses pinned versions.
- Product actions write audit logs.
- Model and MCP product records can be created, installed, selected in session config, and safely resolved as metadata without exposing credentials.
- Existing super-agent session/run/harness tests still pass.
- Playwright MCP verifies the remote 226 flow and records the result.

## Suggested Next Goal Wording

Use this as the next objective:

> 按 `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/plans/2026-06-21-ai-product-runtime-marketplace.md` 实施 AIProduct 第二阶段。先完成后端产品层和标准技能产品闭环：新增 `AIProduct` domain/application/API/SQL，标准技能同步为产品，支持空间安装、卸载、升级、审计，并把 session runtime config 存储和 resolver 接入 super-agent sessions/harness。保持现有 skill API 兼容，不重构无关 UI；完成后运行文档中的 backend/frontend 测试，并用 Playwright MCP 在 `10.10.10.226:8896` 记录远程验证结果。

