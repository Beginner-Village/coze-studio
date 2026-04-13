const fs = require("fs");
const sharp = require("sharp");
const {
  Document, Packer, Paragraph, TextRun, ImageRun, Table, TableRow, TableCell,
  Header, Footer, AlignmentType, HeadingLevel, BorderStyle, WidthType,
  ShadingType, PageNumber, PageBreak, LevelFormat
} = require("docx");

// ============ SVG Diagrams ============

// Diagram 1: ES current architecture - who writes, who reads
const svg1_esArchitecture = `
<svg xmlns="http://www.w3.org/2000/svg" width="900" height="620" viewBox="0 0 900 620">
  <defs>
    <filter id="shadow" x="-5%" y="-5%" width="115%" height="115%">
      <feDropShadow dx="2" dy="2" stdDeviation="3" flood-opacity="0.15"/>
    </filter>
    <linearGradient id="esGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#FED841;stop-opacity:1"/>
      <stop offset="100%" style="stop-color:#F5A623;stop-opacity:1"/>
    </linearGradient>
    <linearGradient id="writeGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#4ECDC4;stop-opacity:1"/>
      <stop offset="100%" style="stop-color:#2CB5AC;stop-opacity:1"/>
    </linearGradient>
    <linearGradient id="readGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#6C9FFF;stop-opacity:1"/>
      <stop offset="100%" style="stop-color:#4A7FE5;stop-opacity:1"/>
    </linearGradient>
    <linearGradient id="indexGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#FF8A65;stop-opacity:1"/>
      <stop offset="100%" style="stop-color:#E64A19;stop-opacity:1"/>
    </linearGradient>
  </defs>

  <!-- Background -->
  <rect width="900" height="620" rx="16" fill="#F8F9FB"/>

  <!-- Title -->
  <text x="450" y="40" font-family="Arial,sans-serif" font-size="20" font-weight="bold" text-anchor="middle" fill="#1a1a2e">Coze Studio - Elasticsearch 使用全景图</text>

  <!-- Left: Write modules -->
  <text x="130" y="80" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#2CB5AC">写入方（通过 EventBus）</text>

  <rect x="30" y="100" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="125" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Agent / Bot</text>

  <rect x="30" y="155" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="180" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Workflow 工作流</text>

  <rect x="30" y="210" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="235" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Plugin 插件</text>

  <rect x="30" y="265" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="290" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Knowledge 知识库</text>

  <rect x="30" y="320" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="345" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Prompt 提示词</text>

  <rect x="30" y="375" width="200" height="40" rx="8" fill="url(#writeGrad)" filter="url(#shadow)"/>
  <text x="130" y="400" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Database 数据库(记忆)</text>

  <!-- Center: ES -->
  <rect x="320" y="130" width="260" height="300" rx="16" fill="url(#esGrad)" filter="url(#shadow)" stroke="#E09400" stroke-width="2"/>
  <text x="450" y="165" font-family="Arial,sans-serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#5D3A00">Elasticsearch</text>

  <!-- ES Indexes -->
  <rect x="340" y="185" width="220" height="55" rx="8" fill="url(#indexGrad)" opacity="0.9"/>
  <text x="450" y="206" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="white">project_draft</text>
  <text x="450" y="228" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#FFE0B2">Agent/Bot 列表、搜索、排序</text>

  <rect x="340" y="250" width="220" height="55" rx="8" fill="url(#indexGrad)" opacity="0.9"/>
  <text x="450" y="271" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="white">coze_resource</text>
  <text x="450" y="293" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#FFE0B2">工作流/插件/知识库/提示词/数据库</text>

  <rect x="340" y="315" width="220" height="55" rx="8" fill="url(#indexGrad)" opacity="0.9"/>
  <text x="450" y="336" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="white">opencoze_{id}</text>
  <text x="450" y="358" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#FFE0B2">知识库文档全文检索</text>

  <text x="450" y="410" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#5D3A00" font-style="italic">所有数据主本在 MySQL，ES 是读模型</text>

  <!-- Right: Read features -->
  <text x="770" y="80" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#4A7FE5">读取方（前端功能）</text>

  <rect x="670" y="100" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="125" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">空间 Agent 列表</text>

  <rect x="670" y="155" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="180" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">资源库列表</text>

  <rect x="670" y="210" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="235" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">Agent 资源面板</text>

  <rect x="670" y="265" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="290" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">搜索/筛选/排序</text>

  <rect x="670" y="320" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="345" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">最近编辑 / 收藏</text>

  <rect x="670" y="375" width="200" height="40" rx="8" fill="url(#readGrad)" filter="url(#shadow)"/>
  <text x="770" y="400" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="white" font-weight="bold">知识库 RAG 检索</text>

  <!-- Arrows: Write -->
  <defs><marker id="arrowR" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto"><path d="M0,0 L8,3 L0,6" fill="#2CB5AC"/></marker></defs>
  <defs><marker id="arrowB" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto"><path d="M0,0 L8,3 L0,6" fill="#4A7FE5"/></marker></defs>

  <line x1="230" y1="120" x2="315" y2="210" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>
  <line x1="230" y1="175" x2="315" y2="275" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>
  <line x1="230" y1="230" x2="315" y2="275" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>
  <line x1="230" y1="285" x2="315" y2="280" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>
  <line x1="230" y1="340" x2="315" y2="275" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>
  <line x1="230" y1="395" x2="315" y2="275" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>

  <!-- Arrow from Agent to project_draft -->
  <line x1="230" y1="120" x2="315" y2="210" stroke="#2CB5AC" stroke-width="2" marker-end="url(#arrowR)" stroke-dasharray="6,3"/>

  <!-- Arrows: Read -->
  <line x1="585" y1="210" x2="665" y2="120" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>
  <line x1="585" y1="275" x2="665" y2="175" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>
  <line x1="585" y1="275" x2="665" y2="230" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>
  <line x1="585" y1="275" x2="665" y2="290" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>
  <line x1="585" y1="210" x2="665" y2="345" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>
  <line x1="585" y1="340" x2="665" y2="395" stroke="#4A7FE5" stroke-width="2" marker-end="url(#arrowB)"/>

  <!-- Bottom: Impact summary -->
  <rect x="100" y="460" width="700" height="140" rx="12" fill="#FFF3E0" stroke="#FF9800" stroke-width="2"/>
  <text x="450" y="490" font-family="Arial,sans-serif" font-size="15" font-weight="bold" text-anchor="middle" fill="#E65100">⚠ ES 不可用时的影响</text>
  <text x="450" y="520" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="#BF360C">空间内看不到任何 Agent / Bot（列表为空）</text>
  <text x="450" y="545" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="#BF360C">资源库看不到工作流、插件、知识库、提示词、数据库</text>
  <text x="450" y="570" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="#BF360C">搜索、筛选、收藏、最近编辑全部失效</text>
  <text x="450" y="595" font-family="Arial,sans-serif" font-size="13" text-anchor="middle" fill="#BF360C">知识库 RAG 全文检索不可用（纯向量检索不受影响）</text>
</svg>`;

