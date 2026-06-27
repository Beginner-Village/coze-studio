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

import { useMemo, useRef, useEffect, useState } from 'react';

import {
  LazyCozeMdBox,
  protectSandboxFilenames,
} from '@coze-common/chat-uikit';
import { Toast } from '@coze-arch/coze-design';

import {
  IcChevronDown,
  IcClock,
  IcCopy,
  IcList,
  IcSearch,
} from '../icons';
import { useTraceStore, type RawMessage } from './trace-store';

import s from './codex-trace.module.less';

const Markdown: React.FC<{ text: string }> = ({ text }) => (
  <LazyCozeMdBox
    markDown={protectSandboxFilenames(text)}
    autoFixSyntax={{ autoFixEnding: false }}
  />
);

// 工具名 → 简短动词(对齐 Codex/DeepSeek 的 Read/Bash 风格)
const TOOL_VERB: Record<string, string> = {
  run_bash: 'Bash',
  read_file: 'Read',
  write_file: 'Write',
  edit_file: 'Edit',
  list_files: 'List',
  grep: 'Grep',
  glob: 'Glob',
  web_fetch: 'Fetch',
  web_search: 'Search',
  search: 'Search',
  fetch: 'Fetch',
  read_skill: 'Skill',
  skill_manage: 'Skill',
  memory_save: 'Remember',
  memory_recall: 'Recall',
  deep_task: 'Delegate',
  update_plan: 'Plan',
};

const verbOf = (tool: string): string => {
  const trimmed = tool.trim();
  const mapped = TOOL_VERB[trimmed] || TOOL_VERB[trimmed.toLowerCase()];
  if (mapped) {
    return mapped;
  }
  return trimmed
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
};

interface ToolStep {
  kind: 'tool';
  callId: string;
  tool: string;
  verb: string;
  arg: string;
  request?: string;
  result?: string;
  startedAt?: number;
  finishedAt?: number;
  durationLabel?: string;
  status: 'running' | 'success' | 'error';
}
interface TextStep {
  kind: 'reasoning' | 'answer';
  text: string;
}
interface ContextStep {
  kind: 'context';
  summary: string;
  summaryPath: string;
  originalMessages: number;
  compactedMessages: number;
  retainedMessages: number;
  originalBytes: number;
  maxBytes: number;
}
type Step = ToolStep | TextStep | ContextStep;
interface Turn {
  id: string;
  user: string;
  userImages?: string[];
  steps: Step[];
  done: boolean;
  startedAt?: number;
  finishedAt?: number;
  metaLabel?: string;
}

const safeParse = (str?: string): Record<string, unknown> => {
  if (!str) {
    return {};
  }
  try {
    return JSON.parse(str) as Record<string, unknown>;
  } catch {
    return {};
  }
};

const cleanText = (value?: unknown): string =>
  typeof value === 'string' ? value.trim() : '';

// 用户消息可能是 content_type=mix 的 JSON({"item_list":[{type:"text",text},{type:"image",...}]})。
// trace 气泡只应显示用户输入的文字,而不是把整条 mix JSON 原文(含图片签名 URL)打出来。
const extractUserText = (content?: unknown): string => {
  const raw = cleanText(content);
  if (!raw.startsWith('{') || !raw.includes('"item_list"')) {
    return raw;
  }
  const list = safeParse(raw).item_list;
  if (!Array.isArray(list)) {
    return raw;
  }
  const texts = list
    .filter(
      it =>
        !!it &&
        typeof it === 'object' &&
        (it as { type?: unknown }).type === 'text' &&
        typeof (it as { text?: unknown }).text === 'string',
    )
    .map(it => ((it as { text: string }).text || '').trim())
    .filter(Boolean);
  if (texts.length) {
    return texts.join('\n');
  }
  const hasImage = list.some(
    it =>
      !!it &&
      typeof it === 'object' &&
      (it as { type?: unknown }).type === 'image',
  );
  return hasImage ? '' : raw;
};

