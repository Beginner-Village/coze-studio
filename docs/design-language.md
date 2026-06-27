# Coze Studio / Ynet 设计语言（Design Language）

> 本文从仓库真实设计 token 中提炼，**不是设计稿主观描述**。所有数值/命名均来自
> `frontend/config/tailwind-config/src/`（`light.js` `dark.js` `coze.js` `index.js`）、
> Semi Design 组件层、以及 `@coze-arch/semi-theme-hand01` 品牌主题覆盖。
> 改动 token 请改这些源文件，本文随之更新。

---

## 0. 一句话定位

一套**面向 AI 工作台（智能体 / 工作流 / 对话）的桌面级 B 端设计语言**：以
**蓝紫双主色**（产品蓝 + AI 紫）承载"工具理性 + AI 智能"的双重气质，
**4 层语义色 + 8px 网格 + 中等圆角 + 克制阴影**，深浅双主题，
中文优先（PingFang SC）。底层组件复用 **Semi Design**，通过品牌主题与 Tailwind 语义层统一外观。

---

## 1. 技术底座（先理解架构，再谈风格）

| 层 | 位置 | 职责 |
|---|---|---|
| **CSS 变量层** | `tailwind-config/src/light.js` `dark.js` | 定义原子色值（RGB）、间距像素、alpha；深浅主题各一份，`.dark` class 切换 |
| **语义层（plugin）** | `tailwind-config/src/coze.js` | 把原子色映射成 `coz-fg-*` / `coz-mg-*` / `coz-bg-*` / `coz-stroke-*` 语义 class 与 CSS 变量 |
| **Theme 映射** | `tailwind-config/src/index.js` | 把 CSS 变量接进 Tailwind 的 `colors/spacing/radius/shadow/fontSize` 等 |
| **组件层** | Semi Design (`@douyinfe/semi-ui`) + `@coze-arch/bot-semi` | 实际 UI 组件 |
| **品牌主题** | `@coze-arch/semi-theme-hand01/raw.json` | 在 Semi 之上覆盖品牌色/圆角，使 Semi 组件与本设计语言一致 |
| **封装库** | `@coze-arch/coze-design` | 业务侧统一组件入口 |

**给设计 / 前端的关键规则：永远用语义 class，不要直接用原子色。**
写 `text-coz-fg`、`bg-coz-mg-primary`、`border-coz-stroke-primary`，
不要写 `text-[#358aff]` 或 `bg-brand-5`。语义层已自动处理深浅主题与 hover/pressed 状态。

---

## 2. 色彩系统

### 2.1 双主色（品牌气质的核心）

| 角色 | Light 主值 | Dark 主值 | 语义 | 用途 |
|---|---|---|---|---|
| **产品蓝 brand** | `rgb(53,138,255)` (`brand-5`) | `rgb(166,166,255)` | `coz-fg-hglt` / `coz-mg-hglt-plus` | 主按钮、主链接、选中、强调——**功能性主色** |
| **AI 紫 purple** | `rgb(167,0,250)` (`purple-5`) | — | `coz-fg-hglt-ai` / `coz-mg-hglt-plus-ai` | 一切"AI / 智能 / 生成"语境的强调——**情感性主色** |

> 这是本设计语言最具识别度的一点：**蓝管"操作"，紫管"AI"**。
> 凡是 AI 生成、智能体、模型相关的入口/状态/标签，用紫色系（`-ai` 后缀语义）区分于普通操作。

品牌蓝按交互梯度（语义层自动调用）：
`brand-7` 188 pressed → `brand-6` hover → `brand-5` 默认 → `brand-3/2/1/0` 渐浅填充/底色。

### 2.2 四层语义色（命名心智模型）

颜色不按"红蓝绿"记，按**层级**记。这是整套系统的核心抽象：

| 前缀 | 含义 | 作用于 | 典型 class |
|---|---|---|---|
| `coz-fg-*` | **Foreground** 前景 | 文字 / 图标颜色 | `coz-fg`(正文) `coz-fg-secondary` `coz-fg-dim`(弱) `coz-fg-plus`(强) `coz-fg-hglt`(高亮) |
| `coz-mg-*` | **Middleground** 中景 | 填充 / 背景块（按钮、卡片、tag） | `coz-mg-primary` `coz-mg-hglt-plus`(主色实心) `coz-mg-card` `coz-mg-mask` |
| `coz-bg-*` | **Background** 背景 | 页面 / 容器层 | `coz-bg`(页) `coz-bg-plus` `coz-bg-max`(最高层卡片) `coz-bg-secondary` |
| `coz-stroke-*` | **Stroke** 描边 | 边框 / 分割线 | `coz-stroke-primary` `coz-stroke-plus` `coz-stroke-hglt` |

