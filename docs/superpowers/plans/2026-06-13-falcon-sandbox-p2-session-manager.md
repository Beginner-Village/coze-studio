# 猎鹰沙箱 P2 · 会话管理器 Implementation Plan

> REQUIRED SUB-SKILL: superpowers:executing-plans。Steps 用 `- [ ]`。

**Goal:** 在 `domain/sandbox` 实现 user_id 维度的会话级沙箱管理器：四态生命周期、single-flight 冷启动、Redis 注册表、MinIO workspace 持久化、空闲回收；接入 `appinfra.AppDependencies`。

**Architecture:** `Manager` 组合 `sandbox.Runner`(P1) + 窄接口 `Registry`(Redis 实现) + `storage.Storage`(MinIO)。`EnsureSandbox(key)` 用 `singleflight.Group` 去重并发冷启动：注册表有且 running→touch；paused→resume；无/dead→create+从 MinIO 还原 workspace+登记。工具方法 Exec/Read/Write/List 先 ensure 再委派再 touch。`Checkpoint` 把 `/workspace` tar 进 MinIO。`Reaper` goroutine 扫注册表，按阈值 pause/kill。Registry 窄接口让单测用内存 fake，避免 fake 庞大的 `cache.Cmdable`。

**Tech Stack:** Go、golang.org/x/sync/singleflight、contract/cache(Redis)、contract/storage(MinIO)。

---

### Task 1: Registry（接口 + Entry + Redis 实现 + 内存 fake）
- `domain/sandbox/registry.go`：`Registry` 接口（Put/Get/List/Delete/Touch/SetState）、`Entry{SandboxID,State,LastActiveUnix}`、`RedisRegistry`（用 cache.Cmdable 的 HSet/HGetAll/Del；hash key `sandbox:registry`，field=id value=json）。
- `domain/sandbox/fakes_test.go`：`memRegistry`、`fakeRunner`（内存模拟 create/exec/file/state/pause/resume/kill）、`fakeStorage`。
- `domain/sandbox/registry_test.go`：RedisRegistry 用 fakeCache 或直接测内存 fake 的契约一致性。
- 验收：`go test ./domain/sandbox/ -run Registry` 过。

### Task 2: Manager 生命周期（EnsureSandbox 四态 + single-flight）
- `domain/sandbox/manager.go`：`Manager`、`Config`、`New`、`EnsureSandbox`、`Exec/WriteFile/ReadFile/ListFiles`（先 ensure 后委派后 touch）。`now func() int64` 可注入时钟。
- `manager_test.go`：冷启动建沙箱并登记；热复用不重复建；paused→resume；并发 EnsureSandbox 只建一次（singleflight）。
- 验收：`go test ./domain/sandbox/ -run Manager` 过。

### Task 3: 持久化 Checkpoint/Restore（MinIO）
- `manager.go` 加 `Checkpoint(key)`（exec tar /workspace → ReadFile → storage.PutObject `workspaces/{key}.tgz`）与冷启动时 `restore`（GetObject→WriteFile→exec tar x）。
- `integration_test.go`（gated on docker）：写文件→Checkpoint→Kill→EnsureSandbox(还原)→文件还在。
- 验收：有 docker 时集成测试过。

### Task 4: Reaper 空闲回收
- `domain/sandbox/reaper.go`：`Reaper{mgr,pauseSec,killSec}`、`Start(ctx)`/一次性 `sweep(now)`。idle>pauseSec→Pause+SetState；idle>killSec→Checkpoint+Kill+Delete。
- `reaper_test.go`：用 fake + 可控时钟验证 pause/kill 阈值。
- 验收：`go test ./domain/sandbox/ -run Reaper` 过。

### Task 5: 接入 appinfra
- `appinfra/app_infra.go`：`AppDependencies` 加 `SandboxManager *sandbox.Manager`；`Init` 末尾 `deps.SandboxManager = sandbox.New(...)` 并启动 Reaper。后端用 Docker runner（env `SANDBOX_BACKEND` 默认 docker）。
- 验收：`go build ./...` 全绿。

## P2 验收
- `go test ./domain/sandbox/...` 全绿；docker 集成测试持久化往返通过；`go build ./...` 绿。
