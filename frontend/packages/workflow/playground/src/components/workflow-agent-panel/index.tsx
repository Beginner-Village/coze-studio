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

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useService } from '@flowgram-adapter/free-layout-editor';
import { userStoreService } from '@coze-studio/user-store';
import { IconCozLink } from '@coze-arch/coze-design/icons';
import { Tooltip } from '@coze-arch/coze-design';
import {
  createWriteableLifeCycleServices,
  PluginMode,
  useChatAreaStoreSet,
  WriteableChatAreaPlugin,
  type OnBeforeSendMessageContext,
  type PluginName,
  type PluginRegistryEntry,
  type WriteableLifeCycleServiceGenerator,
} from '@coze-common/chat-area';
import { BotDebugChatAreaProviderAdapter } from '@coze-agent-ide/chat-area-provider-adapter';
import { SuperChatArea } from '@coze-agent-ide/bot-creator';

import {
  WorkflowAgentCommandService,
  WORKFLOW_AGENT_NODE_BINDING_GUIDE,
  auditWorkflowAgentNodeCapabilities,
  detectWorkflowAgentTestRunRequirement,
  discoverWorkflowAgentResources,
  formatWorkflowAgentNodeCapabilityAudit,
  formatWorkflowAgentNodeCatalog,
  formatWorkflowAgentResourceSummary,
  isWorkflowAgentSingletonNodeType,
  pollWorkflowCanvasBrowserCommands,
  postWorkflowCanvasBrowserCommandResult,
  type AgentCanvasCommand,
  type WorkflowCanvasCommandEnvelope,
  type WorkflowCanvasCommandResult,
  type WorkflowAgentTestRunRequirement,
} from '../../services';
import { WorkflowGlobalStateEntity } from '../../entities/workflow-global-state-entity';
import { ExternalMCPGuideModal } from './external-mcp-guide-modal';
import {
  readWorkflowAgentPanelOpenState,
  writeWorkflowAgentPanelOpenState,
} from './open-state';
import {
  buildWorkflowAgentRequestQuery,
  getWorkflowAgentMessageText,
} from './message-payload';

const DEFAULT_FINMALLCLAW_AGENT_ID = '7652617174313336832';
const DEFAULT_FINMALLCLAW_SPACE_ID = '7652614054615187456';
const WORKFLOW_AGENT_SEND_EVENT = 'coze:workflow-agent-send';
const WORKFLOW_AGENT_SESSION_TITLE_PREFIX = '工作流画布 ';
const WORKFLOW_AGENT_SESSION_PAGE_SIZE = 100;
const WORKFLOW_AGENT_SCENE_PLAYGROUND = 4;
const WORKFLOW_MCP_RELAY_POLL_INTERVAL = 1500;
const WORKFLOW_MCP_RELAY_LIMIT = 50;

interface CanvasAck {
  status?: string;
  op?: string;
  args?: Record<string, unknown>;
}

interface SuperAgentSession {
  session_id?: string;
  conversation_id?: string;
  title?: string;
  scene?: number;
}

interface WorkflowAgentSession {
  conversationId: string;
  scene?: number;
}

interface WorkflowCanvasPluginContext {
  buildPayload: (rawQuery: string) => Promise<WorkflowAgentSendPayload>;
  prepareSend: () => void;
  spaceId?: string;
  workflowId?: string;
}

interface WorkflowAgentSendPayload {
  query: string;
  extra: Record<string, string | undefined>;
}

const workflowCanvasPluginName = 'WorkflowCanvasCopilot' as PluginName;
const workflowAgentIdCache = new Map<string, string>();
const workflowSessionCache = new Map<string, WorkflowAgentSession>();

const isRecord = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === 'object' && !Array.isArray(v);

const getSuperAgentSessionId = (session: SuperAgentSession): string =>
  session.session_id || session.conversation_id || '';

const getWorkflowAgentIdCacheKey = (spaceId?: string): string =>
  spaceId || 'default';

const readWorkflowAgentId = (spaceId?: string): string =>
  workflowAgentIdCache.get(getWorkflowAgentIdCacheKey(spaceId)) ?? '';

const writeWorkflowAgentId = (spaceId: string | undefined, agentId: string) => {
  if (agentId) {
    workflowAgentIdCache.set(getWorkflowAgentIdCacheKey(spaceId), agentId);
  }
};

