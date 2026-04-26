# Space Import/Export Test Coverage — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Space 导入导出/同步/版本/回滚功能补齐两层测试：纯逻辑单元测试覆盖核心函数 + testcontainers 端到端覆盖 7 个关键场景，全部进 CI。

**Architecture:** 单元测试用 gomock 隔离 DB/Storage 测核心逻辑（id mapping、reference rewriting、validator、serializer round-trip、release rollback/snapshot）；E2E 测试用 testcontainers 启 MySQL+MinIO+ES+Redis，覆盖完整业务流程（round-trip、增量、回滚、版本冲突、跨引用、删除、ES 失败降级）。

**Tech Stack:** Go 1.22+, testify/assert, gomock, testcontainers-go, MySQL 8.4, MinIO, Elasticsearch 8.x

**Spec:** [docs/superpowers/specs/2026-04-27-import-export-test-coverage-design.md](docs/superpowers/specs/2026-04-27-import-export-test-coverage-design.md)

---

## Phase 1: 单元测试

### Task 1: id_mapper 单元测试

**Files:**
- Create: `backend/application/space/import/id_mapper_test.go`

- [ ] **Step 1: 阅读现有 id_mapper.go**

Run: `cat backend/application/space/import/id_mapper.go`

确认 IDMapper 的导出方法（典型为 `Map`、`Get`、`Has`、`MapAll`）。

- [ ] **Step 2: 写测试**

```go
package import_

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDMapper_NewMapping(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...) // 按真实构造函数填参（mock idGen）

	src := int64(100)
	target, err := mapper.Map(ctx, "agent", src)
	require.NoError(t, err)
	assert.NotZero(t, target)
	assert.NotEqual(t, src, target)
}

func TestIDMapper_ExistingMapping(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)

	src := int64(200)
	first, _ := mapper.Map(ctx, "agent", src)
	second, _ := mapper.Map(ctx, "agent", src)
	assert.Equal(t, first, second, "second call should return cached mapping")
}

func TestIDMapper_DifferentResourceTypes(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)

	a, _ := mapper.Map(ctx, "agent", 100)
	w, _ := mapper.Map(ctx, "workflow", 100)
	assert.NotEqual(t, a, w, "same source ID for different types should map differently")
}

func TestIDMapper_ZeroIDEdgeCase(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)

	target, err := mapper.Map(ctx, "agent", 0)
	// 期望按现有实现：0 应直接返回 0 或报错（看真实逻辑）
	if err != nil {
		assert.Equal(t, int64(0), target)
	} else {
		assert.NotZero(t, target)
	}
}

func TestIDMapper_GetReturnsMappedID(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)

	src := int64(300)
	target, _ := mapper.Map(ctx, "knowledge", src)

	got, ok := mapper.Get("knowledge", src)
	require.True(t, ok)
	assert.Equal(t, target, got)
}

func TestIDMapper_GetMissReturnsFalse(t *testing.T) {
	mapper := NewIDMapper(...)
	_, ok := mapper.Get("agent", 999999)
	assert.False(t, ok)
}
```

按真实 NewIDMapper 签名填 mock idGen。如果项目的 idGen 接口在 `pkg/idgen/idgen.go`，用 gomock 生成 mock 或写一个简单 fake 实现：

```go
type fakeIDGen struct{ counter int64 }
func (f *fakeIDGen) GenID(ctx context.Context) (int64, error) {
    f.counter++
    return 1000 + f.counter, nil
}
```

- [ ] **Step 3: 跑测试**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/application/space/import/ -run TestIDMapper -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/import/id_mapper_test.go
git commit -m "test(import): add unit tests for IDMapper"
```

---

### Task 2: reference_rewriter 单元测试

**Files:**
- Create: `backend/application/space/import/reference_rewriter_test.go`

- [ ] **Step 1: 阅读 reference_rewriter.go**

Run: `cat backend/application/space/import/reference_rewriter.go | head -80`

确认 `RewriteReferences` 签名和支持的 reference 类型。

- [ ] **Step 2: 写测试**

```go
package import_

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteReferences_WorkflowToTool(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)
	mapper.Map(ctx, "plugin", 1001)
	mapper.Map(ctx, "tool",   2001)

	workflow := &ExportedWorkflow{
		Schema: `{"nodes":[{"plugin_id":1001,"tool_id":2001}]}`,
	}

	r := NewReferenceRewriter(mapper)
	rewritten, err := r.RewriteWorkflow(workflow)
	require.NoError(t, err)
	// 断言 schema 里 plugin_id / tool_id 被改写为 mapper 返回的新 ID
	newPluginID, _ := mapper.Get("plugin", 1001)
	assert.Contains(t, rewritten.Schema, formatID(newPluginID))
}

