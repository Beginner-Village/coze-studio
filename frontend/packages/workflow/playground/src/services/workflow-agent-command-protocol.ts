export type WorkflowCanvasSurface = 'workflow' | 'conversation';

export type WorkflowCanvasCommandOp =
  | 'get_node_catalog'
  | 'get_node_spec'
  | 'get_node_capability_audit'
  | 'get_canvas_context'
  | 'get_bindable_variables'
  | 'add_node'
  | 'delete_node'
  | 'delete_line'
  | 'connect'
  | 'configure_node'
  | 'set_node_params'
  | 'clear_canvas'
  | 'auto_layout'
  | 'validate'
  | 'test_run'
  | 'run_node_smoke'
  | 'explain_failure';

export interface WorkflowCanvasDiagnostic {
  level: 'info' | 'warning' | 'error';
  message: string;
  node_id?: string;
  node_tag?: string;
  op?: string;
}

export interface WorkflowCanvasCommand {
  op: WorkflowCanvasCommandOp;
  target?: string;
  args?: Record<string, unknown>;
}

export interface WorkflowCanvasCommandEnvelope {
  protocol: 'canvas_automation.v0';
  surface: WorkflowCanvasSurface;
  space_id?: string;
  canvas_id?: string;
  mode: 'browser_live' | 'backend_draft' | 'readonly';
  request_id: string;
  requires_response?: boolean;
  commands: WorkflowCanvasCommand[];
}

export interface WorkflowCanvasCommandResultItem {
  op: WorkflowCanvasCommandOp;
  ok: boolean;
  node_id?: string;
  line_id?: string;
  target?: string;
  message?: string;
  diagnostics: WorkflowCanvasDiagnostic[];
}

export interface WorkflowCanvasCommandResult {
  protocol: 'canvas_automation.v0';
  request_id: string;
  status: 'ok' | 'partial' | 'failed';
  results: WorkflowCanvasCommandResultItem[];
  canvas_context?: string;
  bindable_variables?: string;
  node_smoke_report?: unknown;
  node_smoke_summary?: unknown;
  binding_diagnostics: WorkflowCanvasDiagnostic[];
}

export interface WorkflowCanvasCommandEnvelopeIdentity {
  requestId: string;
  surface: WorkflowCanvasSurface;
  mode: WorkflowCanvasCommandEnvelope['mode'];
  spaceId?: string;
  canvasId?: string;
  requiresResponse?: boolean;
}

export type WorkflowCanvasBrowserCommand =
  | { op: 'addNode'; args: Record<string, unknown>; tag?: string }
  | { op: 'connect'; args: Record<string, unknown> }
  | { op: 'deleteNode'; args: Record<string, unknown> }
  | { op: 'deleteLine'; args: Record<string, unknown> }
  | { op: 'clearCanvas' }
  | { op: 'setNodeParams'; args: Record<string, unknown> }
  | { op: 'configureNode'; args: Record<string, unknown> }
  | { op: 'autoLayout' }
  | { op: 'testRun'; args?: { input?: Record<string, string> } };

interface LegacyWorkflowCanvasAck {
  status?: string;
  op?: string;
  args?: Record<string, unknown>;
}

const COMMAND_DISPLAY_NAMES: Record<WorkflowCanvasCommandOp, string> = {
  get_node_catalog: '读取节点目录',
  get_node_spec: '读取节点规格',
  get_node_capability_audit: '读取节点能力',
  get_canvas_context: '读取画布上下文',
  get_bindable_variables: '读取可绑定变量',
  add_node: '添加节点',
  delete_node: '删除节点',
  delete_line: '删除连线',
  connect: '连接节点',
  configure_node: '配置节点',
  set_node_params: '修改节点参数',
  clear_canvas: '清空画布',
  auto_layout: '优化布局',
  validate: '校验工作流',
  test_run: '试运行工作流',
  run_node_smoke: '节点自测',
  explain_failure: '分析失败原因',
};

const WRITE_OPS = new Set<WorkflowCanvasCommandOp>([
  'add_node',
  'delete_node',
  'delete_line',
  'connect',
  'configure_node',
  'set_node_params',
  'clear_canvas',
  'auto_layout',
  'run_node_smoke',
]);

const PROTOCOL_TO_BROWSER_OP: Partial<
  Record<WorkflowCanvasCommandOp, WorkflowCanvasBrowserCommand['op']>
