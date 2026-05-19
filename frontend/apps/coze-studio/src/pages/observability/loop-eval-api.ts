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
  item_count?: number;
  creator?: string | LoopUser;
  created_at?: string | number;
  updated_at?: string | number;
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

export interface EvaluatorRecord {
  id?: string;
  evaluator_version_id?: string;
  status?: string | number;
  evaluator_output_data?: unknown;
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
}

export interface TrajectoryConfig {
  id?: string;
  config_id?: string;
  name?: string;
  description?: string;
  created_at?: string | number;
  updated_at?: string | number;
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
  evaluator_version_ids: string[];
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
    throw new Error(`API error: ${resp.status} ${resp.statusText}`);
  }
  const json = await resp.json();
  if (json.code !== undefined && json.code !== 0) {
    throw new Error(json.msg || `API error code: ${json.code}`);
  }
  return json.data ?? json;
}

function post<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    body: JSON.stringify(body),
  });
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
  return post(`${EVALUATION_BASE}/evaluation_sets`, req);
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

export function getEvaluator(req: {
  evaluator_id: string;
  space_id: string;
}): Promise<GetEvaluatorResp> {
  return post(`${EVALUATION_BASE}/evaluators/${req.evaluator_id}`, {
    workspace_id: req.space_id,
  });
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
  return post(`${EVALUATION_BASE}/experiments`, {
    workspace_id: req.workspace_id,
    name: req.name,
    desc: req.description,
    eval_set_id: req.eval_set_id,
    evaluator_version_ids: req.evaluator_version_ids,
  });
}

export function getExperiment(req: {
  workspace_id: string;
  experiment_id: string;
}): Promise<{ experiment?: Experiment }> {
  return post(`${EVALUATION_BASE}/experiments`, req);
}

export function getExperimentAggrResult(req: {
  workspace_id: string;
  experiment_id: string;
}): Promise<Record<string, unknown>> {
  return post(`${EVALUATION_BASE}/experiments/get_experiment_aggr_result`, req);
}

export function listTrajectoryConfigs(req: {
  workspace_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<TrajectoryConfig>> {
  return post(`${OBSERVABILITY_BASE}/traces/trajectory`, req);
}

export function exportTracesToDataset(
  req: ExportTracesToDatasetRequest,
): Promise<Record<string, unknown>> {
  return post(`${OBSERVABILITY_BASE}/traces/export_to_dataset`, req);
}
