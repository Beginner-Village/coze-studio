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

interface UnknownRecord {
  [key: string]: unknown;
}

export interface WorkflowAgentPluginApiResource {
  id: string;
  name?: string;
  description?: string;
  functionName?: string;
  parameters?: string[];
}

export interface WorkflowAgentPluginResource {
  id: string;
  name?: string;
  description?: string;
  type?: string;
  apis: WorkflowAgentPluginApiResource[];
}

export interface WorkflowAgentKnowledgeResource {
  id: string;
  name?: string;
  description?: string;
  formatType?: string;
  status?: string;
}

export interface WorkflowAgentAgentResource {
  id: string;
  name?: string;
  description?: string;
  platform: 'coze' | 'hiagent';
  status?: string;
}

export interface WorkflowAgentResourceSummary {
  plugins: WorkflowAgentPluginResource[];
  knowledgeBases: WorkflowAgentKnowledgeResource[];
  agents: WorkflowAgentAgentResource[];
  warnings: string[];
}

export interface WorkflowAgentResourceDiscoveryDeps {
  getPlugins?: (spaceId: string) => Promise<unknown[]>;
  getKnowledgeBases?: (spaceId: string) => Promise<unknown[]>;
  getCozeAgents?: (spaceId: string) => Promise<unknown[]>;
  getHiAgents?: (spaceId: string) => Promise<unknown[]>;
}

const RESOURCE_LIMIT = {
  plugins: 24,
  pluginApis: 8,
  apiParams: 8,
  knowledgeBases: 40,
  agents: 40,
};

const PLUGIN_TYPES = {
  PLUGIN: 1,
  APP: 2,
  FUNC: 3,
  WORKFLOW: 4,
  IMAGEFLOW: 5,
  LOCAL: 6,
} as const;

const INTELLIGENCE_STATUS = {
  DRAFT: 1,
  PUBLISHED: 3,
  REVIEWING: 4,
} as const;

const INTELLIGENCE_TYPES = {
  SINGLE_AGENT: 1,
  BOT: 2,
} as const;

const AVAILABLE_INTELLIGENCE_STATUS = [
  INTELLIGENCE_STATUS.DRAFT,
  INTELLIGENCE_STATUS.PUBLISHED,
  INTELLIGENCE_STATUS.REVIEWING,
];
const COZE_AGENT_TYPES = [
  INTELLIGENCE_TYPES.SINGLE_AGENT,
  INTELLIGENCE_TYPES.BOT,
];

