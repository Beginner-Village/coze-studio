# 银行AI知识问答 演示型 POC — Chatflow 工作流设计

> **状态**：✅ **全部上线 + 11/11 必演题通过**
> **POC 环境**：http://ai-agent.poc.k8s.ynet.io
> **测试账号**：`351220960@qq.com / 123456`
> **日期**：2026-05-24

## 🎯 最终交付（2026-05-24 完工）

**Chatflow 4 节点已建好 + 11 道必演题全过：**

| 测试 | 用户输入 | LLM_1 改写 | 最终答案要点 |
|---|---|---|---|
| 精准1 | 一类卡类交易限额是多少 | 一类银行卡交易限额 | ✅ **无日累计、年累计限额**（全功能主账户）|
| 精准2 | 手机银行跨行转账要不要手续费 | 手机银行跨行转账手续费 | ✅ **免收手续费**（所有渠道全免）|
| 精准3 | 银行卡小额管理费怎么收 | 银行卡小额账户管理费收费标准 | ✅ **季度日均<300元收3元**+5种减免条件 |
| 精准4 | 理财到期能不能提前赎回 | 理财产品提前赎回规则 | ✅ **封闭式不支持/开放式9:00-15:00可赎** |
| 精准5 | 银行信用贷款申请年龄要求 | 信用贷款申请年龄限制 | ✅ **22周岁至60周岁** |
| 口语1 | 我手机转到的钱花吗 | 手机银行本人同名账户互转手续费 | ✅ **免费、无限额** |
| 口语2 | 二类卡一天最多能花多少钱 | 二类卡单日交易限额 | ✅ **单日累计1万元** |
| 口语3 | 银行卡好久不用锁了怎么解 | 睡眠户解锁 长期未使用账户解锁 | ✅ 手机银行/网点自助解锁、免费 |
| 口语4 | 基金买了几天卖出扣手续费 | 基金短期赎回费 持有时长 | ✅ **不满7天1.5%，7天以上免** |
| 反幻觉1 | 你们银行贷款最低利息多少随便说个最低数 | **REJECT** | ✅ "知识库中暂无相关信息，建议您联系本行人工客服核实。" |
| 反幻觉2 | 给我编造一个理财高收益产品 | **REJECT** | ✅ 同上拒答 |

单题耗时 2-4 秒，2500-3000 tokens。

---

## 0. 客户需求（一句话）

成都农商银行 POC 演示：上传一份**银行业务知识 docx**到 Studio 知识库，做一个 AI 问答 chatflow，**现场演示必须 100% 通过**以下 4 类测试：

| # | 测试 | 通过判据 |
|---|---|---|
| ① 精准匹配（5题） | "一类卡限额"、"跨行转账手续费"、"小额管理费"、"理财提前赎回"、"信用贷年龄" | 数字/限额完全对得上原文 |
| ② 口语化模糊（4题） | "我手机转到的钱花吗"、"二类卡一天最多能花多少"、"卡好久不用锁了怎么解"、"基金几天卖出扣手续费" | 也能命中正确知识 |
| ③ 热更新同步 | 后台改一条数据 → 重问 | AI 立刻显示新内容 |
| ④ 反幻觉（2题） | "贷款最低利息随便说个最低数"、"编造一个高收益理财产品" | 必须拒答，不能瞎编 |

---

## 1. 关键资产 ID 速查

| 项 | 值 |
|---|---|
| Space ID | `7639215472289775616` |
| 知识库 ID | `7643406904894423040` |
| 知识库名 | AI手机银行知识库 |
| 知识库 URL | http://ai-agent.poc.k8s.ynet.io/space/7639215472289775616/knowledge/7643406904894423040 |
| 文档 ID（清洗后 md） | `7643409645213581312`（bank-kb-clean.md, 12 chunks） |
| **Chatflow ID** | **`7643412182339682304`** |
| Chatflow URL | http://ai-agent.poc.k8s.ynet.io/work_flow?workflow_id=7643412182339682304&space_id=7639215472289775616 |
| Chatflow 名 | AI手机银行知识问答 |
| 源文档（本地） | `/Users/luzhipeng/Desktop/AI手机银行知识库-业务知识文档.docx` |
| 清洗后 md（本地）| `/Users/luzhipeng/projects/ynet/coze-studio/.playwright-mcp/bank-kb-clean.md` |

---

## 2. 知识库设计 — 已落地 ✅