func TestRewriteReferences_AgentKnowledge(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(...)
	mapper.Map(ctx, "knowledge", 5000)

	agent := &ExportedAgent{
		KnowledgeRefs: []int64{5000},
	}

	r := NewReferenceRewriter(mapper)
	rewritten, err := r.RewriteAgent(agent)
	require.NoError(t, err)

	newID, _ := mapper.Get("knowledge", 5000)
	assert.Equal(t, []int64{newID}, rewritten.KnowledgeRefs)
}

func TestRewriteReferences_MissingReferenceKeepsPlaceholder(t *testing.T) {
	mapper := NewIDMapper(...)
	r := NewReferenceRewriter(mapper)

	agent := &ExportedAgent{KnowledgeRefs: []int64{99999}}
	_, err := r.RewriteAgent(agent)
	// 行为按现有实现：要么报错 要么保留占位
	if err == nil {
		// 验证保留了原 ID 或某种 placeholder
	}
}

func TestRewriteReferences_NestedReferences(t *testing.T) {
	// 跨 workflow → workflow 嵌套
	// 跨 agent → workflow → tool
	// 测试递归改写覆盖到所有层级
	t.Skip("写真实场景的嵌套用例")
}
```

注意：根据现有 `reference_rewriter.go` 真实接口和数据结构调整。如果是字符串模板替换（regex），断言策略要匹配实际实现。

- [ ] **Step 3: 跑测试**

Run: `go test ./backend/application/space/import/ -run TestRewriteReferences -v`
Expected: PASS（可能需迭代调整测试断言以匹配真实行为）

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/import/reference_rewriter_test.go
git commit -m "test(import): add unit tests for reference rewriting"
```

---

### Task 3: validator 单元测试

**Files:**
- Create: `backend/application/space/import/validator_test.go`

- [ ] **Step 1: 阅读 validator.go**

Run: `cat backend/application/space/import/validator.go | head -100`

- [ ] **Step 2: 写测试**

```go
package import_

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validManifest() *Manifest {
	return &Manifest{
		Version: "2.0.0",
		Resources: ResourceManifest{
			Agents:    []*AgentManifest{{ID: 1, Name: "test"}},
			Workflows: []*WorkflowManifest{},
			Plugins:   []*PluginManifest{},
		},
	}
}

func TestValidateManifest_Valid(t *testing.T) {
	v := NewValidator()
	err := v.Validate(validManifest())
	require.NoError(t, err)
}

func TestValidateManifest_MissingVersion(t *testing.T) {
	m := validManifest()
	m.Version = ""
	err := NewValidator().Validate(m)
	assert.Error(t, err)
}

func TestValidateManifest_UnsupportedVersion(t *testing.T) {
	m := validManifest()
	m.Version = "0.9.0"
	err := NewValidator().Validate(m)
	assert.Error(t, err)
}

func TestValidateManifest_InvalidAgentName(t *testing.T) {
	m := validManifest()
	m.Resources.Agents[0].Name = ""  // 必填
	err := NewValidator().Validate(m)
	assert.Error(t, err)
}

func TestValidateManifest_LargeResourceCount(t *testing.T) {
	m := validManifest()
	for i := 0; i < 1000; i++ {
		m.Resources.Agents = append(m.Resources.Agents, &AgentManifest{ID: int64(i + 100), Name: "a"})
	}
	err := NewValidator().Validate(m)
	assert.NoError(t, err, "1000 agents should be acceptable")
}
```

按真实 Manifest 结构调整字段名。

- [ ] **Step 3: 跑测试**

Run: `go test ./backend/application/space/import/ -run TestValidateManifest -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/import/validator_test.go
git commit -m "test(import): add unit tests for manifest validator"
```

---

### Task 4: serializer 单元测试（round-trip）

**Files:**
- Create: `backend/application/space/export/serializer_test.go`

- [ ] **Step 1: 阅读 serializer.go**

Run: `cat backend/application/space/export/serializer.go | head -80`

- [ ] **Step 2: 写测试**

