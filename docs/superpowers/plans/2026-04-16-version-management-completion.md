# Space 版本管理补全实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全版本管理系统的关键缺口：导入时自动关联版本、预导入快照、sync mapping 更新、版本冲突检测、同步历史增强、纯逻辑单元测试。

**Architecture:** SpaceImporter.Confirm 暴露 ID 映射 → SyncService.ImportConfirm 利用映射更新 sync_mapping + 提取 manifest 版本号 + 创建预导入快照 → ImportPreview 添加版本冲突警告 → 同步历史 API 返回版本信息 → 纯逻辑函数的单元测试。

**Tech Stack:** Go 1.22+, GORM, Hertz, SHA256 hashing

---

## 文件结构

| 文件 | 职责 | 操作 |
|------|------|------|
| `backend/application/space/import/types.go` | ImportResult 增加 IDMappings 字段 | Modify |
| `backend/application/space/import/space_importer.go` | executeImport 填充 IDMappings | Modify |
| `backend/application/space/sync/sync_service.go` | ImportConfirm 版本关联 + 快照 + mapping 更新；ImportPreview 版本冲突检测 | Modify |
| `backend/api/model/space/space_sync.go` | SyncHistoryItem 增加版本字段；SyncImportPreviewData 增加版本信息 | Modify |
| `backend/api/handler/space/space_sync_service.go` | SyncHistory handler 返回版本字段；ImportPreview 返回版本信息 | Modify |
| `backend/application/space/release/hasher_test.go` | HashPackage + BuildResourceHashes 单元测试 | Create |
| `backend/application/space/release/release_service_test.go` | parseSemver + diffIDList 单元测试 | Create |

---

### Task 1: SpaceImporter 暴露 ID 映射

**Files:**
- Modify: `backend/application/space/import/types.go:42-50`
- Modify: `backend/application/space/import/space_importer.go:205-306`

- [ ] **Step 1: 给 ImportResult 添加 IDMappings 字段**

修改 `backend/application/space/import/types.go`，在 ImportResult 结构体末尾添加字段：

```go
// ImportResult represents the final result of an import operation
type ImportResult struct {
	AgentsCreated      int            `json:"agents_created"`
	PluginsCreated     int            `json:"plugins_created"`
	WorkflowsCreated   int            `json:"workflows_created"`
	VariablesCreated   int            `json:"variables_created"`
	SpaceModelsCreated int            `json:"space_models_created"`
	KnowledgeCreated   int            `json:"knowledge_created"`
	Errors             []*ImportError `json:"errors,omitempty"`
	// IDMappings maps resource type → (source ID → target ID) for sync mapping
	IDMappings map[string]map[int64]int64 `json:"id_mappings,omitempty"`
}
```

- [ ] **Step 2: 在 executeImport 成功后填充 IDMappings**

修改 `backend/application/space/import/space_importer.go`，在 `executeImport` 方法中，事务成功后（`if err != nil` check 之后、`syncToES` 之前），添加：

```go
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "import transaction failed"))
	}

	// Expose ID mappings for sync mapping updates
	result.IDMappings = map[string]map[int64]int64{
		"agent":              importCtx.AgentIDMap,
		"plugin":             importCtx.PluginIDMap,
		"workflow":           importCtx.WorkflowIDMap,
		"variable":           importCtx.VariableIDMap,
		"space_model":        importCtx.SpaceModelIDMap,
		"knowledge":          importCtx.KnowledgeIDMap,
		"document":           importCtx.DocumentIDMap,
		"folder":             importCtx.FolderIDMap,
		"external_knowledge": importCtx.ExternalKnowledgeIDMap,
	}

	// Sync to ES after successful transaction (outside transaction to avoid blocking)
	s.syncToES(ctx, pending.Resources, importCtx)
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功，无错误

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/import/types.go backend/application/space/import/space_importer.go
git commit -m "feat(import): expose ID mappings from ImportResult for sync mapping"
```

---

### Task 2: ImportConfirm 更新 sync mapping

**Files:**
- Modify: `backend/application/space/sync/sync_service.go:157-255`

导入完成后，利用 Task 1 暴露的 IDMappings，将 source→target 映射写入 `space_sync_mapping` 表。这使得后续增量导入能正确识别已存在的资源（plan 中显示 update 而非 create）。

- [ ] **Step 1: 在 ImportConfirm 中添加 sync mapping 更新逻辑**

修改 `backend/application/space/sync/sync_service.go` 的 `ImportConfirm` 方法。在 `recordSyncHistory` 调用之前（当前第 233 行附近），添加 mapping 更新逻辑：