export const WORKFLOW_AGENT_NODE_BINDING_GUIDE = [
  '节点绑定提示:',
  '每个工作流已有且只能有一个开始节点 start(100001,type=1) 和一个结束节点 end(900001,type=2);禁止 add_node 创建 type=1/type=2,只能 connect 引用它们。',
  '可新增节点类型: 3=大模型, 4=插件/API, 5=代码, 6=知识库检索, 8=IF, 9=子工作流, 13=输出/纯输出, 15=文本处理, 18=问答/追问, 21=循环, 22=意图识别, 27=知识库写入, 28=批处理, 30=输入, 32=变量聚合, 42=更新数据, 43=查询数据, 44=删除数据, 45=HTTP, 46=新增数据, 58=JSON序列化, 59=JSON解析, 61=MCP工具, 99=卡片选择, 100=智能体。',
  '必须先理解当前画布上下文和可绑定变量: 节点输入只能绑定 start.input 或上游节点已声明的 outputs,不要编造变量。',
  '执行顺序必须是:先 add_node 创建节点,再 connect 连线,然后 configure_node 逐个配置;从输入到输出逐节点确认 input/inputs、prompt/text/template、condition、merge_groups、outputs 和 End returns 都已绑定或声明;配置 condition/merge_groups/returns 前必须先调用 workflow_canvas_get_bindable_variables。',
  '优先使用 workflow_canvas_configure_node 配置节点: input/inputs 绑定输入变量, outputs 声明输出变量, returns 配置结束节点返回变量, prompt/code/condition 配置节点逻辑。',
  '每次完成一组添加节点、连线、配置或清理操作后,必须调用 workflow_canvas_get_canvas_context 做绑定审计:按从 Start 到 End 的路径逐节点确认输入绑定、prompt/text/template/condition 引用、分支出口、变量聚合和 End returns 都只使用可绑定变量;如果 canvas_context 里的绑定诊断不是 none,禁止说完成,必须按诊断局部修复对应节点;注意 canvas_context 是用户消息发送前快照,同一轮刚下发的画布操作可能尚未反映,不要把旧快照误判为空画布,要结合本轮 node_tag/outputs/returns 自检;发现未绑定/未定义/旧节点残留就继续修复,不要直接宣称完成。',
  '试运行失败或校验报错时禁止默认 clear_canvas;除非用户明确要求整体重做,否则先定位失败节点并局部修复输入绑定、变量引用、outputs、merge_groups 或 End returns。常见 BlockID is empty/引用变量不存在,优先补 configure_node 的 input/inputs/returns/merge_groups,不是清空重建。',
  'LLM节点(type=3): configure_node 至少设置 input、prompt 或 user_prompt、outputs; prompt 中可用 {{input变量名}}。',
  'Code节点(type=5): 当前只支持 Python; configure_node 至少设置 input、language="python"、code、outputs; code 必须写 async def main(args: Args) -> Output,通过 params=args.params or {}; params.get("入参名") 取值,不要对 args 直接 strip/get/[]; return 字段必须和 outputs 名称一致。',
  'IF节点(type=8): configure_node 设置 condition.left/operator/right; true/false 分支连线使用 from_port=true 或 from_port=false。',
  '变量聚合节点(type=32): 分支汇合时使用; configure_node 设置 merge_groups=[{name:"output",variables:[{from:"分支节点",output:"变量"}]}],variables 必须全部来自 get_canvas_context 的可绑定变量;所有可能返回的分支末端输出都要放进同一个 output 组;End 绑定聚合节点 output。不要聚合 type=13 输出/消息节点,因为它不是稳定的下游变量来源;分支固定文案先用 type=15 文本处理产出 output:string,再聚合文本处理节点 output。',
  '输出节点(type=13): 纯输出/消息输出, 不调用模型; configure_node 设置 input/inputs、content/text/template 和 streaming_output;content 只能引用本节点 input/inputs 里定义的 name,例如 input name 为 result 时写 content="结果：{{result}}";它主要用于向用户展示消息,不要作为变量聚合或 End returns 的上游变量来源。',
  '输入节点(type=30): 声明额外工作流入参; configure_node 设置 outputs=[{name:"user_id",type:"string",description:"用户ID"}]。',
  '文本处理节点(type=15): configure_node 设置 method="concat" 或 "split"; concat 用 content/template 生成 output, split 用 delimiter 拆分为 array_string;content/template 只能引用本节点 input/inputs 的 name;固定文案没有变量引用时可以删除默认 input,但仍要声明 output:string 供聚合或 End returns 使用。',
  '问答节点(type=18): 用于缺槽追问; configure_node 设置 question/content、answer_type(text/option)、options、limit 和必要 input 绑定。',
  'JSON序列化节点(type=58): configure_node 绑定 input/inputs, 默认输出 output:string。',
  '卡片选择节点(type=99): configure_node 设置 selected_card/card_id、content 和模板变量绑定;资源不确定时先标注待确认。',
  'End节点(end/900001): 支持返回变量和返回文本。返回变量时 configure_node 设置 returns,例如 returns=[{name:"output",from:"merge",output:"output"}]。返回文本时 configure_node 设置 input/inputs、content/text/template 和 streaming_output=true,content 可用 {{变量名}} 拼多个输出变量,例如 input=[{name:"intent",from:"intent",output:"category"},{name:"answer",from:"merge",output:"output"}],content="业务: {{intent}}\\n结果: {{answer}}"。最终必须只引用可绑定变量,不要返回未定义变量。',
  '插件/API节点(type=4): 标题写插件/API名称, configure_node config 中记录 plugin_id/api_id/plugin_name/api_name 和 inputs;资源 id 必须来自资源清单。',
  '知识库节点(type=6): 标题写知识库名称, configure_node config 中记录 dataset_id/dataset_ids/query/input 和 outputs;资源 id 必须来自资源清单。',
  '智能体节点(type=100): 标题写智能体名称, configure_node config 中记录 agent_id/platform/input 和 outputs;资源 id 必须来自资源清单。',
].join('\n');

