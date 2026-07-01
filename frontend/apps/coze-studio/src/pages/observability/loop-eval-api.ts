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
/* eslint-disable max-lines -- single cohesive Loop eval/observability API client */

import { getLocalizedErrorMessage } from '@coze-arch/bot-api';

const EVALUATION_BASE = '/loop/api/evaluation/v1';
const OBSERVABILITY_BASE = '/loop/api/observability/v1';

export interface LoopUser {
  id?: string;
  user_id?: string;
  name?: string;
  username?: string;
  nickname?: string;
}

export interface EvaluationSet {
  id?: string;
  evaluation_set_id?: string;
  name?: string;
  description?: string;
  version?: string;
  latest_version?: string;
  item_count?: number | string;
  creator?: string | LoopUser;
  created_at?: string | number;
  updated_at?: string | number;
  // Loop nests created/updated metadata under base_info for many entities.
  base_info?: BaseInfo;
  evaluation_set_version?: {
    id?: string;
    evaluation_set_id?: string;
    evaluation_set_version_id?: string;
    version?: string;
    version_num?: string;
    item_count?: number | string;
    evaluation_set_schema?: EvaluationSetSchema;
  };
}

export interface EvaluationSetItem {
  id?: string;
  item_id?: string;
  turns?: unknown[];
  input?: unknown;
  output?: unknown;
  created_at?: string | number;
}

export interface Evaluator {
  id?: string;
  evaluator_id?: string;
  workspace_id?: string;
  name?: string;
  description?: string;
  evaluator_type?: string | number;
  current_version?: EvaluatorVersion;
  version?: string;
  latest_version?: string;
  creator?: string | LoopUser;
  base_info?: BaseInfo;
  created_at?: string | number;
}

export interface BaseInfo {
  created_by?: LoopUser;
  updated_by?: LoopUser;
  created_at?: string | number;
  updated_at?: string | number;
}

export interface EvaluatorVersion {
  id?: string;
  version?: string;
  description?: string;
  evaluator_content?: EvaluatorContent;
}

export interface EvaluatorContent {
  receive_chat_history?: boolean;
  input_schemas?: ArgsSchema[];
  output_schemas?: ArgsSchema[];
  prompt_evaluator?: PromptEvaluator;
  code_evaluator?: CodeEvaluator;
}

export interface ArgsSchema {
  key?: string;
  support_content_types?: string[];
  json_schema?: string;
}

export interface PromptEvaluator {
  message_list?: Message[];
  model_config?: ModelConfig;
  prompt_source_type?: number;
}

export interface CodeEvaluator {
  language_type?: string;
  code_content?: string;
}

export interface Message {
  role?: number;
  content?: Content;
}

export interface Content {
  content_type?: string;
  text?: string;
}

export interface ModelConfig {
  model_id?: string;
  model_name?: string;
}

export interface EvaluatorInputData {
  history_messages?: Message[];
  input_fields?: Record<string, Content>;
  evaluate_dataset_fields?: Record<string, Content>;
  evaluate_target_output_fields?: Record<string, Content>;
  ext?: Record<string, string>;
}

export interface EvaluatorRunError {
  code?: number;
  message?: string;
}

export interface EvaluatorResult {
  score?: number;
  reasoning?: string;
  correction?: unknown;
}

export interface EvaluatorOutputData {
  evaluator_result?: EvaluatorResult;
  evaluator_run_error?: EvaluatorRunError;
  evaluator_usage?: Record<string, unknown>;
  time_consuming_ms?: number;
  stdout?: string;
}

export interface EvaluatorRecord {
  id?: string;
  evaluator_version_id?: string;
  status?: string | number;
  evaluator_output_data?: EvaluatorOutputData;
  ext?: Record<string, string>;
}

export interface EvaluatorTemplate {
  id?: string;
  name?: string;
  description?: string;
  evaluator_type?: string | number;
  evaluator_content?: EvaluatorContent;
}