```go
package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializer_RoundTrip(t *testing.T) {
	original := &ExportedSpace{
		ID:   123,
		Name: "Test Space",
		Agents: []*ExportedAgent{
			{ID: 1, Name: "agent1", Prompt: "hello"},
			{ID: 2, Name: "agent2", Prompt: "world"},
		},
	}

	s := NewSerializer()
	bytes, err := s.Serialize(original)
	require.NoError(t, err)

	deserialized, err := s.Deserialize(bytes)
	require.NoError(t, err)

	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.Name, deserialized.Name)
	require.Len(t, deserialized.Agents, 2)
	assert.Equal(t, "agent1", deserialized.Agents[0].Name)
}

func TestSerializer_EmptySpace(t *testing.T) {
	s := NewSerializer()
	empty := &ExportedSpace{ID: 1, Name: "empty"}
	bytes, err := s.Serialize(empty)
	require.NoError(t, err)

	d, err := s.Deserialize(bytes)
	require.NoError(t, err)
	assert.Empty(t, d.Agents)
}

func TestSerializer_SpecialCharacters(t *testing.T) {
	s := NewSerializer()
	special := &ExportedSpace{
		ID:   1,
		Name: "测试 中文 🚀 \"quotes\" \\backslash",
		Agents: []*ExportedAgent{{ID: 1, Name: "', \"\";", Prompt: "newline\nin prompt"}},
	}
	bytes, err := s.Serialize(special)
	require.NoError(t, err)
	d, err := s.Deserialize(bytes)
	require.NoError(t, err)
	assert.Equal(t, special.Name, d.Name)
	assert.Equal(t, special.Agents[0].Prompt, d.Agents[0].Prompt)
}
```

- [ ] **Step 3: 跑测试**

Run: `go test ./backend/application/space/export/ -run TestSerializer -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/export/serializer_test.go
git commit -m "test(export): add round-trip and edge case tests for serializer"
```

---

### Task 5: sync_mapping_repo 单元测试

**Files:**
- Create: `backend/application/space/sync/sync_mapping_repo_test.go`

- [ ] **Step 1: 写测试**

```go
package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// 用 in-memory sqlite 或 mock GORM
)

func setupRepo(t *testing.T) *SyncMappingRepo {
	t.Helper()
	db := openSQLiteInMemory(t)
	migrate(db, "sync_mapping")
	return NewSyncMappingRepo(db)
}

func TestSyncMappingRepo_Upsert(t *testing.T) {
	r := setupRepo(t)
	ctx := context.Background()

	err := r.Upsert(ctx, &SyncMapping{
		SourceSpaceID: 1, TargetSpaceID: 2,
		ResourceType: 1, // agent
		SourceID: 100, TargetID: 1001,
	})
	require.NoError(t, err)

	got, err := r.GetByResource(ctx, 1, 2, 1, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1001), got.TargetID)
}

func TestSyncMappingRepo_UpsertOverwrite(t *testing.T) {
	r := setupRepo(t)
	ctx := context.Background()

	r.Upsert(ctx, &SyncMapping{SourceSpaceID: 1, TargetSpaceID: 2, ResourceType: 1, SourceID: 100, TargetID: 1001})
	r.Upsert(ctx, &SyncMapping{SourceSpaceID: 1, TargetSpaceID: 2, ResourceType: 1, SourceID: 100, TargetID: 9999})

	got, _ := r.GetByResource(ctx, 1, 2, 1, 100)
	assert.Equal(t, int64(9999), got.TargetID)
}

func TestSyncMappingRepo_Delete(t *testing.T) {
	r := setupRepo(t)
	ctx := context.Background()

	r.Upsert(ctx, &SyncMapping{SourceSpaceID: 1, TargetSpaceID: 2, ResourceType: 1, SourceID: 100, TargetID: 1001})
	require.NoError(t, r.Delete(ctx, 1, 2, 1, 100))

	_, err := r.GetByResource(ctx, 1, 2, 1, 100)
	assert.Error(t, err)
}
```

如果项目没有 sqlite 测试 helper，跳过此 task 或改为 mock-based 单元测试（用 gomock 模拟 sync_mapping DAO）。

- [ ] **Step 2: 跑测试**

Run: `go test ./backend/application/space/sync/ -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/application/space/sync/sync_mapping_repo_test.go
git commit -m "test(sync): add unit tests for sync_mapping_repo"
```

---

### Task 6: release service rollback/snapshot 测试增补

**Files:**
- Modify: `backend/application/space/release/release_service_test.go`

- [ ] **Step 1: 阅读现有测试和 release_service.go**

Run: `cat backend/application/space/release/release_service.go | head -100`

定位 `Rollback`、`CreateSnapshot` 相关函数。

- [ ] **Step 2: 加纯逻辑测试**

