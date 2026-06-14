/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const fs = require("fs");
const {
  Document, Packer, Paragraph, TextRun, Table, TableRow, TableCell,
  Header, Footer, AlignmentType, LevelFormat, TabStopType, TabStopPosition,
  TableOfContents, HeadingLevel, BorderStyle, WidthType, ShadingType,
  VerticalAlign, PageNumber, PageBreak
} = require("docx");

const FONT = "宋体";
const FONT_H = "黑体";

// ---------- helpers ----------
const border = { style: BorderStyle.SINGLE, size: 1, color: "BBBBBB" };
const borders = { top: border, bottom: border, left: border, right: border };
const cellMargins = { top: 60, bottom: 60, left: 110, right: 110 };

function h1(text) {
  return new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun({ text, font: FONT_H })] });
}
function h2(text) {
  return new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun({ text, font: FONT_H })] });
}
function h3(text) {
  return new Paragraph({ heading: HeadingLevel.HEADING_3, children: [new TextRun({ text, font: FONT_H })] });
}
function p(text, opts = {}) {
  return new Paragraph({
    spacing: { after: 120, line: 320 },
    alignment: opts.align || AlignmentType.JUSTIFIED,
    indent: opts.indent === false ? undefined : { firstLine: 480 },
    children: [new TextRun({ text, font: FONT, size: opts.size || 22, bold: !!opts.bold })],
  });
}
function bullet(text) {
  return new Paragraph({
    numbering: { reference: "bullets", level: 0 },
    spacing: { after: 60, line: 300 },
    children: [new TextRun({ text, font: FONT, size: 22 })],
  });
}
function num(text) {
  return new Paragraph({
    numbering: { reference: "numbers", level: 0 },
    spacing: { after: 60, line: 300 },
    children: [new TextRun({ text, font: FONT, size: 22 })],
  });
}
function cell(text, { head = false, w } = {}) {
  return new TableCell({
    borders, width: { size: w, type: WidthType.DXA }, margins: cellMargins,
    verticalAlign: VerticalAlign.CENTER,
    shading: head ? { fill: "D9E2F3", type: ShadingType.CLEAR, color: "auto" } : undefined,
    children: [new Paragraph({
      spacing: { after: 0, line: 280 },
      children: [new TextRun({ text, font: FONT, size: 20, bold: head })],
    })],
  });
}
function table(widths, rows) {
  const total = widths.reduce((a, b) => a + b, 0);
  return new Table({
    width: { size: total, type: WidthType.DXA },
    columnWidths: widths,
    rows: rows.map((r, ri) =>
      new TableRow({ children: r.map((c) => cell(c, { head: ri === 0, w: widths[r.indexOf(c)] })) })
    ),
  });
}
// robust table builder (avoid indexOf collisions)
function tbl(widths, rows) {
  const total = widths.reduce((a, b) => a + b, 0);
  return new Table({
    width: { size: total, type: WidthType.DXA },
    columnWidths: widths,
    rows: rows.map((r, ri) =>
      new TableRow({ children: r.map((txt, ci) => cell(txt, { head: ri === 0, w: widths[ci] })) })
    ),
  });
}
function spacer() { return new Paragraph({ children: [new TextRun("")] }); }

// architecture diagram: stacked shaded boxes (one column table)
function archBox(title, desc, fill) {
  return new TableRow({ children: [new TableCell({
    borders, width: { size: 9026, type: WidthType.DXA }, margins: { top: 100, bottom: 100, left: 140, right: 140 },
    shading: { fill, type: ShadingType.CLEAR, color: "auto" },
    children: [
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { after: 40 },
        children: [new TextRun({ text: title, font: FONT_H, size: 21, bold: true, color: "1F3864" })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { after: 0 },
        children: [new TextRun({ text: desc, font: FONT, size: 18, color: "333333" })] }),
    ],
  })] });
}
function archArrow() {
  return new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 20, after: 20 },
    children: [new TextRun({ text: "▼", font: FONT, size: 18, color: "8896B0" })] });
}
function archDiagram(layers) {
  const out = [];
  layers.forEach((l, i) => {
    out.push(new Table({ width: { size: 9026, type: WidthType.DXA }, columnWidths: [9026], rows: [archBox(l[0], l[1], l[2])] }));
    if (i < layers.length - 1) out.push(archArrow());
  });
  return out;
}

const SW = 9026; // content width A4 - 1" margins

// ---------- cover ----------
const cover = [
  new Paragraph({ spacing: { before: 1800, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "AI 智能知识库平台", font: FONT_H, size: 56, bold: true, color: "1F3864" })] }),
  new Paragraph({ spacing: { before: 200, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "（简称：AI 知识库平台）", font: FONT_H, size: 30, color: "1F3864" })] }),
  new Paragraph({ spacing: { before: 800, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "技 术 白 皮 书", font: FONT_H, size: 44, bold: true })] }),
  new Paragraph({ spacing: { before: 120, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "Technical White Paper", font: "Arial", size: 24, color: "555555" })] }),
  new Paragraph({ spacing: { before: 2400, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "版本号：V1.0", font: FONT, size: 26 })] }),
  new Paragraph({ spacing: { before: 160, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "著作权人：北京易诚互动网络技术股份有限公司", font: FONT, size: 26 })] }),
  new Paragraph({ spacing: { before: 160, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "登记号：2026SR0260508", font: FONT, size: 26 })] }),
  new Paragraph({ spacing: { before: 600, after: 0 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "二〇二六年", font: FONT, size: 24 })] }),
  new Paragraph({ children: [new PageBreak()] }),
];