export interface CreateEvaluatorReq {
  evaluator: Evaluator;
  workspace_id?: string;
  cid?: string;
}

export interface CreateEvaluatorResp {
  evaluator_id?: string;
}

export interface GetEvaluatorResp {
  evaluator?: Evaluator;
}

export interface RunEvaluatorReq {
  workspace_id: string;
  evaluator_version_id: string;
  input_data: EvaluatorInputData;
}

export interface RunEvaluatorResp {
  record?: EvaluatorRecord;
}

export interface ListEvaluatorTemplatesResp {
  builtin_template_keys?: EvaluatorContent[];
  templates?: EvaluatorTemplate[];
}

export interface SubmitEvaluatorVersionReq {
  workspace_id: string;
  evaluator_id: string;
  version: string;
  description?: string;
  cid?: string;
}

export interface SubmitEvaluatorVersionResp {
  evaluator?: Evaluator;
}

export interface Experiment {
  id?: string;
  expt_id?: string;
  experiment_id?: string;
  name?: string;
  status?: string | number;
  eval_set?: string | EvaluationSet;
  eval_set_name?: string;
  evaluator_count?: number;
  progress?:
    | number
    | string
    | { total?: number; finished?: number; success?: number };
  created_at?: string | number;
  description?: string;
  desc?: string;
  base_info?: BaseInfo;
  start_time?: string | number;
  end_time?: string | number;
  evaluator_version_ids?: string[];
  target_id?: string;
  eval_target?: {
    eval_target_type?: number;
    source_target_id?: string;
  };
  expt_stats?: {
    success_turn_cnt?: number;
    fail_turn_cnt?: number;
    pending_turn_cnt?: number;
    processing_turn_cnt?: number;
    terminated_turn_cnt?: number;
  };
}

export interface TrajectoryConfig {
  id?: string;
  config_id?: string;
  name?: string;
  description?: string;
  created_at?: string | number;
  updated_at?: string | number;
}

export interface Trajectory {
  trace_id?: string;
  [key: string]: unknown;
}

export interface ListResponse<T> {
  total?: number | string;
  next_page_token?: string;
  nextCursor?: string;
  evaluation_sets?: T[];
  evaluators?: T[];
  experiments?: T[];
  items?: T[];
  configs?: T[];
  trajectory_configs?: T[];
  trajectories?: T[];
}

export interface FieldSchema {
  key: string;
  name: string;
  description?: string;
  content_type?: string;
  default_display_format?: number;
}

export interface EvaluationSetSchema {
  field_schemas?: FieldSchema[];
}

export interface CreateEvaluationSetRequest {
  workspace_id: string;
  name: string;
  description?: string;
  evaluation_set_schema?: EvaluationSetSchema;
}

export interface ExportTracesToDatasetRequest {
  workspace_id: string;
  trace_ids: string[];
  evaluation_set_id?: string;
}

export interface CreateExperimentRequest {
  workspace_id: string;
  name: string;
  description?: string;
  eval_set_id: string;
  eval_set_version_id?: string;
  evaluator_version_ids: string[];
  evaluator_field_mapping?: EvaluatorFieldMapping[];
  target_version_id?: string;
  target_id?: string;
  create_eval_target_param?: CreateEvalTargetParam;
  // 目标连接器：把评测集列映射到目标输入字段。bot/workflow 必须提供≥1条，否则实验校验不过、
  // 或目标节点不展开 turn（就不会真正执行目标）。
  target_field_mapping?: TargetFieldMapping;
  expt_type?: number;
}

export interface FieldMapping {
  field_name?: string;
  from_field_name?: string;
  const_value?: string;
}

export interface TargetFieldMapping {
  from_eval_set?: FieldMapping[];
}

export interface EvaluatorFieldMapping {
  evaluator_version_id: string;
  from_eval_set?: FieldMapping[];
  from_target?: FieldMapping[];
}

