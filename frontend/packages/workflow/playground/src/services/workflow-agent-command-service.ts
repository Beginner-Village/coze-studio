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

/**
 * WorkflowAgentCommandService —— finmallclaw 操作工作流画布的"指令总线"。
 *
 * 内嵌智能体(finmallclaw,无 sandbox)通过本服务驱动 FlowGram 画布的内存文档模型,
 * 实现"一个个加节点、一条条连线"的动态效果(不走存盘+刷新,直接操作活的画布)。
 *
 * 设计目标:把"加节点 / 连线 / 设参数 / 跑调试"封装成可被 agent 工具调用的原子指令,
 * 每条指令执行后画布由 FlowGram 响应式重渲染,带天然动画。指令之间可插入停顿,
 * 形成用户可见的"现场搭建"过程。
 */
import { inject, injectable } from 'inversify';
import {
  WorkflowDocument,
  WorkflowSelectService,
  AutoLayoutService,
  FlowNodeFormData,
  getAntiOverlapPosition,
  type FormModelV2,
  type WorkflowLinePortInfo,
  type WorkflowNodeEntity,
} from '@flowgram-adapter/free-layout-editor';
import { type IPoint } from '@flowgram-adapter/common';
import { StandardNodeType } from '@coze-workflow/base';
import {
  ValidationService,
  type ValidateErrorMap,
} from '@coze-workflow/base/services';
import {
  type GetWorkFlowProcessData,
  NodeExeStatus,
  type NodeResult,
  WorkflowExeStatus,
} from '@coze-workflow/base/api';

// 直接从文件路径引入,避免经 '../shortcuts' barrel 造成与 services 的循环依赖
// (循环依赖会让 @inject 装饰器拿到 undefined,导致整个工作流编辑器容器初始化失败)。
import { WorkflowPlaygroundContext } from '../workflow-playground-context';
import { getFollowNode } from '../shortcuts/contributions/layout/get-follow-node';
import { WorkflowRunService } from './workflow-run-service';
import { WorkflowLinesService } from './workflow-line-service';
import { WorkflowEditService } from './workflow-edit-service';
import {
  createWorkflowAgentSemanticParams,
  type WorkflowAgentSemanticConfig,
} from './workflow-agent-semantic-config';
import {
  createWorkflowAgentPlaceholderNodeJson,
  isWorkflowAgentSingletonNodeType,
  resolveWorkflowAgentSingletonNodeRef,
} from './workflow-agent-node-json';
import { collectWorkflowAgentClearableNodes } from './workflow-agent-clear-canvas';
import { autoLayoutAfterAgentCommandIfNeeded } from './workflow-agent-command-auto-layout';
import {
  isWorkflowCanvasWriteOp,
  workflowCanvasCommandToAgentCanvasCommand,
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandEnvelope,
  type WorkflowCanvasCommandResult,
  type WorkflowCanvasCommandResultItem,
} from './workflow-agent-command-protocol';
import {
  formatWorkflowAgentBindableVariables,
  summarizeWorkflowAgentCanvas,
  type WorkflowAgentCanvasValidationError,
} from './workflow-agent-canvas-summary';
import { waitForWorkflowAgentSavingIdle } from './workflow-agent-test-run-guard';
import { executeWorkflowCanvasReadonlyCommand } from './workflow-agent-command-readonly';
import { executeWorkflowCanvasNodeSmokeCommand } from './workflow-agent-node-smoke-browser-command';

/** 加节点指令参数 */
export interface AddNodeCommand {
  /** 节点类型,见 StandardNodeType(如 '3'=LLM, '5'=Code, '8'=If, '4'=Api) */
  type: StandardNodeType;
  /** 节点标题(可选) */
  title?: string;
  /** 画布坐标(可选,缺省自动避让排布) */
  position?: IPoint;
  /** 节点初始数据(标题/输入等,Partial<WorkflowNodeJSON>) */
  nodeJson?: Record<string, unknown>;
}

/** 连线指令参数 */
export type ConnectCommand = WorkflowLinePortInfo & {
  from: string;
  to: string;
};

/** 配置节点表单参数 */
export interface SetNodeParamsCommand {
  /** 真实 nodeId 或 addNode 时登记的 node_tag */
  node: string;
  /** 表单路径和值,如 {'nodeMeta.title':'标题','inputs.inputParameters': [...]} */
  params: Record<string, unknown>;
}