// ---------- copyright notice ----------
const notice = [
  h1("版权与法律声明"),
  p("本文档所涉及的“AI 智能知识库平台（简称：AI 知识库平台）V1.0”软件及其相关技术、文档资料的著作权，归北京易诚互动网络技术股份有限公司（以下简称“本公司”）所有，并已依据《计算机软件保护条例》《计算机软件著作权登记办法》取得中国版权保护中心颁发的《计算机软件著作权登记证书》（登记号：2026SR0260508）。"),
  p("本白皮书旨在系统阐述本软件知识库子系统的产品定位、总体架构、功能体系、关键技术与典型应用，作为软件著作权登记的配套技术说明材料及对外技术介绍材料使用。"),
  p("未经本公司书面许可，任何单位和个人不得以任何形式复制、传播、摘编、修改本文档的全部或部分内容，亦不得将本文档用于任何商业目的。本文档内容如有更新，恕不另行通知。"),
  p("文中所涉及的第三方产品名称、商标等，归各自权利人所有，仅用于技术说明，不代表任何授权或合作关系。"),
  new Paragraph({ children: [new PageBreak()] }),
];

// ---------- doc info & revision ----------
const frontTitle = (t) => new Paragraph({ spacing: { before: 240, after: 160 },
  children: [new TextRun({ text: t, font: FONT_H, size: 28, bold: true, color: "1F3864" })] });
const docinfo = [
  frontTitle("文档信息"),
  tbl([2600, 6426], [
    ["项目", "内容"],
    ["软件全称", "AI 智能知识库平台"],
    ["软件简称", "AI 知识库平台"],
    ["版本号", "V1.0"],
    ["著作权人", "北京易诚互动网络技术股份有限公司"],
    ["软件著作权登记号", "2026SR0260508"],
    ["文档类型", "技术白皮书"],
    ["文档版本", "V1.0"],
    ["密级", "公开"],
  ]),
  spacer(),
  frontTitle("修订记录"),
  tbl([1600, 1800, 2200, 3426], [
    ["版本", "修订日期", "修订人", "修订说明"],
    ["V1.0", "2026 年", "产品技术部", "首次发布，形成知识库子系统技术白皮书。"],
  ]),
  new Paragraph({ children: [new PageBreak()] }),
];

// ---------- TOC ----------
const toc = [
  new Paragraph({ spacing: { after: 200 }, alignment: AlignmentType.CENTER,
    children: [new TextRun({ text: "目  录", font: FONT_H, size: 32, bold: true })] }),
  new TableOfContents("目录", { hyperlink: true, headingStyleRange: "1-2" }),
  new Paragraph({ children: [new PageBreak()] }),
];

// ---------- body ----------
const body = [];

// 1 引言
body.push(h1("第一章 引言"));
body.push(h2("1.1 编写目的"));
body.push(p("随着企业数字化转型的深入推进以及大语言模型（LLM）技术的快速发展，如何将组织内部海量、异构、分散的文档资料转化为可被智能体（Agent）与对话系统精准调用的结构化知识，已成为企业级 AI 应用落地的关键环节。本白皮书围绕“AI 智能知识库平台 V1.0”中的知识库子系统，对其产品定位、总体架构、功能体系、关键技术及典型应用场景进行全面、系统的阐述，为软件著作权登记提供配套技术说明，并为用户、合作伙伴及技术评估人员提供权威的技术参考。"));
body.push(h2("1.2 读者对象"));
body.push(p("本白皮书适用于以下读者：企业信息化与人工智能项目的决策者与评估人员；系统集成与技术选型人员；平台的实施、运维与二次开发人员；以及对检索增强生成（RAG）与知识库技术感兴趣的研究人员。"));
body.push(h2("1.3 术语与缩略语"));
body.push(tbl([2200, 6826], [
  ["术语 / 缩略语", "说明"],
  ["知识库（Knowledge Base）", "对特定领域文档资料进行采集、解析、切分、向量化与索引后形成的可检索知识集合。"],
  ["RAG", "检索增强生成（Retrieval-Augmented Generation），先检索相关知识再交由大模型生成答案的技术范式。"],
  ["切片 / 分段（Chunk / Slice）", "文档经分段策略处理后形成的最小知识单元，是向量化与检索的基本对象。"],
  ["Embedding（向量化）", "将文本转换为高维稠密 / 稀疏向量的过程，用于语义相似度计算。"],
  ["向量数据库", "面向高维向量进行存储与近似最近邻（ANN）检索的数据库，如 Milvus、VikingDB、OceanBase Vector。"],
  ["召回（Recall）", "依据用户查询从知识库中检索候选知识片段的过程。"],
  ["重排（Rerank）", "对多路召回结果按相关性重新打分与排序，以提升结果质量。"],
  ["NL2SQL", "将自然语言查询转换为 SQL 语句以查询结构化表格数据的技术。"],
  ["HNSW", "Hierarchical Navigable Small World，一种高性能的近似最近邻向量索引算法。"],
  ["OCR", "光学字符识别（Optical Character Recognition），用于从图片中提取文本。"],
]));