// 从 mix 消息里取出图片缩略图 url(历史消息里是后端签名 URL,可直接展示)。
const extractUserImages = (content?: unknown): string[] => {
  const raw = cleanText(content);
  if (!raw.startsWith('{') || !raw.includes('"item_list"')) {
    return [];
  }
  const list = safeParse(raw).item_list;
  if (!Array.isArray(list)) {
    return [];
  }
  const urls: string[] = [];
  for (const it of list) {
    if (!it || typeof it !== 'object') {
      continue;
    }
    if ((it as { type?: unknown }).type !== 'image') {
      continue;
    }
    const img = (
      it as {
        image?: {
          image_thumb?: { url?: unknown };
          image_ori?: { url?: unknown };
        };
      }
    ).image;
    const url = img?.image_thumb?.url ?? img?.image_ori?.url;
    if (typeof url === 'string' && url) {
      urls.push(url);
    }
  }
  return urls;
};

const argOf = (toolName: string, pluginRequest?: string): string => {
  const a = safeParse(pluginRequest);
  const pick = (k: string) =>
    typeof a[k] === 'string' ? cleanText(a[k] as string) : '';
  return (
    pick('command') ||
    pick('path') ||
    pick('url') ||
    pick('pattern') ||
    pick('skill_name') ||
    pick('name') ||
    pick('content') ||
    pick('query') ||
    ''
  );
};

const stringifyToolArguments = (value: unknown): string => {
  if (typeof value === 'string') {
    return value;
  }
  if (value && typeof value === 'object') {
    return JSON.stringify(value);
  }
  return '';
};

const toolCallFromContent = (
  content?: string,
): { name?: string; request?: string } => {
  const text = cleanText(content);
  if (!text) {
    return {};
  }
  try {
    const parsed = JSON.parse(text) as {
      name?: string;
      tool_name?: string;
      arguments?: unknown;
      args?: unknown;
      function?: {
        name?: string;
        arguments?: unknown;
      };
    };
    return {
      name: cleanText(
        parsed.function?.name || parsed.name || parsed.tool_name || '',
      ),
      request: stringifyToolArguments(
        parsed.function?.arguments ?? parsed.arguments ?? parsed.args,
      ),
    };
  } catch {
    return {};
  }
};

const messageTime = (message: RawMessage): number =>
  Number(message?.created_at ?? message?.createdAt ?? 0);

const normalizeTimestamp = (value?: number): number => {
  if (!value || !Number.isFinite(value)) {
    return 0;
  }
  return value > 1e12 ? value : value * 1000;
};

const formatElapsed = (start?: number, end?: number): string => {
  const startMs = normalizeTimestamp(start);
  const endMs = normalizeTimestamp(end);
  if (!startMs || !endMs || endMs <= startMs) {
    return '';
  }
  return `${((endMs - startMs) / 1000).toFixed(1)}s`;
};

const numberOf = (value: unknown): number => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === 'string') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
};

const formatSeconds = (value: number): string => `${value.toFixed(1)}s`;

// 取不到真实计时就返回空串,调用方不渲染计时 label(绝不显示编造时间)。
const toolDurationLabel = (step: ToolStep): string => {
  if (step.status === 'running') {
    return '运行中';
  }
  if (step.durationLabel) {
    return step.durationLabel;
  }
  const elapsed = formatElapsed(step.startedAt, step.finishedAt);
  return elapsed ? `共 ${elapsed}` : '';
};

const toolDurationFromExtra = (extraInfo: Record<string, unknown>): string => {
  const timeCost =
    numberOf(extraInfo.time_cost) ||
    numberOf(extraInfo.latency) ||
    numberOf(extraInfo.cost_time);
  return timeCost ? `共 ${formatSeconds(timeCost)}` : '';
};

const answerMetaFromExtra = (
  extraInfo: Record<string, unknown>,
  start?: number,
  end?: number,
): string => {
  const timeCost =
    numberOf(extraInfo.time_cost) ||
    numberOf(extraInfo.latency) ||
    numberOf(extraInfo.cost_time);
  const elapsed = timeCost
    ? formatSeconds(timeCost)
    : formatElapsed(start, end);
  const inputTokens = numberOf(extraInfo.input_tokens);
  const outputTokens = numberOf(extraInfo.output_tokens);
  const tokens =
    numberOf(extraInfo.token) ||
    numberOf(extraInfo.total_tokens) ||
    inputTokens + outputTokens;
  if (elapsed && tokens) {
    return `${elapsed} · ${tokens} Tokens`;
  }
  if (elapsed) {
    return elapsed;
  }
  return tokens ? `${tokens} Tokens` : '';
};