const isRecord = (value: unknown): value is UnknownRecord =>
  !!value && typeof value === 'object' && !Array.isArray(value);

const asString = (value: unknown): string | undefined => {
  if (value === undefined || value === null) {
    return undefined;
  }
  const text = String(value).trim();
  return text || undefined;
};

const asArray = (value: unknown): unknown[] =>
  Array.isArray(value) ? value : [];

const formatError = (error: unknown): string => {
  if (error instanceof Error) {
    return error.message;
  }
  return typeof error === 'string' ? error : 'unknown error';
};

const enumName = <T extends Record<string, string | number>>(
  enumObj: T,
  value: unknown,
): string | undefined => {
  const raw = typeof value === 'number' ? value : Number(value);
  const entry = Object.entries(enumObj).find(([, v]) => v === raw);
  return entry?.[0];
};

const defaultGetPlugins = async (spaceId: string): Promise<unknown[]> => {
  const { PluginDevelopApi } = await import('@coze-arch/bot-api');
  const response = await PluginDevelopApi.GetPlaygroundPluginList(
    {
      space_id: spaceId,
      page: 1,
      size: RESOURCE_LIMIT.plugins,
      plugin_types: [
        PLUGIN_TYPES.PLUGIN,
        PLUGIN_TYPES.APP,
        PLUGIN_TYPES.FUNC,
        PLUGIN_TYPES.WORKFLOW,
        PLUGIN_TYPES.IMAGEFLOW,
        PLUGIN_TYPES.LOCAL,
      ],
    },
    {
      __disableErrorToast: true,
    },
  );
  return response.data?.plugin_list ?? [];
};

const defaultGetKnowledgeBases = async (
  spaceId: string,
): Promise<unknown[]> => {
  const { KnowledgeApi } = await import('@coze-arch/bot-api');
  const response = await KnowledgeApi.ListDataset(
    {
      space_id: spaceId,
      page: 1,
      size: RESOURCE_LIMIT.knowledgeBases,
      filter: {},
    },
    {
      __disableErrorToast: true,
    },
  );
  return response.dataset_list ?? [];
};

const fetchJSON = async (
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<unknown> => {
  const response = await fetch(input, {
    credentials: 'include',
    ...init,
  });
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return response.json();
};

const defaultGetCozeAgents = async (spaceId: string): Promise<unknown[]> => {
  const result = await fetchJSON(
    '/api/intelligence_api/search/get_draft_intelligence_list',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Agw-Js-Conv': 'str',
      },
      body: JSON.stringify({
        space_id: String(spaceId),
        name: '',
        status: AVAILABLE_INTELLIGENCE_STATUS,
        types: COZE_AGENT_TYPES,
        search_scope: 0,
        order_by: 0,
        size: RESOURCE_LIMIT.agents,
      }),
    },
  );
  if (isRecord(result) && result.code !== undefined && result.code !== 0) {
    throw new Error(asString(result.msg) ?? 'Failed to fetch Coze agents');
  }
  return isRecord(result) && isRecord(result.data)
    ? asArray(result.data.intelligences)
    : [];
};