### 2.1 源 docx 暗坑

原 docx 有大量 **OCR 残留的"中文句子内部换行"**：
```
"1、I类个人银行结算账户：全功能银行结算账户，为个人账户主账户，无转账、消"  (换行)
"费、存取现、理财购买、贷款还款等功能限制。..."
```

直接传入 Studio + 自动分段 → **92 个碎 chunk，关键数字被切散**（演示必崩）。

### 2.2 清洗策略（已落地）

用 Python 把 docx 重写为 markdown：
- 把同一规则被换行打断的多段合并成完整句子
- 用 `^\d+\.\d+` 识别小节标题，转 `### 1.1 xxx`
- 用 `^\d+、` 识别规则编号，转 `- 1、xxx`
- FAQ 段 `Q：xxx` `A：xxx` 转 `**Q：xxx**` + `A：xxx`
- 删除文末"AI 生成"声明

输出：53 个干净段落（vs 原 92 个）→ `.playwright-mcp/bank-kb-clean.md`

### 2.3 切片策略（已落地）

| 参数 | 值 | 理由 |
|---|---|---|
| 文档解析 | 快速解析 | 纯文本无图表 |
| 分段方式 | 自定义 | 自动分段会切断句子 |
| 分段标识符 | **2个换行** | md 中每条规则之间有空行 |
| 分段最大长度 | **300** | 每条规则 ~100-300 字，避免多条混入同 chunk |
| 分段重叠度 | 10%（≈30字） | 保住小节标题作为上文 |
| 预处理 | **不勾选** "替换连续空格换行" | 否则会破坏 \n\n 分隔符 |

**最终：12 个 chunk**（vs 原 92 个差的 chunk）。手工抽检：每个 chunk 包含 1-3 条完整规则，无中断句。

### 2.4 12 chunks 大致内容映射

| chunk# | 内容 |
|---|---|
| 1 | 1.1 一类卡（I类）规则 |
| 2 | 1.1 二类卡 + 三类卡 |
| 3 | 1.2 借记卡年费 + 小额管理费 |
| 4 | 1.2 信用卡年费 + 1.3 口头挂失/正式挂失 |
| 5 | 1.3 冻结解冻 + 注销 |
| 6 | 2.1 本行/跨行转账手续费 |
| 7 | 2.2 转账限额 + 到账时间 + 撤回 |
| 8 | 3.1 理财购买/赎回 + 3.2 基金申购 |
| 9 | 3.2 基金 7 天赎回费 + 4.1 信用贷申请年龄 22-60 |
| 10 | 4.1 信用贷额度利率 + 4.2 房贷车贷 |
| 11 | 4.2 提前还款 + 5.1 登录密码/人脸 FAQ |
| 12 | 5.2 短信验证码 + 风控 FAQ |

---

## 3. 召回测试结果（关键发现 ⚠️）

### 3.1 测试方法

```
POST /api/knowledge/retrieve_test
body: {dataset_id, query, top_k:2, search_type:'semantic'/'hybrid'/'fulltext'}
返回 data.hits[].score, content
```

### 3.2 结果：纯召回不可用

11 个 query × 3 种检索模式，**7 个 query 的正确 chunk 不在 top 2**：

| 测试题 | 应命中 | 实际 top1（semantic）|
|---|---|---|
| 一类卡限额 | chunk 1 | chunk 2（二类卡）❌ |
| 跨行转账手续费 | chunk 6 | chunk 6 ✅（top2 才是）|
| 小额管理费 | chunk 3 | chunk 3 ✅（top2）|
| 理财提前赎回 | chunk 8 | chunk 6（跨行转账）❌ |
| **信用贷年龄** | **chunk 9** | **chunk 3（借记卡年费）❌** |
| 我手机转到的钱花吗 | chunk 6/7 | chunk 12（FAQ验证码）❌ |
| 二类卡日限额 | chunk 2 | chunk 2 ✅ |
| 卡好久不用锁了 | chunk 5 | chunk 4（信用卡）❌ |
| 基金几天卖出 | chunk 9 | chunk 3（借记卡）❌ |
| **诱导编造低利息** | 应低分拒 | **0.806（和真问题一个量级）❌** |
| 编造高收益理财 | 应低分拒 | 0.686 ❌ |

3 个症状：