// 2 产品概述
body.push(h1("第二章 产品概述"));
body.push(h2("2.1 产品背景"));
body.push(p("企业在长期经营过程中积累了大量以 PDF、Word、Excel、Markdown、图片等多种格式存在的非结构化与半结构化文档。传统的关键词检索难以理解用户的真实意图，更无法直接服务于大模型问答场景；而单纯依赖大模型又面临知识时效性差、易产生幻觉、无法引用企业私有数据等问题。AI 智能知识库平台正是为解决上述痛点而设计——通过标准化的知识接入、智能化的文档解析与切分、高质量的向量化与多路检索，将企业私有知识沉淀为可被智能体稳定、精准调用的能力底座。"));
body.push(h2("2.2 产品简介"));
body.push(p("AI 智能知识库平台知识库子系统是一套面向企业级场景的、支持检索增强生成（RAG）的知识管理与检索引擎。系统覆盖从文档上传、解析、分段、向量化、索引构建到多路召回、重排、结果返回的完整链路，支持文本、表格、图片、问答对等多种知识形态，兼容多种向量数据库与全文检索引擎，可灵活适配私有化部署与多租户隔离场景，为上层智能体、对话机器人及业务系统提供稳定、高效、可扩展的知识检索服务。"));
body.push(h2("2.3 产品定位"));
body.push(p("本系统在整体平台中定位为“知识能力中台”，向上为智能体编排、对话应用提供标准化的知识检索接口，向下屏蔽各类向量数据库、全文引擎、解析与重排组件的实现差异，形成可插拔、可扩展的知识基础设施。"));
body.push(h2("2.4 产品特性"));
body.push(bullet("全链路覆盖：贯通文档上传、解析、分段、向量化、索引、召回、重排、返回全流程，开箱即用。"));
body.push(bullet("多格式接入：支持 PDF、Word、TXT、Markdown、CSV、Excel、JSON、图片（OCR）及问答对等多种文档格式。"));
body.push(bullet("混合检索：融合向量语义召回、全文检索召回与 NL2SQL 表格召回，多路并行、结果重排，显著提升召回质量。"));
body.push(bullet("多引擎兼容：向量存储支持 Milvus、VikingDB、OceanBase Vector，全文检索支持 Elasticsearch 7/8，组件可插拔替换。"));
body.push(bullet("精细化分段：提供默认分段、自定义分段与层级分段策略，支持分段长度、分隔符、重叠等参数化配置。"));
body.push(bullet("多租户隔离：以“空间（Space）”为单位进行向量化配置与数据隔离，满足企业多团队、多业务并存的需求。"));
body.push(bullet("异步高并发：基于消息队列的异步事件驱动架构，文档索引可水平扩展，支撑大规模文档处理。"));
body.push(bullet("可观测可运维：内置进度跟踪与 Prometheus 指标采集，解析耗时、文件大小、失败原因等关键指标可观测。"));