const isContextCompactionEvent = (
  message: RawMessage,
  extraInfo: Record<string, unknown>,
): boolean => {
  const eventName = cleanText(message?.event || extraInfo.event);
  const type = cleanText(message?.type);
  return eventName === 'context.compacted' || type === 'context_compacted';
};

const contextStepFromMessage = (
  message: RawMessage,
  extraInfo: Record<string, unknown>,
): ContextStep => ({
  kind: 'context',
  summary: cleanText(message?.content),
  summaryPath: cleanText(extraInfo.summary_path),
  originalMessages: numberOf(extraInfo.original_messages),
  compactedMessages: numberOf(extraInfo.compacted_messages),
  retainedMessages: numberOf(extraInfo.retained_messages),
  originalBytes: numberOf(extraInfo.original_bytes),
  maxBytes: numberOf(extraInfo.max_bytes),
});

// 只显示能拿到的真实部分(优先后端 metaLabel,否则用真实 elapsed);都拿不到就空串。
const turnMetaLabel = (turn: Turn): string => {
  if (turn.metaLabel) {
    return turn.metaLabel;
  }
  return formatElapsed(turn.startedAt, turn.finishedAt);
};

const answerTextOf = (turn: Turn): string =>
  turn.steps
    .filter((step): step is TextStep => step.kind === 'answer')
    .map(step => step.text)
    .join('\n\n')
    .trim();

const handleCopyAnswer = async (turn: Turn): Promise<void> => {
  const text = answerTextOf(turn);
  if (!text) {
    return;
  }
  try {
    await navigator.clipboard.writeText(text);
    Toast.success('已复制');
  } catch {
    Toast.error('复制失败');
  }
};

const isSearchTool = (tool: string): boolean =>
  /search|grep|glob|fetch|web/i.test(tool);

// 把扁平消息派生成 Codex 时间线。
// 注意:chat-area 的 messages 多为「最新在前」,这里按时间正序处理。
const deriveTurns = (rawMessages: RawMessage[]): Turn[] => {
  const turns: Turn[] = [];
  let cur: Turn | null = null;
  const toolByCall = new Map<string, ToolStep>();

  const hasMessageTime = rawMessages.some(message => messageTime(message) > 0);
  const messages = hasMessageTime
    ? [...rawMessages].sort((a, b) => messageTime(a) - messageTime(b))
    : [...rawMessages].reverse();

  for (const m of messages) {
    const role = m?.role;
    const type = m?.type;
    const ei = {
      ...(m?.meta_data ?? {}),
      ...(m?.extra_info ?? {}),
    };
    if (role === 'user' && type !== 'tool_response') {
      cur = {
        id:
          m.message_id ||
          m.extra_info?.local_message_id ||
          String(turns.length),
        user: extractUserText(m?.content),
        userImages: extractUserImages(m?.content),
        steps: [],
        done: false,
        startedAt: messageTime(m),
      };
      turns.push(cur);
      toolByCall.clear();
      continue;
    }
    if (!cur) {
      cur = { id: 'init', user: '', steps: [], done: false };
      turns.push(cur);
    }
    if (isContextCompactionEvent(m, ei)) {
      cur.steps.push(contextStepFromMessage(m, ei));
      cur.finishedAt = messageTime(m) || cur.finishedAt;
    } else if (type === 'function_call') {
      const toolCall = toolCallFromContent(m?.content);
      const rawTool = cleanText(
        ei.tool_name || ei.message_title || toolCall.name,
      );
      const requestSource = ei.plugin_request || toolCall.request;
      const arg = argOf(rawTool, requestSource);
      if (!rawTool && !arg) {
        continue;
      }
      const tool = rawTool || 'tool';
      const step: ToolStep = {
        kind: 'tool',
        callId: ei.call_id || tool + Math.random(),
        tool,
        verb: verbOf(tool),
        arg,
        request: requestSource,
        startedAt: messageTime(m),
        status: 'running',
      };
      cur.steps.push(step);
      if (ei.call_id) {
        toolByCall.set(ei.call_id, step);
      }
    } else if (type === 'tool_response') {
      const step = ei.call_id ? toolByCall.get(ei.call_id) : undefined;
      if (step) {
        step.status = ei.plugin_status === '1' ? 'error' : 'success';
        step.result = typeof m?.content === 'string' ? m.content : '';
        step.finishedAt = messageTime(m);
        step.durationLabel = toolDurationFromExtra(ei);
      }
    } else if (type === 'answer') {
      const text = typeof m?.content === 'string' ? m.content : '';
      if (!text.trim()) {
        continue;
      }
      const last = cur.steps[cur.steps.length - 1];
      if (last && last.kind === 'answer') {
        last.text = text;
      } else {
        cur.steps.push({ kind: 'answer', text });
      }
      if (!m?.is_finish && m?.is_finish !== undefined) {
        cur.done = false;
      }
      cur.finishedAt = messageTime(m) || cur.finishedAt;
      cur.metaLabel =
        answerMetaFromExtra(ei, cur.startedAt, cur.finishedAt) || cur.metaLabel;
    }
  }
  return turns.filter(
    turn => turn.user || turn.userImages?.length || turn.steps.length > 0,
  );
};