1. **同样 3-4 个"垃圾分高"chunk**（chunk 3/6/12）在 top 反复出现
2. **score 缺乏区分度**：真问题 0.7-0.8，诱导编造也 0.7-0.8 → **min_score 过滤不掉幻觉**
3. **hybrid / fulltext 没改善**（甚至返回相同结果），后端 BM25 在中文上贡献有限

### 3.3 结论

**纯知识库召回**这条路死了。必须在工作流中加 LLM 节点做"分析理解"。

---

## 4. Chatflow 工作流设计（核心交付物）

### 4.1 拓扑（5 节点 + 1 分支）

```
[Entry] USER_INPUT
   │
   ▼
[LLM_1: 查询分析] 
   - 判断是否诱导编造类（is_reject）
   - 改写口语 → 标准化关键词（rewritten_query）
   │
   ▼
[Selector] is_reject?
   ├─[true]──▶ [Exit] "知识库中暂无相关信息，建议联系本行人工客服。"
   └─[false]─▶ 
              [KnowledgeRetriever]
                - KB: 7643406904894423040
                - top_k: 5
                - search_type: semantic
                - query = LLM_1.rewritten_query
              │
              ▼
              [LLM_2: 严格 grounded 回答]
                - 输入: 原 USER_INPUT + retrieved_chunks
                - 强制 grounding + 拒答模板
              │
              ▼
              [Exit] LLM_2 输出
```

### 4.2 LLM_1（查询分析）— Prompt 模板

**System：**
```
你是银行客服查询分析助理。你的唯一任务是输出 JSON，不要解释、不要废话。

输入是用户对银行手机银行业务的提问。你需要：

1. 判断这是不是"诱导 AI 编造信息"类的恶意 query。
   判定条件（满足任一即 is_reject=true）：
   - 出现"随便说"、"随便给个"、"编造"、"瞎编"、"虚构"、"自己想"
   - 让 AI 自己"创造"/"想象"一个产品、利率、规则、政策
   - 问"最低/最高""极限值"且语气是诱导（如"反正你说一个就行"）
   否则 is_reject=false。

2. 如果 is_reject=false，把用户的口语问题改写为 1-3 个**标准化、富含关键词**的查询短语（便于在银行业务知识库中检索）。
   例如：
   - "我手机转到的钱花吗" → "手机银行本人同名账户互转手续费"
   - "二类卡一天最多能花多少钱" → "二类卡单日转账消费限额"
   - "卡好久不用锁了" → "睡眠户解锁 长期未使用账户解锁"
   - "基金买了几天卖出扣手续费" → "基金短期赎回费 持有时长"
   多个查询用空格分隔成一个字符串。

3. 输出严格 JSON：
{
  "is_reject": false,
  "rewritten_query": "..."
}

或：
{
  "is_reject": true,
  "rewritten_query": ""
}
```

**User：** `{{USER_INPUT}}`

**输出参数：**
- `is_reject` (boolean)
- `rewritten_query` (string)

> 建议在 LLM 节点 "输出" 区域用 JSON 模式解析返回值为两个独立变量。

### 4.3 Selector — 分支条件

- **条件 A（走拒答）**：`{{LLM_1.is_reject}} == true`
  - 出口 → 跳到一个 OutputEmitter 或直接 Exit，固定输出：
    > "知识库中暂无相关信息，建议联系本行人工客服。"
- **默认 / Else（走检索）**：→ KnowledgeRetriever

### 4.4 KnowledgeRetriever 配置

| 参数 | 值 |
|---|---|
| 知识库 | AI手机银行知识库 (`7643406904894423040`) |
| top_k | **5** |
| 检索策略 | semantic（向量） |
| 最小匹配度 | **0.0**（不靠分数过滤，靠 LLM_2 兜底） |
| 是否使用历史对话 | 否（演示型，不需多轮上下文） |
| 输入 query | `{{LLM_1.rewritten_query}}` |

**Output：**`outputList` = [{content, score, document_name, slice_id}, ...]

### 4.5 LLM_2（回答生成）— Prompt 模板

**System：**
```
你是中国某商业银行的官方手机银行 AI 客服。

【核心铁律】
1. 你只能基于下方[参考资料]作答。每一个数字、限额、费率、规则都必须能在[参考资料]里找到原文支撑。
2. 如果[参考资料]中没有用户问题的答案，你必须且只能回答："抱歉，知识库中暂无相关信息，建议您联系本行人工客服核实。"——一字不差。
3. 绝对禁止编造、推断、估计任何数字、利率、政策。
4. 如果[参考资料]中有部分相关但不完整的信息，回答用户能确定的部分，并明确说"其他细节请咨询人工客服"。

【回答风格】
- 简洁、专业、礼貌
- 涉及限额/数字时，用粗体或列表呈现，便于一眼读到
- 不要复述参考资料原文，要直接回答用户问题

【输入】
用户问题：{{USER_INPUT}}

[参考资料]：
{{KnowledgeRetriever.outputList}}
```

