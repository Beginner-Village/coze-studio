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

import { describe, expect, it } from 'vitest';

import {
  summarizeHarnessPlan,
  mapRuntimeSkillsToSandboxFiles,
  mapArtifactsToSandboxFiles,
  shouldUseArtifactCatalog,
  sortSandboxFiles,
} from '../sandbox-workspace-utils';

describe('sandbox workspace artifact helpers', () => {
  it('uses the artifact catalog only for the outputs root', () => {
    expect(shouldUseArtifactCatalog('/outputs')).toBe(true);
    expect(shouldUseArtifactCatalog('/outputs/reports')).toBe(true);
    expect(shouldUseArtifactCatalog('/workspace')).toBe(false);
    expect(shouldUseArtifactCatalog('/uploads')).toBe(false);
    expect(shouldUseArtifactCatalog('/skills')).toBe(false);
  });

  it('maps artifact metadata into displayable sandbox files', () => {
    const files = mapArtifactsToSandboxFiles([
      {
        artifact_id: '/outputs/report.html',
        name: 'report.html',
        path: '/outputs/report.html',
        size: 52,
        mtime: 1781816441,
        mime: 'text/html',
        sha256: 'aec6999cd6a932e9b9f0957fc42777544a7f844fcb40df0eaadcf8e79a7cfbfe',
        previewable: true,
        downloadable: true,
        download_route: 'POST /api/super-agent/artifacts/download',
      },
    ]);

    expect(files).toEqual([
      {
        artifact_id: '/outputs/report.html',
        name: 'report.html',
        path: '/outputs/report.html',
        is_dir: false,
        size: 52,
        mtime: 1781816441,
        mime: 'text/html',
        sha256:
          'aec6999cd6a932e9b9f0957fc42777544a7f844fcb40df0eaadcf8e79a7cfbfe',
        previewable: true,
        downloadable: true,
        download_route: 'POST /api/super-agent/artifacts/download',
        source: 'artifact',
      },
    ]);
  });

  it('maps runtime skill summaries into displayable skill folders', () => {
    const files = mapRuntimeSkillsToSandboxFiles([
      {
        name: 'report-kit',
        path: '/skills/report-kit',
        entry_path: '/skills/report-kit/SKILL.md',
        standard: true,
        description: 'Build reports',
        version: '0.1.0',
        category: 'productivity',
        file_paths: ['SKILL.md', 'scripts/run.sh', 'assets/logo.png'],
        asset_paths: ['assets/logo.png'],
        image_paths: ['assets/logo.png'],
      },
    ]);

    expect(files).toEqual([
      {
        name: 'report-kit',
        path: '/skills/report-kit',
        is_dir: true,
        size: 3,
        mime: 'application/vnd.ynet.skill',
        source: 'runtime_skill',
        previewable: true,
      },
    ]);
  });

  it('sorts directories first and files by newest update time', () => {
    expect(
      sortSandboxFiles([
        {
          name: 'old.txt',
          path: '/outputs/old.txt',
          is_dir: false,
          size: 1,
          mtime: 100,
        },
        {
          name: 'reports',
          path: '/outputs/reports',
          is_dir: true,
          size: 0,
        },
        {
          name: 'new.html',
          path: '/outputs/new.html',
          is_dir: false,
          size: 1,
          mtime: 200,
        },
      ]).map(file => file.name),
    ).toEqual(['reports', 'new.html', 'old.txt']);
  });

  it('summarizes persisted harness plan status', () => {
    expect(
      summarizeHarnessPlan({
        exists: true,
        content: JSON.stringify([
          { content: '梳理需求', status: 'completed' },
          { content: '实现前端', status: 'in_progress' },
          { content: '回归验证', status: 'pending' },
        ]),
      }),
    ).toBe('计划 1/3 完成 · 1 进行中');
  });

  it('shows missing harness plan clearly', () => {
    expect(summarizeHarnessPlan({ exists: false, content: '' })).toBe(
      '计划未生成',
    );
  });
});
