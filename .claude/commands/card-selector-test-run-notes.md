# 卡片选择节点试运行排错与改造笔记

## 现象
- 单节点右上角“运行”与底部“运行”调用链不同，但两者都拿不到预期输入/输出
- 后端日志显示：CardSelector.Invoke 收到空输入；GetProcess/NodeHistory 返回的 input/output 为空或仅含默认字段

## 根因
- 试运行链路中 `vo.Node.Data` 被反序列化为空对象 `{}`，导致节点适配器 `Adapt` 无法从 `node.data` 里拿到前端配置（`cardSelectorParams/selected_card_id/selected_card`）
- 卡片选择节点缺少输出回调，导致输出无法以模板形式在前端正确展示

## 关键改动
1) 数据结构增强
- 在 `vo.Data` 中新增以下可选字段（不会影响其他节点）：
  - `CardSelectorParams map[string]any`
  - `SelectedCardID string`
  - `SelectedCard map[string]any`

2) 节点适配器 `Adapt` 兜底解析
- 优先从 `n.Data` 解析上述三个字段
- 解析不到时打印调试日志，并从传入的 `WithCanvas(canvas)` 原始 `canvas` 结构回退提取（注意：试运行快照偶发缺失 data，但完整 canvas 仍然包含）
- 全链路增加关键日志，便于排错：
  - `Adapt`: Node data / Parsed dataMap / Final config
  - `Build`: 最终构造的 `CardSelector` 配置
  - `Invoke`: 输入关键字段存在性及分支命中

3) 试运行 I/O 显示
- 新增 `ToCallbackOutput`：若存在 `OutputKeyTemplateResponse`，以模板格式作为展示输出（rawOutput 保留原始值）
- 完善 `ToCallbackInput`：同时兼容 `selected_card` 与 `selected_card_id`，回显 API 配置与变量输入

## 验证要点
- 单节点运行与流程运行都能在 GetProcess 的 `nodeResults[].input/output/raw_output` 中看到期望值
- 日志中 `Adapt/Build/Invoke` 都能看到非空配置与输入

## 可能再踩的坑与规避
- 试运行使用“快照”，快照可能不带 node.data，必须保留 `WithCanvas` 回退解析
- 若将来前端调整字段命名（snake/camel），适配器需同时兼容（已同时支持 `selected_card*` 与 `cardSelectorParams.selectedCard*`）
- 避免将编译产物/备份文件纳入提交（例如 `backend/coze-studio`、`*.bak`）

## 变更清单（后端）
- `backend/domain/workflow/entity/vo/canvas.go`
  - `Data` 增加 3 个可选字段，承接前端配置
- `backend/domain/workflow/internal/nodes/cardselector/card_selector.go`
  - `Adapt`：解析/兜底解析 + 调试日志
  - `Build`：打印构造信息
  - `ToCallbackInput/ToCallbackOutput`：增强输入回显与模板化输出

## 排错命令速记
```bash
# 编译后端
(cd backend && go build -o coze-studio ./main.go)

# 关注日志关键字
rg "CardSelector (Adapt|Build|Debug)" -n backend | cat
```