// Diagram 2: Deployment options
const svg2_deploymentOptions = `
<svg xmlns="http://www.w3.org/2000/svg" width="900" height="750" viewBox="0 0 900 750">
  <defs>
    <filter id="s2" x="-5%" y="-5%" width="115%" height="115%">
      <feDropShadow dx="2" dy="2" stdDeviation="3" flood-opacity="0.12"/>
    </filter>
    <linearGradient id="centerA" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#E3F2FD"/>
      <stop offset="100%" style="stop-color:#BBDEFB"/>
    </linearGradient>
    <linearGradient id="centerB" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#F3E5F5"/>
      <stop offset="100%" style="stop-color:#E1BEE7"/>
    </linearGradient>
    <linearGradient id="esBox" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#FFF8E1"/>
      <stop offset="100%" style="stop-color:#FFECB3"/>
    </linearGradient>
  </defs>

  <rect width="900" height="750" rx="16" fill="#F8F9FB"/>

  <!-- ===== Option 1 ===== -->
  <text x="450" y="35" font-family="Arial,sans-serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#1a1a2e">方案一：跨中心单集群</text>
  <text x="450" y="55" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#666">一个 ES 集群，节点分布在两个中心</text>

  <!-- Center A -->
  <rect x="30" y="70" width="380" height="140" rx="12" fill="url(#centerA)" stroke="#1976D2" stroke-width="2" filter="url(#s2)"/>
  <text x="220" y="95" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#1565C0">中心 A</text>
  <rect x="50" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="100" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Master</text>
  <rect x="165" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="215" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Data</text>
  <rect x="280" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="330" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Data</text>
  <rect x="120" y="155" width="180" height="30" rx="6" fill="#E3F2FD" stroke="#1976D2" stroke-width="1"/>
  <text x="210" y="175" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#1565C0">App Server (R/W)</text>

  <!-- Center B -->
  <rect x="490" y="70" width="380" height="140" rx="12" fill="url(#centerB)" stroke="#7B1FA2" stroke-width="2" filter="url(#s2)"/>
  <text x="680" y="95" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#6A1B9A">中心 B</text>
  <rect x="510" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="560" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Master</text>
  <rect x="625" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="675" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Data</text>
  <rect x="740" y="105" width="100" height="35" rx="6" fill="#FFF8E1" stroke="#F9A825" stroke-width="1.5"/>
  <text x="790" y="128" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">ES Data</text>
  <rect x="580" y="155" width="180" height="30" rx="6" fill="#F3E5F5" stroke="#7B1FA2" stroke-width="1"/>
  <text x="670" y="175" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#6A1B9A">App Server (R/W)</text>

  <!-- Connection -->
  <line x1="410" y1="125" x2="490" y2="125" stroke="#FF6F00" stroke-width="3" stroke-dasharray="8,4"/>
  <text x="450" y="118" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#FF6F00" font-weight="bold">专线</text>

  <!-- Impact box -->
  <rect x="100" y="220" width="320" height="35" rx="6" fill="#FFCDD2" stroke="#D32F2F" stroke-width="1.5"/>
  <text x="260" y="242" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#B71C1C">A 挂：选举成功→30s恢复；失败→全挂</text>
  <rect x="480" y="220" width="320" height="35" rx="6" fill="#C8E6C9" stroke="#388E3C" stroke-width="1.5"/>
  <text x="640" y="242" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#1B5E20">B 挂：同上，取决于选举</text>

  <!-- ===== Option 3 (recommended) ===== -->
  <text x="450" y="295" font-family="Arial,sans-serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#1a1a2e">方案三：独立集群 + EventBus 双消费（推荐）</text>
  <text x="450" y="315" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#666">每个中心独立 ES 集群，各自从 EventBus 消费写入</text>

  <!-- Center A -->
  <rect x="30" y="330" width="380" height="190" rx="12" fill="url(#centerA)" stroke="#1976D2" stroke-width="2" filter="url(#s2)"/>
  <text x="220" y="355" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#1565C0">中心 A</text>
  <rect x="120" y="365" width="180" height="30" rx="6" fill="#E3F2FD" stroke="#1976D2" stroke-width="1"/>
  <text x="210" y="385" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#1565C0">App Server (R/W)</text>
  <rect x="60" y="410" width="140" height="30" rx="6" fill="#E8F5E9" stroke="#388E3C" stroke-width="1.5"/>
  <text x="130" y="430" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#2E7D32">EventBus Consumer</text>
  <rect x="220" y="410" width="160" height="50" rx="8" fill="#FFF8E1" stroke="#F9A825" stroke-width="2"/>
  <text x="300" y="433" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="#E65100">ES Cluster A</text>
  <text x="300" y="450" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#BF360C">（完整独立集群）</text>
  <rect x="100" y="475" width="220" height="30" rx="6" fill="#FFF3E0" stroke="#FF9800" stroke-width="1"/>
  <text x="210" y="495" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">MySQL / OceanBase（数据主本）</text>

  <!-- Arrow from consumer to ES A -->
  <line x1="200" y1="425" x2="218" y2="430" stroke="#388E3C" stroke-width="2" marker-end="url(#arrowR)"/>

  <!-- Center B -->
  <rect x="490" y="330" width="380" height="190" rx="12" fill="url(#centerB)" stroke="#7B1FA2" stroke-width="2" filter="url(#s2)"/>
  <text x="680" y="355" font-family="Arial,sans-serif" font-size="14" font-weight="bold" text-anchor="middle" fill="#6A1B9A">中心 B</text>
  <rect x="580" y="365" width="180" height="30" rx="6" fill="#F3E5F5" stroke="#7B1FA2" stroke-width="1"/>
  <text x="670" y="385" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#6A1B9A">App Server (R/W)</text>
  <rect x="520" y="410" width="140" height="30" rx="6" fill="#E8F5E9" stroke="#388E3C" stroke-width="1.5"/>
  <text x="590" y="430" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#2E7D32">EventBus Consumer</text>
  <rect x="680" y="410" width="160" height="50" rx="8" fill="#FFF8E1" stroke="#F9A825" stroke-width="2"/>
  <text x="760" y="433" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="#E65100">ES Cluster B</text>
  <text x="760" y="450" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#BF360C">（完整独立集群）</text>
  <rect x="560" y="475" width="220" height="30" rx="6" fill="#FFF3E0" stroke="#FF9800" stroke-width="1"/>
  <text x="670" y="495" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#E65100">MySQL / OceanBase（数据主本）</text>

  <!-- EventBus connection -->
  <rect x="370" y="400" width="160" height="45" rx="8" fill="#E8F5E9" stroke="#2E7D32" stroke-width="2" filter="url(#s2)"/>
  <text x="450" y="420" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="#1B5E20">EventBus</text>
  <text x="450" y="436" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#2E7D32">（同一消息双消费）</text>

  <line x1="370" y1="422" x2="200" y2="422" stroke="#2E7D32" stroke-width="2" stroke-dasharray="6,3"/>
  <line x1="530" y1="422" x2="660" y2="422" stroke="#2E7D32" stroke-width="2" stroke-dasharray="6,3"/>

  <!-- Impact box -->
  <rect x="100" y="535" width="320" height="35" rx="6" fill="#C8E6C9" stroke="#388E3C" stroke-width="1.5"/>
  <text x="260" y="557" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#1B5E20" font-weight="bold">A 挂：B 完全不受影响 ✓</text>
  <rect x="480" y="535" width="320" height="35" rx="6" fill="#C8E6C9" stroke="#388E3C" stroke-width="1.5"/>
  <text x="640" y="557" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#1B5E20" font-weight="bold">B 挂：A 完全不受影响 ✓</text>

  <!-- ===== Option 4 ===== -->
  <text x="450" y="610" font-family="Arial,sans-serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#1a1a2e">方案四：去 ES，搜索直接走 OceanBase</text>
  <text x="450" y="630" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#666">利用 OB 原生双活能力，减少有状态组件</text>

  <!-- Simplified diagram -->
  <rect x="130" y="645" width="260" height="50" rx="10" fill="url(#centerA)" stroke="#1976D2" stroke-width="2" filter="url(#s2)"/>
  <text x="260" y="675" font-family="Arial,sans-serif" font-size="13" font-weight="bold" text-anchor="middle" fill="#1565C0">中心 A：App + OB主</text>

  <rect x="510" y="645" width="260" height="50" rx="10" fill="url(#centerB)" stroke="#7B1FA2" stroke-width="2" filter="url(#s2)"/>
  <text x="640" y="675" font-family="Arial,sans-serif" font-size="13" font-weight="bold" text-anchor="middle" fill="#6A1B9A">中心 B：App + OB备</text>

  <line x1="390" y1="670" x2="510" y2="670" stroke="#FF6F00" stroke-width="3" stroke-dasharray="8,4"/>
  <text x="450" y="663" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#FF6F00" font-weight="bold">OB 原生同步</text>

  <rect x="250" y="705" width="400" height="32" rx="6" fill="#E8EAF6" stroke="#3F51B5" stroke-width="1.5"/>
  <text x="450" y="726" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#283593">无 ES 组件，双活能力依托 OB（需改代码）</text>
</svg>`;