const prettyJSON = (raw?: string): string => {
  const text = (raw || '').trim();
  if (!text) {
    return '';
  }
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
};

const workflowToolArgs = (step: ToolStep): Record<string, unknown> => {
  const fromRequest = safeParse(step.request);
  if (Object.keys(fromRequest).length > 0) {
    return fromRequest;
  }
  const fromResult = safeParse(step.result);
  if (fromResult.args && typeof fromResult.args === 'object') {
    return fromResult.args as Record<string, unknown>;
  }
  return {};
};

const stringArg = (args: Record<string, unknown>, key: string): string =>
  typeof args[key] === 'string' ? cleanText(args[key]) : '';

const workflowToolLabel = (
  step: ToolStep,
): { action: string; target?: string } | undefined => {
  if (!step.tool.startsWith('workflow_canvas_')) {
    return undefined;
  }
  const args = workflowToolArgs(step);
  const nodeTag = stringArg(args, 'node_tag') || stringArg(args, 'node');
  const title = stringArg(args, 'title');
  const type = stringArg(args, 'type');
  const from = stringArg(args, 'from');
  const to = stringArg(args, 'to');
  const config = args.config as Record<string, unknown> | undefined;
  const configTitle =
    config && typeof config.title === 'string' ? cleanText(config.title) : '';

  switch (step.tool) {
    case 'workflow_canvas_get_node_catalog':
      return { action: '读取节点目录' };
    case 'workflow_canvas_get_node_spec':
      return { action: '读取节点规格', target: type ? `type ${type}` : undefined };
    case 'workflow_canvas_get_canvas_context':
      return { action: '读取画布上下文' };
    case 'workflow_canvas_add_node':
      return { action: '添加节点', target: title || nodeTag || type };
    case 'workflow_canvas_connect':
      return { action: '连接节点', target: from && to ? `${from} → ${to}` : undefined };
    case 'workflow_canvas_delete_node':
      return { action: '删除节点', target: nodeTag };
    case 'workflow_canvas_delete_line':
      return { action: '删除连线', target: from && to ? `${from} → ${to}` : undefined };
    case 'workflow_canvas_clear_canvas':
      return { action: '清空画布' };
    case 'workflow_canvas_configure_node':
      return { action: '配置节点', target: configTitle || nodeTag };
    case 'workflow_canvas_set_node_params':
      return { action: '修改节点参数', target: nodeTag };
    case 'workflow_canvas_auto_layout':
      return { action: '优化布局' };
    case 'workflow_canvas_test_run':
      return { action: '试运行工作流' };
    case 'workflow_canvas_get_operation_guide':
      return { action: '读取操作指南' };
    case 'workflow_canvas_get_resource_catalog':
      return { action: '读取资源目录' };
    case 'workflow_canvas_list_plugins':
      return { action: '列出可用插件', target: stringArg(args, 'keyword') || undefined };
    case 'workflow_canvas_create_plugin_from_curl':
      return { action: '从CURL建插件', target: stringArg(args, 'name') || undefined };
    case 'workflow_canvas_list_databases':
      return { action: '列出数据库表', target: stringArg(args, 'keyword') || undefined };
    case 'workflow_canvas_get_bindable_variables':
      return { action: '读取可绑定变量', target: nodeTag || undefined };
    case 'workflow_canvas_get_node_capability_audit':
      return { action: '审计节点能力', target: type ? `type ${type}` : undefined };
    case 'workflow_canvas_get_node_smoke_manifest':
      return { action: '读取节点冒烟清单' };
    case 'workflow_canvas_get_node_smoke_coverage':
      return { action: '读取节点冒烟覆盖' };
    default:
      return { action: step.verb || '调用画布工具', target: nodeTag || title };
  }
};