在 release_service_test.go 末尾追加：
```go
func TestCreateSnapshot_DataCorrectness(t *testing.T) {
	mapper := newFakeMapper()  // 假设已有
	repo := newMockReleaseRepo(t)

	svc := NewReleaseService(repo, ...)
	snap, err := svc.createSnapshot(context.Background(), 100, "1.0.0", &SpaceData{
		Agents: []*Agent{{ID: 1, Name: "a1"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", snap.Version)
	assert.Len(t, snap.Resources, 1)
}

func TestRollback_ValidatesTargetVersion(t *testing.T) {
	repo := newMockReleaseRepo(t)
	svc := NewReleaseService(repo, ...)

	err := svc.Rollback(context.Background(), 100, "999.999.999")
	assert.Error(t, err, "rollback to non-existent version should fail")
}
```

按真实 ReleaseService 接口调整。

- [ ] **Step 3: 跑测试**

Run: `go test ./backend/application/space/release/ -v`
Expected: PASS（包括原有 hash/diff 测试 + 新增）

- [ ] **Step 4: 提交**

```bash
git add backend/application/space/release/release_service_test.go
git commit -m "test(release): add unit tests for snapshot and rollback logic"
```

---

## Phase 2: E2E 测试基础设施

### Task 7: testcontainers 启停 + 共享实例

**Files:**
- Create: `backend/test/integration/space_sync/setup_test.go`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p backend/test/integration/space_sync
```

- [ ] **Step 2: 写 setup_test.go**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tces "github.com/testcontainers/testcontainers-go/modules/elasticsearch"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	testDB    *sql.DB
	testS3    *MinIOClient
	testES    *ESClient
	testRDB   *RedisClient
	teardowns []func()
)

func TestMain(m *testing.M) {
	if testing.Short() {
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := startMySQL(ctx); err != nil {
		log.Fatalf("mysql: %v", err)
	}
	if err := startMinIO(ctx); err != nil {
		log.Fatalf("minio: %v", err)
	}
	if err := startElasticsearch(ctx); err != nil {
		log.Fatalf("es: %v", err)
	}
	if err := startRedis(ctx); err != nil {
		log.Fatalf("redis: %v", err)
	}

	if err := runMigrations(testDB); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	code := m.Run()

	for _, t := range teardowns {
		t()
	}
	os.Exit(code)
}

func startMySQL(ctx context.Context) error {
	c, err := tcmysql.Run(ctx, "mysql:8.4",
		tcmysql.WithDatabase("ynet_test"),
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("root"),
	)
	if err != nil {
		return err
	}
	teardowns = append(teardowns, func() { c.Terminate(ctx) })

	connStr, err := c.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	testDB = db
	return nil
}

// startMinIO, startElasticsearch, startRedis: 类似实现

func runMigrations(db *sql.DB) error {
	// 用项目现有 init SQL 路径，比如 docker/sql/init.sql
	// 或 GORM AutoMigrate
	return fmt.Errorf("TODO: implement migrations")
}
```

- [ ] **Step 3: 编译验证**

Run: `go build -tags integration ./backend/test/integration/space_sync/`
Expected: 编译成功（runMigrations 暂时报 TODO 是预期）

- [ ] **Step 4: 实现 runMigrations**

定位项目 schema 初始化（典型 `docker/sql/*.sql` 或 `infra/impl/repository/schema/`）：
```go
func runMigrations(db *sql.DB) error {
    sqls, err := os.ReadFile("../../../../docker/sql/init.sql")
    if err != nil {
        return err
    }
    for _, stmt := range strings.Split(string(sqls), ";") {
        if strings.TrimSpace(stmt) == "" { continue }
        if _, err := db.Exec(stmt); err != nil {
            return fmt.Errorf("exec %q: %w", stmt[:50], err)
        }
    }
    return nil
}
```

- [ ] **Step 5: 跑空测试验证 setup 通过**

写一个 placeholder：`backend/test/integration/space_sync/sanity_test.go`:
```go
//go:build integration
package space_sync_test

import "testing"

func TestSanity(t *testing.T) {
    if testDB == nil {
        t.Fatal("testDB not initialized")
    }
}
```

Run: `go test -tags integration ./backend/test/integration/space_sync/ -run TestSanity -v -timeout 5m`
Expected: PASS（容器启动需要时间）

- [ ] **Step 6: 提交**

```bash
git add backend/test/integration/space_sync/
git commit -m "test(integration): add testcontainers setup for space sync E2E"
```

---

### Task 8: 测试 helpers 与 fixture

**Files:**
- Create: `backend/test/integration/space_sync/helpers.go`

- [ ] **Step 1: 写 fixture helpers**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"
)

type TestSpace struct {
	ID        int64
	Name      string
	Agents    []*TestAgent
	Workflows []*TestWorkflow
	Knowledge []*TestKnowledge
}