/** 语义化配置节点:模型给意图,服务翻译成真实节点表单结构 */
export interface ConfigureNodeCommand {
  /** 真实 nodeId 或 addNode 时登记的 node_tag */
  node: string;
  /** 输入/输出/prompt/return/condition 等语义配置 */
  config: WorkflowAgentSemanticConfig;
}

/** 删除节点指令参数 */
export interface DeleteNodeCommand {
  /** 真实 nodeId 或 addNode 时登记的 node_tag;start/end 不可删除 */
  node: string;
}

/** 删除连线指令参数 */
export type DeleteLineCommand = WorkflowLinePortInfo & {
  from: string;
  to: string;
};

/** 单条可被 agent 下发的指令 */
export type AgentCanvasCommand =
  | { op: 'addNode'; args: AddNodeCommand; tag?: string }
  | { op: 'connect'; args: ConnectCommand }
  | { op: 'deleteNode'; args: DeleteNodeCommand }
  | { op: 'deleteLine'; args: DeleteLineCommand }
  | { op: 'clearCanvas' }
  | { op: 'setNodeParams'; args: SetNodeParamsCommand }
  | { op: 'configureNode'; args: ConfigureNodeCommand }
  | { op: 'autoLayout' }
  | { op: 'testRun'; args?: { input?: Record<string, string> } };

/** 指令执行结果 */
export interface CommandResult {
  ok: boolean;
  /** addNode 时返回新节点 id;tag→id 由调用方据此映射 */
  nodeId?: string;
  deletedCount?: number;
  error?: string;
  testRun?: WorkflowAgentTestRunSummary;
}

export interface WorkflowAgentTestRunNodeSummary {
  nodeId: string;
  nodeName: string;
  nodeType: string;
  status: string;
  errorInfo?: string;
  errorLevel?: string;
  input?: string;
  output?: string;
  rawOutput?: string;
  cost?: string;
}

export interface WorkflowAgentTestRunSummary {
  status: string;
  executeId?: string;
  logId?: string;
  reason?: string;
  lastNodeId?: string;
  cost?: string;
  failedNodes: WorkflowAgentTestRunNodeSummary[];
  outputNodes: WorkflowAgentTestRunNodeSummary[];
  nodes: WorkflowAgentTestRunNodeSummary[];
}

/** 一个节点类型的可读信息(供 agent 选型) */
export interface NodeTypeInfo {
  type: StandardNodeType;
  title: string;
}

const workflowExeStatusText: Record<number, string> = {
  [WorkflowExeStatus.Running]: 'running',
  [WorkflowExeStatus.Success]: 'success',
  [WorkflowExeStatus.Fail]: 'failed',
  [WorkflowExeStatus.Cancel]: 'canceled',
};

const nodeExeStatusText: Record<number, string> = {
  [NodeExeStatus.Waiting]: 'waiting',
  [NodeExeStatus.Running]: 'running',
  [NodeExeStatus.Success]: 'success',
  [NodeExeStatus.Fail]: 'failed',
};

const trimForAgent = (value: unknown, max = 1200): string | undefined => {
  if (value === undefined || value === null || value === '') {
    return undefined;
  }
  const text = typeof value === 'string' ? value : JSON.stringify(value);
  return text.length > max ? `${text.slice(0, max)}...` : text;
};

const summarizeNodeResult = (
  node: NodeResult,
): WorkflowAgentTestRunNodeSummary => ({
  nodeId: node.nodeId,
  nodeName: node.NodeName,
  nodeType: node.NodeType,
  status: nodeExeStatusText[node.nodeStatus] ?? String(node.nodeStatus),
  errorInfo: trimForAgent(node.errorInfo),
  errorLevel: trimForAgent(node.errorLevel, 120),
  input: trimForAgent(node.input),
  output: trimForAgent(node.output),
  rawOutput: trimForAgent(node.raw_output),
  cost: trimForAgent(node.nodeExeCost, 120),
});