const getWorkflowSessionCacheKey = ({
  spaceId,
  agentId,
  workflowId,
}: {
  spaceId?: string;
  agentId?: string;
  workflowId?: string;
}): string => [spaceId, agentId, workflowId].filter(Boolean).join(':');

const readWorkflowSession = (key: string): WorkflowAgentSession | null =>
  key ? workflowSessionCache.get(key) ?? null : null;

const writeWorkflowSession = (
  key: string,
  session: WorkflowAgentSession,
): void => {
  if (key && session.conversationId) {
    workflowSessionCache.set(key, session);
  }
};

const stringifyBrief = (v: unknown, max = 6000): string => {
  try {
    const s = JSON.stringify(v);
    return s.length > max ? `${s.slice(0, max)}...` : s;
  } catch {
    return '';
  }
};

const getMessageText = (content: unknown): string => {
  if (typeof content === 'string') {
    return content;
  }
  return stringifyBrief(content, 1200);
};

const ackToCommand = (ack: CanvasAck): AgentCanvasCommand | null => {
  if (ack.status !== 'dispatched_to_canvas') {
    return null;
  }
  const args = ack.args ?? {};
  switch (ack.op) {
    case 'add_node': {
      const x = args.x as number | undefined;
      const y = args.y as number | undefined;
      return {
        op: 'addNode',
        tag: String(args.node_tag ?? ''),
        args: {
          type: String(args.type) as never,
          title: args.title ? String(args.title) : undefined,
          position:
            typeof x === 'number' && typeof y === 'number'
              ? { x, y }
              : undefined,
        },
      };
    }
    case 'connect':
      return {
        op: 'connect',
        args: {
          from: String(args.from ?? ''),
          to: String(args.to ?? ''),
          fromPort: args.from_port ? String(args.from_port) : undefined,
          toPort: args.to_port ? String(args.to_port) : undefined,
        },
      };
    case 'delete_node':
      return {
        op: 'deleteNode',
        args: { node: String(args.node_tag ?? args.node ?? '') },
      };
    case 'delete_line':
      return {
        op: 'deleteLine',
        args: {
          from: String(args.from ?? ''),
          to: String(args.to ?? ''),
          fromPort: args.from_port ? String(args.from_port) : undefined,
          toPort: args.to_port ? String(args.to_port) : undefined,
        },
      };
    case 'clear_canvas':
      return { op: 'clearCanvas' };
    case 'set_node_params':
      return {
        op: 'setNodeParams',
        args: {
          node: String(args.node_tag ?? args.node ?? ''),
          params: isRecord(args.params) ? args.params : {},
        },
      };
    case 'configure_node':
      return {
        op: 'configureNode',
        args: {
          node: String(args.node_tag ?? args.node ?? ''),
          config: isRecord(args.config) ? args.config : {},
        },
      };
    case 'auto_layout':
      return { op: 'autoLayout' };
    default:
      return null;
  }
};

const workflowCanvasLifeCycleGenerator: WriteableLifeCycleServiceGenerator<
  WorkflowCanvasPluginContext
> = plugin => ({
  messageLifeCycleService: {
    async onBeforeSendMessage(
      ctx: OnBeforeSendMessageContext,
    ): Promise<OnBeforeSendMessageContext> {
      const { buildPayload, prepareSend, spaceId, workflowId } =
        plugin.pluginBizContext;
      prepareSend();

      const rawQuery = getWorkflowAgentMessageText(ctx.message.content);
      const payload = await buildPayload(rawQuery);
      const requestQuery = buildWorkflowAgentRequestQuery({
        content: ctx.message.content,
        contentType: ctx.message.content_type,
        query: payload.query,
      });
      const extendFiled = ctx.options?.extendFiled ?? {};
      const extra = isRecord(extendFiled.extra) ? extendFiled.extra : {};

      return {
        ...ctx,
        options: {
          ...ctx.options,
          extendFiled: {
            ...extendFiled,
            query: requestQuery,
            space_id: spaceId,
            extra: {
              ...extra,
              ...payload.extra,
              workflow_canvas_mode: 'true',
              workflow_id: workflowId,
              space_id: spaceId,
              source: 'workflow_agent_panel',
            },
          },
        },
      };
    },
  },
});

class WorkflowCanvasChatPlugin extends WriteableChatAreaPlugin<WorkflowCanvasPluginContext> {
  public pluginMode = PluginMode.Writeable;
  public pluginName = workflowCanvasPluginName;
  public lifeCycleServices = createWriteableLifeCycleServices(
    this,
    workflowCanvasLifeCycleGenerator,
  );
}