```go
	// Update sync mappings with source→target ID pairs
	if importResult.IDMappings != nil {
		s.updateSyncMappings(ctx, pending, importResult.IDMappings)
	}

	// Record sync history
	s.recordSyncHistory(ctx, pending, validationResult.Resources, importResult)
```

- [ ] **Step 2: 添加 updateSyncMappings 方法**

在 `sync_service.go` 文件末尾（`generateSyncToken` 之前），添加新方法：

```go
// updateSyncMappings records source→target ID mappings after a successful import.
// This enables future incremental imports to detect existing resources.
func (s *SyncService) updateSyncMappings(ctx context.Context, pending *PendingSyncImport, idMappings map[string]map[int64]int64) {
	mappingStore := NewSyncMappingStore(s.db, pending.SourceSpaceID, pending.SpaceID)

	for resourceType, mappings := range idMappings {
		for sourceID, targetID := range mappings {
			if err := mappingStore.UpsertMapping(ctx, nil, resourceType, sourceID, targetID, time.Now().UnixMilli()); err != nil {
				logs.CtxWarnf(ctx, "Failed to upsert sync mapping %s:%d->%d: %v", resourceType, sourceID, targetID, err)
			}
		}
	}

	logs.CtxInfof(ctx, "Updated sync mappings for space_id=%d, source_space_id=%d", pending.SpaceID, pending.SourceSpaceID)
}
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/sync/sync_service.go
git commit -m "feat(sync): update sync mappings after import for incremental detection"
```

---

### Task 3: ImportConfirm 自动关联版本 + 预导入快照

**Files:**
- Modify: `backend/application/space/sync/sync_service.go:157-255`

当导入的包 manifest 中含有 `ReleaseVersion`（表示来自一个已发布的 release），ImportConfirm 应该：
1. 创建当前状态的快照（以便回滚）
2. 将版本号记录到 sync_history

- [ ] **Step 1: 修改 ImportConfirm 添加版本提取和快照逻辑**

修改 `backend/application/space/sync/sync_service.go` 的 `ImportConfirm` 方法。在重新解析 ZIP 之后（`validationResult` 获取之后）、处理删除资源之前，添加版本检测和快照逻辑：

```go
	// Re-parse ZIP contents
	validator := spaceimport.NewValidator()
	validationResult, err := validator.ValidateAndParse(ctx, fileContent)
	if err != nil {
		return nil, err
	}

	// Detect release version from manifest
	releaseVersion := pending.Manifest.ReleaseVersion
	var snapshotKey string
	if releaseVersion != "" {
		logs.CtxInfof(ctx, "Importing release version %s into space_id=%d", releaseVersion, spaceID)
		// Create pre-import snapshot for safety (non-blocking: if snapshot fails, still proceed)
		snapshotKey, err = s.CreateSnapshot(ctx, spaceID)
		if err != nil {
			logs.CtxWarnf(ctx, "Failed to create pre-import snapshot for space_id=%d: %v (proceeding with import)", spaceID, err)
			snapshotKey = ""
		}
	}

	// Handle deleted resources before creating/updating
```

- [ ] **Step 2: 修改 recordSyncHistory 调用，传入版本信息**

在同一方法中，将当前的 `recordSyncHistory` 调用替换为带 opts 的版本：

旧代码（约第 233 行）：
```go
	// Record sync history
	s.recordSyncHistory(ctx, pending, validationResult.Resources, importResult)
```

新代码：
```go
	// Record sync history with version info
	var histOpts []recordHistoryOpts
	if releaseVersion != "" {
		histOpts = append(histOpts, recordHistoryOpts{
			Version:     releaseVersion,
			SnapshotKey: snapshotKey,
		})
	}
	s.recordSyncHistory(ctx, pending, validationResult.Resources, importResult, histOpts...)
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/sync/sync_service.go
git commit -m "feat(sync): auto-associate version and create pre-import snapshot"
```

---

### Task 4: ImportPreview 版本冲突检测

**Files:**
- Modify: `backend/application/space/sync/sync_service.go:97-155`
- Modify: `backend/application/space/sync/sync_service.go` (SyncPreviewResult)

当导入包的 manifest 含有 `ReleaseVersion` 时，在 Preview 阶段检查目标 space 当前已部署的版本，如果导入版本 <= 当前版本，添加警告。

- [ ] **Step 1: 在 SyncPreviewResult 中添加版本信息字段**

修改 `sync_service.go` 中的 `SyncPreviewResult`（约第 564 行）：