// 3 总体架构
body.push(h1("第三章 总体架构"));
body.push(h2("3.1 设计原则"));
body.push(p("知识库子系统在架构设计上遵循分层解耦、契约先行、可插拔扩展、异步并行与多租户隔离五项基本原则，确保系统在功能丰富的同时具备良好的可维护性、可扩展性与高性能。"));
body.push(h2("3.2 分层架构"));
body.push(p("系统采用经典的领域驱动分层架构，自上而下划分为接口层、应用层、领域层与基础设施层，各层职责清晰、单向依赖。整体分层架构如下图所示："));
archDiagram([
  ["接口层（API Handler）", "知识库 / 文档 / 切片 / 检索测试 / 外部知识库 等 HTTP 接口", "DEEBF7"],
  ["应用层（Application）", "面向用例的服务编排：知识库创建、文档处理、检索调度", "E2EFDA"],
  ["领域层（Domain）", "核心业务逻辑：实体 · 领域服务 · 文档处理器 · 解析分段策略 · 检索链路 · 事件处理", "FFF2CC"],
  ["基础设施层（Infra）", "解析器 · 向量存储 · 全文检索 · 重排 · NL2SQL · OCR · 进度跟踪（契约可插拔）", "FCE4D6"],
]).forEach((x) => body.push(x));
body.push(spacer());
body.push(tbl([1900, 7126], [
  ["层次", "主要职责"],
  ["接口层（API Handler）", "对外暴露知识库、文档、切片、检索测试、外部知识库绑定等 HTTP 接口，负责参数校验与协议转换。"],
  ["应用层（Application）", "面向用例的服务编排，协调领域服务完成知识库创建、文档处理、检索等完整业务流程。"],
  ["领域层（Domain）", "知识库子系统的核心业务逻辑，包含实体、领域服务、文档处理器、分段与解析策略、检索链路、事件处理等。"],
  ["基础设施层（Infra）", "提供文档解析、向量存储、全文检索、重排、NL2SQL、OCR、进度跟踪等能力的具体实现，通过契约接口与领域层解耦。"],
]));
body.push(h2("3.3 功能架构"));
body.push(p("从功能视角，知识库子系统由知识库管理、文档管理、文档解析与分段、向量化与索引、知识检索与召回、结果重排、切片管理以及若干高级能力等模块构成，各模块协同形成完整的知识生产与消费闭环。"));
body.push(tbl([2400, 6626], [
  ["功能模块", "核心能力"],
  ["知识库管理", "知识库的创建、更新、删除、复制、移动、查询与批量获取。"],
  ["文档管理", "文档上传、更新、删除、列表查询、处理进度跟踪、批量操作与重新分段。"],
  ["解析与分段", "多格式文档解析、快速 / 精准两级解析、图片抽取与 OCR、表格抽取、默认 / 自定义 / 层级分段策略。"],
  ["向量化与索引", "稠密 / 稀疏向量化、向量库索引构建、全文索引构建、空间级向量化隔离。"],
  ["检索与召回", "向量语义召回、全文检索召回、NL2SQL 表格召回、查询改写、多路并行召回。"],
  ["结果重排", "OpenAI 重排、VikingDB 原生重排、RRF 倒数排名融合重排。"],
  ["切片管理", "切片的创建、更新、删除、查询，以及图片切片与图注管理。"],
  ["高级能力", "文档质量审阅、图注抽取、表结构管理、数据导入、外部知识库绑定、索引一键重建等。"],
]));
body.push(h2("3.4 技术架构"));
body.push(p("系统后端采用 Go 语言开发，基于 CloudWeGo Hertz 高性能 HTTP 框架构建服务接口，借助 Eino 编排框架实现检索链路的模块化组合，使用 GORM 进行关系型数据持久化，并通过消息队列实现文档索引的异步事件驱动处理；前端采用 TypeScript 与 React 技术栈，以组件化方式构建知识库管理与文档编辑界面。"));
body.push(tbl([2400, 6626], [
  ["技术领域", "选型"],
  ["后端语言", "Go"],
  ["HTTP 框架", "CloudWeGo Hertz"],
  ["编排框架", "Eino（检索链路 compose 编排）"],
  ["ORM", "GORM"],
  ["异步处理", "消息队列事件驱动（文档索引异步化）"],
  ["向量数据库", "Milvus、VikingDB、OceanBase Vector"],
  ["全文检索", "Elasticsearch 7 / 8"],
  ["缓存", "Redis"],
  ["前端", "TypeScript + React"],
  ["可观测性", "Prometheus 指标采集"],
]));

// 4 功能详述
body.push(h1("第四章 功能详述"));

body.push(h2("4.1 知识库管理"));
body.push(p("知识库是平台进行知识组织与隔离的基本单元。系统支持对知识库进行完整的生命周期管理，用户可在指定空间下创建知识库并设置名称、描述、图标与格式类型；支持对知识库进行更新、删除、复制、移动等操作；删除知识库时将级联清理其下属文档、切片及相关索引，保证数据一致性；同时支持分页列表查询与按多 ID 的批量元数据获取，便于上层应用集成。"));

body.push(h2("4.2 文档管理"));
body.push(p("文档是知识库的内容来源。系统支持多种格式文档的上传与可配置解析策略，提供实时的处理进度跟踪与状态管理；支持文档元数据与表结构信息的更新、文档删除及其关联索引的自动清理；支持按知识库、状态、类型等条件进行文档列表查询，并支持多文档批量上传与批量进度查询。针对分段策略调整的场景，系统提供“重新分段”能力，用户无需重新上传即可按新的分段策略对已有文档重新切分与索引。"));

body.push(h2("4.3 文档解析与分段"));
body.push(h3("4.3.1 支持的文档格式"));
body.push(p("系统内置丰富的解析器，覆盖企业常见的文档格式，具体如下："));
body.push(tbl([2400, 6626], [
  ["类别", "支持格式"],
  ["文档类", "PDF、Word（DOC / DOCX）、纯文本（TXT）、Markdown（MD）"],
  ["数据类", "CSV、Excel（XLSX）、JSON、JSON-Maps（自定义 JSON 格式）"],
  ["图片类", "JPG、JPEG、PNG（支持 OCR 文本提取）"],
  ["问答类", "QA-CSV、QA-JSON、QA-XLSX（面向问答对的专用列式格式）"],
]));
body.push(h3("4.3.2 解析策略"));
body.push(p("系统提供两级解析能力：快速解析（侧重处理效率）与精准解析（侧重还原文档结构）。在解析过程中支持图片抽取并可选接入 OCR（如 PaddleOCR、阿里 VE-OCR）以识别图片中的文字；支持表格抽取与保留、页面过滤，以及对文档层级结构与元数据的保留，为后续高质量分段与检索奠定基础。"));
body.push(h3("4.3.3 分段策略"));
body.push(p("分段质量直接决定检索与生成效果。系统提供三类分段策略，满足不同文档结构与业务需求："));
body.push(bullet("默认分段：系统按内置规则自动完成文档切分，开箱即用。"));
body.push(bullet("自定义分段：支持配置分段最大长度、自定义分隔符、分段重叠（保留上下文）、空白裁剪以及 URL / Email 裁剪等参数。"));
body.push(bullet("层级分段：基于文档标题层级进行多级切分，支持最大深度控制与标题信息跨段保留，适合结构化长文档。"));