const createWorkflowCanvasPluginRegistry = (
  context: WorkflowCanvasPluginContext,
): PluginRegistryEntry<WorkflowCanvasPluginContext> => ({
  createPluginBizContext: () => context,
  Plugin: WorkflowCanvasChatPlugin,
});

// 对话工作流(chatflow)专属规则:注入给超级智能体,让它知道当前是对话流并正确
// 设置对话历史/流式等对话专属参数。仅 isChatflow 时注入,普通工作流完全不受影响。
const WORKFLOW_AGENT_CHATFLOW_GUIDE = [
  '【对话工作流(chatflow)专属规则——当前画布是对话流,必须遵守】',
  '本工作流 flow_mode=chatflow,面向多轮对话。仅按普通工作流搭建会导致节点无对话记忆、单轮、不流式,对话体验是坏的。除了正常搭节点与连线,还需:',
  '1. 大模型节点(type 3):需结合对话历史(多轮记忆)时,configure_node 传 {"enable_chat_history": true, "chat_history_round": 5};或 set_node_params 设 "$$input_decorator$$.chatHistorySetting" = {"enableChatHistory": true, "chatHistoryRound": 5}。chat_history_round 为携带的历史轮数,常用 3~20。',
  '2. 意图识别节点(type 22):需对话历史时,用 set_node_params 设 "inputs.chatHistorySetting.enableChatHistory" = true 与 "inputs.chatHistorySetting.chatHistoryRound" = 5(意图节点不在 configure_node 语义配置范围,只能用 set_node_params)。',
  '3. 结束/输出节点:对话流通常流式回复,configure_node 传 "streaming_output": true,或 set_node_params 设 "inputs.streamingOutput" = true。',
  '4. 开始节点是对话流预设节点,已自带 USER_INPUT(用户输入)等系统输入且可自动写入会话历史,不要重新定义开始节点的输入,直接从开始节点引用 USER_INPUT。',
  '5. 主线:理解用户输入 →(可选)意图分流 → 大模型回答;凡需"记住上文/延续对话"的大模型与意图节点都要开启上面的对话历史。',
].join('\n');

// 用户选中/正在编辑某个节点时注入:让超级智能体把"这个节点/这里/它"指代消解到选中节点。
const WORKFLOW_AGENT_SELECTED_NODE_GUIDE = [
  '【选中节点上下文】用户当前选中/正在编辑一个节点(详见 workflow_canvas_selected_node)。',
  '当用户说"这个节点""这里""它""当前节点""选中的"而未指明节点 id 时,默认就指这个选中节点。',
  '据此就地操作:用 configure_node/set_node_params 优化或修改它、delete_node 删除后按要求重建、或从它 connect 出新分支。优先在选中节点上就地改,不要新建无关节点。',
].join('\n');

const buildWorkflowAgentPayload = async ({
  command,
  workflowId,
  spaceId,
  rawQuery,
  testRunRequirement,
  isChatflow,
}: {
  command: WorkflowAgentCommandService;
  workflowId?: string;
  spaceId?: string;
  rawQuery: string;
  testRunRequirement?: WorkflowAgentTestRunRequirement;
  isChatflow?: boolean;
}): Promise<WorkflowAgentSendPayload> => {
  const availableNodeTypes = command
    .listNodeTypes()
    .filter(n => !isWorkflowAgentSingletonNodeType(String(n.type)))
    .map(n => ({ type: String(n.type) as never, title: n.title }));
  const nodeCatalog = formatWorkflowAgentNodeCatalog(availableNodeTypes);
  const nodeCapabilityAudit = formatWorkflowAgentNodeCapabilityAudit(
    auditWorkflowAgentNodeCapabilities(availableNodeTypes),
  );
  const [canvasSummary, bindableVariables, resources, selectedNode] =
    await Promise.all([
      command.getCanvasSummary().catch(() => undefined),
      command.getBindableVariablesSummary().catch(() => undefined),
      discoverWorkflowAgentResources(spaceId),
      command.getSelectedNodeSummary().catch(() => ''),
    ]);
  const resourceSummary = formatWorkflowAgentResourceSummary(resources);
  const testRun =
    testRunRequirement ?? detectWorkflowAgentTestRunRequirement(rawQuery);

  return {
    query: rawQuery,
    extra: {
      workflow_canvas_node_catalog: nodeCatalog,
      workflow_canvas_node_capability_audit: nodeCapabilityAudit,
      workflow_canvas_context: canvasSummary || 'unknown',
      workflow_canvas_bindable_variables: bindableVariables || 'unknown',
      workflow_canvas_resource_summary: resourceSummary,
      workflow_canvas_binding_guide: WORKFLOW_AGENT_NODE_BINDING_GUIDE,
      workflow_canvas_test_run_required: String(testRun.required),
      workflow_canvas_test_run_input: JSON.stringify(
        testRun.input ?? { input: '示例输入' },
      ),
      workflow_canvas_workflow_id: workflowId,
      workflow_canvas_space_id: spaceId,
      ...(isChatflow
        ? {
            workflow_canvas_flow_mode: 'chatflow',
            workflow_canvas_chatflow_guide: WORKFLOW_AGENT_CHATFLOW_GUIDE,
          }
        : {}),
      ...(selectedNode
        ? {
            workflow_canvas_selected_node: selectedNode,
            workflow_canvas_selected_node_guide:
              WORKFLOW_AGENT_SELECTED_NODE_GUIDE,
          }
        : {}),
    },
  };
};