> = {
  add_node: 'addNode',
  connect: 'connect',
  delete_node: 'deleteNode',
  delete_line: 'deleteLine',
  clear_canvas: 'clearCanvas',
  set_node_params: 'setNodeParams',
  configure_node: 'configureNode',
  auto_layout: 'autoLayout',
  test_run: 'testRun',
};

const isWorkflowCanvasCommandOp = (
  op: string | undefined,
): op is WorkflowCanvasCommandOp =>
  !!op && Object.prototype.hasOwnProperty.call(COMMAND_DISPLAY_NAMES, op);

export const getWorkflowCanvasCommandDisplayName = (op: string): string =>
  isWorkflowCanvasCommandOp(op) ? COMMAND_DISPLAY_NAMES[op] : '执行工具';

export const isWorkflowCanvasWriteOp = (op: string): boolean =>
  isWorkflowCanvasCommandOp(op) && WRITE_OPS.has(op);

export const createWorkflowCanvasCommandEnvelope = (
  commands: WorkflowCanvasCommand[],
  identity: WorkflowCanvasCommandEnvelopeIdentity,
): WorkflowCanvasCommandEnvelope => ({
  protocol: 'canvas_automation.v0',
  surface: identity.surface,
  space_id: identity.spaceId,
  canvas_id: identity.canvasId,
  mode: identity.mode,
  request_id: identity.requestId,
  requires_response: identity.requiresResponse,
  commands,
});

export const normalizeWorkflowCanvasLegacyAck = (
  ack: LegacyWorkflowCanvasAck,
): WorkflowCanvasCommandResultItem => {
  const op = isWorkflowCanvasCommandOp(ack.op) ? ack.op : 'explain_failure';
  const args = ack.args ?? {};
  const target =
    stringArg(args.target) ??
    stringArg(args.node) ??
    stringArg(args.node_tag) ??
    stringArg(args.tag);

  return {
    op,
    ok: ack.status === 'dispatched_to_canvas',
    target,
    message:
      ack.status === 'dispatched_to_canvas'
        ? '已下发到画布执行'
        : '工具结果未按画布协议返回',
    diagnostics: [],
  };
};

export const workflowCanvasCommandToAgentCanvasCommand = (
  command: WorkflowCanvasCommand,
): WorkflowCanvasBrowserCommand | undefined => {
  const browserOp = PROTOCOL_TO_BROWSER_OP[command.op];
  if (!browserOp) {
    return undefined;
  }

  if (browserOp === 'clearCanvas' || browserOp === 'autoLayout') {
    return { op: browserOp };
  }

  if (browserOp === 'addNode') {
    return {
      op: 'addNode',
      tag: command.target,
      args: command.args ?? {},
    };
  }

  if (browserOp === 'configureNode') {
    const args = command.args ?? {};
    return {
      op: 'configureNode',
      args: {
        node: command.target ?? stringArg(args.node) ?? '',
        config: args.config,
      },
    };
  }

  if (browserOp === 'setNodeParams') {
    const args = command.args ?? {};
    return {
      op: 'setNodeParams',
      args: {
        node: command.target ?? stringArg(args.node) ?? '',
        params: args.params,
      },
    };
  }

  if (browserOp === 'deleteNode') {
    const args = command.args ?? {};
    return {
      op: 'deleteNode',
      args: {
        node:
          command.target ??
          stringArg(args.node_tag) ??
          stringArg(args.node) ??
          '',
      },
    };
  }

  if (browserOp === 'testRun') {
    return {
      op: 'testRun',
      args: normalizeTestRunArgs(command.args),
    };
  }

  return {
    op: browserOp,
    args: normalizePortArgs(command.args ?? {}),
  } as WorkflowCanvasBrowserCommand;
};

const normalizeTestRunArgs = (
  args: Record<string, unknown> | undefined,
): { input?: Record<string, string> } | undefined => {
  const input = args?.input;
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    return undefined;
  }
  return { input: input as Record<string, string> };
};

const normalizePortArgs = (
  args: Record<string, unknown>,
): Record<string, unknown> => ({
  ...args,
  fromPort: args.fromPort ?? args.from_port,
  toPort: args.toPort ?? args.to_port,
});

const stringArg = (value: unknown): string | undefined =>
  typeof value === 'string' && value ? value : undefined;