每个语义又带 **状态后缀**：`-hovered` `-pressed` `-dim`(淡化)。
即同一颜色的 hover/按下态由 token 内置，组件无需手写。

**文字层级用 alpha 表达层次**（light）：
`fg-3` 正文 0.82 → `fg-2` 次要 0.62 → `fg-1` 禁用/占位 0.38 → `fg-4` 强调 0.9 → `fg-5` 纯黑标题 1.0。

### 2.3 功能色（状态语义）

| 语义 | 色相 | 主值(light) | 用途 |
|---|---|---|---|
| 危险 / 错误 red | 红 | `rgb(229,50,65)` | 删除、报错、危险操作 |
| 警告 yellow | 橙黄 | `rgb(255,115,0)` | 警告、待处理（注意：偏橙非纯黄） |
| 成功 green | 绿 | `rgb(0,178,60)` | 成功、在线、通过 |

每个功能色都有 `-hglt-plus`(实心强调) / `-hglt`(浅底) / `-dim` 三档，规则同主色。

### 2.4 扩展色板（图表 / Tag / 代码高亮）

`cyan` `blue` `purple` `magenta` `emerald` `orange` `alternative`(亮黄绿) —— 仅用于
**数据可视化、Tag 分类着色、代码语法高亮**，不参与功能语义。
每色提供 `50/30/20/10`（饱和梯度，给浅底）与 `5/3`（前景）。

### 2.5 深浅主题

- 切换方式：根元素加 `.dark` class（`darkMode: 'class'`）。
- Light 页底 `rgb(255,255,255)`，正文 `rgb(28,28,35)`；Dark 页底 `rgb(2,8,23)`（深蓝黑，非纯黑），正文 `rgb(249,249,249)`。
- **设计稿必须同时给深浅两版**，且只用语义 class——切主题零改代码。

---

## 3. 间距与尺寸（8px 网格）

基准单位 `--coze-8 = 8px`，`spacing.DEFAULT = 8px`。提供两套刻度：

- **像素直引**：`p-8px` `gap-12px` `m-16px` …（1/2/3/4/5/6/8/9/10/12/14/16/18/20/22/24/26/28/30/32/40/48/64/80/96px + 120~1080 大尺寸）
- **语义档位**（推荐用于组件内部，保证一致节奏）：

| 档 | 值 | 典型场景 |
|---|---|---|
| `mini` | 16px | 紧凑内边距 |
| `small` | 20px | 小间隔 |
| `normal` | 32px | 默认区块间距 |
| `large` | 40px | 大区块 |
| `mm` | 48px / `md` 64 / `xl` 80 / `xxl` 96 | 页面级留白 |

**控件标准高度**（`inputHeight` / `height`）：`small 24px` · `normal 32px` · `large 40px`。
→ 表单、按钮默认 **32px 高**，密集表格用 24px，突出操作用 40px。

---

## 4. 圆角（中等圆角，亲和但不圆萌）

`borderRadius.DEFAULT = 8px`。梯度：

| token | 值 | 用途 |
|---|---|---|
| `rounded-tiny` | 2px | 极小标记 |
| `rounded-mini` | 4px | Tag、小 chip |
| `rounded-small` | 6px | 输入框（small） |
| `rounded-normal` | 8px | **默认：按钮、输入、卡片** |
| `rounded-m / md` | 10 / 12px | 中卡片、弹层 |
| `rounded-xl / xxl` | 16 / 24px | 大卡片、模态 |
| `rounded-ultra` | 40px | 胶囊 / 全圆按钮 |

控件专用：按钮 `coz-btn-rounded-normal = 8px`（large 10 / small 5 / mini 4），输入框 `coz-input-rounded-normal = 8px`。
→ 整体观感：**8px 中圆角主导**，偏理性现代，不走大圆角的消费级软萌路线。

---

## 5. 字体与排版

```
font-family: 'PingFang SC', 'Noto Sans SC', sans-serif;
```