const summarizeTestRunResult = (
  result?: GetWorkFlowProcessData,
): WorkflowAgentTestRunSummary | undefined => {
  if (!result) {
    return undefined;
  }
  const nodes = (result.nodeResults ?? []).map(summarizeNodeResult);
  const failedNodes = nodes
    .filter(node => node.status === 'failed' || Boolean(node.errorInfo))
    .slice(0, 8);
  const outputNodes = nodes
    .filter(node => node.nodeType === 'End' || node.nodeType === 'Output')
    .slice(-4);

  return {
    status:
      result.executeStatus === undefined
        ? 'unknown'
        : workflowExeStatusText[result.executeStatus] ??
          String(result.executeStatus),
    executeId: result.executeId,
    logId: result.logID,
    reason: trimForAgent(result.reason),
    lastNodeId: result.lastNodeID,
    cost: trimForAgent(result.workflowExeCost, 120),
    failedNodes,
    outputNodes,
    nodes: nodes.slice(0, 20),
  };
};

@injectable()
export class WorkflowAgentCommandService {
  @inject(WorkflowDocument) private readonly document: WorkflowDocument;

  @inject(WorkflowSelectService)
  private readonly selectService: WorkflowSelectService;

  @inject(WorkflowEditService)
  private readonly editService: WorkflowEditService;

  @inject(WorkflowLinesService)
  private readonly linesService: WorkflowLinesService;

  @inject(WorkflowRunService) private readonly runService: WorkflowRunService;

  @inject(ValidationService)
  private readonly validationService: ValidationService;

  @inject(WorkflowPlaygroundContext)
  private readonly context: WorkflowPlaygroundContext;

  @inject(AutoLayoutService)
  private readonly autoLayoutService: AutoLayoutService;

  /** 节点冒出/连线之间的默认停顿(ms),营造"现场搭建"的动态观感 */
  private pace = 450;

  /** 持久 tag→真实 nodeId 映射(流式工具调用一条条来,需跨调用保持) */
  private readonly tagMap = new Map<string, string>();

  /** 重置 tag 映射(开始新一轮搭建时调用) */
  resetTags(): void {
    this.tagMap.clear();
  }

  /**
   * 执行单条 agent 画布指令(供流式 func_call 逐条驱动)。
   * addNode 会把 tag→真实 id 记入 tagMap;connect 用 tagMap 解析 from/to 的逻辑名。
   */
  async execCommand(cmd: AgentCanvasCommand): Promise<CommandResult> {
    const result = await this.execCommandCore(cmd);
    return await autoLayoutAfterAgentCommandIfNeeded(cmd, result, () =>
      this.autoLayout(),
    );
  }

  async applyCommandEnvelope(
    envelope: WorkflowCanvasCommandEnvelope,
  ): Promise<WorkflowCanvasCommandResult> {
    const results: WorkflowCanvasCommandResultItem[] = [];
    let canvasContext: string | undefined;
    let bindableVariables: string | undefined;
    let nodeSmokeReport: unknown;
    let nodeSmokeSummary: unknown;
    let shouldAutoLayout = false;
    const hasExplicitAutoLayout = envelope.commands.some(
      command => command.op === 'auto_layout',
    );

    for (const command of envelope.commands) {
      const readonlyResult = await executeWorkflowCanvasReadonlyCommand(
        command,
        {
          getCanvasSummary: () => this.getCanvasSummary(),
          getBindableVariablesSummary: () => this.getBindableVariablesSummary(),
        },
      );
      if (readonlyResult) {
        results.push(readonlyResult.item);
        canvasContext = readonlyResult.canvasContext ?? canvasContext;
        bindableVariables =
          readonlyResult.bindableVariables ?? bindableVariables;
        continue;
      }

      const nodeSmokeResult = await executeWorkflowCanvasNodeSmokeCommand(
        command,
        {
          surface: envelope.surface,
          spaceId: envelope.space_id,
          canvasId: envelope.canvas_id,
          service: {
            applyCommandEnvelope: innerEnvelope =>
              this.applyCommandEnvelope(innerEnvelope),
          },
        },
      );
      if (nodeSmokeResult) {
        results.push(nodeSmokeResult.item);
        nodeSmokeReport = nodeSmokeResult.nodeSmokeReport;
        nodeSmokeSummary = nodeSmokeResult.nodeSmokeSummary;
        continue;
      }

      const browserCommand = workflowCanvasCommandToAgentCanvasCommand(command);
      if (!browserCommand) {
        results.push({
          op: command.op,
          ok: false,
          target: command.target,
          message: '当前浏览器画布适配器不支持该命令',
          diagnostics: [
            {
              level: 'error',
              message: '当前浏览器画布适配器不支持该命令',
              op: command.op,
            },
          ],
        });
        continue;
      }

      const result = await this.execCommandCore(
        browserCommand as AgentCanvasCommand,
      );
      if (
        result.ok &&
        command.op !== 'auto_layout' &&
        isWorkflowCanvasWriteOp(command.op)
      ) {
        shouldAutoLayout = true;
      }
      results.push(this.toWorkflowCanvasCommandResultItem(command, result));
    }

    if (shouldAutoLayout && !hasExplicitAutoLayout) {
      const layoutResult = await this.autoLayout();
      results.push(
        this.toWorkflowCanvasCommandResultItem(
          { op: 'auto_layout' },
          layoutResult,
        ),
      );
    }

    return {
      protocol: 'canvas_automation.v0',
      request_id: envelope.request_id,
      status: this.getEnvelopeStatus(results),
      results,
      canvas_context: canvasContext,
      bindable_variables: bindableVariables,
      node_smoke_report: nodeSmokeReport,
      node_smoke_summary: nodeSmokeSummary,
      binding_diagnostics: [],
    };
  }