```go
type SyncPreviewResult struct {
	ImportToken    string                `json:"import_token"`
	Manifest       *spaceexport.Manifest `json:"manifest"`
	Plan           *ImportPlan           `json:"plan"`
	Warnings       []string              `json:"warnings"`
	TokenExpiresAt int64                 `json:"token_expires_at"`
	// Version info for display
	IncomingVersion string `json:"incoming_version,omitempty"`
	CurrentVersion  string `json:"current_version,omitempty"`
}
```

- [ ] **Step 2: 在 ImportPreview 中添加版本冲突检测**

修改 `ImportPreview` 方法，在 `buildImportPlan` 之后、创建 `SyncPreviewResult` 之前，添加：

```go
	// Build import plan
	plan := s.buildImportPlan(validationResult.Resources, mappingStore)

	// Detect version conflict
	incomingVersion := manifest.ReleaseVersion
	currentVersion := ""
	if incomingVersion != "" {
		current, _ := s.historyRepo.GetCurrentVersion(ctx, spaceID)
		if current != nil && current.Version != nil {
			currentVersion = *current.Version
		}
		if currentVersion != "" && !isNewerVersion(incomingVersion, currentVersion) {
			validationResult.Warnings = append(validationResult.Warnings,
				fmt.Sprintf("当前已部署版本 %s >= 即将导入的版本 %s，请确认是否需要降级", currentVersion, incomingVersion))
		}
	}

	// Store pending import
```

注意：需要确保文件顶部已经 import `fmt`（当前已有）。

同时需要在 `sync_service.go` 文件末尾添加 `isNewerVersion` 辅助函数（在 `generateSyncToken` 之前）：

```go
// isNewerVersion returns true if version a is strictly newer than version b.
// Uses semver parsing for correct multi-digit comparison (e.g., v1.0.10 > v1.0.9).
func isNewerVersion(a, b string) bool {
	aMaj, aMin, aPatch, aOk := parseSemverSimple(a)
	bMaj, bMin, bPatch, bOk := parseSemverSimple(b)
	if !aOk || !bOk {
		return a > b // fallback to string comparison
	}
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPatch > bPatch
}

// parseSemverSimple parses a semver string like "v1.2.3" into components.
func parseSemverSimple(version string) (major, minor, patch int, ok bool) {
	if len(version) < 2 || version[0] != 'v' {
		return 0, 0, 0, false
	}
	parts := [3]int{}
	idx := 0
	for _, ch := range version[1:] {
		if ch == '.' {
			idx++
			if idx > 2 {
				return 0, 0, 0, false
			}
			continue
		}
		if ch < '0' || ch > '9' {
			return 0, 0, 0, false
		}
		parts[idx] = parts[idx]*10 + int(ch-'0')
	}
	if idx != 2 {
		return 0, 0, 0, false
	}
	return parts[0], parts[1], parts[2], true
}
```

- [ ] **Step 3: 更新 SyncPreviewResult 返回值**

修改 `ImportPreview` 方法末尾的 return，添加版本字段：

```go
	return &SyncPreviewResult{
		ImportToken:     token,
		Manifest:        manifest,
		Plan:            plan,
		Warnings:        validationResult.Warnings,
		TokenExpiresAt:  time.Now().Add(syncImportTTL).Unix(),
		IncomingVersion: incomingVersion,
		CurrentVersion:  currentVersion,
	}, nil
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功

- [ ] **Step 5: 提交**

```bash
git add backend/application/space/sync/sync_service.go
git commit -m "feat(sync): add version conflict detection in import preview"
```

---

### Task 5: 同步历史 API 增强

**Files:**
- Modify: `backend/api/model/space/space_sync.go:94-102`
- Modify: `backend/api/model/space/space_sync.go:49-54`
- Modify: `backend/api/handler/space/space_sync_service.go:258-291`
- Modify: `backend/api/handler/space/space_sync_service.go:63-112`

在 API 响应中暴露版本管理相关字段，使前端能展示版本信息。

- [ ] **Step 1: SyncHistoryItem 添加版本字段**

修改 `backend/api/model/space/space_sync.go`，给 `SyncHistoryItem` 添加字段：

```go
type SyncHistoryItem struct {
	ID                  uint64  `json:"id,string"`
	SourceSpaceID       int64   `json:"source_space_id,string"`
	TargetSpaceID       int64   `json:"target_space_id,string"`
	SyncType            string  `json:"sync_type"`
	Version             *string `json:"version,omitempty"`
	ExportTime          int64   `json:"export_time"`
	ImportTime          *int64  `json:"import_time,omitempty"`
	Status              int8    `json:"status"`
	SnapshotKey         *string `json:"snapshot_key,omitempty"`
	RollbackFromVersion *string `json:"rollback_from_version,omitempty"`
}
```

- [ ] **Step 2: SyncHistory handler 填充新字段**

修改 `backend/api/handler/space/space_sync_service.go` 中的 `SyncHistory` handler（约第 272 行），在构建 items 时填充新字段：

```go
	items := make([]*spaceModel.SyncHistoryItem, 0, len(records))
	for _, r := range records {
		items = append(items, &spaceModel.SyncHistoryItem{
			ID:                  r.ID,
			SourceSpaceID:       r.SourceSpaceID,
			TargetSpaceID:       r.TargetSpaceID,
			SyncType:            r.SyncType,
			Version:             r.Version,
			ExportTime:          r.ExportTime,
			ImportTime:          r.ImportTime,
			Status:              r.Status,
			SnapshotKey:         r.SnapshotKey,
			RollbackFromVersion: r.RollbackFromVersion,
		})
	}
