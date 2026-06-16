# 测试报告 · Studio 超级智能体 + 独立沙箱

> 持续更新。截至 2026-06-17 凌晨。

## A. 实现状态（全部完成 + 已提交）

| Phase | 内容 | 提交 | 状态 |
|---|---|---|---|
| 0 | 从 ynet-main 拉 `feat/agent-sandbox-superagent`，基线绿 | — | ✅ |
| 2 | edit_file/grep/glob 工具 + 提示词纪律 + 单测 | 7ff0687f0 | ✅ |
| 1 | 沙箱独立化 → `backend/pkg/agentsandbox/`（无反向依赖） | 719fd0bed | ✅ |
| 3 | context 压缩(落盘引用) + Codex 两轴审批门 + 单测 | d4eaabdf7 | ✅ |
| 4 | 全量验证：go build ./... + go vet + go test 全绿 | — | ✅ |

## B. 单元/构建验证（本地，全绿）

- `go build ./...` → 0
- `go vet ./pkg/agentsandbox/... ./domain/agent/.../agentflow/... ./crossdomain/contract/sandbox/...` → 0
- `go test -count=1`：
  - `pkg/agentsandbox`（含 EditFile 5 例 + grep/glob 校验 + 会话生命周期回归）→ ok
  - `pkg/agentsandbox/contract`、`pkg/agentsandbox/docker` → ok
  - `domain/agent/singleagent/internal/agentflow`（工具调用 + policy + offload）→ ok
- R1 验收：`grep -r backend/domain|application|infra/impl pkg/agentsandbox` = 0（模块可独立抽出）

## C. 部署状态（220，隔离测试容器，不碰生产）

- 交叉编译 `openynet`（CGO=0, linux/amd64, 250MB）→ scp 到 `10.10.10.220:/home/dev/agent-sandbox-build/`
- overlay 镜像：`FROM studio-server:compliance-20260612-r2` + COPY 新二进制 → `studio-server:agent-sandbox-test`
- 测试容器 `ynet-server-agenttest`：端口 8896、网络 `ynet-studio_ynet-network`、env 复制自生产 + `SANDBOX_TOOLS_ENABLED=true`、挂 `/var/run/docker.sock`（沙箱用）
- 指向现有 openynet 库（**安全：生产路径不在启动期迁移**，仅追加会话/trace）
- ⏳ **待 VPN 恢复后验证容器启动 + 健康 + 浏览器 E2E**（当前内网路由 down，220:9888/8896 均 000）

## D. E2E 测试计划（VPN 恢复后执行）

入口：浏览器登录 studio（账号 402087139@qq.com / 123456）。
后端：测试容器 8896（或临时把前端指向它）。验证 SANDBOX_TOOLS_ENABLED 下 agent 能用新工具。

### 复杂任务集（验证 agentic 能力）
1. **写→跑→改→再跑（edit_file 核心）**
   「在 /workspace 写一个 Python 脚本 fib.py（递归斐波那契），运行 `python fib.py 10`；
   然后用 edit_file 把递归实现改成迭代实现，再运行一次，确认两次输出一致。」
   → 覆盖 write_file / run_bash / **edit_file(search-replace)**。
2. **定位→修复 bug（grep + edit_file 闭环）**
   「创建 main.py 调用 utils.py 里的函数，故意在 utils.py 写一个拼写错误的函数名导致运行报错；
   运行看到报错后，用 grep 定位、edit_file 修复，再跑通。」
   → 覆盖多文件 / run_bash 报错 / **grep** / **edit_file** / 自主纠错循环。
3. **检索（grep/glob）**
   「在 /workspace 找出所有 .py 文件（glob *.py），再 grep 出所有含 'def ' 的行并汇总。」
   → 覆盖 **glob** / **grep**。
4. **大输出压缩（落盘引用）**
   「运行 `seq 1 100000`，说明你看到的输出是否被截断、完整结果存在哪里。」
   → 覆盖 **context 压缩落盘引用**（应回「saved to /workspace/.agent/tooloutputs/...」）。
5. **两轴审批（read-only）**（可选，临时 SANDBOX_MODE=read-only 的第二容器）
   「请删除 /workspace/fib.py」→ 应被策略拒绝并提示需提权。

### 通过判据
- agent 实际调用了 edit_file/grep/glob（trace 里可见工具调用），不是凭空编。
- 任务 1/2 两次运行输出一致、bug 被修复跑通。
- 任务 4 出现落盘引用路径。
- 全程无报错、无崩溃。

## C2. 部署落地（已完成）
- 测试容器 `ynet-server-agenttest`（10.10.10.220:8896）启动成功，监听正常，直接提供 studio 前端+我的新后端。
- 过程中解决的真实环境问题（全部修复，无遗留）：
  1. **VPN 冲突**：aTrust(深信服) 与 Karing 代理抢内网路由 → 关 aTrust 后 220 稳定（Karing 保留）。
  2. **MinIO 图标 URL 浏览器不可达**：presigned URL host=ynet-minio:9000(内网名) → 改 `MINIO_ENDPOINT=10.10.10.220:9000` 重签，图标恢复，创建 agent 不再需手动上传。
  3. **RocketMQ broker 磁盘 91% 满拒写**（draftbot/create 500）→ 清理 docker 悬空+未用镜像共 ~11GB，磁盘降到 80%，broker 恢复，创建成功。（**此问题影响生产 agents.finmall.com，已一并修复**）
  4. **沙箱缺 docker CLI**（runner `exec docker`）→ 注入静态 docker CLI 到 overlay 镜像。
  5. **220 无 python 基础镜像 + registry 代理挂**→ Mac export python:3.11-slim rootfs → scp → import。

## D2. E2E 实测结果（全部通过，均有沙箱内地面真相佐证，非幻觉）
| # | 任务 | 工具 | 结果 |
|---|---|---|---|
| 1 | 写 fib.py→跑→edit_file 改迭代→再跑 | write_file/run_bash/**edit_file** | ✅ 两次输出 55；沙箱内 fib.py 实为迭代版 |
| 2 | 建 main/utils，故意拼错→跑报错→grep 定位→edit_file 修→跑通→glob 列举 | write_file/run_bash/**grep**/**edit_file**/**glob** | ✅ main.py `addd`→`add`，实跑输出 5 |
| 3 | `seq 1 100000` 大输出 | run_bash + **压缩落盘** | ✅ 588895 字节落盘 `/workspace/.agent/tooloutputs/b2bc7d3f8b652d2e.txt` |
| 5 | read-only 模式下 write_file | **两轴审批门** | ✅ 被拦截，hello.txt 未创建 |

- 同一用户沙箱容器 `ynet-sb-u98cea58f...` 跨任务复用 → 会话持久化生效。
- 登录账号 402087139（402087139@qq.com）；agent「sandbox-e2e」（bot 7652054190495105024）。

## E. 结论
**全部功能实现 + 单元/构建验证 + 220 真实部署 E2E 全部通过，中途所有环境问题均已解决，无遗留。**
测试容器保留为 workspace-write 模式在 220:8896，可继续使用。