export interface CreateEvalTargetParam {
  eval_target_type: number;
  source_target_id?: string;
  source_target_version?: string;
}

// Resolve user_id → display name via Studio's MGetUserBasicInfo.
// Loop entities only carry user_id under base_info.created_by; Studio holds
// the human-readable names, so we go through Studio's own endpoint.
export interface UserBasicInfo {
  user_id: string | number;
  user_name?: string;
  user_unique_name?: string;
  user_avatar?: string;
}

export async function mGetUserBasicInfo(
  userIds: string[],
): Promise<Record<string, UserBasicInfo>> {
  const ids = Array.from(new Set(userIds.filter(Boolean)));
  if (ids.length === 0) {
    return {};
  }
  const resp = await fetch('/api/playground_api/mget_user_info', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ user_ids: ids }),
  });
  if (!resp.ok) {
    return {};
  }
  const json = await resp.json().catch(() => null);
  const list: UserBasicInfo[] =
    json?.data?.user_basic_info_list ||
    json?.user_basic_info_list ||
    json?.data ||
    [];
  const map: Record<string, UserBasicInfo> = {};
  for (const u of list) {
    map[String(u.user_id)] = u;
  }
  return map;
}

// Loop's evaluator_type is an int enum (Prompt/LLM=1, Code=2, CustomRPC=3,
// Agent=4). Evaluator objects carry it as a string or number depending on the
// source endpoint, so normalize to the numeric enum the debug APIs require.
export function normalizeEvaluatorType(type?: string | number): number {
  if (typeof type === 'number') {
    return type;
  }
  const normalized = String(type ?? '').toLowerCase();
  if (normalized === '2' || normalized.includes('code')) {
    return 2;
  }
  if (normalized === '3' || normalized.includes('rpc')) {
    return 3;
  }
  if (normalized === '4' || normalized.includes('agent')) {
    return 4;
  }
  // Default to Prompt/LLM (1) for prompt/llm or unknown types.
  return 1;
}

function isContentObject(value: unknown): value is Content {
  return (
    !!value &&
    typeof value === 'object' &&
    !Array.isArray(value) &&
    'content_type' in (value as Record<string, unknown>)
  );
}

// Loop expects every input field value to be a Content object
// ({content_type:"Text", text:"..."}). Users type plain strings in the debug
// modals, so wrap raw values here; pass through values already in Content shape.
function toContent(value: unknown): Content {
  if (isContentObject(value)) {
    return value;
  }
  const text = typeof value === 'string' ? value : JSON.stringify(value ?? '');
  return { content_type: 'Text', text };
}

function toContentMap(
  fields?: Record<string, unknown>,
): Record<string, Content> | undefined {
  if (!fields || typeof fields !== 'object') {
    return undefined;
  }
  const out: Record<string, Content> = {};
  for (const [key, value] of Object.entries(fields)) {
    out[key] = toContent(value);
  }
  return out;
}

// Accepts the loosely-typed input_data a user supplies in the debug modals and
// normalizes every *_fields map into Content objects before it hits Loop.
export function normalizeInputData(
  raw: Record<string, unknown>,
): EvaluatorInputData {
  const result: EvaluatorInputData = {};
  const inputFields = toContentMap(
    raw.input_fields as Record<string, unknown> | undefined,
  );
  if (inputFields) {
    result.input_fields = inputFields;
  }
  const datasetFields = toContentMap(
    raw.evaluate_dataset_fields as Record<string, unknown> | undefined,
  );
  if (datasetFields) {
    result.evaluate_dataset_fields = datasetFields;
  }
  const targetFields = toContentMap(
    raw.evaluate_target_output_fields as Record<string, unknown> | undefined,
  );
  if (targetFields) {
    result.evaluate_target_output_fields = targetFields;
  }
  if (Array.isArray(raw.history_messages)) {
    result.history_messages = raw.history_messages as Message[];
  }
  if (raw.ext && typeof raw.ext === 'object') {
    result.ext = raw.ext as Record<string, string>;
  }
  return result;
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const resp = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    },
    ...options,
  });
  if (!resp.ok) {
    throw new Error(`请求失败：${resp.status}`);
  }
  const json = await resp.json();
  if (json.code !== undefined && json.code !== 0) {
    const message = json.msg || `接口返回错误码：${json.code}`;
    throw new Error(getLocalizedErrorMessage(message) ?? message);
  }
  return json.data ?? json;
}