func CreateTestSpace(t *testing.T, opts ...SpaceOpt) *TestSpace {
	t.Helper()
	cfg := defaultSpaceConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	// 用 testDB 直接 INSERT
	ctx := context.Background()
	spaceID := insertSpace(t, ctx, cfg.Name)
	for _, a := range cfg.Agents {
		insertAgent(t, ctx, spaceID, a)
	}
	// ... 同理 workflows / knowledge
	return loadSpace(t, ctx, spaceID)
}

type SpaceOpt func(*spaceConfig)

func WithAgents(n int) SpaceOpt {
	return func(c *spaceConfig) {
		for i := 0; i < n; i++ {
			c.Agents = append(c.Agents, &TestAgent{Name: fmt.Sprintf("agent_%d", i)})
		}
	}
}

func WithKnowledge(n int) SpaceOpt {
	return func(c *spaceConfig) {
		for i := 0; i < n; i++ {
			c.Knowledge = append(c.Knowledge, &TestKnowledge{Name: fmt.Sprintf("kb_%d", i)})
		}
	}
}

// AssertSpacesEquivalent 断言两个 space 关键字段相等
func AssertSpacesEquivalent(t *testing.T, expected, actual *TestSpace) {
	t.Helper()
	if len(expected.Agents) != len(actual.Agents) {
		t.Fatalf("agent count mismatch: %d vs %d", len(expected.Agents), len(actual.Agents))
	}
	for i, e := range expected.Agents {
		a := actual.Agents[i]
		if e.Name != a.Name {
			t.Errorf("agent[%d].Name: %q vs %q", i, e.Name, a.Name)
		}
		if e.ID == a.ID {
			t.Errorf("agent[%d].ID should differ (was re-mapped): both %d", i, e.ID)
		}
	}
}

func CleanupSpace(t *testing.T, spaceID int64) {
	t.Helper()
	ctx := context.Background()
	testDB.ExecContext(ctx, "DELETE FROM single_agent_draft WHERE space_id = ?", spaceID)
	testDB.ExecContext(ctx, "DELETE FROM workflow_meta WHERE space_id = ?", spaceID)
	testDB.ExecContext(ctx, "DELETE FROM knowledge WHERE space_id = ?", spaceID)
	testDB.ExecContext(ctx, "DELETE FROM space WHERE id = ?", spaceID)
}

func insertSpace(t *testing.T, ctx context.Context, name string) int64 {
	id := generateID()
	_, err := testDB.ExecContext(ctx,
		"INSERT INTO space (id, name, creator_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, name, 1, time.Now().UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("insertSpace: %v", err)
	}
	return id
}

// 类似实现 insertAgent, insertWorkflow, insertKnowledge, loadSpace 等
```

按项目真实表结构填字段。如果有 `domain/space/entity` 之类已有 helpers 可复用。

- [ ] **Step 2: 编译**

Run: `go build -tags integration ./backend/test/integration/space_sync/`
Expected: 成功

- [ ] **Step 3: 提交**

```bash
git add backend/test/integration/space_sync/helpers.go
git commit -m "test(integration): add space fixture helpers"
```

---

### Task 9: E2E 1 - Round Trip

**Files:**
- Create: `backend/test/integration/space_sync/round_trip_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportImportRoundTrip(t *testing.T) {
	ctx := context.Background()

	// 1. 创建 source space
	source := CreateTestSpace(t,
		WithName("source"),
		WithAgents(2),
		WithWorkflows(1),
		WithKnowledge(1),
	)
	t.Cleanup(func() { CleanupSpace(t, source.ID) })

	// 2. 导出
	exporter := newSpaceExporter()
	exportData, err := exporter.Export(ctx, source.ID)
	require.NoError(t, err)
	require.NotEmpty(t, exportData)

	// 3. 创建 target space
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, target.ID) })

	// 4. 导入
	syncSvc := newSyncService()
	preview, err := syncSvc.ImportPreview(ctx, target.ID, exportData)
	require.NoError(t, err)
	require.Equal(t, 2, preview.AgentsToCreate)

	result, err := syncSvc.ImportConfirm(ctx, target.ID, exportData, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, result.AgentsCreated)

	// 5. 加载 target space 对比
	targetLoaded := loadSpace(t, ctx, target.ID)
	AssertSpacesEquivalent(t, source, targetLoaded)
}
```

`newSpaceExporter` / `newSyncService` 是 helpers 里构造业务对象（连接 testDB / testS3 / testES / testRDB）。

- [ ] **Step 2: 实现 newSpaceExporter / newSyncService helper**

加到 `helpers.go`：
```go
func newSpaceExporter() *export.SpaceExporter {
	// 项目里 SpaceExporter 构造函数应接受 db、idGen、storage
	return export.NewSpaceExporter(testDB, testIDGen, testS3, testES)
}