  private async execCommandCore(
    cmd: AgentCanvasCommand,
  ): Promise<CommandResult> {
    if (cmd.op === 'addNode') {
      const r = await this.addNode(cmd.args);
      if (cmd.tag && r.nodeId) {
        this.tagMap.set(cmd.tag, r.nodeId);
      }
      return r;
    }
    if (cmd.op === 'connect') {
      return this.connect({
        ...cmd.args,
        from: this.resolveNodeRef(cmd.args.from),
        to: this.resolveNodeRef(cmd.args.to),
      });
    }
    if (cmd.op === 'deleteNode') {
      return this.deleteNode({
        node: this.resolveNodeRef(cmd.args.node),
      });
    }
    if (cmd.op === 'deleteLine') {
      return this.deleteLine({
        ...cmd.args,
        from: this.resolveNodeRef(cmd.args.from),
        to: this.resolveNodeRef(cmd.args.to),
      });
    }
    if (cmd.op === 'clearCanvas') {
      return this.clearCanvas();
    }
    if (cmd.op === 'setNodeParams') {
      return this.setNodeParams({
        ...cmd.args,
        node: this.resolveNodeRef(cmd.args.node),
      });
    }
    if (cmd.op === 'configureNode') {
      return this.configureNode({
        ...cmd.args,
        node: this.resolveNodeRef(cmd.args.node),
      });
    }
    if (cmd.op === 'autoLayout') {
      return this.autoLayout();
    }
    if (cmd.op === 'testRun') {
      return this.testRun(cmd.args?.input);
    }
    return { ok: false, error: 'unknown op' };
  }

  private toWorkflowCanvasCommandResultItem(
    command: WorkflowCanvasCommand,
    result: CommandResult,
  ): WorkflowCanvasCommandResultItem {
    return {
      op: command.op,
      ok: result.ok,
      node_id: result.nodeId,
      target: command.target,
      message:
        result.error ??
        (result.deletedCount !== undefined
          ? `deleted ${result.deletedCount} nodes`
          : undefined),
      diagnostics: result.error
        ? [
            {
              level: 'error',
              message: result.error,
              node_id: result.nodeId,
              op: command.op,
            },
          ]
        : [],
    };
  }

  private getEnvelopeStatus(
    results: WorkflowCanvasCommandResultItem[],
  ): WorkflowCanvasCommandResult['status'] {
    const failed = results.filter(result => !result.ok);
    if (!failed.length) {
      return 'ok';
    }
    return failed.length === results.length ? 'failed' : 'partial';
  }

  setPace(ms: number): void {
    this.pace = Math.max(0, ms);
  }

  private delay(): Promise<void> {
    return this.pace > 0
      ? new Promise(resolve => setTimeout(resolve, this.pace))
      : Promise.resolve();
  }

  private resolveNodeRef(ref: string): string {
    return (
      this.tagMap.get(ref) ?? resolveWorkflowAgentSingletonNodeRef(ref) ?? ref
    );
  }

  /** 列出当前工程可用的节点类型(来自 NodeTemplateList 加载的模板表) */
  listNodeTypes(): NodeTypeInfo[] {
    const all = Object.values(StandardNodeType);
    return this.context
      .getTemplateList(all)
      .filter(Boolean)
      .map(t => {
        const template = t as { title?: string; name?: string };
        return {
          type: t.type as StandardNodeType,
          title: template.title ?? template.name ?? String(t.type),
        };
      });
  }