```

- [ ] **Step 3: SyncImportPreviewData 添加版本信息**

修改 `backend/api/model/space/space_sync.go`，给 `SyncImportPreviewData` 添加字段：

```go
type SyncImportPreviewData struct {
	ImportToken     string    `json:"import_token"`
	Plan            *SyncPlan `json:"plan"`
	Warnings        []string  `json:"warnings"`
	TokenExpiresAt  int64     `json:"token_expires_at"`
	IncomingVersion string    `json:"incoming_version,omitempty"`
	CurrentVersion  string    `json:"current_version,omitempty"`
}
```

- [ ] **Step 4: SyncImportPreview handler 填充版本信息**

修改 `backend/api/handler/space/space_sync_service.go` 中的 `SyncImportPreview` handler（约第 98 行），添加版本字段到响应：

```go
	c.JSON(consts.StatusOK, &spaceModel.SyncImportPreviewResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.SyncImportPreviewData{
			ImportToken: result.ImportToken,
			Plan: &spaceModel.SyncPlan{
				Create: result.Plan.Create,
				Update: result.Plan.Update,
				Delete: result.Plan.Delete,
			},
			Warnings:        result.Warnings,
			TokenExpiresAt:  result.TokenExpiresAt,
			IncomingVersion: result.IncomingVersion,
			CurrentVersion:  result.CurrentVersion,
		},
	})
```

- [ ] **Step 5: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功

- [ ] **Step 6: 提交**

```bash
git add backend/api/model/space/space_sync.go backend/api/handler/space/space_sync_service.go
git commit -m "feat(api): expose version info in sync history and import preview"
```

---

### Task 6: 纯逻辑单元测试

**Files:**
- Create: `backend/application/space/release/hasher_test.go`
- Create: `backend/application/space/release/release_service_test.go`

为无外部依赖的纯逻辑函数编写单元测试。

- [ ] **Step 1: 创建 hasher_test.go**

创建 `backend/application/space/release/hasher_test.go`：

```go
package release

import (
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

func TestHashPackage(t *testing.T) {
	// Same input → same hash
	data := []byte("test zip content")
	h1 := HashPackage(data)
	h2 := HashPackage(data)
	if h1 != h2 {
		t.Errorf("HashPackage not deterministic: %s != %s", h1, h2)
	}
	// SHA256 hex is 64 chars
	if len(h1) != 64 {
		t.Errorf("HashPackage length = %d, want 64", len(h1))
	}

	// Different input → different hash
	h3 := HashPackage([]byte("other content"))
	if h1 == h3 {
		t.Errorf("HashPackage collision for different inputs")
	}

	// Empty input
	h4 := HashPackage([]byte{})
	if len(h4) != 64 {
		t.Errorf("HashPackage empty input length = %d, want 64", len(h4))
	}
}

func TestBuildResourceHashes(t *testing.T) {
	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{ID: 100, Name: "agent1"},
			{ID: 200, Name: "agent2"},
		},
		Plugins: []*export.ExportedPlugin{
			{ID: 300, Name: "plugin1"},
		},
		Workflows:      []*export.ExportedWorkflow{},
		Variables:      []*export.ExportedVariable{},
		SpaceModels:    []*export.ExportedSpaceModel{},
		KnowledgeBases: []*export.ExportedKnowledge{},
		Folders:        []*export.ExportedFolder{},
		ExternalKnowledge: []*export.ExportedExternalKnowledge{},
	}

	hashes, err := BuildResourceHashes(resources)
	if err != nil {
		t.Fatalf("BuildResourceHashes error: %v", err)
	}

	// Should contain entries for agents and plugins
	if _, ok := hashes["agent:100"]; !ok {
		t.Error("missing hash for agent:100")
	}
	if _, ok := hashes["agent:200"]; !ok {
		t.Error("missing hash for agent:200")
	}
	if _, ok := hashes["plugin:300"]; !ok {
		t.Error("missing hash for plugin:300")
	}

	// Different agents → different hashes
	if hashes["agent:100"] == hashes["agent:200"] {
		t.Error("agent:100 and agent:200 should have different hashes")
	}

	// Deterministic: same input → same hashes
	hashes2, _ := BuildResourceHashes(resources)
	for k, v := range hashes {
		if hashes2[k] != v {
			t.Errorf("hash for %s not deterministic: %s != %s", k, v, hashes2[k])
		}
	}
}