const defaultGetHiAgents = async (spaceId: string): Promise<unknown[]> => {
  const result = await fetchJSON(
    `/api/space/${spaceId}/hi-agents?page_size=${RESOURCE_LIMIT.agents}`,
    {
      method: 'GET',
    },
  );
  if (isRecord(result) && result.code !== undefined && result.code !== 0) {
    throw new Error(asString(result.msg) ?? 'Failed to fetch HiAgent list');
  }
  return isRecord(result) ? asArray(result.agents) : [];
};

const normalizeApiParams = (api: UnknownRecord): string[] =>
  asArray(api.parameters ?? api.request_params)
    .slice(0, RESOURCE_LIMIT.apiParams)
    .map(param => {
      if (!isRecord(param)) {
        return undefined;
      }
      const name = asString(param.name ?? param.key);
      const type = asString(param.type ?? param.schema_type);
      const description = asString(param.desc ?? param.description);
      return [name, type, description].filter(Boolean).join(':');
    })
    .filter(Boolean) as string[];

const normalizePlugins = (items: unknown[]): WorkflowAgentPluginResource[] =>
  items
    .map(item => {
      if (!isRecord(item)) {
        return undefined;
      }
      const id = asString(item.id ?? item.plugin_id);
      if (!id) {
        return undefined;
      }
      const apis = asArray(item.plugin_apis ?? item.apis)
        .slice(0, RESOURCE_LIMIT.pluginApis)
        .map(api => {
          if (!isRecord(api)) {
            return undefined;
          }
          const apiId = asString(api.api_id ?? api.record_id ?? api.id);
          const name = asString(api.name ?? api.function_name);
          if (!apiId && !name) {
            return undefined;
          }
          return {
            id: apiId ?? name ?? '',
            name,
            description: asString(api.desc ?? api.description),
            functionName: asString(api.function_name),
            parameters: normalizeApiParams(api),
          } satisfies WorkflowAgentPluginApiResource;
        })
        .filter(Boolean) as WorkflowAgentPluginApiResource[];
      return {
        id,
        name: asString(item.name ?? item.plugin_name),
        description: asString(
          item.desc_for_human ?? item.desc ?? item.description,
        ),
        type:
          enumName(PLUGIN_TYPES, item.plugin_type) ??
          asString(item.plugin_type ?? item.type),
        apis,
      } satisfies WorkflowAgentPluginResource;
    })
    .filter(Boolean) as WorkflowAgentPluginResource[];

const normalizeKnowledgeBases = (
  items: unknown[],
): WorkflowAgentKnowledgeResource[] =>
  items
    .map(item => {
      if (!isRecord(item)) {
        return undefined;
      }
      const id = asString(item.dataset_id ?? item.id);
      if (!id) {
        return undefined;
      }
      return {
        id,
        name: asString(item.name),
        description: asString(item.description),
        formatType: asString(item.format_type),
        status: asString(item.status),
      } satisfies WorkflowAgentKnowledgeResource;
    })
    .filter(Boolean) as WorkflowAgentKnowledgeResource[];

const normalizeAgents = (
  items: unknown[],
  platform: 'coze' | 'hiagent',
): WorkflowAgentAgentResource[] =>
  items
    .map(item => {
      if (!isRecord(item)) {
        return undefined;
      }
      const basic = isRecord(item.basic_info) ? item.basic_info : item;
      const id = asString(basic.id ?? item.id ?? item.agent_id);
      if (!id) {
        return undefined;
      }
      return {
        id,
        name: asString(basic.name ?? item.name),
        description: asString(basic.description ?? item.description),
        platform,
        status: asString(basic.status ?? item.status),
      } satisfies WorkflowAgentAgentResource;
    })
    .filter(Boolean) as WorkflowAgentAgentResource[];

