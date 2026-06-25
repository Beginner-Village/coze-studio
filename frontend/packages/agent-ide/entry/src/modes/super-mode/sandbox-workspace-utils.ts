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

export interface SandboxFile {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mtime?: number;
  mime?: string;
  sha256?: string;
  previewable?: boolean;
  downloadable?: boolean;
  download_route?: string;
  artifact_id?: string;
  source?: 'workspace' | 'artifact' | 'runtime_skill';
}

export interface SuperAgentArtifactMeta {
  artifact_id: string;
  name: string;
  path: string;
  size: number;
  mtime?: number;
  mime?: string;
  sha256?: string;
  previewable?: boolean;
  downloadable?: boolean;
  download_route?: string;
}

export interface SuperAgentHarnessPlanState {
  exists?: boolean;
  content?: string;
}

export interface SuperAgentRuntimeSkillSummary {
  name: string;
  path: string;
  entry_path: string;
  standard: boolean;
  description?: string;
  version?: string;
  category?: string;
  file_paths?: string[];
  asset_paths?: string[];
  image_paths?: string[];
}

export const shouldUseArtifactCatalog = (path: string): boolean =>
  path === '/outputs' || path.startsWith('/outputs/');

export const mapArtifactsToSandboxFiles = (
  artifacts: SuperAgentArtifactMeta[],
): SandboxFile[] =>
  artifacts.map(artifact => ({
    name: artifact.name,
    path: artifact.path,
    is_dir: false,
    size: artifact.size,
    mtime: artifact.mtime,
    mime: artifact.mime,
    sha256: artifact.sha256,
    previewable: artifact.previewable,
    downloadable: artifact.downloadable,
    download_route: artifact.download_route,
    artifact_id: artifact.artifact_id,
    source: 'artifact',
  }));

export const mapRuntimeSkillsToSandboxFiles = (
  skills: SuperAgentRuntimeSkillSummary[],
): SandboxFile[] =>
  skills.map(skill => ({
    name: skill.name,
    path: skill.path,
    is_dir: true,
    size: skill.file_paths?.length ?? 0,
    mime: 'application/vnd.ynet.skill',
    previewable: skill.standard,
    source: 'runtime_skill',
  }));

export const sortSandboxFiles = (files: SandboxFile[]): SandboxFile[] =>
  [...files].sort((a, b) => {
    if (a.is_dir !== b.is_dir) {
      return a.is_dir ? -1 : 1;
    }
    if ((b.mtime ?? 0) !== (a.mtime ?? 0)) {
      return (b.mtime ?? 0) - (a.mtime ?? 0);
    }
    return a.name.localeCompare(b.name);
  });

export const summarizeHarnessPlan = (
  plan?: SuperAgentHarnessPlanState | null,
): string => {
  if (!plan?.exists) {
    return '计划未生成';
  }
  try {
    const parsed = JSON.parse(plan.content ?? '');
    const steps = Array.isArray(parsed)
      ? parsed
      : Array.isArray(parsed?.plan)
        ? parsed.plan
        : Array.isArray(parsed?.steps)
          ? parsed.steps
          : [];
    if (steps.length === 0) {
      return '计划为空';
    }
    let done = 0;
    let inProgress = 0;
    steps.forEach((step: { status?: string }) => {
      if (step.status === 'completed' || step.status === 'done') {
        done += 1;
      } else if (step.status === 'in_progress') {
        inProgress += 1;
      }
    });
    return `计划 ${done}/${steps.length} 完成${
      inProgress > 0 ? ` · ${inProgress} 进行中` : ''
    }`;
  } catch {
    return '计划可用';
  }
};