const WorkflowCanvasCommandBridge: React.FC<{
  command: WorkflowAgentCommandService;
}> = ({ command }) => {
  const { useMessagesStore } = useChatAreaStoreSet();
  const messages = useMessagesStore(state => state.messages);
  const commandRef = useRef(command);
  const messagesRef = useRef(messages);
  const armedRef = useRef(false);
  const seenAcksRef = useRef<Set<string>>(new Set());
  const queueRef = useRef<Promise<void>>(Promise.resolve());

  commandRef.current = command;
  messagesRef.current = messages;

  const getCanvasAckId = useCallback((message: (typeof messages)[number]) => {
    const content = getMessageText(message.content);
    if (
      String(message.type) !== 'tool_response' ||
      !content.includes('dispatched_to_canvas')
    ) {
      return '';
    }
    return String(
      message.message_id || message.extra_info?.local_message_id || content,
    );
  }, []);

  useEffect(() => {
    const onSend = () => {
      armedRef.current = true;
      seenAcksRef.current.clear();
      messagesRef.current.forEach(message => {
        const id = getCanvasAckId(message);
        if (id) {
          seenAcksRef.current.add(id);
        }
      });
      queueRef.current = Promise.resolve();
    };
    window.addEventListener(WORKFLOW_AGENT_SEND_EVENT, onSend);
    return () => {
      window.removeEventListener(WORKFLOW_AGENT_SEND_EVENT, onSend);
    };
  }, [getCanvasAckId]);

  useEffect(() => {
    if (!armedRef.current) {
      return;
    }

    messages.forEach(message => {
      const content = getMessageText(message.content);
      const id = getCanvasAckId(message);
      if (!id) {
        return;
      }
      if (seenAcksRef.current.has(id)) {
        return;
      }
      seenAcksRef.current.add(id);
      try {
        const cmd = ackToCommand(JSON.parse(content) as CanvasAck);
        if (cmd) {
          queueRef.current = queueRef.current
            .catch(() => undefined)
            .then(async () => {
              await commandRef.current.execCommand(cmd);
            })
            .then(() => undefined);
        }
      } catch (error) {
        if (process.env.NODE_ENV === 'development') {
          console.warn(
            '[WorkflowCanvasCommandBridge] Ignore malformed tool response',
            error,
          );
        }
      }
    });
  }, [getCanvasAckId, messages]);

  return null;
};

const workflowRelayCursorKey = (spaceId?: string, workflowId?: string): string =>
  `workflow-mcp-relay-cursor:${spaceId ?? ''}:${workflowId ?? ''}`;

const readWorkflowRelayCursor = (
  spaceId?: string,
  workflowId?: string,
): number => {
  if (!spaceId || !workflowId) {
    return 0;
  }
  const value = window.sessionStorage.getItem(
    workflowRelayCursorKey(spaceId, workflowId),
  );
  const cursor = Number(value);
  return Number.isFinite(cursor) && cursor > 0 ? cursor : 0;
};

const writeWorkflowRelayCursor = (
  spaceId: string,
  workflowId: string,
  cursor: number,
): void => {
  window.sessionStorage.setItem(
    workflowRelayCursorKey(spaceId, workflowId),
    String(cursor),
  );
};