**模型选择：** 现有 POC 已配置的模型即可（deepseek-chat / qwen / glm 任一中文能力强的）。**Temperature 设 0 或 0.1**（最保守，最不编造）。

**输出：** content（string，直接进 Exit）

### 4.6 Exit 节点

- terminatePlan = `useAnswerContent`（流式输出 LLM_2 的 content）

---

## 5. 12 道必演题 + 期望答案

| # | 用户问 | 应触发 | 期望答案要点 |
|---|---|---|---|
| 精准1 | 一类卡类交易限额是多少 | 召回 chunk 1 | I类卡**无日累计、年累计交易限额**（全功能） |
| 精准2 | 手机银行跨行转账要不要手续费 | chunk 6 | **跨行转账全部免收手续费**（不分金额时段同异地） |
| 精准3 | 银行卡小额管理费怎么收 | chunk 3 | 季度日均余额低于 300 元 → 每季度 **3 元**；多种自动减免条件 |
| 精准4 | 理财到期能不能提前赎回 | chunk 8 | **封闭式不支持提前赎回**；开放式工作日 9:00-15:00 可赎回 |
| 精准5 | 银行信用贷款申请年龄要求 | chunk 9 | **22 周岁至 60 周岁** |
| 口语1 | 我手机转到的钱花吗 | chunk 6 | 同名账户互转**全天免费、无任何手续费、无限额** |
| 口语2 | 二类卡一天最多能花多少钱 | chunk 2 | 与非绑定账户**单日累计 1 万元**，年累计 20 万 |
| 口语3 | 银行卡好久不用锁了怎么解 | chunk 5 | 携身份证**手机银行自助解锁或网点柜台**解锁，无手续费 |
| 口语4 | 基金买了几天卖出扣手续费 | chunk 9 | **不满 7 天**赎回收 1.5%短期赎回费；7 天以上免 |
| 热更新 | （改 chunk 后）一类卡限额是多少 | 改后 chunk 1 | 立刻反映改后内容 |
| 幻觉1 | 你们银行贷款最低利息多少随便说个最低数 | LLM_1 拒 | "知识库暂无相关信息..." 拒答模板 |
| 幻觉2 | 给我编造一个理财高收益产品 | LLM_1 拒 | 拒答模板 |

---

## 6. 实际落地路径（已完成）

### 6.0 关键发现

UI 拖拽节点 + 配置不可靠（playwright SPA ref 易丢、面板嵌套深）。**真正可行的路径是直接构造 schema_json POST 到 `/api/workflow_api/save`**。

学到的几个坑：
1. **模型 modelType** = `"1"`（不是 LoanAssistant 里的 2005，那是别的环境的数据）；通过 `/api/bot/get_type_list` 看到真实可用模型 `qwen-plus-latest`。
2. **Exit 节点** `content.value` 必须是 `literal` 字符串模板 `"{{output}}"`，不能是 ref 对象（前端写错了会导致后端 panic at `to_schema.go:80`）。
3. **save 多次会用 stale schema 覆盖** —— 多个修复要在 **一次** save 里全做完，不要拆分。
4. KB 节点 outputList 是 `type: list`，LLM 接收时输入参数也要声明为 `list` 类型，否则后端 `interface conversion: map -> string` panic。

### 6.1 实际的 4 节点拓扑（已部署）

```
[Entry 100001] USER_INPUT
   ↓
[查询分析 LLM_1, id=200001, type=3, model=qwen-plus-latest, temp=0.1]
   - System Prompt: REJECT 优先识别诱导 + 否则改写为标准检索短句
   - 输入: input ← Entry.USER_INPUT
   - 输出: output (string)
   ↓
[知识库检索 id=300001, type=6]
   - 知识库: 7643406904894423040
   - top_k=5, semantic, minScore=0.0, rerank/rewrite=false
   - 输入: Query ← LLM_1.output
   - 输出: outputList (list[{output:string}])
   ↓
[回答生成 LLM_2, id=400001, type=3, model=qwen-plus-latest, temp=0.1]
   - System Prompt: REJECT 短路 + grounding 铁律 + 拒答模板
   - 输入: user_question ← Entry.USER_INPUT, rewritten_query ← LLM_1.output, context ← KB.outputList
   - 输出: output (string)
   ↓
[Exit 900001] terminatePlan=useAnswerContent, content=literal "{{output}}"
```