func TestBuildResourceHashesNil(t *testing.T) {
	hashes, err := BuildResourceHashes(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("expected empty hashes for nil resources, got %d entries", len(hashes))
	}
}
```

- [ ] **Step 2: 运行 hasher 测试**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/application/space/release/ -run TestHash -v`
Expected: PASS

- [ ] **Step 3: 创建 release_service_test.go**

创建 `backend/application/space/release/release_service_test.go`：

```go
package release

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input         string
		major, minor, patch int
		ok            bool
	}{
		{"v1.0.0", 1, 0, 0, true},
		{"v2.3.4", 2, 3, 4, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v10.20.30", 10, 20, 30, true},
		{"1.0.0", 0, 0, 0, false},      // missing v prefix
		{"v1.0", 0, 0, 0, false},        // missing patch
		{"v1.0.0-beta", 0, 0, 0, false}, // pre-release not supported
		{"", 0, 0, 0, false},
		{"vx.y.z", 0, 0, 0, false},
	}

	for _, tt := range tests {
		major, minor, patch, ok := parseSemver(tt.input)
		if ok != tt.ok {
			t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok {
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("parseSemver(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		}
	}
}

func TestDiffIDList(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	fromIDs := []int64{1, 2, 3}
	toIDs := []int64{2, 3, 4}
	fromHashes := map[string]string{
		"agent:1": "hash_a1",
		"agent:2": "hash_a2",
		"agent:3": "hash_a3",
	}
	toHashes := map[string]string{
		"agent:2": "hash_a2",       // same
		"agent:3": "hash_a3_new",   // changed
		"agent:4": "hash_a4",
	}

	diffIDList(diff, "agents", fromIDs, toIDs, fromHashes, toHashes, "agent")

	// Added: 4
	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 4 {
		t.Errorf("Added = %v, want [{ID:4}]", diff.Added["agents"])
	}

	// Removed: 1
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}

	// Modified: 3 (hash changed)
	if len(diff.Modified["agents"]) != 1 || diff.Modified["agents"][0].ID != 3 {
		t.Errorf("Modified = %v, want [{ID:3}]", diff.Modified["agents"])
	}
}

func TestDiffIDListNoHashes(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{1, 2}, []int64{2, 3}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 3 {
		t.Errorf("Added = %v, want [{ID:3}]", diff.Added["agents"])
	}
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}
	// No hashes → no modified detection
	if len(diff.Modified["agents"]) != 0 {
		t.Errorf("Modified = %v, want empty (no hashes)", diff.Modified["agents"])
	}
}

func TestDiffIDListEmpty(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{}, []int64{}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 0 || len(diff.Removed["agents"]) != 0 || len(diff.Modified["agents"]) != 0 {
		t.Error("expected empty diff for empty inputs")
	}
}
```

- [ ] **Step 4: 运行 release service 测试**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/application/space/release/ -run "TestParseSemver|TestDiffIDList" -v`
Expected: PASS

- [ ] **Step 5: 运行全部 release 测试确认**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/application/space/release/ -v`
Expected: All PASS

- [ ] **Step 6: 提交**

```bash
git add backend/application/space/release/hasher_test.go backend/application/space/release/release_service_test.go
git commit -m "test(release): add unit tests for hasher and diff logic"
```

---

### Task 7: 全量编译验证 + 最终提交

- [ ] **Step 1: 全量编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功

- [ ] **Step 2: 全量测试**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/application/space/release/ -v`
Expected: All PASS

- [ ] **Step 3: 确认 git 状态**

Run: `git status && git log --oneline -10`
Expected: 工作区干净，能看到本次所有提交