func newSyncService() *sync_app.SyncService {
	return sync_app.NewSyncService(testDB, testS3, testES, ...)
}
```

按真实构造函数填。

- [ ] **Step 3: 跑测试**

Run: `go test -tags integration ./backend/test/integration/space_sync/ -run TestExportImportRoundTrip -v -timeout 10m`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add backend/test/integration/space_sync/round_trip_test.go backend/test/integration/space_sync/helpers.go
git commit -m "test(integration): add export/import round-trip E2E test"
```

---

### Task 10: E2E 2 - Increment Update

**Files:**
- Create: `backend/test/integration/space_sync/increment_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportTwiceIsUpdate(t *testing.T) {
	ctx := context.Background()
	source := CreateTestSpace(t, WithName("source"), WithAgents(2))
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, source.ID); CleanupSpace(t, target.ID) })

	exporter := newSpaceExporter()
	syncSvc := newSyncService()

	// 第一次导入
	exportData, err := exporter.Export(ctx, source.ID)
	require.NoError(t, err)
	r1, err := syncSvc.ImportConfirm(ctx, target.ID, exportData, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, r1.AgentsCreated)

	// 修改 source 的 agent[0] 名字
	updateAgent(t, ctx, source.Agents[0].ID, "renamed")

	// 第二次导出 + 导入
	exportData2, _ := exporter.Export(ctx, source.ID)
	r2, err := syncSvc.ImportConfirm(ctx, target.ID, exportData2, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 0, r2.AgentsCreated, "should not create new agents on second import")
	require.Equal(t, 2, r2.AgentsUpdated, "should update existing agents")

	// 验证 target 的 agent 名字已更新
	targetReloaded := loadSpace(t, ctx, target.ID)
	require.Equal(t, "renamed", targetReloaded.Agents[0].Name)

	// 验证 sync_mapping 表
	mapping, err := getSyncMapping(t, ctx, source.ID, target.ID, "agent", source.Agents[0].ID)
	require.NoError(t, err)
	require.NotZero(t, mapping.TargetID)
}
```

- [ ] **Step 2: 跑测试**