**中文优先**（PingFang SC 苹方为首选，Noto Sans SC 兜底），无衬线、桌面 B 端取向。

字号刻度（`fontSize`，px 命名）：

| token | 值 | 用途 |
|---|---|---|
| `text-mini` | 10px | 角标、辅助 |
| `text-base` | 12px | **正文默认（B 端紧凑）** |
| `text-lg` | 14px | 强调正文 / 控件文字 |
| `text-xl / xxl` | 15 / 16px | 小标题 |
| `text-18px ~ 24px` | 18–24 | 各级标题 |
| `text-32px / 36px / 48px / 64px` | — | 营销 / 大标题 |

行高刻度 `lineHeight`：16/20/22/24/28/36px，与字号配对（如 12px 文 / 20px 行）。
→ 信息密度偏高：**正文 12px 是常态**，符合专业工具型产品。

---

## 6. 阴影（克制、柔和、双层叠加）

阴影色 `--coze-shadow-0 = rgb(0,0,0)`，全部用极低 alpha 双层叠加，营造"漂浮但不脏"的层次：

| token | 值 |
|---|---|
| `coz-shadow-small` | `0 2px 6px /0.04` + `0 4px 12px /0.02` |
| `coz-shadow`(normal/default) | `0 4px 12px /0.08` + `0 8px 24px /0.04` |
| `coz-shadow-large` | `0 8px 24px /0.16` + `0 16px 48px /0.08` |

→ 阴影只用这三档，**禁止自定义硬阴影**。卡片用 small/normal，弹层/模态用 large。

---

## 7. 描边与分割

- 默认描边 `coz-stroke-primary`（`stroke-5`，alpha 0.13，极淡）——B 端讲究"线要轻"。
- 强描边 `coz-stroke-plus`（`stroke-6`，0.25）；选中/聚焦描边 `coz-stroke-hglt`（品牌蓝）。
- 边框宽度：默认 `1px`，细线 `half = 0.5px`。
- 不透明分割线用 `coz-stroke-opaque`（`rgb(226,228,239)` 浅蓝灰）。

---

## 8. 动效

仅内置极少量微动效，原则是**功能性、快、不抢戏**：

- 图标展开/收起：`icon-down` / `icon-up`，旋转 90°，`0.2s ease-out`。
- 基调：交互反馈 **200ms、ease-out**，无弹跳、无炫技。
- hover/pressed 通过语义色 token 切换，不靠额外动画。

---

## 9. 落地速查（给前端 / 设计 agent）

构建任意界面时的默认选择：

```
页面底       bg-coz-bg
卡片         bg-coz-bg-max  rounded-normal(8px)  coz-shadow  border coz-stroke-primary
正文         text-coz-fg  text-base(12px)
次要文字     text-coz-fg-secondary
主按钮       bg-coz-mg-hglt-plus  text-coz-fg-white  h-normal(32)  coz-btn-rounded-normal
次按钮       bg-coz-mg-primary    text-coz-fg
AI 相关入口  紫色系：text-coz-fg-hglt-ai / bg-coz-mg-hglt-plus-ai
危险操作     red 系：bg-coz-mg-hglt-plus-red / text-coz-fg-hglt-red
间距         8px 网格：gap-8px/12px/16px，区块 normal(32)
```

**三条铁律：**
1. 用语义 class，不用原子色 / 硬编码 hex —— 自动适配深浅主题与交互态。
2. 蓝=操作，紫=AI —— 这是本产品的识别符。
3. 8px 网格 + 32px 控件高 + 8px 圆角 + 三档柔阴影 —— 不要发明新刻度。

---

## 10. 真相提示（避免误用）

- 本仓库**没有自研的统一组件库**；可复用 UI 原语散落在 `frontend/packages/components/*`
  （table-view / scroll-view / virtual-list / loading-button 等），视觉一致性靠的是
  **共享的 Tailwind token + Semi 品牌主题**，而非一个集中式 DS。
- 真正的组件实现来自 **Semi Design（外部）**。要改组件级外观，改
  `@coze-arch/semi-theme-hand01` 主题，而非逐组件覆盖。
- 颜色变量里历史命名仍叫 `coze-*`（产品改名前留存），新主题文件头版权已是 `ynet-dev`；
  两者指同一套 token，勿混淆为两套系统。