const ToolRow: React.FC<{ step: ToolStep }> = ({ step }) => {
  const [open, setOpen] = useState(false);
  const SearchOrClock = isSearchTool(step.tool) ? IcSearch : IcClock;
  const iconClassName = isSearchTool(step.tool)
    ? `${s.toolCallIcon} ${s.toolCallIconBrand}`
    : s.toolCallIcon;
  const toolName = step.tool || step.verb;
  const durationLabel = toolDurationLabel(step);
  const request = prettyJSON(step.request);
  const result = (step.result || '').trim();
  const hasDetail = Boolean(request || result);
  const workflowLabel = workflowToolLabel(step);

  const copy = (text: string) => {
    navigator.clipboard?.writeText(text).then(
      () => Toast.success('已复制'),
      () => Toast.error('复制失败'),
    );
  };

  return (
    <div className={s.toolCallWrap}>
      <div
        className={s.toolCall}
        title={step.arg || undefined}
        data-expandable={hasDetail || undefined}
        onClick={hasDetail ? () => setOpen(o => !o) : undefined}
      >
        <span className={iconClassName} data-status={step.status}>
          <SearchOrClock size={15} />
        </span>
        <span className={s.toolCallName}>
          {workflowLabel ? (
            <>
              <span className={s.toolCallAction}>{workflowLabel.action}</span>
              {workflowLabel.target ? (
                <span className={s.toolCallTarget}>
                  {workflowLabel.target}
                </span>
              ) : null}
            </>
          ) : (
            <>
              已调用 <code>{toolName}</code>
            </>
          )}
        </span>
        {durationLabel ? (
          <span className={s.toolCallTime}>{durationLabel}</span>
        ) : null}
        {hasDetail ? (
          <IcChevronDown
            size={15}
            className={`${s.toolCallChevron} ${open ? s.toolCallChevronOpen : ''}`}
          />
        ) : null}
      </div>
      {open && hasDetail ? (
        <div className={s.toolDetail}>
          {request ? (
            <div className={s.toolBlock}>
              <div className={s.toolBlockHd}>
                <span className={s.toolBlockLabel}>入参</span>
                <button
                  type="button"
                  className={s.toolBlockCopy}
                  onClick={() => copy(request)}
                >
                  <IcCopy size={13} />
                </button>
              </div>
              <pre className={s.toolPre}>{request}</pre>
            </div>
          ) : null}
          {result ? (
            <div className={s.toolBlock}>
              <div className={s.toolBlockHd}>
                <span className={s.toolBlockLabel} data-error={step.status === 'error' || undefined}>
                  {step.status === 'error' ? '错误结果' : '结果'}
                </span>
                <button
                  type="button"
                  className={s.toolBlockCopy}
                  onClick={() => copy(result)}
                >
                  <IcCopy size={13} />
                </button>
              </div>
              <pre className={s.toolPre}>{result}</pre>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
};

const ContextCompactionCard: React.FC<{ step: ContextStep }> = ({ step }) => (
  <div className={s.contextCard}>
    <div className={s.contextHead}>
      <IcList size={15} className={s.contextIcon} />
      <span className={s.contextTitle}>上下文已自动压缩</span>
    </div>
    <div className={s.contextMetrics}>
      {step.originalMessages ? (
        <span>{`${step.originalMessages} 条历史消息`}</span>
      ) : null}
      {step.compactedMessages ? (
        <span>{`压缩 ${step.compactedMessages} 条`}</span>
      ) : null}
      {step.retainedMessages ? (
        <span>{`保留 ${step.retainedMessages} 条`}</span>
      ) : null}
    </div>
    {step.summaryPath ? (
      <div className={s.contextPath}>{step.summaryPath}</div>
    ) : null}
    {step.summary ? <div className={s.contextSummary}>{step.summary}</div> : null}
  </div>
);

export const CodexTracePanel: React.FC = () => {
  const messages = useTraceStore(st => st.messages);
  const pendingReply = useTraceStore(st => st.pendingReply);
  const turns = useMemo(() => deriveTurns(messages), [messages]);
  const visibleTurns =
    turns.length > 0
      ? turns
      : pendingReply
        ? [{ id: 'pending-reply', user: '', steps: [], done: false }]
        : [];
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, pendingReply]);

  if (visibleTurns.length === 0) {
    return (
      <div className={s.empty}>
        <IcList size={38} className={s.emptyIcon} />
        <div className={s.emptyTitle}>FinMallClaw 执行台</div>
        <div className={s.emptyDesc}>
          发送任务,这里会实时展示它的思考与每一步工具调用
        </div>
      </div>
    );
  }

  return (
    <div className={s.panel}>
      {visibleTurns.map((turn, index) => {
        const hasAnswer = turn.steps.some(step => step.kind === 'answer');
        const metaLabel = turnMetaLabel(turn);
        const showTyping =
          pendingReply && index === visibleTurns.length - 1 && !hasAnswer;
        return (
          <div key={turn.id} className={s.turn}>
            {turn.user || turn.userImages?.length ? (
              <div className={s.userBubble}>
                {turn.userImages?.length ? (
                  <div
                    style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 6,
                      marginBottom: turn.user ? 6 : 0,
                    }}
                  >
                    {turn.userImages.map((url, i) => (
                      <img
                        key={i}
                        src={url}
                        alt=""
                        style={{
                          maxWidth: 160,
                          maxHeight: 160,
                          borderRadius: 8,
                          objectFit: 'cover',
                        }}
                      />
                    ))}
                  </div>
                ) : null}
                {turn.user ? <div>{turn.user}</div> : null}
              </div>
            ) : null}
            {turn.steps.length > 0 || showTyping ? (
              <div className={s.aiMessage}>
                <div className={s.aiAvatar} />
                <div className={s.aiCol}>
                  <div className={s.aiName}>演示超级体</div>
                  <div className={s.timeline}>
                    {turn.steps.map((step, i) =>
                      step.kind === 'answer' ? (
                        <div key={i} className={s.answer}>
                          <Markdown text={step.text} />
                        </div>
                      ) : step.kind === 'context' ? (
                        <ContextCompactionCard key={i} step={step} />
                      ) : step.kind === 'tool' ? (
                        <ToolRow key={step.callId || i} step={step} />
                      ) : (
                        <div key={i} className={s.reasoning}>
                          <Markdown text={step.text} />
                        </div>
                      ),
                    )}
                    {showTyping ? (
                      <div className={s.typingBubble} aria-label="正在生成回复">
                        <span />
                        <span />
                        <span />
                      </div>
                    ) : null}
                    {hasAnswer ? (
                      <div className={s.aiFoot}>
                        {metaLabel ? (
                          <span className={s.aiMeta}>{metaLabel}</span>
                        ) : null}
                        <div className={s.aiActs}>
                          <button
                            className={s.aiAct}
                            type="button"
                            aria-label="复制"
                            onClick={() => handleCopyAnswer(turn)}
                          >
                            <IcCopy size={15} />
                          </button>
                        </div>
                      </div>
                    ) : null}
                  </div>
                </div>
              </div>
            ) : null}
          </div>
        );
      })}
      <div ref={endRef} />
    </div>
  );
};