body.push(h2("4.4 向量化与索引"));
body.push(p("文档切分完成后，系统对各切片进行向量化并构建索引。向量化支持稠密向量与稀疏向量两种形态，可对接内置向量化模型与第三方向量化服务（如 VikingDB Embedding）。向量化配置以空间为单位进行隔离，确保多租户场景下各业务的向量化策略互不干扰。索引构建同时面向向量数据库与全文检索引擎：稠密向量默认采用 HNSW 索引算法并支持参数化配置，稀疏向量采用稀疏倒排索引；全文内容则写入 Elasticsearch 构建 BM25 索引，从而支撑后续的混合检索。"));

body.push(h2("4.5 知识检索与召回"));
body.push(p("检索是知识库对外提供价值的核心环节。系统采用多路并行召回的设计，针对用户查询同时执行多种召回策略，并融合上下文进行查询改写，以最大化召回的覆盖度与准确度。"));
body.push(bullet("向量 / 语义召回：基于向量数据库进行稠密向量相似度检索，理解查询的语义意图。"));
body.push(bullet("全文检索召回：基于 Elasticsearch 的 BM25 算法进行关键词 / 短语匹配，并支持模糊匹配。"));
body.push(bullet("NL2SQL 表格召回：针对表格类知识，将自然语言查询转换为 SQL 并在表数据上执行查询。"));
body.push(bullet("查询改写：结合对话历史对原始查询进行上下文增强改写，提升多轮对话场景下的召回相关性。"));
body.push(p("此外，系统支持可配置的最低分数阈值与无效查询过滤，并可按知识库、文档 ID 进行检索范围的精确约束。", { indent: true }));

body.push(h2("4.6 结果重排"));
body.push(p("多路召回的结果在返回前需统一重排，以保证最终结果的相关性顺序。系统提供多种可插拔的重排实现，包括基于 OpenAI 的重排、VikingDB 原生重排，以及 RRF（倒数排名融合）重排。重排在并行召回之后、结果打包之前执行，对来自不同召回通路的候选片段进行统一打分与排序后返回上层应用。"));

body.push(h2("4.7 切片管理"));
body.push(p("切片（Slice）是知识的最小单元。系统支持对切片进行手动创建、内容与元数据更新、删除以及按知识库、文档、关键词、状态等条件的列表查询；针对图片型切片，提供图注（Caption）管理与图片切片详情查询，支持图注的自动抽取与人工修正，保证图文知识的可检索性与可维护性。"));

body.push(h2("4.8 高级能力"));
body.push(bullet("文档质量审阅：支持创建、更新、批量查询文档质量审阅记录，辅助知识治理。"));
body.push(bullet("图注与关键词抽取：自动为图片生成图注 / 关键词，提升图片知识的检索命中率。"));
body.push(bullet("表结构管理：支持表格列定义的校验与变更，保障结构化数据的一致性。"));
body.push(bullet("数据导入：支持带模式校验的结构化数据导入。"));
body.push(bullet("外部知识库绑定：通过 API 与外部知识源对接，实现外部知识的统一检索。"));
body.push(bullet("索引一键重建：支持对整个空间的全文索引进行一键重建（Resync），便于运维与数据修复。"));

// 5 关键技术
body.push(h1("第五章 关键技术与特点"));
body.push(h2("5.1 异步事件驱动的文档索引"));
body.push(p("文档解析、分段、向量化与索引构建属于计算密集且耗时较长的操作。系统将文档创建与索引构建解耦，文档创建后即发布索引事件，由独立的事件处理器异步消费并完成解析与索引；该设计使文档索引能力可随消费者水平扩展，从而支撑大规模文档的批量处理，同时通过异常重试区分（可重试 / 不可重试错误）与 panic 恢复机制保障处理过程的稳健性。"));
body.push(h2("5.2 基于 Eino 的可组合检索链路"));
body.push(p("检索链路采用 Eino 编排框架以 compose 链方式构建，将查询改写、向量召回、全文召回、NL2SQL 召回、重排、结果打包等环节抽象为可组合的节点，多路召回节点并行执行，既保证了链路的清晰与可维护，又最大化了检索的并发效率，使整体检索时延接近最慢单路而非各路之和。"));
body.push(h2("5.3 混合检索与多路召回融合"));
body.push(p("系统融合稠密向量语义召回、稀疏向量 / 全文召回与 NL2SQL 结构化召回，兼顾语义理解与关键词精确匹配，并通过统一重排进行结果融合，显著优于任何单一召回方式，尤其适用于既包含非结构化文档又包含结构化表格的复杂企业知识场景。"));
body.push(h2("5.4 多向量库与组件可插拔"));
body.push(p("系统通过契约（Contract）接口对解析器、向量存储、重排器、NL2SQL、OCR 等能力进行抽象，各能力均提供多种实现并以管理器（Manager）模式装配。向量存储可在 Milvus、VikingDB、OceanBase Vector 之间灵活切换，全文检索兼容 Elasticsearch 7/8，重排支持 OpenAI、VikingDB、RRF 多种策略，用户可根据部署环境与成本要求自由组合，避免被单一技术栈锁定。"));
body.push(h2("5.5 空间级多租户隔离"));
body.push(p("系统以“空间（Space）”为单位管理向量化配置与索引，每个空间可拥有独立的向量化模型与存储配置，实现多团队、多业务在同一平台下的数据与配置隔离，满足企业级多租户场景的安全与合规要求。"));
body.push(h2("5.6 可观测性与运维支撑"));
body.push(p("系统内置实时进度条用于跟踪文档解析与索引的长任务进度，并通过 Prometheus 采集解析耗时、文件大小、失败原因等关键指标，结合详细的文档状态机（分段中、已启用、失败等）与批量进度查询，为运维人员提供完善的可观测能力。"));