function post<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

function get<T>(url: string): Promise<T> {
  return request<T>(url, { method: 'GET' });
}

export function listEvaluationSets(req: {
  workspace_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<EvaluationSet>> {
  return post(`${EVALUATION_BASE}/evaluation_sets/list`, req);
}

export function createEvaluationSet(
  req: CreateEvaluationSetRequest,
): Promise<{ evaluation_set?: EvaluationSet }> {
  return post(`${EVALUATION_BASE}/evaluation_sets`, req);
}

export function getEvaluationSet(req: {
  workspace_id: string;
  evaluation_set_id: string;
}): Promise<{ evaluation_set?: EvaluationSet }> {
  return get(
    `${EVALUATION_BASE}/evaluation_sets/${req.evaluation_set_id}?workspace_id=${req.workspace_id}`,
  );
}

export function listEvaluationSetItems(req: {
  workspace_id: string;
  evaluation_set_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<EvaluationSetItem>> {
  return post(
    `${EVALUATION_BASE}/evaluation_sets/${req.evaluation_set_id}/items/list`,
    req,
  );
}

export function listEvaluators(req: {
  workspace_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<Evaluator>> {
  return post(`${EVALUATION_BASE}/evaluators/list`, req);
}

export function createEvaluator(
  req: CreateEvaluatorReq,
): Promise<CreateEvaluatorResp> {
  return post(`${EVALUATION_BASE}/evaluators`, req);
}

export async function getEvaluator(req: {
  evaluator_id: string;
  space_id: string;
}): Promise<GetEvaluatorResp> {
  const resp = await post<{ evaluators?: Evaluator[] }>(
    `${EVALUATION_BASE}/evaluators/batch_get`,
    {
      workspace_id: req.space_id,
      evaluator_ids: [req.evaluator_id],
    },
  );
  return { evaluator: resp.evaluators?.[0] };
}

export function runEvaluator(req: RunEvaluatorReq): Promise<RunEvaluatorResp> {
  return post(
    `${EVALUATION_BASE}/evaluators_versions/${req.evaluator_version_id}/run`,
    req,
  );
}

export function listEvaluatorTemplates(req: {
  space_id: string;
}): Promise<ListEvaluatorTemplatesResp> {
  return post(`${EVALUATION_BASE}/evaluators/list_template`, {
    workspace_id: req.space_id,
  });
}

export function submitEvaluatorVersion(
  req: SubmitEvaluatorVersionReq,
): Promise<SubmitEvaluatorVersionResp> {
  return post(
    `${EVALUATION_BASE}/evaluators/${req.evaluator_id}/submit_version`,
    req,
  );
}

export function listExperiments(req: {
  workspace_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<Experiment>> {
  return post(`${EVALUATION_BASE}/experiments/list`, req);
}

export function createExperiment(
  req: CreateExperimentRequest,
): Promise<{ experiment_id?: string; expt_id?: string }> {
  return post(`${EVALUATION_BASE}/experiments/submit`, {
    workspace_id: req.workspace_id,
    name: req.name,
    desc: req.description,
    eval_set_id: req.eval_set_id,
    eval_set_version_id: req.eval_set_version_id,
    evaluator_version_ids: req.evaluator_version_ids,
    evaluator_field_mapping: req.evaluator_field_mapping,
    target_version_id: req.target_version_id,
    target_id: req.target_id,
    create_eval_target_param: req.create_eval_target_param,
    target_field_mapping: req.target_field_mapping,
    expt_type: req.expt_type,
  });
}

export function getExperiment(req: {
  workspace_id: string;
  experiment_id: string;
}): Promise<{ experiment?: Experiment }> {
  return post<{ experiments?: Experiment[] }>(
    `${EVALUATION_BASE}/experiments/batch_get`,
    {
      workspace_id: req.workspace_id,
      expt_ids: [req.experiment_id],
    },
  ).then(resp => ({ experiment: resp.experiments?.[0] }));
}

export function getExperimentAggrResult(req: {
  workspace_id: string;
  experiment_id: string;
}): Promise<Record<string, unknown>> {
  return post(`${EVALUATION_BASE}/experiments/aggr_results/batch_get`, {
    workspace_id: req.workspace_id,
    expt_ids: [req.experiment_id],
  });
}

// ── 逐行结果（batch test 的核心产出：每条评测项的目标输出 + 各评估器打分）──
// 后端 BatchGetExperimentResult (/experiments/results/batch_get)。响应结构较深且
// 未锁定 IDL，这里用宽松类型，渲染层按 key 尽力取值并对未知字段做 JSON 兜底。
export interface ExperimentColumnField {
  key?: string;
  name?: string;
  content_type?: string;
}

export interface ExperimentColumnEvaluator {
  evaluator_id?: string;
  evaluator_version_id?: string;
  name?: string;
  version?: string;
}

// 单条 item 结果：字段随后端演进，renderer 一律按可选处理。
export type ExperimentResultRow = Record<string, unknown>;

export interface ListExperimentResultsResponse {
  column_eval_set_fields?: ExperimentColumnField[];
  column_evaluators?: ExperimentColumnEvaluator[];
  expt_column_evaluators?: ExperimentColumnEvaluator[];
  item_results?: ExperimentResultRow[];
  total?: number | string;
}

// ── 批量测试编排用 API ──
// 向评测集写入数据项：每条 item 的字段 map 转成 field_data_list(注意 key=field_data_list)。
export function createEvaluationSetItems(req: {
  workspace_id: string;
  evaluation_set_id: string;
  rows: Array<Record<string, string>>;
}): Promise<{ added_items?: Record<string, string> }> {
  return post(
    `${EVALUATION_BASE}/evaluation_sets/${req.evaluation_set_id}/items/batch_create`,
    {
      workspace_id: req.workspace_id,
      evaluation_set_id: req.evaluation_set_id,
      allow_partial_add: true,
      items: req.rows.map(fields => ({
        turns: [
          {
            field_data_list: Object.entries(fields).map(([k, v]) => ({
              key: k,
              name: k,
              content: { content_type: 'Text', text: v },
            })),
          },
        ],
      })),
    },
  );
}

// 提交评测集版本快照(实验按版本读数据，草稿不可直接用)。
export function commitEvaluationSetVersion(req: {
  workspace_id: string;
  evaluation_set_id: string;
  version: string;
  description?: string;
}): Promise<{ id?: string }> {
  return post(
    `${EVALUATION_BASE}/evaluation_sets/${req.evaluation_set_id}/versions`,
    {
      workspace_id: req.workspace_id,
      evaluation_set_id: req.evaluation_set_id,
      version: req.version,
      description: req.description,
    },
  );
}

export function listExperimentResults(req: {
  workspace_id: string;
  experiment_id: string;
  page_number?: number;
  page_size?: number;
}): Promise<ListExperimentResultsResponse> {
  return post(`${EVALUATION_BASE}/experiments/results/batch_get`, {
    workspace_id: req.workspace_id,
    experiment_ids: [req.experiment_id],
    // 单实验视图必须带 baseline_experiment_id=自身，否则后端只回表头、不回 item_results。
    baseline_experiment_id: req.experiment_id,
    page_number: req.page_number ?? 1,
    page_size: req.page_size ?? 20,
  });
}

export function listTrajectories(req: {
  workspace_id: string;
  platform_type: string;
  trace_ids: string[];
  start_time?: number | string;
}): Promise<ListResponse<Trajectory>> {
  return post(`${OBSERVABILITY_BASE}/traces/trajectory`, req);
}

export function exportTracesToDataset(
  req: ExportTracesToDatasetRequest,
): Promise<Record<string, unknown>> {
  return post(`${OBSERVABILITY_BASE}/traces/export_to_dataset`, req);
}

// ── coze-loop fusion endpoints ──
// The three routes below were brought in by the coze-loop merge. Request bodies
// follow the same `workspace_id` + post() convention as the methods above. The
// response shapes are not yet locked down in a shared IDL, so they use loose
// types; tighten once the backend contract is published. TODO: align with IDL.

// Loop's BatchDebugEvaluator contract: it debugs an evaluator *definition*
// (evaluator_content + evaluator_type), not a saved version id, and takes a
// flat array of input_data rows whose field values are Content objects.
export interface BatchDebugEvaluatorsReq {
  workspace_id: string;
  evaluator_content: EvaluatorContent;
  // EvaluatorType enum (Prompt/LLM=1, Code=2, CustomRPC=3, Agent=4).
  evaluator_type: number;
  input_data: EvaluatorInputData[];
}

// Each row maps positionally to the corresponding input_data row. A row carries
// either an evaluator_result (score/reasoning) or an evaluator_run_error.
export interface BatchDebugEvaluatorsResp {
  evaluator_output_data?: EvaluatorOutputData[];
}

export function batchDebugEvaluators(
  req: BatchDebugEvaluatorsReq,
): Promise<BatchDebugEvaluatorsResp> {
  return post(`${EVALUATION_BASE}/evaluators/batch_debug`, req);
}

export interface ColumnExtractConfigItem {
  // A single configurable trace-list column. Field names are best-effort until
  // the IDL lands; renderers should treat every field as optional.
  key?: string;
  name?: string;
  // JSONPath / span field this column extracts from.
  field_path?: string;
  content_type?: string;
  // Whether the column is shown by default in the trace list.
  visible?: boolean;
  default_display_format?: number;
}

export interface GetColumnExtractConfigReq {
  workspace_id: string;
  // Optional scope hint (e.g. which trace view the columns apply to).
  scene?: string;
}

export interface GetColumnExtractConfigResp {
  // Primary list of configurable columns.
  columns?: ColumnExtractConfigItem[];
  // Alternate key some backends use; kept for forward-compat.
  column_configs?: ColumnExtractConfigItem[];
}

export function getColumnExtractConfig(
  req: GetColumnExtractConfigReq,
): Promise<GetColumnExtractConfigResp> {
  return post(`${OBSERVABILITY_BASE}/column_extract_config`, req);
}

export interface TraceAgentToolCall {
  // A single tool/function invocation captured for an agent span.
  name?: string;
  tool_name?: string;
  arguments?: string;
  input?: unknown;
  output?: unknown;
  status?: string | number;
}

export interface GetTraceAgentMetadataReq {
  workspace_id: string;
  trace_id: string;
  // Optional span scope; agent metadata is usually per-span.
  span_id?: string;
  start_time?: string;
  end_time?: string;
}

export interface TraceAgentMetadata {
  agent_id?: string;
  agent_name?: string;
  model?: string;
  tool_calls?: TraceAgentToolCall[];
  // Free-form additional metadata; shape unconfirmed. TODO: align with IDL.
  metadata?: Record<string, unknown>;
}

export interface GetTraceAgentMetadataResp {
  // Some backends return the object directly, others wrap it; support both.
  agent_metadata?: TraceAgentMetadata;
  metadata?: TraceAgentMetadata;
}

export function getTraceAgentMetadata(
  req: GetTraceAgentMetadataReq,
): Promise<GetTraceAgentMetadataResp> {
  return post(`${OBSERVABILITY_BASE}/trace/agent/metadata`, req);
}