Run: `go test -tags integration ./backend/test/integration/space_sync/ -run TestImportTwiceIsUpdate -v -timeout 5m`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/test/integration/space_sync/increment_test.go
git commit -m "test(integration): add incremental import update E2E test"
```

---

### Task 11: E2E 3 - Rollback

**Files:**
- Create: `backend/test/integration/space_sync/rollback_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestRollbackRestoresState(t *testing.T) {
	ctx := context.Background()
	space := CreateTestSpace(t, WithName("rollback-test"), WithAgents(1))
	t.Cleanup(func() { CleanupSpace(t, space.ID) })

	original := space.Agents[0].Prompt
	releaseSvc := newReleaseService()

	// Release v1
	_, err := releaseSvc.Release(ctx, space.ID, "1.0.0", "initial release")
	require.NoError(t, err)

	// 修改 agent
	updateAgent(t, ctx, space.Agents[0].ID, space.Agents[0].Name) // 名字不变
	updateAgentPrompt(t, ctx, space.Agents[0].ID, "modified")

	// Release v2
	_, err = releaseSvc.Release(ctx, space.ID, "1.1.0", "after modify")
	require.NoError(t, err)

	// Rollback to v1
	err = releaseSvc.Rollback(ctx, space.ID, "1.0.0")
	require.NoError(t, err)

	// 验证回退
	reloaded := loadSpace(t, ctx, space.ID)
	require.Equal(t, original, reloaded.Agents[0].Prompt)

	// 验证 release_history 表中有 rollback 记录
	last, err := getLatestRelease(t, ctx, space.ID)
	require.NoError(t, err)
	require.Equal(t, "rollback", last.Type)
}
```

- [ ] **Step 2: 跑测试 + 提交**

Run: `go test -tags integration ./backend/test/integration/space_sync/ -run TestRollback -v -timeout 5m`

```bash
git add backend/test/integration/space_sync/rollback_test.go
git commit -m "test(integration): add rollback E2E test"
```

---

### Task 12: E2E 4 - Version Conflict

**Files:**
- Create: `backend/test/integration/space_sync/version_conflict_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestVersionConflictDetected(t *testing.T) {
	ctx := context.Background()
	source := CreateTestSpace(t, WithName("source"), WithAgents(1))
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, source.ID); CleanupSpace(t, target.ID) })

	releaseSvc := newReleaseService()
	exporter := newSpaceExporter()
	syncSvc := newSyncService()

	// Release source v1
	_, err := releaseSvc.Release(ctx, source.ID, "1.0.0", "v1")
	require.NoError(t, err)
	exportData, _ := exporter.Export(ctx, source.ID)

	// 第一次导入
	_, err = syncSvc.ImportConfirm(ctx, target.ID, exportData, ImportOptions{})
	require.NoError(t, err)

	// 第二次 preview 同 release_version → 应检测到 conflict
	preview, err := syncSvc.ImportPreview(ctx, target.ID, exportData)
	require.NoError(t, err)
	require.True(t, preview.VersionConflict, "second preview with same release_version should conflict")
	require.Equal(t, "1.0.0", preview.ConflictingVersion)
}
```

- [ ] **Step 2: 跑测试 + 提交**

```bash
go test -tags integration ./backend/test/integration/space_sync/ -run TestVersionConflict -v
git add backend/test/integration/space_sync/version_conflict_test.go
git commit -m "test(integration): add version conflict detection E2E test"
```

---

### Task 13: E2E 5 - Cross Reference Integrity

**Files:**
- Create: `backend/test/integration/space_sync/reference_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestCrossReferenceIntegrity(t *testing.T) {
	ctx := context.Background()
	// 创建带跨引用的 space：workflow 调用 plugin tool；agent 关联 knowledge
	source := CreateTestSpace(t,
		WithName("source"),
		WithAgents(1),
		WithKnowledge(1),
		WithWorkflows(1),
		WithPlugins(1),
		WithCrossReferences(true),
	)
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, source.ID); CleanupSpace(t, target.ID) })

	exporter := newSpaceExporter()
	syncSvc := newSyncService()

	exportData, _ := exporter.Export(ctx, source.ID)
	_, err := syncSvc.ImportConfirm(ctx, target.ID, exportData, ImportOptions{})
	require.NoError(t, err)

	// 验证跨引用已重写为 target 的 ID
	targetLoaded := loadSpace(t, ctx, target.ID)
	require.NotEqual(t, source.Agents[0].KnowledgeRefs[0], targetLoaded.Agents[0].KnowledgeRefs[0])

	// target 的 agent 应引用 target 的 knowledge ID
	require.Equal(t, targetLoaded.Knowledge[0].ID, targetLoaded.Agents[0].KnowledgeRefs[0])

	// workflow → tool 引用同样验证
	workflowSchema := targetLoaded.Workflows[0].Schema
	require.Contains(t, workflowSchema, formatID(targetLoaded.Plugins[0].ID))
	require.NotContains(t, workflowSchema, formatID(source.Plugins[0].ID))
}
```

`WithCrossReferences(true)` 在 helpers 里实现：让 fixture 生成器创建带跨引用的数据。

- [ ] **Step 2: 跑 + 提交**

```bash
go test -tags integration ./backend/test/integration/space_sync/ -run TestCrossReferenceIntegrity -v
git add backend/test/integration/space_sync/reference_test.go
git commit -m "test(integration): add cross-reference integrity E2E test"
```

---

### Task 14: E2E 6 - Delete Handling

**Files:**
- Create: `backend/test/integration/space_sync/delete_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandling(t *testing.T) {
	ctx := context.Background()
	source := CreateTestSpace(t, WithName("source"), WithAgents(5))
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, source.ID); CleanupSpace(t, target.ID) })

	exporter := newSpaceExporter()
	syncSvc := newSyncService()

	exportData1, _ := exporter.Export(ctx, source.ID)
	_, err := syncSvc.ImportConfirm(ctx, target.ID, exportData1, ImportOptions{})
	require.NoError(t, err)

	// 删除 source 的 2 个 agent
	deleteAgent(t, ctx, source.Agents[3].ID)
	deleteAgent(t, ctx, source.Agents[4].ID)

	// 二次导入
	exportData2, _ := exporter.Export(ctx, source.ID)
	r, err := syncSvc.ImportConfirm(ctx, target.ID, exportData2, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, r.AgentsDeleted, "should mark 2 missing agents as deleted")

	// 验证 target 的对应 agent 已 deleted
	targetLoaded := loadSpaceIncludingDeleted(t, ctx, target.ID)
	deletedAgents := filterDeleted(targetLoaded.Agents)
	require.Len(t, deletedAgents, 2)
}
```

- [ ] **Step 2: 跑 + 提交**

```bash
go test -tags integration ./backend/test/integration/space_sync/ -run TestDeleteHandling -v
git add backend/test/integration/space_sync/delete_test.go
git commit -m "test(integration): add delete handling E2E test"
```

---

### Task 15: E2E 7 - ES Failure Graceful

**Files:**
- Create: `backend/test/integration/space_sync/es_failure_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package space_sync_test