// 6 数据流程
body.push(h1("第六章 核心数据流程"));
body.push(h2("6.1 文档上传与索引流程"));
body.push(p("文档从上传到可被检索，需经历元数据落库、异步解析、分段、向量化、索引构建与状态更新等环节，主要流程如下："));
[
  "调用文档创建接口，上传文档并指定解析与分段策略；",
  "应用层与领域服务将文档元数据写入关系型数据库，并发布索引文档事件；",
  "事件处理器异步消费事件，依据文件类型选择对应解析器执行解析；",
  "按所选分段策略对解析结果进行切分，生成带重叠的切片；",
  "获取所在空间的向量化配置，对各切片进行向量化并写入向量数据库；",
  "将切片全文内容写入 Elasticsearch 构建全文索引；",
  "更新文档与切片状态为“已启用”，完成索引。",
].forEach((t) => body.push(num(t)));
body.push(h2("6.2 查询与检索流程"));
body.push(p("用户查询经校验后进入并行召回与重排链路，主要流程如下："));
[
  "调用检索接口，系统对查询进行校验并过滤无效查询；",
  "准备 RAG 检索上下文：筛选启用的知识库与文档，提取表列、类型等元数据；",
  "进入检索链路，若存在对话历史则先进行查询改写；",
  "并行执行向量召回、全文召回与 NL2SQL 召回三路检索；",
  "对多路召回结果进行统一重排（OpenAI / VikingDB / RRF）；",
  "将重排后的结果打包为检索切片对象返回上层应用。",
].forEach((t) => body.push(num(t)));

// 7 安全性设计
body.push(h1("第七章 安全性设计"));
body.push(p("知识库承载企业核心私有数据，其安全性是系统设计的重要考量。系统从部署形态、租户隔离、访问控制、传输与存储、接口安全及隐私合规等多个维度构建知识数据的全生命周期防护体系。"));
body.push(h2("7.1 私有化部署与数据不出域"));
body.push(p("系统支持完整的私有化部署，知识数据的采集、解析、向量化、存储与检索均在用户自有环境内闭环完成，数据不出域，从根本上规避了私有知识外泄的风险，满足金融、政务等对数据主权要求严格的行业需求。"));
body.push(h2("7.2 多租户与空间级隔离"));
body.push(p("系统以“空间（Space）”为基本隔离单元，不同空间在知识库、文档、切片数据以及向量化配置上相互独立，结合检索阶段按知识库、文档范围的精确约束，确保多团队、多业务在同一平台共存时数据互不越界。"));
body.push(h2("7.3 访问控制与权限过滤"));
body.push(p("系统在知识检索链路中支持按知识库、文档维度对检索范围进行约束，仅返回授权范围内的知识片段，避免越权访问后端知识库导致的数据泄露；上层应用可结合业务身份对检索请求进行权限校验与结果过滤，实现最小权限原则。"));
body.push(h2("7.4 传输与存储安全"));
body.push(p("系统在设计上支持对数据传输通道进行加密（如 TLS），对上传文件、解析图片等内容采用对象存储统一管理，并可结合用户侧的备份与加密策略，保障数据在传输与静态存储环节的安全。"));
body.push(h2("7.5 接口安全"));
body.push(p("系统对外接口遵循统一的鉴权与参数校验机制，仅在系统边界进行输入校验，对无效查询进行过滤，防止异常或恶意请求影响服务稳定；外部知识库绑定提供独立的校验接口，保障外部知识源接入的可信与可控。"));
body.push(h2("7.6 隐私与合规"));
body.push(p("系统在文档解析与处理过程中支持对敏感信息进行裁剪（如 URL、邮箱裁剪等），并可结合用户侧的数据脱敏与治理流程，帮助用户满足相关数据安全与隐私保护的合规要求。"));