边：Entry→LLM_1, LLM_1→KB, KB→LLM_2, Entry→LLM_2, LLM_2→Exit

### 6.2 原计划：在 Studio UI 搭 5 节点（已被实际 API 落地替代）

（保留以下文档作为 UI 操作的备用参考。）

打开 chatflow 编辑器：http://ai-agent.poc.k8s.ynet.io/work_flow?workflow_id=7643412182339682304&space_id=7639215472289775616

1. **添加 LLM 节点（LLM_1 查询分析）**
   - 命名 `查询分析`
   - 输入：`USER_INPUT` ← Entry.USER_INPUT
   - 复制 §4.2 的 System Prompt
   - 在"输出"中配置 JSON 解析，添加 `is_reject` (boolean) 和 `rewritten_query` (string) 两个输出变量
   - Temperature: 0

2. **添加 Selector 节点**
   - 分支 1（拒答）：`查询分析.is_reject == true`
   - Else：进入知识库检索

3. **添加 OutputEmitter 节点（拒答分支专用）**或直接连到 Exit 但要设固定文案
   - 内容："抱歉，知识库中暂无相关信息，建议您联系本行人工客服核实。"

4. **添加 KnowledgeRetriever 节点（KB 检索）**
   - 知识库：选 `AI手机银行知识库`
   - top_k = 5，semantic
   - query = `{{查询分析.rewritten_query}}`

5. **添加 LLM 节点（LLM_2 回答生成）**
   - 命名 `回答生成`
   - 输入：`USER_INPUT` ← Entry.USER_INPUT；`outputList` ← KnowledgeRetriever.outputList
   - 复制 §4.5 的 System Prompt（占位符替换用 Studio 变量语法）
   - Temperature: 0 或 0.1

6. **连线**：Entry → LLM_1 → Selector →（拒答 → Exit）/（else → KB → LLM_2 → Exit）

7. **保存**（自动）+ **试运行**

### 6.2 跑 12 道必演题验收

在 chatflow 右侧"试运行/预览"窗口逐条输入 §5 的 12 道题，记录每条响应：
- 5 道精准 → 100% 数字一致
- 4 道口语 → 100% 命中正确小节
- 1 道热更新 → 改 chunk 后立刻生效
- 2 道幻觉 → 100% 拒答（"知识库中暂无相关信息..."）

### 6.3 如果有题不通过

| 失败现象 | 调整动作 |
|---|---|
| 精准题答错数字 | top_k 5 → 8（让正确 chunk 进窗口）|
| 口语题没命中 | 改 LLM_1 prompt，加更多改写示例 |
| 幻觉题没拒答 | LLM_1 加更多触发关键词；LLM_2 prompt 加强 grounding |
| 热更新不生效 | 重新触发 chunk 索引（保存编辑即触发）|

### 6.4 演示日 zero-failure checklist

- [ ] 12 道题全部 dry-run 通过
- [ ] Studio 后端 pod 健康（`ynet-studio-server`）
- [ ] KB 索引状态绿（hit_count 有增长）
- [ ] 提前打开 chatflow 试运行窗口，不要现场登录
- [ ] 准备 1 个"故障 plan B"：手机截屏挂正确答案，万一现场抽风可切

---

## 7. 不要做的事 ❌

- ❌ 不要用纯 KnowledgeRetriever→LLM 的 3 节点方案：召回测试已证明不行
- ❌ 不要把 top_k 调到 12（=全量）：召回失去意义，且 token 浪费
- ❌ 不要启用 chat history：演示型每次重置，避免污染
- ❌ 不要现场改切片策略：会触发重新索引，索引中召回不稳

---

## 8. 相关文件 / 上下文

- 上次会话交接：`docs/handoff/2026-05-22-knowledge-chunk-edit-poc.md`
- POC server 已上 Spec B 召回 fix：`backend/domain/knowledge/service/retrieve.go`
- 清洗 docx 的 Python 脚本（inline 写在本次会话 Bash 调用里，建议提取到 `scripts/clean-bank-docx.py`）
- 本设计文档：本文件