import (
	"context"
	"errors"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestESyncFailureGraceful(t *testing.T) {
	ctx := context.Background()
	source := CreateTestSpace(t, WithName("source"), WithAgents(2))
	target := CreateTestSpace(t, WithName("target"))
	t.Cleanup(func() { CleanupSpace(t, source.ID); CleanupSpace(t, target.ID) })

	exporter := newSpaceExporter()

	// 用 mock ES 替换默认 ES client（注入失败）
	failingES := &failingESClient{err: errors.New("ES unavailable")}
	syncSvc := newSyncServiceWithES(failingES)

	exportData, _ := exporter.Export(ctx, source.ID)
	result, err := syncSvc.ImportConfirm(ctx, target.ID, exportData, ImportOptions{})
	// 期望：返回 success，DB 已提交，仅 ES 同步失败被记 log
	require.NoError(t, err)
	require.Equal(t, 2, result.AgentsCreated)

	// 验证 DB 中 agents 已存在
	targetLoaded := loadSpace(t, ctx, target.ID)
	require.Len(t, targetLoaded.Agents, 2)
}

type failingESClient struct{ err error }

func (f *failingESClient) Index(ctx context.Context, index, id string, doc interface{}) error {
	return f.err
}
// ... 实现 ESClient 接口的其他方法
```

- [ ] **Step 2: 跑 + 提交**

```bash
go test -tags integration ./backend/test/integration/space_sync/ -run TestESyncFailureGraceful -v
git add backend/test/integration/space_sync/es_failure_test.go
git commit -m "test(integration): add ES sync failure graceful degradation E2E test"
```

---

## Phase 3: CI 集成

### Task 16: Makefile target

**Files:**
- Modify or Create: `Makefile`

- [ ] **Step 1: 检查 Makefile**

Run: `cat Makefile 2>/dev/null || echo "no Makefile"`

如果有，加 target；如果没有，创建。

- [ ] **Step 2: 加 target**

```makefile
.PHONY: test-unit test-integration test-all

test-unit:
	go test -short ./backend/... -timeout 5m

test-integration:
	go test -tags integration ./backend/test/integration/... -timeout 15m

test-all: test-unit test-integration
```

如果项目已有其他 Makefile target，注意保留并合并。

- [ ] **Step 3: 验证**

```bash
make test-unit
```
Expected: 单元测试通过（应已含本计划新增的）

- [ ] **Step 4: 提交**

```bash
git add Makefile
git commit -m "build: add test-unit and test-integration Makefile targets"
```

---

### Task 17: CI workflow 集成

**Files:**
- Modify: `.github/workflows/test.yml` 或对应 CI 配置

- [ ] **Step 1: 找到 CI 配置**

```bash
ls .github/workflows/ 2>/dev/null || ls .gitlab-ci.yml 2>/dev/null || echo "no CI config found"
```

- [ ] **Step 2: 加 integration job**

如果是 GitHub Actions：
```yaml
  test-integration:
    runs-on: ubuntu-latest
    needs: test-unit
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run integration tests
        run: make test-integration
        env:
          TESTCONTAINERS_RYUK_DISABLED: 'true'
```

如果是其他 CI（GitLab/Drone），按对应语法。

- [ ] **Step 3: 提交**

```bash
git add .github/workflows/test.yml
git commit -m "ci: add integration test job"
```

---

## Phase 4: 自检与验收

### Task 18: 全量验证

- [ ] **Step 1: 单元测试覆盖率**

```bash
go test -coverprofile=cover.out -short ./backend/application/space/...
go tool cover -func=cover.out | tail -1
```
Expected: `total:` > 60%

- [ ] **Step 2: 单元测试全跑**

```bash
make test-unit
```
Expected: All PASS，30 秒内完成

- [ ] **Step 3: 集成测试全跑**

```bash
make test-integration
```
Expected: All PASS，5 分钟内完成

- [ ] **Step 4: 检查 git 状态**

```bash
git status
git log --oneline -20
```
Expected: 所有 task 都 commit

---

## 验收清单

- [ ] 所有新增单元测试通过（5 个文件）
- [ ] 所有 E2E 测试通过（7 个文件）
- [ ] 单元测试覆盖率 > 60%（针对 space/import、space/export、space/sync、space/release）
- [ ] `make test-unit` 30 秒内
- [ ] `make test-integration` 5 分钟内
- [ ] CI integration job 跑通
- [ ] 测试可重复运行（多次跑结果一致）
- [ ] 不破坏现有源码（git diff 仅含测试文件 + Makefile + CI 配置）
