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
  name?: string;
  evaluator_type?: string | number;
  latest_version?: string;
  creator?: string | LoopUser;
  created_at?: string | number;
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
  progress?: number | string | { total?: number; finished?: number; success?: number };
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

export interface CreateEvaluationSetRequest {
  workspace_id: string;
  name: string;
  description?: string;
}

export interface ExportTracesToDatasetRequest {
  workspace_id: string;
  trace_ids: string[];
  evaluation_set_id?: string;
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
  return post(`${EVALUATION_BASE}/evaluation_sets/list_evaluation_sets`, req);
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
  return post(`${EVALUATION_BASE}/evaluation_sets/get_evaluation_set`, req);
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
  return post(`${EVALUATION_BASE}/evaluators/list_evaluators`, req);
}

export function listExperiments(req: {
  workspace_id: string;
  page_size?: number;
  page_number?: number;
}): Promise<ListResponse<Experiment>> {
  return post(`${EVALUATION_BASE}/experiments/list_experiments`, req);
}

export function getExperiment(req: {
  workspace_id: string;
  experiment_id: string;
}): Promise<{ experiment?: Experiment }> {
  return post(`${EVALUATION_BASE}/experiments/get_experiment`, req);
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
  return post(`${OBSERVABILITY_BASE}/trajectory_config/list_trajectory_configs`, req);
}

export function exportTracesToDataset(
  req: ExportTracesToDatasetRequest,
): Promise<Record<string, unknown>> {
  return post(`${OBSERVABILITY_BASE}/traces/export_to_dataset`, req);
}
