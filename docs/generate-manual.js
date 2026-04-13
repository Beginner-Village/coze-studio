const fs = require("fs");
const path = require("path");
const {
  Document, Packer, Paragraph, TextRun, Table, TableRow, TableCell,
  ImageRun, Header, Footer, AlignmentType, HeadingLevel, BorderStyle,
  WidthType, ShadingType, PageNumber, PageBreak, LevelFormat
} = require("docx");

const SCREENSHOTS = path.join(__dirname, "screenshots");

function img(filename, w, h) {
  const filePath = path.join(SCREENSHOTS, filename);
  if (!fs.existsSync(filePath)) {
    console.warn(`WARNING: ${filename} not found`);
    return new Paragraph({ children: [new TextRun({ text: `[截图缺失: ${filename}]`, italics: true, color: "FF0000" })] });
  }
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { before: 120, after: 120 },
    children: [new ImageRun({
      type: "png",
      data: fs.readFileSync(filePath),
      transformation: { width: w || 580, height: h || 362 },
      altText: { title: filename, description: filename, name: filename }
    })]
  });
}

function caption(text) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { after: 200 },
    children: [new TextRun({ text, size: 20, italics: true, color: "666666", font: "Microsoft YaHei" })]
  });
}

function heading1(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_1,
    spacing: { before: 360, after: 200 },
    children: [new TextRun({ text, bold: true, size: 32, font: "Microsoft YaHei" })]
  });
}

function heading2(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_2,
    spacing: { before: 240, after: 160 },
    children: [new TextRun({ text, bold: true, size: 28, font: "Microsoft YaHei" })]
  });
}

function heading3(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_3,
    spacing: { before: 200, after: 120 },
    children: [new TextRun({ text, bold: true, size: 24, font: "Microsoft YaHei" })]
  });
}

function para(text, opts = {}) {
  return new Paragraph({
    spacing: { after: 120 },
    children: [new TextRun({ text, size: 22, font: "Microsoft YaHei", ...opts })]
  });
}

function bulletItem(text, ref = "bullets") {
  return new Paragraph({
    numbering: { reference: ref, level: 0 },
    spacing: { after: 80 },
    children: [new TextRun({ text, size: 22, font: "Microsoft YaHei" })]
  });
}

function numberedItem(text, ref = "numbers") {
  return new Paragraph({
    numbering: { reference: ref, level: 0 },
    spacing: { after: 80 },
    children: [new TextRun({ text, size: 22, font: "Microsoft YaHei" })]
  });
}

const border = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
const borders = { top: border, bottom: border, left: border, right: border };
const cellMargins = { top: 60, bottom: 60, left: 100, right: 100 };

function tableCell(text, opts = {}) {
  const width = opts.width || 2340;
  return new TableCell({
    borders,
    width: { size: width, type: WidthType.DXA },
    margins: cellMargins,
    shading: opts.header ? { fill: "2B5797", type: ShadingType.CLEAR } : undefined,
    children: [new Paragraph({
      children: [new TextRun({
        text,
        size: 20,
        font: "Microsoft YaHei",
        bold: opts.header || false,
        color: opts.header ? "FFFFFF" : "333333"
      })]
    })]
  });
}

function makeTable(headers, rows, colWidths) {
  const totalWidth = 9360;
  const widths = colWidths || headers.map(() => Math.floor(totalWidth / headers.length));
  return new Table({
    width: { size: totalWidth, type: WidthType.DXA },
    columnWidths: widths,
    rows: [
      new TableRow({
        children: headers.map((h, i) => tableCell(h, { header: true, width: widths[i] }))
      }),
      ...rows.map(row => new TableRow({
        children: row.map((cell, i) => tableCell(cell, { width: widths[i] }))
      }))
    ]
  });
}

// ============ BUILD DOCUMENT ============