// Diagram 3: Impact comparison
const svg3_impactMatrix = `
<svg xmlns="http://www.w3.org/2000/svg" width="900" height="520" viewBox="0 0 900 520">
  <rect width="900" height="520" rx="16" fill="#F8F9FB"/>

  <text x="450" y="35" font-family="Arial,sans-serif" font-size="20" font-weight="bold" text-anchor="middle" fill="#1a1a2e">各方案：单中心故障影响对比</text>

  <!-- Table header -->
  <rect x="30" y="55" width="160" height="45" rx="0" fill="#37474F"/>
  <text x="110" y="83" font-family="Arial,sans-serif" font-size="12" font-weight="bold" text-anchor="middle" fill="white">功能模块</text>

  <rect x="190" y="55" width="170" height="45" rx="0" fill="#37474F"/>
  <text x="275" y="73" font-family="Arial,sans-serif" font-size="11" font-weight="bold" text-anchor="middle" fill="white">方案一</text>
  <text x="275" y="90" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#B0BEC5">跨中心单集群</text>

  <rect x="360" y="55" width="170" height="45" rx="0" fill="#37474F"/>
  <text x="445" y="73" font-family="Arial,sans-serif" font-size="11" font-weight="bold" text-anchor="middle" fill="white">方案二</text>
  <text x="445" y="90" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#B0BEC5">CCR 主从</text>

  <rect x="530" y="55" width="180" height="45" rx="0" fill="#1B5E20"/>
  <text x="620" y="73" font-family="Arial,sans-serif" font-size="11" font-weight="bold" text-anchor="middle" fill="white">方案三（推荐）</text>
  <text x="620" y="90" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#A5D6A7">独立集群+双消费</text>

  <rect x="710" y="55" width="160" height="45" rx="0" fill="#37474F"/>
  <text x="790" y="73" font-family="Arial,sans-serif" font-size="11" font-weight="bold" text-anchor="middle" fill="white">方案四</text>
  <text x="790" y="90" font-family="Arial,sans-serif" font-size="10" text-anchor="middle" fill="#B0BEC5">去 ES 用 OB</text>

  <!-- Row function -->
  ${[
    ["Agent/Bot 列表", "#FFCDD2|30s不可用/全挂", "#FFF9C4|只读正常", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["资源库列表", "#FFCDD2|30s不可用/全挂", "#FFF9C4|只读正常", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["搜索 / 筛选", "#FFCDD2|30s不可用/全挂", "#FFF9C4|只读正常", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["收藏 / 最近编辑", "#FFCDD2|30s不可用/全挂", "#FFCDD2|写入失败", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["创建/编辑资源", "#FFCDD2|30s不可用/全挂", "#FFCDD2|ES同步失败", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["知识库全文检索", "#FFCDD2|30s不可用/全挂", "#FFF9C4|只读正常", "#C8E6C9|正常", "#FFF9C4|需OB全文索引"],
    ["知识库文档入库", "#FFCDD2|30s不可用/全挂", "#FFCDD2|写入失败", "#C8E6C9|正常", "#C8E6C9|正常"],
    ["Agent 对话(无知识库)", "#C8E6C9|正常", "#C8E6C9|正常", "#C8E6C9|正常", "#C8E6C9|正常"],
  ].map((row, i) => {
    const y = 100 + i * 45;
    const bgColor = i % 2 === 0 ? "#FFFFFF" : "#F5F5F5";
    return `
      <rect x="30" y="${y}" width="160" height="45" fill="${bgColor}" stroke="#E0E0E0" stroke-width="0.5"/>
      <text x="110" y="${y + 28}" font-family="Arial,sans-serif" font-size="11" font-weight="bold" text-anchor="middle" fill="#37474F">${row[0]}</text>
      ${row.slice(1).map((cell, j) => {
        const [color, text] = cell.split("|");
        const cx = [190, 360, 530, 710][j];
        const cw = [170, 170, 180, 160][j];
        return `
          <rect x="${cx}" y="${y}" width="${cw}" height="45" fill="${color}" stroke="#E0E0E0" stroke-width="0.5"/>
          <text x="${cx + cw/2}" y="${y + 28}" font-family="Arial,sans-serif" font-size="11" text-anchor="middle" fill="#37474F">${text}</text>
        `;
      }).join("")}
    `;
  }).join("")}

  <!-- Legend -->
  <rect x="100" y="470" width="20" height="16" rx="3" fill="#C8E6C9" stroke="#388E3C" stroke-width="1"/>
  <text x="130" y="483" font-family="Arial,sans-serif" font-size="12" fill="#333">正常</text>

  <rect x="250" y="470" width="20" height="16" rx="3" fill="#FFF9C4" stroke="#F9A825" stroke-width="1"/>
  <text x="280" y="483" font-family="Arial,sans-serif" font-size="12" fill="#333">降级（只读）</text>

  <rect x="430" y="470" width="20" height="16" rx="3" fill="#FFCDD2" stroke="#D32F2F" stroke-width="1"/>
  <text x="460" y="483" font-family="Arial,sans-serif" font-size="12" fill="#333">不可用</text>

  <text x="450" y="510" font-family="Arial,sans-serif" font-size="12" text-anchor="middle" fill="#666" font-style="italic">* 方案二 CCR 需 ES 白金版许可证（开源版不可用）</text>
</svg>`;