export const discoverWorkflowAgentResources = async (
  spaceId?: string,
  deps: WorkflowAgentResourceDiscoveryDeps = {},
): Promise<WorkflowAgentResourceSummary> => {
  const warnings: string[] = [];
  const base: WorkflowAgentResourceSummary = {
    plugins: [],
    knowledgeBases: [],
    agents: [],
    warnings,
  };
  if (!spaceId) {
    warnings.push('space_id 未知,跳过资源发现');
    return base;
  }

  const load = async <T>(
    label: string,
    query: () => Promise<unknown[]>,
    normalize: (items: unknown[]) => T[],
  ): Promise<T[]> => {
    try {
      return normalize(await query());
    } catch (error) {
      warnings.push(`${label}: ${formatError(error)}`);
      return [];
    }
  };

  const [plugins, knowledgeBases, cozeAgents, hiAgents] = await Promise.all([
    load(
      '插件/API',
      () => (deps.getPlugins ?? defaultGetPlugins)(spaceId),
      normalizePlugins,
    ),
    load(
      '知识库',
      () => (deps.getKnowledgeBases ?? defaultGetKnowledgeBases)(spaceId),
      normalizeKnowledgeBases,
    ),
    load(
      'Coze智能体',
      () => (deps.getCozeAgents ?? defaultGetCozeAgents)(spaceId),
      items => normalizeAgents(items, 'coze'),
    ),
    load(
      'HiAgent智能体',
      () => (deps.getHiAgents ?? defaultGetHiAgents)(spaceId),
      items => normalizeAgents(items, 'hiagent'),
    ),
  ]);

  return {
    plugins,
    knowledgeBases,
    agents: [...cozeAgents, ...hiAgents],
    warnings,
  };
};

const lineWithDescription = ({
  label,
  id,
  description,
}: {
  label?: string;
  id: string;
  description?: string;
}): string =>
  description
    ? `${label || id} (${id}): ${description}`
    : `${label || id} (${id})`;

const limitText = (value: string, max: number): string =>
  value.length > max ? `${value.slice(0, max)}...` : value;

export const formatWorkflowAgentResourceSummary = (
  resources: WorkflowAgentResourceSummary,
  maxLength = 9000,
): string => {
  const lines: string[] = [];
  if (resources.warnings.length) {
    lines.push(`资源发现警告: ${resources.warnings.join('; ')}`);
  }

  lines.push('插件/API:');
  if (resources.plugins.length) {
    resources.plugins.slice(0, RESOURCE_LIMIT.plugins).forEach(plugin => {
      lines.push(
        `- ${lineWithDescription({
          label: plugin.name,
          id: `plugin_id=${plugin.id}${
            plugin.type ? `,type=${plugin.type}` : ''
          }`,
          description: plugin.description,
        })}`,
      );
      plugin.apis.slice(0, RESOURCE_LIMIT.pluginApis).forEach(api => {
        const params = api.parameters?.length
          ? `; params=${api.parameters.join(',')}`
          : '';
        lines.push(
          `  - API ${lineWithDescription({
            label: api.name ?? api.functionName,
            id: `api_id=${api.id}`,
            description: `${api.description ?? ''}${params}`,
          })}`,
        );
      });
    });
  } else {
    lines.push('- 未发现可用插件/API');
  }

  lines.push('知识库:');
  if (resources.knowledgeBases.length) {
    resources.knowledgeBases
      .slice(0, RESOURCE_LIMIT.knowledgeBases)
      .forEach(dataset => {
        const meta = [dataset.formatType, dataset.status]
          .filter(Boolean)
          .join(',');
        lines.push(
          `- ${lineWithDescription({
            label: dataset.name,
            id: `dataset_id=${dataset.id}${meta ? `,meta=${meta}` : ''}`,
            description: dataset.description,
          })}`,
        );
      });
  } else {
    lines.push('- 未发现可用知识库');
  }

  lines.push('智能体:');
  if (resources.agents.length) {
    resources.agents.slice(0, RESOURCE_LIMIT.agents).forEach(agent => {
      lines.push(
        `- ${lineWithDescription({
          label: agent.name,
          id: `agent_id=${agent.id},platform=${agent.platform}`,
          description: agent.description,
        })}`,
      );
    });
  } else {
    lines.push('- 未发现可用智能体');
  }

  return limitText(lines.join('\n'), maxLength);
};