  /** 当前画布的 JSON(节点+连线),供 agent 读取现状以决定下一步 */
  async getCanvasJSON(): Promise<unknown> {
    return await this.document.toJSON();
  }

  /** 当前画布的可读摘要,包含节点输出和可绑定变量。 */
  async getCanvasSummary(): Promise<string> {
    const canvas = await this.getCanvasJSON();
    const validationErrors = await this.collectValidationErrorsForAgent();
    return summarizeWorkflowAgentCanvas(canvas, { validationErrors });
  }

  /**
   * 当前用户选中/正在编辑的节点摘要(id/type/title/当前配置),供超级智能体做指代
   * 消解:用户说"这个节点/这里/它/当前节点"时即指它。无选中时返回空串(不注入)。
   */
  async getSelectedNodeSummary(): Promise<string> {
    try {
      const selected =
        this.selectService?.activatedNode ??
        this.selectService?.selection?.find(
          (e): e is WorkflowNodeEntity =>
            typeof (e as WorkflowNodeEntity)?.id === 'string',
        );
      const id = selected?.id;
      if (!id) {
        return '';
      }
      const canvas = (await this.getCanvasJSON()) as {
        nodes?: unknown[];
        blocks?: unknown[];
      };
      const find = (
        list: unknown[] | undefined,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      ): Record<string, any> | undefined => {
        for (const item of list ?? []) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const node = item as Record<string, any>;
          if (node?.id === id) {
            return node;
          }
          const inner = find(node?.blocks as unknown[] | undefined);
          if (inner) {
            return inner;
          }
        }
        return undefined;
      };
      const found = find(canvas?.nodes) ?? find(canvas?.blocks);
      const type = String(found?.type ?? '');
      const title = String(found?.data?.nodeMeta?.title ?? '');
      const dataStr = JSON.stringify(found?.data ?? {});
      const cfg =
        dataStr.length > 1500 ? `${dataStr.slice(0, 1500)}…(已截断)` : dataStr;
      return `id=${id} type=${type} title=${title}\n当前配置: ${cfg}`;
    } catch {
      return '';
    }
  }

  /** 当前画布可供 configure_node/End/VariableMerge 绑定的变量候选。 */
  async getBindableVariablesSummary(): Promise<string> {
    return formatWorkflowAgentBindableVariables(await this.getCanvasJSON());
  }

  private async collectValidationErrorsForAgent(): Promise<
    Record<string, WorkflowAgentCanvasValidationError[]> | undefined
  > {
    this.validationService.validating = true;
    try {
      const workflowId = this.runService.globalState.workflowId;
      const { hasError: hasFrontendError, nodeErrorMap } =
        await this.validationService.validateWorkflow();
      if (hasFrontendError) {
        this.setValidationErrorsForCurrentWorkflow(nodeErrorMap);
        return nodeErrorMap;
      }

      const { hasError: hasSchemaError, errors } =
        await this.validationService.validateSchemaV2();
      if (hasSchemaError) {
        this.validationService.setErrorsV2(errors);
        this.linesService.validateAllLine();
        return (
          errors[workflowId]?.errors ??
          Object.values(errors).find(item => Object.keys(item.errors).length)
            ?.errors
        );
      }

      this.validationService.clearErrors();
      this.linesService.validateAllLine();
      return {};
    } catch {
      return undefined;
    } finally {
      this.validationService.validating = false;
    }
  }

  private setValidationErrorsForCurrentWorkflow(errors: ValidateErrorMap): void {
    const workflowId = this.runService.globalState.workflowId;
    if (workflowId) {
      this.validationService.setErrorsV2({
        [workflowId]: {
          workflowId,
          errors,
        },
      });
    } else {
      this.validationService.setErrors(errors, true);
    }
    this.linesService.validateAllLine();
  }

  /**
   * 加一个节点;返回新节点 id。位置缺省时自动避让排布。
   *
   * nodeJson 仅在"完整"(含 data.inputs)时才透传给引擎——否则用节点 registry 的
   * 默认数据创建,避免传入半成品(如只给 title)覆盖掉必需的默认 inputs 导致节点非法/
   * 无法序列化与调试。需要标题/参数时请传完整 nodeJson(可先据节点模板默认补全)。
   */
  async addNode(cmd: AddNodeCommand): Promise<CommandResult> {
    try {
      if (isWorkflowAgentSingletonNodeType(String(cmd.type))) {
        return {
          ok: false,
          error:
            'Start/End nodes are singleton nodes. Use existing start/end nodes instead of addNode.',
        };
      }
      const basePosition: IPoint = cmd.position ?? { x: 240, y: 200 };
      const position = getAntiOverlapPosition(this.document, basePosition);
      const nodeJson =
        cmd.nodeJson ??
        createWorkflowAgentPlaceholderNodeJson(cmd.type, cmd.title);
      const data = (nodeJson as { data?: { inputs?: unknown } } | undefined)
        ?.data;
      const isComplete = !!data && data.inputs !== undefined;
      const node: WorkflowNodeEntity | undefined = isComplete
        ? await this.document.createWorkflowNodeByType(
            cmd.type,
            position,
            nodeJson as never,
          )
        : await this.document.createWorkflowNodeByType(cmd.type, position);
      if (node && cmd.title) {
        this.applyNodeParams(node, { 'nodeMeta.title': cmd.title });
      }
      await this.delay();
      return { ok: true, nodeId: node?.id };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 更新节点表单参数;params 的 key 是 FormModelV2 path。 */
  async setNodeParams(cmd: SetNodeParamsCommand): Promise<CommandResult> {
    try {
      const nodeId = this.resolveNodeRef(cmd.node);
      const node = this.document.getNode(nodeId);
      if (!node) {
        return { ok: false, error: `node ${nodeId} not found` };
      }
      this.applyNodeParams(node, cmd.params);
      await this.delay();
      return { ok: true, nodeId: node.id };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 用模型友好的语义参数配置节点,避免模型直接写内部表单 path。 */
  async configureNode(cmd: ConfigureNodeCommand): Promise<CommandResult> {
    try {
      const nodeId = this.resolveNodeRef(cmd.node);
      const node = this.document.getNode(nodeId);
      if (!node) {
        return { ok: false, error: `node ${nodeId} not found` };
      }
      const params = createWorkflowAgentSemanticParams({
        nodeType: String(node.flowNodeType),
        config: cmd.config,
        resolveNodeRef: ref => this.resolveNodeRef(ref),
      });
      this.applyNodeParams(node, params);
      await this.delay();
      return { ok: true, nodeId: node.id };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  private applyNodeParams(
    node: WorkflowNodeEntity,
    params: Record<string, unknown>,
  ): void {
    const formData = node.getData<FlowNodeFormData>(FlowNodeFormData);
    const formModel = formData.getFormModel<FormModelV2>();
    Object.entries(params).forEach(([path, value]) => {
      // 绑定/参数类 path 的值由语义层完整产出(整段 inputParameters 数组/map、
      // apiParam、fcParam、condition、mergeGroups、outputs)。其表单默认值可能是
      // 另一种形态(如插件节点 apiDetail 未加载时 inputs.inputParameters 是数组,而我们
      // 写的是 map),会让 normalizeFormParamValue 返回 undefined 被静默跳过——
      // 这正是"输入参数绑定有时改不成功"的根因。这类 path 直接全量写入。
      if (value !== undefined && this.isFullOverwritePath(path)) {
        formModel.setValueIn(path, value);
        return;
      }
      const current = formModel.getValueIn?.(path);
      const next = this.normalizeFormParamValue(current, value);
      if (next === undefined && current !== undefined) {
        return;
      }
      formModel.setValueIn(path, next);
    });
  }

  /** 语义层完整拥有取值的 path:总是全量写,绝不形态合并/跳过。 */
  private isFullOverwritePath(path: string): boolean {
    return (
      path.endsWith('inputParameters') ||
      path === 'inputs.apiParam' ||
      path === 'fcParam' ||
      path === 'inputs.fcParam' ||
      path === 'condition' ||
      path === 'inputs.mergeGroups' ||
      path === 'outputs' ||
      path.startsWith('inputs.inputParameters.') ||
      path === 'inputs.databaseInfoList' ||
      path === 'inputs.sql' ||
      path.startsWith('inputs.selectParam') ||
      path.startsWith('inputs.insertParam') ||
      path.startsWith('inputs.updateParam') ||
      path.startsWith('inputs.deleteParam') ||
      path === 'concatResult' ||
      path === 'concatChar' ||
      path === 'inputs.content' ||
      path === 'inputs.streamingOutput' ||
      path.endsWith('.prompt') ||
      path.endsWith('.systemPrompt')
    );
  }

  private normalizeFormParamValue(current: unknown, value: unknown): unknown {
    if (Array.isArray(current)) {
      return Array.isArray(value) ? value : undefined;
    }
    if (this.isPlainRecord(current)) {
      return this.isPlainRecord(value) ? { ...current, ...value } : undefined;
    }
    if (
      current !== undefined &&
      current !== null &&
      value !== null &&
      typeof current !== typeof value
    ) {
      return undefined;
    }
    return value;
  }

  private isPlainRecord(v: unknown): v is Record<string, unknown> {
    return !!v && typeof v === 'object' && !Array.isArray(v);
  }

  /** 连一条线(from→to);端口缺省时用节点默认端口。 */
  async connect(cmd: ConnectCommand): Promise<CommandResult> {
    try {
      this.linesService.createLine({
        ...cmd,
        from: this.resolveNodeRef(cmd.from),
        to: this.resolveNodeRef(cmd.to),
      });
      await this.delay();
      return { ok: true };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 删除指定节点;保留工作流内置 Start/End。 */
  async deleteNode(cmd: DeleteNodeCommand): Promise<CommandResult> {
    try {
      const nodeId = this.resolveNodeRef(cmd.node);
      const node = this.document.getNode(nodeId);
      if (!node) {
        return { ok: false, error: `node ${nodeId} not found` };
      }
      if (isWorkflowAgentSingletonNodeType(String(node.flowNodeType))) {
        return { ok: false, error: 'Start/End singleton nodes cannot be deleted.' };
      }
      if (!this.document.canRemove(node)) {
        return { ok: false, error: `node ${nodeId} cannot be deleted` };
      }
      this.editService.deleteNode(node, true);
      this.deleteTagsByNodeId(nodeId);
      await this.delay();
      return { ok: true, nodeId };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 删除指定连线;端口为空时删除 from→to 间所有匹配连线。 */
  async deleteLine(cmd: DeleteLineCommand): Promise<CommandResult> {
    try {
      const from = this.resolveNodeRef(cmd.from);
      const to = this.resolveNodeRef(cmd.to);
      const lines = this.linesService.getAllLines().filter(line => {
        const fromPortMatched =
          cmd.fromPort === undefined || line.info.fromPort === cmd.fromPort;
        const toPortMatched =
          cmd.toPort === undefined || line.info.toPort === cmd.toPort;
        return (
          line.info.from === from &&
          line.info.to === to &&
          fromPortMatched &&
          toPortMatched
        );
      });
      if (!lines.length) {
        return { ok: false, error: `line ${from} -> ${to} not found` };
      }
      const deleted = lines.filter(line => this.linesService.deleteLine(line));
      await this.delay();
      return deleted.length
        ? { ok: true }
        : { ok: false, error: `line ${from} -> ${to} cannot be deleted` };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 清空画布中除 Start/End 外的所有节点,用于复杂需求重新设计。 */
  async clearCanvas(): Promise<CommandResult> {
    try {
      const nodes = collectWorkflowAgentClearableNodes(
        this.document.getAllNodes(),
      );
      let deletedCount = 0;
      const failedNodeIds: string[] = [];
      nodes.forEach(node => {
        try {
          this.editService.deleteNode(node, true);
          deletedCount += 1;
        } catch {
          failedNodeIds.push(node.id);
        }
      });
      this.tagMap.clear();
      await this.delay();
      if (failedNodeIds.length) {
        return {
          ok: false,
          deletedCount,
          error: `failed to delete nodes: ${failedNodeIds.join(',')}`,
        };
      }
      return { ok: true, deletedCount };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  private deleteTagsByNodeId(nodeId: string): void {
    Array.from(this.tagMap.entries()).forEach(([tag, id]) => {
      if (tag === nodeId || id === nodeId) {
        this.tagMap.delete(tag);
      }
    });
  }

  /** 一键优化布局:整理节点排布、连线清晰(等价编辑器底部"优化布局"按钮)。 */
  async autoLayout(): Promise<CommandResult> {
    try {
      await this.autoLayoutService.layout({ getFollowNode });
      await this.delay();
      return { ok: true };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /** 触发整图试运行(完整调试);结果由 WorkflowRunService 的轮询状态驱动 UI。 */
  async testRun(input?: Record<string, string>): Promise<CommandResult> {
    try {
      const ready = await waitForWorkflowAgentSavingIdle({
        isSaving: () => Boolean(this.runService.globalState.config.saving),
      });
      if (!ready) {
        return { ok: false, error: 'workflow is still saving, test_run skipped' };
      }
      const testRunResult = await this.runService.testRun(
        input,
        undefined,
        undefined,
        { skipGlobalReload: true },
      );
      const summary = summarizeTestRunResult(testRunResult);
      return {
        ok: summary?.status ? summary.status === 'success' : true,
        error:
          summary && summary.status !== 'success'
            ? summary.reason || summary.failedNodes[0]?.errorInfo
            : undefined,
        testRun: summary,
      };
    } catch (e) {
      return { ok: false, error: (e as Error).message };
    }
  }

  /**
   * 顺序执行一批指令——agent 一次下发一段"搭建脚本",这里逐条执行、逐条停顿,
   * 用户即可在画布上看到节点一个个出现、线一条条连上。
   * addNode 用的 tag 会被记录到 tagToId,供后续 connect 引用逻辑名而非真实 id。
   */
  async runScript(
    commands: AgentCanvasCommand[],
  ): Promise<{ results: CommandResult[]; tagToId: Record<string, string> }> {
    const results: CommandResult[] = [];
    const tagToId: Record<string, string> = {};
    for (const cmd of commands) {
      if (cmd.op === 'addNode') {
        const r = await this.addNode(cmd.args);
        if (cmd.tag && r.nodeId) {
          tagToId[cmd.tag] = r.nodeId;
        }
        results.push(r);
      } else if (cmd.op === 'connect') {
        // 允许 from/to 使用 addNode 时登记的 tag
        const resolved: ConnectCommand = {
          ...cmd.args,
          from:
            tagToId[cmd.args.from] ??
            resolveWorkflowAgentSingletonNodeRef(cmd.args.from) ??
            cmd.args.from,
          to:
            tagToId[cmd.args.to] ??
            resolveWorkflowAgentSingletonNodeRef(cmd.args.to) ??
            cmd.args.to,
        };
        results.push(await this.connect(resolved));
      } else if (cmd.op === 'deleteNode') {
        results.push(
          await this.deleteNode({
            node:
              tagToId[cmd.args.node] ??
              resolveWorkflowAgentSingletonNodeRef(cmd.args.node) ??
              cmd.args.node,
          }),
        );
      } else if (cmd.op === 'deleteLine') {
        results.push(
          await this.deleteLine({
            ...cmd.args,
            from:
              tagToId[cmd.args.from] ??
              resolveWorkflowAgentSingletonNodeRef(cmd.args.from) ??
              cmd.args.from,
            to:
              tagToId[cmd.args.to] ??
              resolveWorkflowAgentSingletonNodeRef(cmd.args.to) ??
              cmd.args.to,
          }),
        );
      } else if (cmd.op === 'clearCanvas') {
        results.push(await this.clearCanvas());
        Object.keys(tagToId).forEach(tag => {
          delete tagToId[tag];
        });
      } else if (cmd.op === 'setNodeParams') {
        results.push(
          await this.setNodeParams({
            ...cmd.args,
            node:
              tagToId[cmd.args.node] ??
              resolveWorkflowAgentSingletonNodeRef(cmd.args.node) ??
              cmd.args.node,
          }),
        );
      } else if (cmd.op === 'configureNode') {
        results.push(
          await this.configureNode({
            ...cmd.args,
            node:
              tagToId[cmd.args.node] ??
              resolveWorkflowAgentSingletonNodeRef(cmd.args.node) ??
              cmd.args.node,
          }),
        );
      } else if (cmd.op === 'autoLayout') {
        results.push(await this.autoLayout());
      } else if (cmd.op === 'testRun') {
        results.push(await this.testRun(cmd.args?.input));
      }
    }
    // 脚本执行完自动整理布局,保证新增节点排布整齐(除非脚本里已显式 autoLayout)。
    if (!commands.some(c => c.op === 'autoLayout')) {
      results.push(await this.autoLayout());
    }
    return { results, tagToId };
  }
}