// ============ Build Document ============

async function buildDoc() {
  const border = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
  const borders = { top: border, bottom: border, left: border, right: border };
  const noBorder = { style: BorderStyle.NONE, size: 0 };
  const noBorders = { top: noBorder, bottom: noBorder, left: noBorder, right: noBorder };

  // Convert SVGs to PNG with 2x resolution for crisp rendering
  const svgToPng = async (svg, width) => {
    return await sharp(Buffer.from(svg.trim()))
      .resize({ width: width * 2 })
      .png()
      .toBuffer();
  };

  const [png1, png2, png3] = await Promise.all([
    svgToPng(svg1_esArchitecture, 900),
    svgToPng(svg2_deploymentOptions, 900),
    svgToPng(svg3_impactMatrix, 900),
  ]);

  const doc = new Document({
    styles: {
      default: { document: { run: { font: "Arial", size: 22 } } },
      paragraphStyles: [
        { id: "Heading1", name: "Heading 1", basedOn: "Normal", next: "Normal", quickFormat: true,
          run: { size: 36, bold: true, font: "Arial", color: "1a1a2e" },
          paragraph: { spacing: { before: 360, after: 200 }, outlineLevel: 0 } },
        { id: "Heading2", name: "Heading 2", basedOn: "Normal", next: "Normal", quickFormat: true,
          run: { size: 28, bold: true, font: "Arial", color: "2d3436" },
          paragraph: { spacing: { before: 280, after: 160 }, outlineLevel: 1 } },
        { id: "Heading3", name: "Heading 3", basedOn: "Normal", next: "Normal", quickFormat: true,
          run: { size: 24, bold: true, font: "Arial", color: "37474F" },
          paragraph: { spacing: { before: 200, after: 120 }, outlineLevel: 2 } },
      ]
    },
    numbering: {
      config: [
        { reference: "bullets",
          levels: [{ level: 0, format: LevelFormat.BULLET, text: "\u2022", alignment: AlignmentType.LEFT,
            style: { paragraph: { indent: { left: 720, hanging: 360 } } } }] },
        { reference: "numbers",
          levels: [{ level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
            style: { paragraph: { indent: { left: 720, hanging: 360 } } } }] },
      ]
    },
    sections: [{
      properties: {
        page: {
          size: { width: 12240, height: 15840 },
          margin: { top: 1200, right: 1200, bottom: 1200, left: 1200 }
        }
      },
      headers: {
        default: new Header({
          children: [new Paragraph({
            alignment: AlignmentType.RIGHT,
            children: [new TextRun({ text: "Coze Studio - ES \u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848", font: "Arial", size: 18, color: "999999" })]
          })]
        })
      },
      footers: {
        default: new Footer({
          children: [new Paragraph({
            alignment: AlignmentType.CENTER,
            children: [
              new TextRun({ text: "Page ", font: "Arial", size: 18, color: "999999" }),
              new TextRun({ children: [PageNumber.CURRENT], font: "Arial", size: 18, color: "999999" })
            ]
          })]
        })
      },
      children: [
        // ===== Title =====
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 100 },
          children: [new TextRun({ text: "Coze Studio", font: "Arial", size: 48, bold: true, color: "1a1a2e" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 80 },
          children: [new TextRun({ text: "Elasticsearch \u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848", font: "Arial", size: 36, bold: true, color: "37474F" })]
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 400 },
          children: [new TextRun({ text: "2026-03-16", font: "Arial", size: 22, color: "999999" })]
        }),

        // ===== Section 1 =====
        new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("1. ES \u4F7F\u7528\u5168\u666F")] }),
        new Paragraph({ spacing: { after: 200 }, children: [
          new TextRun("Elasticsearch \u5728 Coze Studio \u4E2D\u4E0D\u4EC5\u7528\u4E8E\u77E5\u8BC6\u5E93\u5168\u6587\u68C0\u7D22\uFF0C\u8FD8\u662F"),
          new TextRun({ text: "\u6574\u4E2A\u5E73\u53F0\u7684\u5217\u8868/\u641C\u7D22\u5F15\u64CE", bold: true }),
          new TextRun("\u3002\u4EE5\u4E0B\u662F\u5B8C\u6574\u7684\u4F7F\u7528\u5168\u666F\u56FE\uFF1A"),
        ]}),

        // SVG 1
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new ImageRun({
            type: "png",
            data: png1,
            transformation: { width: 720, height: 496 },
            altText: { title: "ES Architecture", description: "ES usage overview in Coze Studio", name: "es-arch" }
          })]
        }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("1.1 ES \u7D22\u5F15\u8BF4\u660E")] }),

        // Index table
        new Table({
          width: { size: 9840, type: WidthType.DXA },
          columnWidths: [2200, 3620, 4020],
          rows: [
            new TableRow({ children: [
              new TableCell({ borders, width: { size: 2200, type: WidthType.DXA }, shading: { fill: "37474F", type: ShadingType.CLEAR },
                margins: { top: 60, bottom: 60, left: 100, right: 100 },
                children: [new Paragraph({ children: [new TextRun({ text: "ES \u7D22\u5F15", bold: true, color: "FFFFFF", font: "Arial", size: 20 })] })] }),
              new TableCell({ borders, width: { size: 3620, type: WidthType.DXA }, shading: { fill: "37474F", type: ShadingType.CLEAR },
                margins: { top: 60, bottom: 60, left: 100, right: 100 },
                children: [new Paragraph({ children: [new TextRun({ text: "\u5185\u5BB9", bold: true, color: "FFFFFF", font: "Arial", size: 20 })] })] }),
              new TableCell({ borders, width: { size: 4020, type: WidthType.DXA }, shading: { fill: "37474F", type: ShadingType.CLEAR },
                margins: { top: 60, bottom: 60, left: 100, right: 100 },
                children: [new Paragraph({ children: [new TextRun({ text: "\u5199\u5165\u6765\u6E90", bold: true, color: "FFFFFF", font: "Arial", size: 20 })] })] }),
            ]}),
            ...[
              ["project_draft", "Agent/Bot \u5217\u8868\u3001\u641C\u7D22\u3001\u7B5B\u9009\u3001\u6392\u5E8F\u3001\u6536\u85CF\u3001\u6700\u8FD1\u7F16\u8F91", "singleagent\u3001app\u3001space_importer"],
              ["coze_resource", "\u5DE5\u4F5C\u6D41\u3001\u63D2\u4EF6\u3001\u77E5\u8BC6\u5E93\u3001\u63D0\u793A\u8BCD\u3001\u6570\u636E\u5E93(\u8BB0\u5FC6)", "workflow\u3001plugin\u3001knowledge\u3001prompt\u3001database"],
              ["opencoze_{id}", "\u77E5\u8BC6\u5E93\u6587\u6863\u5207\u7247\u5168\u6587\u68C0\u7D22", "knowledge \u6587\u6863\u5165\u5E93\u6D41\u7A0B"],
            ].map((row, i) => new TableRow({ children: row.map((cell, j) =>
              new TableCell({ borders, width: { size: [2200, 3620, 4020][j], type: WidthType.DXA },
                shading: { fill: i % 2 === 0 ? "F5F5F5" : "FFFFFF", type: ShadingType.CLEAR },
                margins: { top: 60, bottom: 60, left: 100, right: 100 },
                children: [new Paragraph({ children: [new TextRun({ text: cell, font: "Arial", size: 20 })] })] })
            )}))
          ]
        }),

        new Paragraph({ spacing: { before: 200 }, children: [
          new TextRun({ text: "\u5173\u952E\u7ED3\u8BBA\uFF1A", bold: true }),
          new TextRun("ES \u5728\u8FD9\u4E2A\u7CFB\u7EDF\u91CC\u662F\u201C\u8BFB\u6A21\u578B/\u7D22\u5F15\u201D\uFF0C\u4E0D\u662F\u6570\u636E\u4E3B\u672C\u3002\u6240\u6709\u6570\u636E\u7684\u4E3B\u672C\u5728 MySQL/OceanBase \u4E2D\uFF0CES \u6570\u636E\u53EF\u4ECE DB \u5168\u91CF\u91CD\u5EFA\u3002")
        ]}),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== Section 2 =====
        new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("2. \u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848")] }),
        new Paragraph({ spacing: { after: 200 }, children: [
          new TextRun("\u4EE5\u4E0B\u662F\u56DB\u79CD ES \u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848\u7684\u67B6\u6784\u56FE\u548C\u6545\u969C\u5F71\u54CD\u5206\u6790\uFF1A"),
        ]}),

        // SVG 2
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new ImageRun({
            type: "png",
            data: png2,
            transformation: { width: 720, height: 600 },
            altText: { title: "Deployment Options", description: "ES dual-center deployment options", name: "deploy-opts" }
          })]
        }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("2.1 \u65B9\u6848\u4E00\uFF1A\u8DE8\u4E2D\u5FC3\u5355\u96C6\u7FA4")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4E00\u4E2A ES \u96C6\u7FA4\uFF0C\u8282\u70B9\u5206\u5E03\u5728\u4E24\u4E2A\u4E2D\u5FC3")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4F18\u70B9\uFF1A\u5F3A\u4E00\u81F4\u6027\uFF0C\u65E0\u9700\u6539\u4EE3\u7801")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u7F3A\u70B9\uFF1A\u8981\u6C42\u4F4E\u5EF6\u8FDF\u4E13\u7EBF\uFF1B\u8111\u88C2\u98CE\u9669\u9AD8\uFF1B\u4E00\u4E2A\u4E2D\u5FC3\u6302\u53EF\u80FD\u5BFC\u81F4\u9009\u4E3E\u5931\u8D25\u2192\u5168\u90E8\u4E0D\u53EF\u7528")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun({ text: "\u6545\u969C\u5F71\u54CD\uFF1A30s \u4E0D\u53EF\u7528\u6216\u5168\u90E8\u74E5\u75EA", bold: true, color: "D32F2F" })] }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("2.2 \u65B9\u6848\u4E8C\uFF1ACCR \u4E3B\u4ECE\u590D\u5236")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4E3B\u4E2D\u5FC3\u5199\u5165\uFF0C\u4ECE\u4E2D\u5FC3\u5F02\u6B65\u590D\u5236\u53EA\u8BFB\u526F\u672C")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4F18\u70B9\uFF1A\u4ECE\u4E2D\u5FC3\u53EF\u8BFB\uFF0C\u5206\u62C5\u67E5\u8BE2\u538B\u529B")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun({ text: "\u7F3A\u70B9\uFF1A\u9700\u8981 ES \u767D\u91D1\u7248\u8BB8\u53EF\u8BC1\uFF08\u5F00\u6E90\u7248\u4E0D\u53EF\u7528\uFF09", bold: true })] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun({ text: "\u4E3B\u4E2D\u5FC3\u6302\uFF1A\u80FD\u8BFB\u4E0D\u80FD\u5199\uFF0C\u5217\u8868\u6B63\u5E38\u4F46\u65B0\u8D44\u6E90\u4E0D\u540C\u6B65", color: "E65100" })] }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("2.3 \u65B9\u6848\u4E09\uFF1A\u72EC\u7ACB\u96C6\u7FA4 + EventBus \u53CC\u6D88\u8D39\uFF08\u63A8\u8350\uFF09")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u6BCF\u4E2A\u4E2D\u5FC3\u72EC\u7ACB ES \u96C6\u7FA4\uFF0C\u5404\u81EA\u4ECE EventBus \u6D88\u8D39\u540C\u4E00\u4EFD\u6D88\u606F\u5199\u5165\u672C\u5730 ES")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4F18\u70B9\uFF1A\u6539\u9020\u6700\u5C0F\uFF08\u53EA\u9700 EventBus Consumer \u53CC\u90E8\u7F72\uFF09\uFF1B\u5F00\u6E90\u7248\u53EF\u7528\uFF1B\u4E24\u4E2A\u4E2D\u5FC3\u5B8C\u5168\u89E3\u8026")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u7F3A\u70B9\uFF1A\u6700\u7EC8\u4E00\u81F4\uFF08\u79D2\u7EA7\u5EF6\u8FDF\uFF09")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun({ text: "\u4EFB\u4E00\u4E2D\u5FC3\u6302\uFF1A\u53E6\u4E00\u4E2A\u5B8C\u5168\u4E0D\u53D7\u5F71\u54CD\uFF0C\u96F6\u5F71\u54CD", bold: true, color: "1B5E20" })] }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("2.4 \u65B9\u6848\u56DB\uFF1A\u53BB ES\uFF0C\u641C\u7D22\u8D70 OceanBase")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("project_draft \u548C coze_resource \u672C\u8D28\u662F CQRS \u8BFB\u6A21\u578B\uFF0C\u6570\u636E\u5168\u5728 DB\uFF0C\u53EF\u76F4\u63A5\u67E5 DB")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u4F18\u70B9\uFF1A\u51CF\u5C11\u6709\u72B6\u6001\u7EC4\u4EF6\uFF0COB \u539F\u751F\u652F\u6301\u53CC\u6D3B")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u7F3A\u70B9\uFF1A\u9700\u8981\u8F83\u591A\u4EE3\u7801\u6539\u9020\uFF1B\u6A21\u7CCA\u641C\u7D22\u6027\u80FD\u53EF\u80FD\u4E0B\u964D")] }),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== Section 3 =====
        new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("3. \u5404\u65B9\u6848\u6545\u969C\u5F71\u54CD\u5BF9\u6BD4")] }),
        new Paragraph({ spacing: { after: 200 }, children: [
          new TextRun("\u4EE5\u4E0B\u77E9\u9635\u5C55\u793A\u5404\u65B9\u6848\u5728\u5355\u4E2D\u5FC3\u6545\u969C\u65F6\uFF0C\u5404\u529F\u80FD\u6A21\u5757\u7684\u53EF\u7528\u6027\uFF1A"),
        ]}),

        // SVG 3
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [new ImageRun({
            type: "png",
            data: png3,
            transformation: { width: 720, height: 416 },
            altText: { title: "Impact Matrix", description: "Failure impact comparison matrix", name: "impact-matrix" }
          })]
        }),

        new Paragraph({ children: [new PageBreak()] }),

        // ===== Section 4 =====
        new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("4. \u7ED3\u8BBA\u4E0E\u5EFA\u8BAE")] }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("\u77ED\u671F\u63A8\u8350\uFF1A\u65B9\u6848\u4E09")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u6539\u9020\u6700\u5C0F\uFF1A\u53EA\u9700\u8BA9\u4E24\u4E2A\u4E2D\u5FC3\u5404\u8DD1\u4E00\u5957 EventBus Consumer\uFF0C\u5404\u81EA\u5199\u5165\u672C\u5730 ES")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("ES \u6570\u636E\u5168\u90E8\u53EF\u4ECE MySQL \u91CD\u5EFA\uFF0C\u5373\u4F7F\u4E24\u8FB9\u77ED\u6682\u4E0D\u4E00\u81F4\u4E5F\u4E0D\u5F71\u54CD\u6570\u636E\u5B89\u5168")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u5355\u4E2D\u5FC3\u6545\u969C\u96F6\u5F71\u54CD\uFF0C\u5F00\u6E90\u7248 ES \u5373\u53EF\u5B9E\u73B0")] }),

        new Paragraph({ spacing: { before: 200 }, heading: HeadingLevel.HEADING_2, children: [new TextRun("\u4E2D\u957F\u671F\u53EF\u8003\u8651\uFF1A\u65B9\u6848\u56DB")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("project_draft \u548C coze_resource \u672C\u8D28\u662F CQRS \u8BFB\u6A21\u578B\uFF0C\u6570\u636E\u5168\u5728 DB\uFF0C\u76F4\u63A5\u67E5 DB \u5E76\u4E0D\u590D\u6742")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u51CF\u5C11\u4E00\u4E2A\u6709\u72B6\u6001\u7EC4\u4EF6\u7684\u8FD0\u7EF4\u8D1F\u62C5")] }),
        new Paragraph({ numbering: { reference: "bullets", level: 0 }, children: [new TextRun("\u9700\u8981\u8BC4\u4F30\u641C\u7D22\u6027\u80FD\uFF08\u6A21\u7CCA\u641C\u7D22\u3001\u591A\u6761\u4EF6\u7B5B\u9009 + \u6392\u5E8F + \u5206\u9875\u573A\u666F\uFF09")] }),

        new Paragraph({ spacing: { before: 300 }, children: [
          new TextRun({ text: "\u7EFC\u5408\u8BC4\u4F30\uFF1A", bold: true, size: 24 }),
          new TextRun({ text: "\u65B9\u6848\u4E09\u662F\u6027\u4EF7\u6BD4\u6700\u9AD8\u7684\u9009\u62E9\u2014\u2014\u6539\u52A8\u5C0F\u3001\u98CE\u9669\u4F4E\u3001\u6545\u969C\u96F6\u5F71\u54CD\u3002\u65B9\u6848\u56DB\u662F\u67B6\u6784\u6F14\u8FDB\u65B9\u5411\uFF0C\u53EF\u4F5C\u4E3A\u540E\u7EED\u4F18\u5316\u76EE\u6807\u3002", size: 24 }),
        ]}),
      ]
    }]
  });

  const buffer = await Packer.toBuffer(doc);
  fs.writeFileSync("/Users/luzhipeng/projects/coze-studio/docs/ES\u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848.docx", buffer);
  console.log("Done: docs/ES\u53CC\u4E2D\u5FC3\u90E8\u7F72\u65B9\u6848.docx");
}

buildDoc().catch(console.error);