const getWorkflowRelayErrorMessage = (error: unknown): string =>
  error instanceof Error ? error.message : String(error);

const createWorkflowRelayFailedResult = (
  envelope: WorkflowCanvasCommandEnvelope,
  error: unknown,
): WorkflowCanvasCommandResult => {
  const message = getWorkflowRelayErrorMessage(error);
  return {
    protocol: 'canvas_automation.v0',
    request_id: envelope.request_id,
    status: 'failed',
    results: envelope.commands.map(command => ({
      op: command.op,
      ok: false,
      target: command.target,
      message,
      diagnostics: [
        {
          level: 'error',
          message,
          op: command.op,
        },
      ],
    })),
    binding_diagnostics: [
      {
        level: 'error',
        message,
      },
    ],
  };
};

const WorkflowCanvasExternalMCPCommandBridge: React.FC<{
  command: WorkflowAgentCommandService;
  workflowId?: string;
  spaceId?: string;
}> = ({ command, workflowId, spaceId }) => {
  const commandRef = useRef(command);
  const cursorRef = useRef(0);
  const inFlightRef = useRef(false);

  commandRef.current = command;

  useEffect(() => {
    if (!workflowId || !spaceId) {
      return;
    }

    let cancelled = false;
    cursorRef.current = readWorkflowRelayCursor(spaceId, workflowId);

    const poll = async () => {
      if (cancelled || inFlightRef.current) {
        return;
      }
      inFlightRef.current = true;
      try {
        const result = await pollWorkflowCanvasBrowserCommands({
          workflowId,
          spaceId,
          afterId: cursorRef.current,
          limit: WORKFLOW_MCP_RELAY_LIMIT,
        });
        if (cancelled) {
          return;
        }
        const envelopes = result.envelopes.length
          ? result.envelopes
          : [result.envelope];
        for (const envelope of envelopes) {
          if (!envelope.commands.length) {
            continue;
          }
          let commandResult: WorkflowCanvasCommandResult;
          try {
            commandResult =
              await commandRef.current.applyCommandEnvelope(envelope);
          } catch (error) {
            commandResult = createWorkflowRelayFailedResult(envelope, error);
          }
          if (envelope.requires_response) {
            await postWorkflowCanvasBrowserCommandResult({
              workflowId,
              spaceId,
              result: commandResult,
            });
          }
        }
        cursorRef.current = result.cursor;
        writeWorkflowRelayCursor(spaceId, workflowId, result.cursor);
      } catch (error) {
        if (process.env.NODE_ENV === 'development') {
          console.warn(
            '[WorkflowCanvasExternalMCPCommandBridge] Poll failed',
            error,
          );
        }
      } finally {
        inFlightRef.current = false;
      }
    };

    void poll();
    const timer = window.setInterval(poll, WORKFLOW_MCP_RELAY_POLL_INTERVAL);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [spaceId, workflowId]);

  return null;
};

export const PANEL_WIDTH = 420;

export const WorkflowAgentPanel: React.FC<{
  onOpenChange?: (open: boolean) => void;
}> = ({ onOpenChange }) => {
  const command = useService<WorkflowAgentCommandService>(
    WorkflowAgentCommandService,
  );
  const globalState = useService<WorkflowGlobalStateEntity>(
    WorkflowGlobalStateEntity,
  );
  const userInfo = userStoreService.useUserInfo();

  const spaceId = globalState.spaceId || globalState.playgroundProps?.spaceId;
  const workflowId =
    globalState.workflowId || globalState.playgroundProps?.workflowId;
  const workflowOpenStateKey = workflowId || spaceId;

  const [agentId, setAgentIdState] = useState(() =>
    readWorkflowAgentId(spaceId),
  );
  const workflowSessionKey = getWorkflowSessionCacheKey({
    spaceId,
    agentId,
    workflowId,
  });
  const [workflowSession, setWorkflowSession] =
    useState<WorkflowAgentSession | null>(() =>
      readWorkflowSession(workflowSessionKey),
    );
  const [workflowSessionError, setWorkflowSessionError] = useState(false);

  const setAgentId = useCallback(
    (nextAgentId: string) => {
      writeWorkflowAgentId(spaceId, nextAgentId);
      setAgentIdState(nextAgentId);
    },
    [spaceId],
  );

  const [open, setOpenState] = useState(() =>
    readWorkflowAgentPanelOpenState(workflowOpenStateKey),
  );
  const [mcpGuideOpen, setMcpGuideOpen] = useState(false);
  const previousWorkflowOpenStateKeyRef = useRef(workflowOpenStateKey);

  const setOpen = useCallback(
    (nextOpen: boolean) => {
      setOpenState(nextOpen);
      writeWorkflowAgentPanelOpenState(workflowOpenStateKey, nextOpen);
    },
    [workflowOpenStateKey],
  );

  useEffect(() => {
    onOpenChange?.(open);
  }, [open, onOpenChange]);

  useEffect(() => {
    if (previousWorkflowOpenStateKeyRef.current === workflowOpenStateKey) {
      return;
    }
    previousWorkflowOpenStateKeyRef.current = workflowOpenStateKey;
    setOpenState(currentOpen => {
      if (currentOpen) {
        writeWorkflowAgentPanelOpenState(workflowOpenStateKey, true);
        return true;
      }
      return readWorkflowAgentPanelOpenState(workflowOpenStateKey);
    });
  }, [workflowOpenStateKey]);

  useEffect(() => {
    if (agentId || !spaceId) {
      return;
    }
    const cachedAgentId = readWorkflowAgentId(spaceId);
    if (cachedAgentId) {
      setAgentIdState(cachedAgentId);
    }
  }, [agentId, spaceId]);

  useEffect(() => {
    if (!open || agentId || !spaceId) {
      return;
    }
    void (async () => {
      try {
        const r = await fetch(
          '/api/intelligence_api/search/get_draft_intelligence_list',
          {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              space_id: spaceId,
              types: [1, 2, 3, 4],
              size: 20,
              order_by: 0,
            }),
          },
        );
        const data = await r.json();
        const list = data?.data?.intelligences ?? [];
        const pick =
          list.find((it: { basic_info?: { name?: string } }) =>
            /finmallclaw|超级体|super/i.test(it?.basic_info?.name ?? ''),
          ) ?? list[0];
        const id = pick?.basic_info?.id;
        setAgentId(
          String(
            id ||
              (spaceId === DEFAULT_FINMALLCLAW_SPACE_ID
                ? DEFAULT_FINMALLCLAW_AGENT_ID
                : ''),
          ),
        );
      } catch {
        if (spaceId === DEFAULT_FINMALLCLAW_SPACE_ID) {
          setAgentId(DEFAULT_FINMALLCLAW_AGENT_ID);
        }
      }
    })();
  }, [open, agentId, spaceId]);

  useEffect(() => {
    setWorkflowSessionError(false);
    setWorkflowSession(prev => {
      const cached = readWorkflowSession(workflowSessionKey);
      if (cached) {
        return cached;
      }
      return workflowSessionKey ? null : prev;
    });
  }, [workflowSessionKey]);

  useEffect(() => {
    if (!open || !agentId || !spaceId || !workflowId) {
      return;
    }
    const cached = readWorkflowSession(workflowSessionKey);
    if (cached) {
      setWorkflowSession(cached);
      return;
    }
    let cancelled = false;
    void (async () => {
      const title = `${WORKFLOW_AGENT_SESSION_TITLE_PREFIX}${workflowId}`;
      setWorkflowSessionError(false);
      try {
        const listResp = await fetch('/api/super-agent/sessions/list', {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({
            space_id: spaceId,
            bot_id: agentId,
            page: 1,
            page_size: WORKFLOW_AGENT_SESSION_PAGE_SIZE,
          }),
        });
        const listData = await listResp.json();
        const sessions =
          (listData?.data?.sessions as SuperAgentSession[] | undefined) ?? [];
        let session = sessions.find(item => item.title === title);

        if (!session) {
          const createResp = await fetch('/api/super-agent/sessions/create', {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              space_id: spaceId,
              bot_id: agentId,
              title,
            }),
          });
          const createData = await createResp.json();
          session = createData?.data?.session as SuperAgentSession | undefined;
        }

        const conversationId = session
          ? getSuperAgentSessionId(session)
          : '';
        if (!conversationId) {
          throw new Error('missing workflow agent session');
        }
        if (!cancelled) {
          const nextSession = {
            conversationId,
            scene: session?.scene ?? WORKFLOW_AGENT_SCENE_PLAYGROUND,
          };
          writeWorkflowSession(workflowSessionKey, nextSession);
          setWorkflowSession(nextSession);
        }
      } catch {
        if (!cancelled) {
          setWorkflowSessionError(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, agentId, spaceId, workflowId, workflowSessionKey]);

  const prepareSend = useCallback(() => {
    command.resetTags();
    window.dispatchEvent(new Event(WORKFLOW_AGENT_SEND_EVENT));
  }, [command]);

  const buildPayload = useCallback(
    (rawQuery: string) => {
      const testRunRequirement =
        detectWorkflowAgentTestRunRequirement(rawQuery);
      return buildWorkflowAgentPayload({
        command,
        workflowId,
        spaceId,
        rawQuery,
        testRunRequirement,
        isChatflow: globalState.isChatflow,
      });
    },
    [command, spaceId, workflowId, globalState],
  );

  const pluginRegistryList = useMemo(
    () => [
      createWorkflowCanvasPluginRegistry({
        buildPayload,
        prepareSend,
        spaceId,
        workflowId,
      }),
    ],
    [buildPayload, prepareSend, spaceId, workflowId],
  );

  if (!open) {
    return (
      <>
        <WorkflowCanvasExternalMCPCommandBridge
          command={command}
          workflowId={workflowId}
          spaceId={spaceId}
        />
        <button
          type="button"
          onClick={() => setOpen(true)}
          style={{
            position: 'absolute',
            right: 24,
            bottom: 24,
            zIndex: 30,
            width: 48,
            height: 48,
            borderRadius: 24,
            border: 'none',
            cursor: 'pointer',
            background: 'linear-gradient(135deg,#2F6BFF,#7B5Cff)',
            color: '#fff',
            fontSize: 12,
            boxShadow: '0 4px 16px rgba(47,107,255,.4)',
          }}
          title="finmallclaw"
        >
          AI
        </button>
      </>
    );
  }

  return (
    <div
      style={{
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        zIndex: 30,
        width: PANEL_WIDTH,
        display: 'flex',
        flexDirection: 'column',
        background: '#fff',
        borderLeft: '1px solid #e8e8e8',
        boxShadow: '-4px 0 16px rgba(0,0,0,.06)',
        overflow: 'hidden',
      }}
    >
      <WorkflowCanvasExternalMCPCommandBridge
        command={command}
        workflowId={workflowId}
        spaceId={spaceId}
      />
      <ExternalMCPGuideModal
        visible={mcpGuideOpen}
        onClose={() => setMcpGuideOpen(false)}
      />
      {agentId &&
      userInfo?.user_id_str &&
      (!workflowId || workflowSession?.conversationId) ? (
        <BotDebugChatAreaProviderAdapter
          botId={agentId}
          userId={userInfo.user_id_str}
          extraPluginRegistryList={pluginRegistryList}
          initialConversationId={workflowSession?.conversationId}
          initialScene={workflowSession?.scene}
        >
          <SuperChatArea
            messageView="trace"
            title="finmallclaw"
            chatInputTopSlot={
              <div
                style={{
                  pointerEvents: 'none',
                  position: 'absolute',
                  width: 0,
                  height: 0,
                  overflow: 'hidden',
                }}
              >
                <WorkflowCanvasCommandBridge command={command} />
              </div>
            }
            renderChatTitleNode={() => (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Tooltip content="接入外部 AI(Claude Code / Codex 等 MCP 客户端)">
                  <button
                    type="button"
                    onClick={() => setMcpGuideOpen(true)}
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 4,
                      height: 28,
                      padding: '0 10px',
                      borderRadius: 14,
                      border: '1px solid rgba(82,100,154,.16)',
                      background: '#fff',
                      color: 'rgba(15,21,40,.72)',
                      cursor: 'pointer',
                      fontSize: 12,
                      lineHeight: '24px',
                    }}
                  >
                    <IconCozLink fontSize={14} />
                    接入外部AI
                  </button>
                </Tooltip>
                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 14,
                    border: '1px solid rgba(82,100,154,.16)',
                    background: '#fff',
                    color: 'rgba(15,21,40,.72)',
                    cursor: 'pointer',
                    fontSize: 18,
                    lineHeight: '24px',
                  }}
                  title="关闭"
                >
                  ×
                </button>
              </div>
            )}
          />
        </BotDebugChatAreaProviderAdapter>
      ) : (
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 24,
            color: '#999',
            fontSize: 13,
            textAlign: 'center',
          }}
        >
          {workflowSessionError
            ? '工作流 AI 会话创建失败，请稍后重试'
            : '正在连接 finmallclaw 超级体…'}
        </div>
      )}
    </div>
  );
};