// 8 性能、可靠性与扩展性
body.push(h1("第八章 性能、可靠性与扩展性"));
body.push(h2("8.1 性能设计"));
body.push(p("系统在性能上的核心设计体现在异步化、并行化与高效索引三个方面：文档索引采用异步事件驱动，避免长任务阻塞接口；检索链路多路召回并行执行，整体检索时延接近最慢单路而非各路之和；向量索引默认采用 HNSW 高性能近似最近邻算法并支持参数化调优；同时通过批量处理（批量上传、批量切片、批量检索）与 Redis 缓存进一步提升吞吐与响应速度。"));
body.push(h2("8.2 可靠性保障"));
body.push(p("系统通过多重机制保障处理过程的可靠性：文档索引过程包裹 panic 恢复机制，避免单文档异常影响整体服务；对处理异常区分可重试与不可重试错误，支持失败重试；提供详细的文档状态机（分段中、已启用、失败等）与实时进度跟踪，处理结果可追溯；并支持对整个空间的全文索引进行一键重建（Resync），便于数据修复与运维恢复。"));
body.push(h2("8.3 可扩展性"));
body.push(p("系统在扩展性上具备两个层面的优势：其一是处理能力的水平扩展，基于消息队列的异步索引消费者可按需扩容以支撑大规模文档处理；其二是技术组件的可插拔扩展，解析器、向量存储、重排器、NL2SQL、OCR 等能力均通过契约接口抽象并提供多种实现，用户可在不改动核心逻辑的前提下替换或新增底层组件，从而灵活适配不同的部署环境与成本要求。"));
body.push(h2("8.4 可观测性"));
body.push(p("系统内置 Prometheus 指标采集，对解析耗时、文件大小、失败原因等关键指标进行监控，结合进度跟踪与状态机，为容量评估、性能调优与故障定位提供数据支撑。"));

// 9 接口说明
body.push(h1("第九章 接口说明"));
body.push(p("知识库子系统对外提供标准化的 HTTP 接口，按功能划分为知识库管理、文档管理、切片管理、图片管理、表结构、文档审阅、检索测试与外部知识库等若干接口组。主要接口如下表所示（节选）。"));
body.push(h2("7.1 知识库与文档接口"));
body.push(tbl([3800, 1400, 3826], [
  ["接口路径", "方法", "功能说明"],
  ["/api/knowledge/create", "POST", "创建知识库"],
  ["/api/knowledge/update", "POST", "更新知识库"],
  ["/api/knowledge/delete", "POST", "删除知识库"],
  ["/api/knowledge/detail", "POST", "获取知识库详情"],
  ["/api/knowledge/list", "POST", "分页查询知识库列表"],
  ["/api/knowledge/document/create", "POST", "上传并创建文档"],
  ["/api/knowledge/document/update", "POST", "更新文档元数据"],
  ["/api/knowledge/document/delete", "POST", "删除文档"],
  ["/api/knowledge/document/list", "POST", "查询文档列表"],
  ["/api/knowledge/document/progress/get", "POST", "查询文档处理进度"],
  ["/api/knowledge/document/resegment", "POST", "按新策略重新分段"],
]));
body.push(h2("7.2 切片、图片与其他接口"));
body.push(tbl([3800, 1400, 3826], [
  ["接口路径", "方法", "功能说明"],
  ["/api/knowledge/slice/create", "POST", "创建切片"],
  ["/api/knowledge/slice/update", "POST", "更新切片"],
  ["/api/knowledge/slice/delete", "POST", "删除切片"],
  ["/api/knowledge/slice/list", "POST", "查询切片列表"],
  ["/api/knowledge/photo/list", "POST", "查询图片切片列表"],
  ["/api/knowledge/photo/caption", "POST", "更新图片图注"],
  ["/api/knowledge/photo/extract_caption", "POST", "自动抽取图注"],
  ["/api/knowledge/table_schema/get", "POST", "获取表结构"],
  ["/api/knowledge/table_schema/validate", "POST", "校验表结构"],
  ["/api/knowledge/retrieve_test", "POST", "检索测试"],
  ["/api/external-knowledge/retrieval", "POST", "外部知识库检索"],
]));

// 10 部署方案与运行环境
body.push(h1("第十章 部署方案与运行环境"));
body.push(h2("10.1 部署方案"));
body.push(p("系统支持私有化与容器化（Docker）部署，可依据用户的数据规模、并发要求与高可用诉求灵活选择部署形态：对于中小规模场景，可采用单实例部署快速上线；对于大规模、高并发场景，可对索引消费者、检索服务及各中间件进行集群化与多副本部署，并通过对消息队列消费者的水平扩展提升文档处理吞吐，从而实现弹性伸缩与高可用。"));
body.push(h2("10.2 运行环境要求"));
body.push(p("知识库子系统的典型运行环境要求如下表所示。实际部署时可依据数据规模与并发要求进行弹性伸缩。"));
body.push(tbl([2400, 6626], [
  ["类别", "说明"],
  ["开发语言", "后端 Go，前端 TypeScript / React"],
  ["运行平台", "Linux 服务器，支持容器化（Docker）部署"],
  ["数据库", "关系型数据库（存储知识库、文档、切片元数据）"],
  ["向量数据库", "Milvus / VikingDB / OceanBase Vector（任选其一或组合）"],
  ["全文检索", "Elasticsearch 7 / 8"],
  ["缓存中间件", "Redis"],
  ["消息中间件", "消息队列（用于文档索引异步处理）"],
  ["对象存储", "用于存放文档解析产生的图片及上传文件"],
]));