const doc = new Document({
  numbering: {
    config: [
      {
        reference: "bullets",
        levels: [{
          level: 0, format: LevelFormat.BULLET, text: "\u2022", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      },
      {
        reference: "numbers",
        levels: [{
          level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      },
      {
        reference: "numbers2",
        levels: [{
          level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      },
      {
        reference: "numbers3",
        levels: [{
          level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      },
      {
        reference: "numbers4",
        levels: [{
          level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      },
      {
        reference: "numbers5",
        levels: [{
          level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } }
        }]
      }
    ]
  },
  styles: {
    default: { document: { run: { font: "Microsoft YaHei", size: 22 } } },
    paragraphStyles: [
      { id: "Heading1", name: "Heading 1", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 32, bold: true, font: "Microsoft YaHei" },
        paragraph: { spacing: { before: 360, after: 200 }, outlineLevel: 0 } },
      { id: "Heading2", name: "Heading 2", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 28, bold: true, font: "Microsoft YaHei" },
        paragraph: { spacing: { before: 240, after: 160 }, outlineLevel: 1 } },
      { id: "Heading3", name: "Heading 3", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 24, bold: true, font: "Microsoft YaHei" },
        paragraph: { spacing: { before: 200, after: 120 }, outlineLevel: 2 } },
    ]
  },
  sections: [
    // ========== COVER PAGE ==========
    {
      properties: {
        page: {
          size: { width: 11906, height: 16838 },
          margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 }
        }
      },
      children: [
        new Paragraph({ spacing: { before: 3000 } }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 200 },
          children: [new TextRun({ text: "\u730E\u9E70\u667A\u80FD\u4F53\u5E73\u53F0", bold: true, size: 52, font: "Microsoft YaHei", color: "2B5797" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 600 },
          children: [new TextRun({ text: "\u4F7F\u7528\u8BF4\u660E\u4E66", bold: true, size: 36, font: "Microsoft YaHei", color: "555555" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 100 },
          children: [new TextRun({ text: "\u7248\u672C: V1.0", size: 24, font: "Microsoft YaHei", color: "888888" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 100 },
          children: [new TextRun({ text: "\u65E5\u671F: 2026\u5E744\u67082\u65E5", size: 24, font: "Microsoft YaHei", color: "888888" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 100 },
          children: [new TextRun({ text: "\u5BC6\u7EA7: \u5185\u90E8\u4F7F\u7528", size: 24, font: "Microsoft YaHei", color: "888888" })]
        }),
        new Paragraph({ spacing: { before: 2000 } }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new TextRun({ text: "\u6210\u90FD\u519C\u5546\u94F6\u884C AI \u677F\u5757\u9879\u76EE", size: 24, font: "Microsoft YaHei", color: "999999" })]
        }),
      ]
    },
    // ========== MAIN CONTENT ==========
    {
      properties: {
        page: {
          size: { width: 11906, height: 16838 },
          margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 }
        }
      },
      headers: {
        default: new Header({
          children: [new Paragraph({
            alignment: AlignmentType.RIGHT,
            children: [new TextRun({ text: "\u730E\u9E70\u667A\u80FD\u4F53\u5E73\u53F0 - \u4F7F\u7528\u8BF4\u660E\u4E66", size: 18, color: "999999", font: "Microsoft YaHei" })]
          })]
        })
      },
      footers: {
        default: new Footer({
          children: [new Paragraph({
            alignment: AlignmentType.CENTER,
            children: [new TextRun({ text: "\u7B2C ", size: 18, font: "Microsoft YaHei" }), new TextRun({ children: [PageNumber.CURRENT], size: 18, font: "Microsoft YaHei" }), new TextRun({ text: " \u9875", size: 18, font: "Microsoft YaHei" })]
          })]
        })
      },
      children: [
        // ===== 1. 平台概述 =====
        heading1("1. \u5E73\u53F0\u6982\u8FF0"),
        para("\u730E\u9E70\u667A\u80FD\u4F53\u5E73\u53F0\u662F\u4E00\u5957\u4F01\u4E1A\u7EA7 AI \u667A\u80FD\u4F53\u5F00\u53D1\u4E0E\u7BA1\u7406\u5E73\u53F0\uFF0C\u63D0\u4F9B\u4ECE\u667A\u80FD\u4F53\u521B\u5EFA\u3001\u6A21\u578B\u914D\u7F6E\u3001\u77E5\u8BC6\u5E93\u7BA1\u7406\u3001\u5DE5\u4F5C\u6D41\u7F16\u6392\u5230\u5B89\u5168\u56F4\u680F\u9632\u62A4\u7684\u5168\u6D41\u7A0B\u80FD\u529B\u3002"),
        para("\u5E73\u53F0\u7531\u4E09\u4E2A\u6838\u5FC3\u5B50\u7CFB\u7EDF\u7EC4\u6210\uFF1A"),

        makeTable(
          ["\u5B50\u7CFB\u7EDF", "\u529F\u80FD\u8BF4\u660E", "\u8BBF\u95EE\u5730\u5740"],
          [
            ["\u667A\u80FD\u4F53\u5E73\u53F0 (Studio)", "\u667A\u80FD\u4F53\u5F00\u53D1\u3001\u6A21\u578B\u7BA1\u7406\u3001\u77E5\u8BC6\u5E93\u3001\u5DE5\u4F5C\u6D41", "http://30.3.165.209:9888"],
            ["\u53EF\u89C2\u6D4B\u6027\u5E73\u53F0 (Loop)", "\u8FD0\u884C\u76D1\u63A7\u3001Trace \u8FFD\u8E2A\u3001\u65E5\u5FD7\u67E5\u8BE2", "\u5DF2\u5D4C\u5165 Studio \u5E73\u53F0"],
            ["\u5B89\u5168\u56F4\u680F (Guard)", "AI \u5B89\u5168\u68C0\u6D4B\u3001\u5185\u5BB9\u5BA1\u6838\u3001\u5173\u952E\u8BCD\u8FC7\u6EE4", "http://30.3.165.211:8080"],
          ],
          [2500, 4000, 2860]
        ),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 附录A: 部署架构 =====
        heading1("\u9644\u5F55A. \u90E8\u7F72\u67B6\u6784"),

        heading2("A.1 \u670D\u52A1\u5668\u5206\u5DE5"),
        makeTable(
          ["\u670D\u52A1\u5668IP", "\u89D2\u8272", "\u90E8\u7F72\u5185\u5BB9", "\u5BF9\u5916\u7AEF\u53E3"],
          [
            ["30.3.165.209", "Studio \u5E94\u7528\u670D\u52A1\u5668", "ynet-server (\u540E\u7AEF) + ynet-web (\u524D\u7AEF)", "9888 (Web), 9889 (MinIO\u4EE3\u7406)"],
            ["30.3.165.210", "Loop \u5E94\u7528\u670D\u52A1\u5668", "ynet-loop-app + ynet-loop-nginx", "8888 (API)"],
            ["30.3.165.211", "Guard \u5E94\u7528\u670D\u52A1\u5668", "guard-app (Python+Nginx)", "8080 (Web+API)"],
          ],
          [1800, 2200, 3200, 2160]
        ),

        heading2("A.2 \u4E2D\u95F4\u4EF6\u90E8\u7F72"),
        makeTable(
          ["\u4E2D\u95F4\u4EF6", "\u670D\u52A1\u5668IP", "\u7AEF\u53E3", "\u7248\u672C", "\u4F7F\u7528\u65B9"],
          [
            ["OceanBase", "30.5.9.252 (\u96C6\u7FA4)", "2883", "OB\u96C6\u7FA4", "Studio + Loop + Guard"],
            ["Redis", "30.3.165.185", "7080", "6.2.3", "Studio + Loop + Guard"],
            ["Elasticsearch", "30.3.165.196", "9200", "7.12.1", "Studio + Guard"],
            ["MinIO", "30.3.165.191", "9000", "2023.x", "Studio + Loop"],
            ["RocketMQ", "30.3.165.182", "9876", "5.x", "Studio + Loop"],
            ["ClickHouse", "30.3.165.198", "9000", "\u5B89\u88C5\u5305", "Loop"],
            ["OneAPI (LLM\u4EE3\u7406)", "30.3.162.95", "4000", "-", "Studio"],
          ],
          [1800, 1800, 1000, 1200, 3560]
        ),

        heading2("A.3 \u4E2D\u95F4\u4EF6\u4F7F\u7528\u77E9\u9635"),
        para("\u4E0B\u8868\u5C55\u793A\u5404\u5E94\u7528\u5BF9\u4E2D\u95F4\u4EF6\u7684\u4F9D\u8D56\u5173\u7CFB\uFF1A"),
        makeTable(
          ["\u4E2D\u95F4\u4EF6", "Studio (209)", "Loop (210)", "Guard (211)"],
          [
            ["OceanBase", "ai_studio (\u8BFB\u5199)", "ai_loop (\u8BFB\u5199)", "ai_guard (\u8BFB\u5199)"],
            ["Redis", "\u7F13\u5B58/Session", "\u7F13\u5B58", "\u7F13\u5B58"],
            ["Elasticsearch", "\u9879\u76EE/\u8D44\u6E90\u641C\u7D22", "-", "\u5B89\u5168\u89C4\u5219\u68C0\u7D22"],
            ["MinIO", "\u6587\u4EF6\u5B58\u50A8 (openynet)", "\u65E5\u5FD7\u5B58\u50A8", "-"],
            ["RocketMQ", "\u6D88\u606F\u961F\u5217", "\u6D88\u606F\u961F\u5217", "-"],
            ["ClickHouse", "-", "Trace/\u65F6\u5E8F\u6570\u636E", "-"],
            ["OneAPI", "\u6A21\u578B\u8C03\u7528", "-", "-"],
          ],
          [1800, 2520, 2520, 2520]
        ),

        heading2("A.4 \u8BBF\u95EE\u5730\u5740\u6C47\u603B"),
        makeTable(
          ["\u7CFB\u7EDF", "\u5730\u5740", "\u5907\u6CE8"],
          [
            ["Studio (\u667A\u80FD\u4F53\u5E73\u53F0)", "http://30.3.165.209:9888", "\u4E3B\u5165\u53E3"],
            ["Guard (\u5B89\u5168\u56F4\u680F)", "http://30.3.165.211:8080", "\u72EC\u7ACB\u8BBF\u95EE"],
            ["Loop (\u53EF\u89C2\u6D4B\u6027)", "\u5D4C\u5165 Studio \u5DE6\u4FA7\u83DC\u5355", "\u901A\u8FC7 Studio \u8BBF\u95EE"],
          ],
          [2600, 3000, 3760]
        ),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 附录B: 部署指南 =====
        heading1("\u9644\u5F55B. \u90E8\u7F72\u6307\u5357"),

        heading2("B.1 \u90E8\u7F72\u987A\u5E8F\u603B\u89C8"),
        para("\u90E8\u7F72\u5206\u56DB\u4E2A\u9636\u6BB5\uFF0C\u5FC5\u987B\u6309\u987A\u5E8F\u6267\u884C\uFF1A"),
        makeTable(
          ["\u9636\u6BB5", "\u5185\u5BB9", "\u6D89\u53CA\u670D\u52A1\u5668"],
          [
            ["\u4E00. \u4E2D\u95F4\u4EF6\u521D\u59CB\u5316", "ES \u542F\u52A8\u3001ClickHouse \u5EFA\u8868\u3001RocketMQ \u521B\u5EFA Topic", "196, 198, 182"],
            ["\u4E8C. \u6570\u636E\u5E93\u521D\u59CB\u5316", "OceanBase \u5EFA\u8868 (ai_studio/ai_loop/ai_guard)\u3001ClickHouse \u5EFA\u8868", "OB\u96C6\u7FA4, 198"],
            ["\u4E09. \u5E94\u7528\u90E8\u7F72", "Studio(209) \u2192 Loop(210) \u2192 Guard(211) \u987A\u5E8F\u90E8\u7F72", "209, 210, 211"],
            ["\u56DB. \u90E8\u7F72\u540E\u914D\u7F6E", "Loop Token \u751F\u6210\u3001\u56FE\u6807\u4E0A\u4F20\u3001\u9A8C\u8BC1\u6D4B\u8BD5", "209, 210"],
          ],
          [1800, 5000, 2560]
        ),

        heading2("B.2 \u4E2D\u95F4\u4EF6\u521D\u59CB\u5316"),
        heading3("B.2.1 Elasticsearch (30.3.165.196)"),
        para("\u786E\u8BA4 ES \u670D\u52A1\u8FD0\u884C\u4E2D\uFF1A"),
        para("curl -s http://30.3.165.196:9200", { font: "Consolas", size: 20 }),
        para("\u8FD4\u56DE\u96C6\u7FA4\u4FE1\u606F\u5373\u6B63\u5E38\u3002\u65E0\u9700\u914D\u7F6E\u8BA4\u8BC1\u3002"),

        heading3("B.2.2 ClickHouse (30.3.165.198)"),
        para("\u521B\u5EFA observability_spans \u8868\u7528\u4E8E\u5B58\u50A8 Trace \u6570\u636E\uFF1A"),
        para("\u6267\u884C\u811A\u672C: init-clickhouse-v2.sh\uFF0C\u8BE5\u811A\u672C\u901A\u8FC7 ClickHouse HTTP \u63A5\u53E3 (8123) \u521B\u5EFA\u8868\u3002"),
        para("\u8868\u7ED3\u6784\u5305\u542B: trace_id, span_id, space_id, span_name, duration, input, output \u7B49\u5B57\u6BB5\uFF0C\u4F7F\u7528 MergeTree \u5F15\u64CE\uFF0C\u6309\u65E5\u671F\u5206\u533A\u3002"),

        heading3("B.2.3 RocketMQ (30.3.165.182)"),
        para("\u521B\u5EFA\u5FC5\u8981\u7684 Topic \u548C\u6D88\u8D39\u7EC4\uFF1A"),
        para("\u6267\u884C\u811A\u672C: init-rmq-all.sh\uFF0C\u5171\u521B\u5EFA 13 \u4E2A Topic\uFF1A"),
        bulletItem("trace_ingestion_event, trace_annotation_event (\u53EF\u89C2\u6D4B\u6027)"),
        bulletItem("data_async_tasks, evaluation_expt_* (\u8BC4\u4F30)"),
        bulletItem("openynet_* (\u77E5\u8BC6\u5E93)"),

        heading2("B.3 \u6570\u636E\u5E93\u521D\u59CB\u5316"),
        heading3("B.3.1 OceanBase \u5EFA\u8868"),
        para("\u4E09\u4E2A\u5E94\u7528\u5206\u522B\u4F7F\u7528\u72EC\u7ACB\u7684\u6570\u636E\u5E93\uFF1A"),
        makeTable(
          ["\u6570\u636E\u5E93", "\u7528\u9014", "\u5EFA\u8868\u65B9\u5F0F"],
          [
            ["ai_studio", "Studio \u4E3B\u5E93 (112+ \u8868)", "\u5E94\u7528\u9996\u6B21\u542F\u52A8\u81EA\u52A8\u5EFA\u8868"],
            ["ai_loop", "Loop \u4E3B\u5E93 (44+ \u8868)", "\u6267\u884C init-loop-db.sh + 02-ynet-loop.sql"],
            ["ai_guard", "Guard \u4E3B\u5E93 (18+ \u8868)", "\u5E94\u7528\u9996\u6B21\u542F\u52A8\u81EA\u52A8\u5EFA\u8868"],
          ],
          [2000, 3500, 3860]
        ),

        heading3("B.3.2 \u8865\u5145\u8868 (Embedding/Rerank)"),
        para("\u5982\u9700 Embedding/Rerank \u529F\u80FD\uFF0C\u6267\u884C create-tables-cdrcb3.sh \u5728 ai_studio \u5E93\u4E2D\u521B\u5EFA space_embedding \u548C space_rerank \u8868\u3002"),

        heading3("B.3.3 OceanBase \u7248\u672C\u8981\u6C42"),
        para("\u77E5\u8BC6\u5E93\u5411\u91CF\u68C0\u7D22\u529F\u80FD\u8981\u6C42 OceanBase >= 4.3.0 (\u652F\u6301 VECTOR \u6570\u636E\u7C7B\u578B)\u3002\u53EF\u4F7F\u7528 check-ob-vector-v2.sh \u9A8C\u8BC1\u3002"),

        heading2("B.4 \u5E94\u7528\u90E8\u7F72"),
        heading3("B.4.1 Studio \u90E8\u7F72 (209)"),
        para("\u6267\u884C\u811A\u672C: deploy-studio.sh"),
        numberedItem("\u52A0\u8F7D\u955C\u50CF: ynet-server.tar.gz + ynet-web.tar.gz", "numbers"),
        numberedItem("\u7F51\u7EDC\u8FDE\u901A\u6027\u68C0\u67E5 (OB/Redis/ES/MinIO/RMQ/Loop)", "numbers"),
        numberedItem("\u751F\u6210 Nginx \u914D\u7F6E (\u542B MinIO \u4EE3\u7406\u3001Loop \u96C6\u6210)", "numbers"),
        numberedItem("\u751F\u6210 docker-compose.yml (\u542B\u5168\u90E8\u4E2D\u95F4\u4EF6\u73AF\u5883\u53D8\u91CF)", "numbers"),
        numberedItem("\u542F\u52A8\u5BB9\u5668\u5E76\u7B49\u5F85\u5065\u5EB7\u68C0\u67E5\u901A\u8FC7", "numbers"),
        para("\u5173\u952E\u73AF\u5883\u53D8\u91CF\uFF1A"),
        makeTable(
          ["\u914D\u7F6E\u9879", "\u503C"],
          [
            ["MYSQL_HOST", "30.5.9.252:2883"],
            ["REDIS_ADDR", "30.3.165.185:7080"],
            ["ES_ADDR", "http://30.3.165.196:9200"],
            ["MINIO_ENDPOINT", "30.3.165.191:9000"],
            ["MQ_NAME_SERVER", "30.3.165.182:9876"],
            ["YNET_LOOP_PROXY_URL", "http://30.3.165.210:8888"],
            ["YNET_LOOP_TELEMETRY_ENABLE", "1"],
          ],
          [3500, 5860]
        ),

        heading3("B.4.2 Loop \u90E8\u7F72 (210)"),
        para("\u6267\u884C\u811A\u672C: deploy-loop-v2.sh"),
        numberedItem("\u52A0\u8F7D\u955C\u50CF: ynet-loop-app-v2.tar.gz", "numbers2"),
        numberedItem("Retag \u955C\u50CF\u5339\u914D docker-compose.yml \u4E2D\u7684\u955C\u50CF\u540D", "numbers2"),
        numberedItem("\u8986\u76D6 docker-compose.yml (\u542B AutoAuthMW \u73AF\u5883\u53D8\u91CF)", "numbers2"),
        numberedItem("\u542F\u52A8\u5BB9\u5668\u5E76\u9A8C\u8BC1 /ping \u548C OTEL endpoint", "numbers2"),
        para("\u5173\u952E\u914D\u7F6E\uFF1A"),
        bulletItem("AutoAuthMW \u73AF\u5883\u53D8\u91CF: YNET_LOOP_DEFAULT_USER_ID / NAME / EMAIL"),
        bulletItem(".env \u6587\u4EF6\u542B\u6240\u6709\u4E2D\u95F4\u4EF6\u8FDE\u63A5\u4FE1\u606F"),
        bulletItem("observability.yaml \u914D\u7F6E Trace \u91C7\u96C6 (RMQ + ClickHouse)"),

        heading3("B.4.3 Guard \u90E8\u7F72 (211)"),
        para("\u6267\u884C\u811A\u672C: deploy-guard-final.sh"),
        numberedItem("\u52A0\u8F7D\u955C\u50CF: guard.tar", "numbers3"),
        numberedItem("\u7F51\u7EDC\u8FDE\u901A\u6027\u68C0\u67E5 (OB/Redis/ES)", "numbers3"),
        numberedItem("\u751F\u6210 docker-compose.yml (\u542B\u6A21\u578B API \u914D\u7F6E)", "numbers3"),
        numberedItem("\u542F\u52A8\u5BB9\u5668\u5E76\u9A8C\u8BC1\u767B\u5F55\u548C\u5B89\u5168\u68C0\u6D4B", "numbers3"),

        heading2("B.5 \u90E8\u7F72\u540E\u914D\u7F6E"),
        heading3("B.5.1 Loop Token \u751F\u6210"),
        para("\u5728 210 \u4E0A\u6267\u884C loop-token4.sh\uFF0C\u83B7\u53D6\u4E09\u4E2A\u503C\u5E76\u914D\u7F6E\u5230 209 \u7684 docker-compose.yml\uFF1A"),
        bulletItem("YNET_LOOP_TELEMETRY_TOKEN: PAT \u4EE4\u724C"),
        bulletItem("YNET_LOOP_WORKSPACE_ID: \u5DE5\u4F5C\u7A7A\u95F4 ID"),
        bulletItem("YNET_LOOP_SESSION_KEY: \u4F1A\u8BDD\u5BC6\u94A5"),

        heading3("B.5.2 \u56FE\u6807\u4E0A\u4F20"),
        para("\u5728 209 \u4E0A\u6267\u884C upload-icons.sh\uFF0C\u5C06\u9ED8\u8BA4\u56FE\u6807\u4E0A\u4F20\u5230 MinIO \u7684 openynet bucket\u3002\u7F3A\u5C11\u56FE\u6807\u4F1A\u5BFC\u81F4\u524D\u7AEF\u663E\u793A 403\u3002"),

        heading3("B.5.3 \u9A8C\u8BC1\u6E05\u5355"),
        makeTable(
          ["\u9A8C\u8BC1\u9879", "\u5982\u4F55\u9A8C\u8BC1", "\u9884\u671F\u7ED3\u679C"],
          [
            ["Studio \u9996\u9875", "\u6D4F\u89C8\u5668\u8BBF\u95EE 209:9888", "\u767B\u5F55\u9875\u6B63\u5E38\u663E\u793A"],
            ["Studio \u540E\u7AEF", "curl 209:8888/api/health", "\u8FD4\u56DE 200"],
            ["Loop ping", "curl 210:8888/ping", '{"message":"pong"}'],
            ["Loop OTEL", "POST 210:8888/v1/loop/opentelemetry/v1/traces", "code:0"],
            ["Guard \u5065\u5EB7", "curl 211:8080/health", "\u8FD4\u56DE 200"],
            ["\u667A\u80FD\u4F53\u5BF9\u8BDD", "\u521B\u5EFA\u667A\u80FD\u4F53\u5E76\u53D1\u9001\u6D88\u606F", "\u6B63\u5E38\u56DE\u590D"],
            ["\u53EF\u89C2\u6D4B\u6027", "Studio \u5DE6\u4FA7\u83DC\u5355 \u2192 \u53EF\u89C2\u6D4B\u6027", "\u663E\u793A Trace \u6570\u636E"],
          ],
          [2000, 4000, 3360]
        ),

        heading2("B.6 \u5E38\u89C1\u95EE\u9898\u4E0E\u89E3\u51B3\u65B9\u6848"),
        makeTable(
          ["\u95EE\u9898", "\u539F\u56E0", "\u89E3\u51B3\u65B9\u6848"],
          [
            [".env \u4E2D # \u53F7\u88AB\u622A\u65AD", "mbank@mbank#dev \u7684 # \u88AB\u5F53\u6CE8\u91CA", '\u7528\u53CC\u5F15\u53F7\u5305\u88F9: MYSQL_USER="mbank@mbank#dev"'],
            ["\u5BB9\u5668\u65E0\u6CD5\u8BBF\u95EE OB", "bridge \u7F51\u7EDC\u65E0\u6CD5\u8DEF\u7531\u5230\u5916\u90E8", "\u6539\u4E3A network_mode: host \u6216\u914D\u7F6E\u7F51\u7EDC"],
            ["\u9ED8\u8BA4\u56FE\u6807 403", "MinIO \u7F3A\u5C11 default_icon \u6587\u4EF6", "\u6267\u884C upload-icons.sh \u4E0A\u4F20\u56FE\u6807"],
            ["\u667A\u80FD\u4F53\u5BF9\u8BDD 401", "OneAPI Key \u53EA\u5141\u8BB8\u7279\u5B9A\u6A21\u578B", "\u6A21\u578B\u540D\u5FC5\u987B\u7CBE\u786E\u5339\u914D OneAPI \u914D\u7F6E"],
            ["\u77E5\u8BC6\u5E93 VECTOR \u62A5\u9519", "OB \u7248\u672C < 4.3.0 \u4E0D\u652F\u6301\u5411\u91CF", "\u5347\u7EA7 OB \u5230 4.3.0+ \u6216\u4F7F\u7528 ES \u66FF\u4EE3"],
            ["Loop \u65E0\u6570\u636E", "ClickHouse \u672A\u5EFA\u8868", "\u6267\u884C init-clickhouse-v2.sh \u5EFA\u8868"],
            ["Loop \u8BA4\u8BC1\u5931\u8D25", "\u65E7\u955C\u50CF\u65E0 AutoAuthMW", "\u4F7F\u7528 ynet-loop-app-v2 \u955C\u50CF"],
          ],
          [2500, 3000, 3860]
        ),

        heading2("B.7 \u955C\u50CF\u6E05\u5355"),
        makeTable(
          ["\u955C\u50CF\u540D\u79F0", "\u7528\u9014", "\u90E8\u7F72\u670D\u52A1\u5668"],
          [
            ["ynet-studio/ynet-server:latest", "Studio \u540E\u7AEF (Go/Hertz)", "209"],
            ["ynet-studio/ynet-web:latest", "Studio \u524D\u7AEF (Nginx+SPA)", "209"],
            ["ynet-loop/app:latest", "Loop \u540E\u7AEF (Go/Hertz)", "210"],
            ["ynet-loop/ynet-loop-nginx:latest", "Loop \u524D\u7AEF (Nginx)", "210"],
            ["coze-studio/guard:latest", "Guard (Python/FastAPI+Nginx)", "211"],
          ],
          [3500, 3500, 2360]
        ),

        heading2("B.8 \u811A\u672C\u6E05\u5355"),
        para("\u4EE5\u4E0B\u811A\u672C\u6309\u6267\u884C\u987A\u5E8F\u6392\u5217\uFF1A"),
        makeTable(
          ["\u811A\u672C", "\u670D\u52A1\u5668", "\u7528\u9014"],
          [
            ["setup-es.sh", "196", "ES \u542F\u52A8\u4E0E\u914D\u7F6E"],
            ["init-clickhouse-v2.sh", "210", "ClickHouse \u5EFA\u8868"],
            ["init-rmq-all.sh", "182", "RocketMQ Topic \u521B\u5EFA"],
            ["init-loop-db.sh", "210", "Loop \u6570\u636E\u5E93\u5EFA\u8868"],
            ["create-tables-cdrcb3.sh", "209", "Embedding/Rerank \u8868"],
            ["deploy-studio.sh", "209", "Studio \u5168\u91CF\u90E8\u7F72"],
            ["deploy-loop-v2.sh", "210", "Loop \u90E8\u7F72 (\u542B AutoAuthMW)"],
            ["deploy-guard-final.sh", "211", "Guard \u90E8\u7F72"],
            ["loop-token4.sh", "210", "Loop Token \u751F\u6210"],
            ["upload-icons.sh", "209", "MinIO \u56FE\u6807\u4E0A\u4F20"],
            ["check-ob-vector-v2.sh", "209", "OB \u5411\u91CF\u652F\u6301\u9A8C\u8BC1"],
            ["deploy-cdrcb-update.sh", "209", "Studio \u955C\u50CF\u70ED\u66F4\u65B0"],
          ],
          [2800, 1200, 5360]
        ),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 2. 登录与工作空间 =====
        heading1("2. \u767B\u5F55\u4E0E\u5DE5\u4F5C\u7A7A\u95F4"),

        heading2("2.1 \u767B\u5F55\u5E73\u53F0"),
        para("\u6253\u5F00\u6D4F\u89C8\u5668\uFF0C\u8BBF\u95EE\u667A\u80FD\u4F53\u5E73\u53F0\u5730\u5740 http://30.3.165.209:9888\uFF0C\u4F7F\u7528\u7BA1\u7406\u5458\u8D26\u53F7\u767B\u5F55\u3002\u767B\u5F55\u540E\u5C06\u81EA\u52A8\u8FDB\u5165\u5DE5\u4F5C\u7A7A\u95F4\u9996\u9875\u3002"),

        heading2("2.2 \u5DE5\u4F5C\u7A7A\u95F4\u6982\u89C8"),
        para("\u5DE5\u4F5C\u7A7A\u95F4\u662F\u5E73\u53F0\u7684\u6838\u5FC3\u64CD\u4F5C\u533A\u57DF\uFF0C\u5DE6\u4FA7\u5BFC\u822A\u680F\u5305\u542B\u4EE5\u4E0B\u6A21\u5757\uFF1A"),
        bulletItem("\u9879\u76EE\u5F00\u53D1\uFF1A\u67E5\u770B\u548C\u7BA1\u7406\u6240\u6709\u667A\u80FD\u4F53\u9879\u76EE"),
        bulletItem("\u8D44\u6E90\u5E93\uFF1A\u5361\u7247\u3001\u5DE5\u4F5C\u6D41\u3001\u63D2\u4EF6\u3001\u77E5\u8BC6\u5E93\u3001\u6280\u80FD\u3001\u63D0\u793A\u8BCD\u3001\u6570\u636E\u5E93"),
        bulletItem("\u7BA1\u7406\uFF1A\u6A21\u578B\u7BA1\u7406\u3001Embedding\u3001\u5916\u90E8\u667A\u80FD\u4F53\u3001\u6210\u5458\u7BA1\u7406\u3001\u5BFC\u51FA/\u5BFC\u5165\u3001\u53EF\u89C2\u6D4B\u6027"),

        img("01-workspace.png", 560, 350),
        caption("\u56FE 2-1\uFF1A\u5DE5\u4F5C\u7A7A\u95F4\u9996\u9875"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 3. 模型配置 =====
        heading1("3. \u6A21\u578B\u914D\u7F6E"),
        para("\u6A21\u578B\u914D\u7F6E\u662F\u5E73\u53F0\u7684\u6838\u5FC3\u529F\u80FD\uFF0C\u7528\u4E8E\u7BA1\u7406\u667A\u80FD\u4F53\u6240\u4F7F\u7528\u7684\u5927\u8BED\u8A00\u6A21\u578B\u3002\u5E73\u53F0\u652F\u6301\u5BF9\u63A5\u4EFB\u4F55\u517C\u5BB9 OpenAI API \u683C\u5F0F\u7684\u6A21\u578B\u670D\u52A1\u3002"),

        heading2("3.1 \u67E5\u770B\u5DF2\u914D\u7F6E\u6A21\u578B"),
        para("\u5728\u5DE6\u4FA7\u5BFC\u822A\u680F\u70B9\u51FB\u300C\u7BA1\u7406\u300D\u2192\u300C\u6A21\u578B\u7BA1\u7406\u300D\uFF0C\u53EF\u67E5\u770B\u5F53\u524D\u5DF2\u914D\u7F6E\u7684\u6240\u6709\u6A21\u578B\u5217\u8868\u3002\u6BCF\u4E2A\u6A21\u578B\u5361\u7247\u663E\u793A\u6A21\u578B\u540D\u79F0\u3001\u4E0A\u4E0B\u6587\u957F\u5EA6\u3001\u5382\u5546\u4FE1\u606F\u548C\u542F\u7528\u72B6\u6001\u3002"),
        img("02-model-management.png", 560, 350),
        caption("\u56FE 3-1\uFF1A\u6A21\u578B\u914D\u7F6E\u5217\u8868"),

        heading2("3.2 \u6DFB\u52A0\u65B0\u6A21\u578B"),
        para("\u70B9\u51FB\u53F3\u4E0A\u89D2\u300C+ \u6DFB\u52A0\u6A21\u578B\u300D\u6309\u94AE\uFF0C\u8FDB\u5165\u6A21\u578B\u6DFB\u52A0\u9875\u9762\u3002\u9700\u8981\u586B\u5199\u4EE5\u4E0B\u4FE1\u606F\uFF1A"),

        makeTable(
          ["\u5B57\u6BB5", "\u8BF4\u660E", "\u793A\u4F8B"],
          [
            ["\u540D\u79F0 *", "\u6A21\u578B\u663E\u793A\u540D\u79F0", "qwen-max"],
            ["\u5382\u5546 *", "\u6A21\u578B\u63D0\u4F9B\u5546", "qwen / openai / custom"],
            ["\u7C7B\u578B *", "\u6587\u672C\u751F\u6210 / \u5D4C\u5165 / Rerank", "\u6587\u672C\u751F\u6210"],
            ["\u57FA\u7840\u6A21\u578B\u6A21\u677F *", "\u9009\u62E9\u9884\u8BBE\u6A21\u677F\u6216\u81EA\u5B9A\u4E49", "Qwen-Max"],
            ["\u6A21\u578B\u6807\u8BC6", "API\u8C03\u7528\u65F6\u5B9E\u9645\u4F7F\u7528\u7684\u6A21\u578B\u540D", "qwen-max"],
            ["\u6700\u5927token\u957F\u5EA6", "\u4E0A\u4E0B\u6587\u7A97\u53E3\u5927\u5C0F", "128000"],
            ["\u94FE\u63A5 *", "\u6A21\u578B API \u5730\u5740", "https://dashscope.aliyuncs.com/compatible-mode/v1"],
            ["\u5BC6\u94A5 *", "API Key", "sk-xxx..."],
          ],
          [2000, 4360, 3000]
        ),

        img("03-add-model.png", 560, 350),
        caption("\u56FE 3-2\uFF1A\u6DFB\u52A0\u6A21\u578B\u8868\u5355"),

        para("\u914D\u7F6E\u5B8C\u6210\u540E\uFF0C\u70B9\u51FB\u9875\u9762\u5E95\u90E8\u7684\u300C\u4FDD\u5B58\u300D\u6309\u94AE\u5373\u53EF\u3002\u6A21\u578B\u6DFB\u52A0\u6210\u529F\u540E\u4F1A\u663E\u793A\u5728\u6A21\u578B\u5217\u8868\u4E2D\uFF0C\u9ED8\u8BA4\u4E3A\u300C\u5DF2\u542F\u7528\u300D\u72B6\u6001\u3002"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 4. Embedding 配置 =====
        heading1("4. Embedding \u914D\u7F6E"),
        para("Embedding \u914D\u7F6E\u7528\u4E8E\u8BBE\u7F6E\u77E5\u8BC6\u5E93\u7684\u5411\u91CF\u5316\u6A21\u578B\u3002\u5F53\u60A8\u9700\u8981\u4F7F\u7528\u77E5\u8BC6\u5E93\u529F\u80FD\u65F6\uFF0C\u5FC5\u987B\u5148\u914D\u7F6E Embedding \u6A21\u578B\u3002"),

        heading2("4.1 \u6DFB\u52A0 Embedding \u914D\u7F6E"),
        para("\u5728\u5DE6\u4FA7\u5BFC\u822A\u680F\u70B9\u51FB\u300C\u7BA1\u7406\u300D\u2192\u300CEmbedding\u300D\uFF0C\u70B9\u51FB\u300C+ \u6DFB\u52A0\u914D\u7F6E\u300D\u6309\u94AE\u3002"),
        img("05-embedding.png", 560, 350),
        caption("\u56FE 4-1\uFF1AEmbedding \u914D\u7F6E\u9875\u9762"),

        heading2("4.2 \u914D\u7F6E\u53C2\u6570\u8BF4\u660E"),
        para("\u5728\u5F39\u51FA\u7684\u914D\u7F6E\u5BF9\u8BDD\u6846\u4E2D\uFF0C\u586B\u5199\u4EE5\u4E0B\u4FE1\u606F\uFF1A"),

        makeTable(
          ["\u5B57\u6BB5", "\u8BF4\u660E", "\u793A\u4F8B\u503C"],
          [
            ["\u540D\u79F0 *", "Embedding \u914D\u7F6E\u540D\u79F0", "DashScope Embedding"],
            ["\u63CF\u8FF0", "\u914D\u7F6E\u7528\u9014\u8BF4\u660E", "\u963F\u91CC\u4E91\u6587\u672C\u5411\u91CF\u5316\u6A21\u578B"],
            ["\u7C7B\u578B *", "\u9009\u62E9 OpenAI \u517C\u5BB9\u683C\u5F0F", "OpenAI - OpenAI Embeddings API"],
            ["\u6700\u5927\u6279\u5904\u7406\u5927\u5C0F", "\u6BCF\u6279\u6700\u5927\u6587\u672C\u6570", "100"],
            ["Base URL", "Embedding API \u5730\u5740", "https://dashscope.aliyuncs.com/compatible-mode/v1"],
            ["API Key *", "API \u5BC6\u94A5", "sk-xxx..."],
            ["Model *", "\u6A21\u578B\u540D\u79F0", "text-embedding-v4"],
            ["Dimensions *", "\u5411\u91CF\u7EF4\u5EA6", "2048"],
          ],
          [2200, 4160, 3000]
        ),

        img("06-add-embedding.png", 560, 420),
        caption("\u56FE 4-2\uFF1A\u6DFB\u52A0 Embedding \u914D\u7F6E\u5BF9\u8BDD\u6846"),

        para("\u52FE\u9009\u300C\u8BBE\u4E3A\u9ED8\u8BA4\u914D\u7F6E\u300D\u540E\uFF0C\u65B0\u521B\u5EFA\u7684\u77E5\u8BC6\u5E93\u5C06\u81EA\u52A8\u4F7F\u7528\u8BE5 Embedding \u914D\u7F6E\u3002"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 5. 智能体管理 =====
        heading1("5. \u667A\u80FD\u4F53\u7BA1\u7406"),

        heading2("5.1 \u521B\u5EFA\u667A\u80FD\u4F53"),
        para("\u5728\u300C\u9879\u76EE\u5F00\u53D1\u300D\u9875\u9762\u70B9\u51FB\u300C\u521B\u5EFA\u300D\u6309\u94AE\uFF0C\u9009\u62E9\u300C\u667A\u80FD\u4F53\u300D\u7C7B\u578B\u5373\u53EF\u521B\u5EFA\u65B0\u7684\u667A\u80FD\u4F53\u9879\u76EE\u3002"),

        heading2("5.2 \u667A\u80FD\u4F53\u7F16\u8F91\u9875\u9762"),
        para("\u667A\u80FD\u4F53\u7F16\u8F91\u9875\u5206\u4E3A\u4E09\u4E2A\u533A\u57DF\uFF1A"),
        bulletItem("\u5DE6\u4FA7\uFF1A\u4EBA\u8BBE\u4E0E\u56DE\u590D\u903B\u8F91\u3001\u63D0\u793A\u8BCD\u6A21\u677F\u9009\u62E9\uFF08\u901A\u7528\u7ED3\u6784\u3001\u4EFB\u52A1\u6267\u884C\u3001\u89D2\u8272\u626E\u6F14\uFF09"),
        bulletItem("\u4E2D\u95F4\uFF1A\u80FD\u529B\u914D\u7F6E\uFF08\u6280\u80FD\u3001\u63D2\u4EF6\u5546\u5E97\u3001\u5DE5\u4F5C\u6D41\u3001\u5361\u7247\u7ED1\u5B9A\u3001\u77E5\u8BC6\u5E93\u3001\u8868\u683C\u3001\u56FE\u7247\uFF09"),
        bulletItem("\u53F3\u4FA7\uFF1A\u9884\u89C8\u4E0E\u8C03\u8BD5\uFF08\u5B9E\u65F6\u5BF9\u8BDD\u6D4B\u8BD5\uFF09"),

        img("07-agent-edit.png", 560, 350),
        caption("\u56FE 5-1\uFF1A\u667A\u80FD\u4F53\u7F16\u8F91\u9875\u9762"),

        heading2("5.3 \u6A21\u578B\u9009\u62E9"),
        para("\u5728\u7F16\u8F91\u9875\u9762\u9876\u90E8\u53EF\u4EE5\u9009\u62E9\u667A\u80FD\u4F53\u4F7F\u7528\u7684\u5927\u8BED\u8A00\u6A21\u578B\u3002\u4EC5\u663E\u793A\u5728\u300C\u6A21\u578B\u7BA1\u7406\u300D\u4E2D\u5DF2\u914D\u7F6E\u5E76\u542F\u7528\u7684\u6A21\u578B\u3002"),

        heading2("5.4 \u77E5\u8BC6\u5E93\u7ED1\u5B9A"),
        para("\u5728\u667A\u80FD\u4F53\u7F16\u8F91\u9875\u7684\u300C\u77E5\u8BC6\u300D\u533A\u57DF\uFF0C\u53EF\u4EE5\u7ED1\u5B9A\u5DF2\u521B\u5EFA\u7684\u77E5\u8BC6\u5E93\u3002\u667A\u80FD\u4F53\u5728\u56DE\u7B54\u7528\u6237\u95EE\u9898\u65F6\uFF0C\u4F1A\u81EA\u52A8\u68C0\u7D22\u77E5\u8BC6\u5E93\u4E2D\u7684\u76F8\u5173\u5185\u5BB9\u4F5C\u4E3A\u53C2\u8003\u3002"),
        para("\u652F\u6301\u4E24\u79CD\u8C03\u7528\u65B9\u5F0F\uFF1A"),
        bulletItem("\u6309\u9700\u8C03\u7528\uFF1A\u667A\u80FD\u4F53\u81EA\u884C\u5224\u65AD\u662F\u5426\u9700\u8981\u68C0\u7D22\u77E5\u8BC6\u5E93"),
        bulletItem("\u5F3A\u5236\u8C03\u7528\uFF1A\u6BCF\u6B21\u5BF9\u8BDD\u90FD\u4F1A\u68C0\u7D22\u77E5\u8BC6\u5E93"),

        heading2("5.5 \u53D1\u5E03\u667A\u80FD\u4F53"),
        para("\u7F16\u8F91\u5B8C\u6210\u540E\uFF0C\u70B9\u51FB\u53F3\u4E0A\u89D2\u300C\u53D1\u5E03\u300D\u6309\u94AE\u53EF\u4EE5\u5C06\u667A\u80FD\u4F53\u53D1\u5E03\u4E3A API \u670D\u52A1\u6216 Web \u5E94\u7528\u3002"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 6. 资源库 =====
        heading1("6. \u8D44\u6E90\u5E93"),

        heading2("6.1 \u77E5\u8BC6\u5E93"),
        para("\u77E5\u8BC6\u5E93\u7528\u4E8E\u5B58\u50A8\u548C\u7BA1\u7406\u667A\u80FD\u4F53\u7684\u53C2\u8003\u77E5\u8BC6\u3002\u652F\u6301\u4E0A\u4F20\u6587\u672C\u3001\u8868\u683C\u3001\u56FE\u7247\u7B49\u591A\u79CD\u683C\u5F0F\u7684\u6587\u6863\u3002"),
        para("\u521B\u5EFA\u77E5\u8BC6\u5E93\u7684\u524D\u63D0\u662F\u5DF2\u914D\u7F6E\u597D Embedding \u6A21\u578B\uFF08\u53C2\u89C1\u7B2C 4 \u7AE0\uFF09\u3002"),
        img("08-knowledge.png", 560, 350),
        caption("\u56FE 6-1\uFF1A\u77E5\u8BC6\u5E93\u9875\u9762"),

        heading2("6.2 \u5DE5\u4F5C\u6D41"),
        para("\u5DE5\u4F5C\u6D41\u662F\u4E00\u79CD\u53EF\u89C6\u5316\u7684\u4EFB\u52A1\u7F16\u6392\u5DE5\u5177\uFF0C\u652F\u6301\u901A\u8FC7\u62D6\u62FD\u8282\u70B9\u7684\u65B9\u5F0F\u6784\u5EFA\u590D\u6742\u7684 AI \u5904\u7406\u6D41\u7A0B\u3002"),
        para("\u652F\u6301\u7684\u8282\u70B9\u7C7B\u578B\u5305\u62EC\uFF1A"),
        bulletItem("\u5F00\u59CB\u8282\u70B9\uFF1A\u5B9A\u4E49\u5DE5\u4F5C\u6D41\u8F93\u5165\u53C2\u6570"),
        bulletItem("LLM \u8282\u70B9\uFF1A\u8C03\u7528\u5927\u8BED\u8A00\u6A21\u578B\u8FDB\u884C\u6587\u672C\u751F\u6210"),
        bulletItem("\u77E5\u8BC6\u5E93\u8282\u70B9\uFF1A\u68C0\u7D22\u77E5\u8BC6\u5E93\u5185\u5BB9"),
        bulletItem("\u4EE3\u7801\u8282\u70B9\uFF1A\u6267\u884C\u81EA\u5B9A\u4E49\u4EE3\u7801\u903B\u8F91"),
        bulletItem("\u6761\u4EF6\u8282\u70B9\uFF1A\u6839\u636E\u6761\u4EF6\u5206\u652F\u6267\u884C"),
        bulletItem("\u7ED3\u675F\u8282\u70B9\uFF1A\u5B9A\u4E49\u5DE5\u4F5C\u6D41\u8F93\u51FA"),
        img("09-workflow-list.png", 560, 350),
        caption("\u56FE 6-2\uFF1A\u5DE5\u4F5C\u6D41\u5217\u8868\u9875\u9762"),

        heading2("6.3 \u5176\u4ED6\u8D44\u6E90"),
        para("\u5E73\u53F0\u8FD8\u63D0\u4F9B\u4EE5\u4E0B\u8D44\u6E90\u7C7B\u578B\uFF1A"),
        bulletItem("\u5361\u7247\uFF1A\u53EF\u590D\u7528\u7684\u5BF9\u8BDD\u754C\u9762\u7EC4\u4EF6"),
        bulletItem("\u63D2\u4EF6\uFF1A\u6269\u5C55\u667A\u80FD\u4F53\u80FD\u529B\u7684\u5DE5\u5177\u96C6"),
        bulletItem("\u6280\u80FD\uFF1A\u9884\u5B9A\u4E49\u7684\u667A\u80FD\u4F53\u80FD\u529B\u5305"),
        bulletItem("\u63D0\u793A\u8BCD\uFF1A\u53EF\u590D\u7528\u7684\u63D0\u793A\u8BCD\u6A21\u677F"),
        bulletItem("\u6570\u636E\u5E93\uFF1A\u7ED3\u6784\u5316\u6570\u636E\u5B58\u50A8\uFF0C\u652F\u6301\u667A\u80FD\u4F53\u8FDB\u884C\u6570\u636E\u67E5\u8BE2\u548C\u8BA1\u7B97"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 7. Guard 安全围栏 =====
        heading1("7. \u5B89\u5168\u56F4\u680F (Guard)"),
        para("\u5B89\u5168\u56F4\u680F\u662F\u5E73\u53F0\u7684 AI \u5185\u5BB9\u5B89\u5168\u9632\u62A4\u7CFB\u7EDF\uFF0C\u63D0\u4F9B\u6587\u672C\u5185\u5BB9\u7684\u5B89\u5168\u68C0\u6D4B\u3001\u5173\u952E\u8BCD\u8FC7\u6EE4\u3001AI \u6A21\u578B\u8BED\u4E49\u5BA1\u6838\u7B49\u80FD\u529B\u3002"),

        heading2("7.1 \u767B\u5F55 Guard"),
        para("\u8BBF\u95EE Guard \u5730\u5740 http://30.3.165.211:8080\uFF0C\u4F7F\u7528\u4EE5\u4E0B\u9ED8\u8BA4\u8D26\u53F7\u767B\u5F55\uFF1A"),
        makeTable(
          ["\u5B57\u6BB5", "\u503C"],
          [
            ["\u79DF\u6237 ID", "default"],
            ["\u7528\u6237\u540D", "admin"],
            ["\u5BC6\u7801", "admin123"],
          ],
          [3000, 6360]
        ),
        img("10-guard-login.png", 480, 360),
        caption("\u56FE 7-1\uFF1AGuard \u767B\u5F55\u9875\u9762"),

        heading2("7.2 \u4EEA\u8868\u76D8"),
        para("\u767B\u5F55\u540E\u8FDB\u5165\u4EEA\u8868\u76D8\uFF0C\u53EF\u67E5\u770B\uFF1A"),
        bulletItem("\u603B\u8BF7\u6C42\u6570\u3001\u901A\u8FC7\u8BF7\u6C42\u3001\u62E6\u622A\u8BF7\u6C42\u3001\u901A\u8FC7\u7387"),
        bulletItem("\u68C0\u6D4B\u6765\u6E90\u5206\u5E03"),
        bulletItem("\u7CFB\u7EDF\u72B6\u6001\uFF08API \u7F51\u5173\u3001AI \u5B89\u5168\u6A21\u578B\u3001\u77E5\u8BC6\u5E93\u3001\u5BA1\u8BA1\u65E5\u5FD7\uFF09"),
        img("11-guard-dashboard.png", 560, 350),
        caption("\u56FE 7-2\uFF1AGuard \u4EEA\u8868\u76D8"),

        heading2("7.3 \u4EA7\u54C1\u7BA1\u7406"),
        para("\u4EA7\u54C1\u662F Guard \u7684\u6838\u5FC3\u6982\u5FF5\uFF0C\u6BCF\u4E2A\u4EA7\u54C1\u53EF\u4EE5\u5305\u542B\u591A\u4E2A\u4E1A\u52A1\uFF0C\u6BCF\u4E2A\u4E1A\u52A1\u53EF\u4EE5\u914D\u7F6E\u72EC\u7ACB\u7684\u68C0\u6D4B\u7B56\u7565\u3002"),
        para("\u64CD\u4F5C\u6B65\u9AA4\uFF1A"),
        numberedItem("\u70B9\u51FB\u5DE6\u4FA7\u300C\u4EA7\u54C1\u7BA1\u7406\u300D\u83DC\u5355", "numbers2"),
        numberedItem("\u70B9\u51FB\u300C+ \u521B\u5EFA\u4EA7\u54C1\u300D\u6309\u94AE", "numbers2"),
        numberedItem("\u586B\u5199\u4EA7\u54C1\u540D\u79F0\u548C\u63CF\u8FF0", "numbers2"),
        numberedItem("\u6DFB\u52A0\u4E1A\u52A1\u5E76\u914D\u7F6E\u68C0\u6D4B\u89C4\u5219", "numbers2"),
        img("13-guard-products.png", 560, 350),
        caption("\u56FE 7-3\uFF1AGuard \u4EA7\u54C1\u7BA1\u7406"),

        heading2("7.4 \u5728\u7EBF\u6D4B\u8BD5"),
        para("\u300C\u5728\u7EBF\u6D4B\u8BD5\u300D\u529F\u80FD\u53EF\u4EE5\u5FEB\u901F\u9A8C\u8BC1\u5B89\u5168\u68C0\u6D4B\u6548\u679C\uFF1A"),
        numberedItem("\u9009\u62E9\u4EA7\u54C1\u548C\u4E1A\u52A1", "numbers3"),
        numberedItem("\u5728\u300C\u68C0\u6D4B\u5185\u5BB9\u300D\u6846\u4E2D\u8F93\u5165\u6D4B\u8BD5\u6587\u672C", "numbers3"),
        numberedItem("\u70B9\u51FB\u300C\u5F00\u59CB\u68C0\u6D4B\u300D\u67E5\u770B\u7ED3\u679C", "numbers3"),
        para("\u5E73\u53F0\u8FD8\u63D0\u4F9B\u300C\u5FEB\u901F\u6D4B\u8BD5\u6837\u672C\u300D\u6309\u94AE\uFF0C\u53EF\u4E00\u952E\u586B\u5145\u6D4B\u8BD5\u5185\u5BB9\uFF08\u6B63\u5E38\u5185\u5BB9\u3001\u5E7F\u544A\u6D4B\u8BD5\u3001\u8054\u7CFB\u65B9\u5F0F\u3001\u654F\u611F\u8BCD\u6D4B\u8BD5\uFF09\u3002"),
        img("12-guard-test.png", 560, 350),
        caption("\u56FE 7-4\uFF1AGuard \u5728\u7EBF\u6D4B\u8BD5"),

        heading2("7.5 \u68C0\u6D4B\u7B56\u7565"),
        para("Guard \u91C7\u7528\u591A\u5C42\u6B21\u68C0\u6D4B\u7B56\u7565\uFF0C\u4F18\u5148\u7EA7\u4ECE\u9AD8\u5230\u4F4E\uFF1A"),

        makeTable(
          ["\u5C42\u7EA7", "\u68C0\u6D4B\u65B9\u5F0F", "\u54CD\u5E94\u65F6\u95F4", "\u8BF4\u660E"],
          [
            ["1", "\u672C\u5730\u5173\u952E\u8BCD\u5339\u914D", "<1ms", "\u7CBE\u786E\u5339\u914D\u9884\u5B9A\u4E49\u7684\u5371\u9669\u5173\u952E\u8BCD"],
            ["2", "AI \u6A21\u578B\u8BED\u4E49\u68C0\u6D4B", "100-200ms", "\u8C03\u7528 AI \u6A21\u578B\u8FDB\u884C\u8BED\u4E49\u7EA7\u522B\u5B89\u5168\u5224\u65AD"],
            ["3", "LRU \u7F13\u5B58", "0ms", "\u76F8\u540C\u5185\u5BB9 5 \u5206\u949F\u5185\u547D\u4E2D\u7F13\u5B58\u76F4\u63A5\u8FD4\u56DE"],
            ["4", "\u7194\u65AD\u673A\u5236", "-", "API \u8FDE\u7EED\u5931\u8D25 3 \u6B21\u540E\u6682\u505C 60 \u79D2"],
          ],
          [1000, 2500, 1860, 4000]
        ),

        heading2("7.6 API \u5BC6\u94A5\u7BA1\u7406"),
        para("\u5728\u300CAPI \u5BC6\u94A5\u300D\u9875\u9762\u53EF\u4EE5\u7BA1\u7406\u7528\u4E8E\u8C03\u7528 Guard \u63A5\u53E3\u7684 API Key\u3002"),
        img("14-guard-apikeys.png", 560, 350),
        caption("\u56FE 7-5\uFF1AAPI \u5BC6\u94A5\u7BA1\u7406"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 8. 可观测性 =====
        heading1("8. \u53EF\u89C2\u6D4B\u6027 (Loop)"),
        para("\u53EF\u89C2\u6D4B\u6027\u5E73\u53F0\u63D0\u4F9B\u667A\u80FD\u4F53\u8FD0\u884C\u7684\u5168\u94FE\u8DEF\u8FFD\u8E2A\u548C\u76D1\u63A7\u80FD\u529B\u3002Loop \u5DF2\u5D4C\u5165\u5230 Studio \u5E73\u53F0\u4E2D\uFF0C\u65E0\u9700\u5355\u72EC\u8BBF\u95EE\u3002\u5728\u5DE5\u4F5C\u7A7A\u95F4\u5DE6\u4FA7\u5BFC\u822A\u680F\u70B9\u51FB\u300C\u53EF\u89C2\u6D4B\u6027\u300D\u5373\u53EF\u76F4\u63A5\u4F7F\u7528\u3002"),
        para("\u4E3B\u8981\u529F\u80FD\uFF1A"),
        bulletItem("Trace \u8FFD\u8E2A\uFF1A\u67E5\u770B\u6BCF\u6B21\u667A\u80FD\u4F53\u5BF9\u8BDD\u7684\u5B8C\u6574\u8C03\u7528\u94FE\u8DEF"),
        bulletItem("\u6027\u80FD\u76D1\u63A7\uFF1A\u54CD\u5E94\u65F6\u95F4\u3001Token \u6D88\u8017\u7B49\u6307\u6807"),
        bulletItem("\u9519\u8BEF\u65E5\u5FD7\uFF1A\u67E5\u770B\u548C\u6392\u67E5\u8FD0\u884C\u5F02\u5E38"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 9. 快速开始 =====
        heading1("9. \u5FEB\u901F\u5F00\u59CB\u6307\u5357"),
        para("\u4EE5\u4E0B\u662F\u4ECE\u96F6\u5F00\u59CB\u914D\u7F6E\u5E73\u53F0\u7684\u63A8\u8350\u6B65\u9AA4\uFF1A"),

        numberedItem("\u914D\u7F6E\u6A21\u578B\uFF1A\u8FDB\u5165\u300C\u6A21\u578B\u7BA1\u7406\u300D\u2192\u300C\u6DFB\u52A0\u6A21\u578B\u300D\uFF0C\u914D\u7F6E\u81F3\u5C11\u4E00\u4E2A\u6587\u672C\u751F\u6210\u6A21\u578B", "numbers4"),
        numberedItem("\u914D\u7F6E Embedding\uFF1A\u8FDB\u5165\u300CEmbedding\u300D\u2192\u300C\u6DFB\u52A0\u914D\u7F6E\u300D\uFF0C\u914D\u7F6E\u5411\u91CF\u5316\u6A21\u578B\uFF08\u53EF\u9009\uFF0C\u77E5\u8BC6\u5E93\u529F\u80FD\u9700\u8981\uFF09", "numbers4"),
        numberedItem("\u521B\u5EFA\u667A\u80FD\u4F53\uFF1A\u8FDB\u5165\u300C\u9879\u76EE\u5F00\u53D1\u300D\u2192\u300C\u521B\u5EFA\u300D\uFF0C\u9009\u62E9\u667A\u80FD\u4F53\u7C7B\u578B", "numbers4"),
        numberedItem("\u7F16\u5199\u63D0\u793A\u8BCD\uFF1A\u5728\u300C\u4EBA\u8BBE\u4E0E\u56DE\u590D\u903B\u8F91\u300D\u4E2D\u7F16\u5199\u667A\u80FD\u4F53\u7684\u7CFB\u7EDF\u63D0\u793A\u8BCD", "numbers4"),
        numberedItem("\u7ED1\u5B9A\u77E5\u8BC6\u5E93\uFF1A\u5728\u300C\u77E5\u8BC6\u300D\u533A\u57DF\u7ED1\u5B9A\u76F8\u5173\u77E5\u8BC6\u5E93\uFF08\u53EF\u9009\uFF09", "numbers4"),
        numberedItem("\u6D4B\u8BD5\u5BF9\u8BDD\uFF1A\u5728\u53F3\u4FA7\u300C\u9884\u89C8\u4E0E\u8C03\u8BD5\u300D\u533A\u57DF\u8FDB\u884C\u5BF9\u8BDD\u6D4B\u8BD5", "numbers4"),
        numberedItem("\u53D1\u5E03\u4E0A\u7EBF\uFF1A\u70B9\u51FB\u300C\u53D1\u5E03\u300D\u6309\u94AE\u5C06\u667A\u80FD\u4F53\u53D1\u5E03\u4E3A\u670D\u52A1", "numbers4"),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== 10. 常见问题 =====
        heading1("10. \u5E38\u89C1\u95EE\u9898"),

        heading3("Q1: \u6DFB\u52A0\u6A21\u578B\u65F6\u63D0\u793A\u8FDE\u63A5\u5931\u8D25\uFF1F"),
        para("A: \u8BF7\u68C0\u67E5\u6A21\u578B API \u5730\u5740\u662F\u5426\u53EF\u8FBE\uFF0CAPI Key \u662F\u5426\u6B63\u786E\u3002\u5185\u7F51\u73AF\u5883\u9700\u786E\u4FDD\u670D\u52A1\u5668\u53EF\u4EE5\u8BBF\u95EE\u6A21\u578B API \u5730\u5740\u3002"),

        heading3("Q2: \u77E5\u8BC6\u5E93\u4E0A\u4F20\u6587\u6863\u540E\u641C\u7D22\u4E0D\u5230\u5185\u5BB9\uFF1F"),
        para("A: \u8BF7\u786E\u8BA4 Embedding \u914D\u7F6E\u662F\u5426\u6B63\u786E\u3002\u6587\u6863\u4E0A\u4F20\u540E\u9700\u8981\u7B49\u5F85\u5411\u91CF\u5316\u5904\u7406\u5B8C\u6210\uFF0C\u53EF\u5728\u77E5\u8BC6\u5E93\u8BE6\u60C5\u9875\u67E5\u770B\u5904\u7406\u72B6\u6001\u3002"),

        heading3("Q3: Guard \u68C0\u6D4B\u7ED3\u679C\u4E0D\u51C6\u786E\uFF1F"),
        para("A: \u53EF\u4EE5\u901A\u8FC7\u4EE5\u4E0B\u65B9\u5F0F\u4F18\u5316\uFF1A\u5728\u300C\u77E5\u8BC6\u5E93\u300D\u4E2D\u6DFB\u52A0\u66F4\u591A\u5B89\u5168\u89C4\u5219\uFF1B\u5728\u300C\u540D\u5355\u5E93\u300D\u4E2D\u7EF4\u62A4\u5173\u952E\u8BCD\u9ED1\u767D\u540D\u5355\uFF1B\u8C03\u6574\u4EA7\u54C1\u4E1A\u52A1\u7684\u68C0\u6D4B\u7B56\u7565\u3002"),

        heading3("Q4: \u5982\u4F55\u67E5\u770B\u667A\u80FD\u4F53\u7684\u8FD0\u884C\u65E5\u5FD7\uFF1F"),
        para("A: \u5728\u5DE5\u4F5C\u7A7A\u95F4\u5DE6\u4FA7\u70B9\u51FB\u300C\u53EF\u89C2\u6D4B\u6027\u300D\u8FDB\u5165 Loop \u5E73\u53F0\uFF0C\u53EF\u67E5\u770B\u8BE6\u7EC6\u7684 Trace \u8FFD\u8E2A\u548C\u6027\u80FD\u6307\u6807\u3002"),
      ]
    }
  ]
});

// ============ GENERATE ============
const outputPath = path.join(__dirname, "\u730E\u9E70\u667A\u80FD\u4F53\u5E73\u53F0-\u4F7F\u7528\u8BF4\u660E\u4E66.docx");
Packer.toBuffer(doc).then(buffer => {
  fs.writeFileSync(outputPath, buffer);
  console.log(`Generated: ${outputPath} (${(buffer.length / 1024).toFixed(0)} KB)`);
}).catch(err => {
  console.error("Error:", err);
  process.exit(1);
});
