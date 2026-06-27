# TODO · 超级智能体 + 沙箱独立化

> 状态: ☐ 未开始 / ◐ 进行中 / ☑ 完成 / ⏸ 待确认

## Phase 0 · 基线与分支  〔checkpoint 0〕
- [ ] ⏸ 确认分支基底 + display-copy 改动处理方式（见 plan.md「待你确认」）
- [ ] 建分支 `feat/agent-sandbox-superagent`
- [ ] 基线 `go build ./...` + 相关包 `go test` 绿

## Phase 1 · 沙箱独立化
- [ ] 1A 新建 `backend/pkg/agentsandbox/`，定义 Cache/Blob 注入接口
- [ ] 1B 迁入 Runner+Docker+SessionManager+Registry+Reaper+Facade，去除对 coze infra 直接依赖
- [ ] 1C coze 侧薄适配（infra→Cache/Blob 实现）+ application.go 装配 + crossdomain Facade
- [ ] 验收：边界无反向依赖 / 行为不变 / build 通过  〔checkpoint 1〕

## Phase 2 · edit_file + grep + glob
- [ ] 2A Facade+crossdomain 接口新增 EditFile/Grep/Glob
- [ ] 2B node_tool_sandbox.go 加三个工具 + 注册
- [ ] 2C system_prompt.go 加使用纪律
- [ ] 2D 单测：唯一/多匹配报错/replaceAll/未找到
- [ ] 验收：单测+build+schema 冒烟  〔checkpoint 2〕

## Phase 3 · context 压缩 + 两轴审批（可选）
- [ ] 工具输出 LLM 摘要
- [ ] 危险工具前置审批 hook + plan 只读模式
- [ ] 验收：审批/压缩单测

## Phase 4 · 测试与验证
- [ ] 单元回归（EditFile/Grep/Glob + 沙箱模块）
- [ ] 集成（docker runner，环境允许时）
- [ ] go build ./... + go vet
- [ ] （可选）真实沙箱端到端冒烟
- [ ] 输出测试报告

## Phase 5 · 后续（不在本次）
- [ ] repo_map(codegraph) / adk 接通 DeepAgent / 子 agent / provider 路由