// 11 应用场景
body.push(h1("第十一章 典型应用场景"));
body.push(h2("11.1 企业智能客服与问答机器人"));
body.push(p("将企业产品手册、业务规章、FAQ 等文档接入知识库，结合检索增强生成为客服机器人提供准确、可溯源的答案，降低人工客服压力，提升服务一致性与响应效率。"));
body.push(h2("11.2 智能体（Agent）知识底座"));
body.push(p("作为智能体编排平台的知识能力中台，为各类业务智能体提供标准化的知识检索接口，使智能体在执行任务时能够实时调用企业私有知识，增强决策与生成的准确性。"));
body.push(h2("11.3 企业内部知识管理与检索"));
body.push(p("面向企业内部的制度文件、技术文档、项目资料等，提供语义检索与混合检索能力，帮助员工快速定位所需知识，沉淀组织经验，提升知识复用效率。"));
body.push(h2("11.4 结构化与非结构化数据融合查询"));
body.push(p("借助 NL2SQL 表格召回与向量 / 全文召回的融合能力，系统可同时处理报表、台账等结构化数据与文档类非结构化数据，支持以自然语言对混合数据进行统一查询。"));

// 12 演进路线
body.push(h1("第十二章 产品演进路线"));
body.push(p("围绕“接入更广、检索更准、性能更强、治理更优”的目标，知识库子系统将持续迭代演进，主要方向包括："));
body.push(bullet("知识接入广度：持续扩展文档格式与数据源支持，增强多模态（图文、表格）知识的解析与理解能力。"));
body.push(bullet("检索精度：优化分段策略与多路召回融合算法，引入更丰富的重排与查询改写能力，进一步提升复杂场景下的召回相关性。"));
body.push(bullet("知识治理：完善文档质量审阅、知识更新与一致性校验机制，构建更完整的知识全生命周期治理体系。"));
body.push(bullet("性能与可观测：持续优化索引与检索性能，丰富监控指标与运维工具，提升大规模部署下的稳定性与可维护性。"));

// 13 总结
body.push(h1("第十三章 总结"));
body.push(p("AI 智能知识库平台知识库子系统以检索增强生成为核心理念，构建了覆盖知识接入、解析分段、向量化索引、多路召回、结果重排到对外服务的完整技术体系。系统在架构上分层解耦、契约先行，在能力上多格式接入、混合检索、多引擎兼容，在性能上异步并行、可水平扩展，在运维上多租户隔离、全面可观测，能够为企业级 AI 应用提供稳定、高效、可扩展的知识能力底座。"));
body.push(p("本软件已取得国家版权局核发的计算机软件著作权登记证书（登记号：2026SR0260508），其著作权依法受到保护。未来，本公司将持续围绕知识接入广度、检索精度与系统性能进行迭代优化，为用户创造更大价值。"));

// ---------- document ----------
const doc = new Document({
  creator: "北京易诚互动网络技术股份有限公司",
  title: "AI 智能知识库平台 技术白皮书 V1.0",
  description: "AI 智能知识库平台知识库子系统技术白皮书",
  styles: {
    default: { document: { run: { font: FONT, size: 22 } } },
    paragraphStyles: [
      { id: "Heading1", name: "Heading 1", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 30, bold: true, font: FONT_H, color: "1F3864" },
        paragraph: { spacing: { before: 320, after: 200 }, outlineLevel: 0 } },
      { id: "Heading2", name: "Heading 2", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 26, bold: true, font: FONT_H, color: "2E5496" },
        paragraph: { spacing: { before: 220, after: 140 }, outlineLevel: 1 } },
      { id: "Heading3", name: "Heading 3", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 23, bold: true, font: FONT_H, color: "44546A" },
        paragraph: { spacing: { before: 160, after: 100 }, outlineLevel: 2 } },
    ],
  },
  numbering: {
    config: [
      { reference: "bullets", levels: [{ level: 0, format: LevelFormat.BULLET, text: "●", alignment: AlignmentType.LEFT,
        style: { run: { font: FONT }, paragraph: { indent: { left: 600, hanging: 280 } } } }] },
      { reference: "numbers", levels: [{ level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
        style: { paragraph: { indent: { left: 600, hanging: 320 } } } }] },
    ],
  },
  sections: [
    // cover section - no header/footer page number distraction
    {
      properties: { page: { size: { width: 11906, height: 16838 }, margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 } } },
      children: [...cover, ...notice, ...docinfo, ...toc],
    },
    // main section with header & footer
    {
      properties: { page: { size: { width: 11906, height: 16838 }, margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 } } },
      headers: { default: new Header({ children: [new Paragraph({
        alignment: AlignmentType.RIGHT,
        border: { bottom: { style: BorderStyle.SINGLE, size: 4, color: "AAAAAA", space: 4 } },
        children: [new TextRun({ text: "AI 智能知识库平台 技术白皮书 V1.0", font: FONT, size: 18, color: "888888" })],
      })] }) },
      footers: { default: new Footer({ children: [new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [
          new TextRun({ text: "第 ", font: FONT, size: 18, color: "888888" }),
          new TextRun({ children: [PageNumber.CURRENT], font: FONT, size: 18, color: "888888" }),
          new TextRun({ text: " 页", font: FONT, size: 18, color: "888888" }),
        ],
      })] }) },
      children: body,
    },
  ],
});

Packer.toBuffer(doc).then((buf) => {
  fs.writeFileSync("AI智能知识库平台-技术白皮书-V1.0.docx", buf);
  console.log("written", buf.length, "bytes");
});
