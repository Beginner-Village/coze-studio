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
  buildStandardSkillFiles,
  splitStandardSkillFiles,
  upsertSkillMdFrontmatter,
} from '../standard-skill-files';

describe('standard skill files', () => {
  it('builds a SKILL.md snapshot with synchronized frontmatter and extra files', () => {
    const files = buildStandardSkillFiles({
      name: 'pdf-tools',
      description: 'Handle PDF files',
      skillMd: '# PDF Tools\n\nUse scripts/run.py.',
      extraFiles: [
        { path: '/scripts/run.py', content: 'print("ok")\n' },
        { path: ' assets/schema.json ', content: '{"ok":true}' },
        { path: 'lib/helper.py', content: 'ignored' },
        { path: 'empty.txt', content: '' },
        { path: '../escape.py', content: 'bad' },
      ],
    });

    expect(files).toEqual({
      'SKILL.md':
        '---\nname: pdf-tools\ndescription: Handle PDF files\n---\n# PDF Tools\n\nUse scripts/run.py.',
      'scripts/run.py': 'print("ok")\n',
      'assets/schema.json': '{"ok":true}',
    });
  });

  it('updates existing frontmatter without losing custom metadata or body', () => {
    const skillMd = upsertSkillMdFrontmatter({
      name: 'xlsx',
      description: 'Build spreadsheets',
      skillMd:
        '---\nlicense: Proprietary\nname: old\ndescription: old desc\n---\n# Body\n',
    });

    expect(skillMd).toBe(
      '---\nname: xlsx\ndescription: Build spreadsheets\nlicense: Proprietary\n---\n# Body\n',
    );
  });

  it('splits existing files and falls back to prompt-only skills', () => {
    expect(
      splitStandardSkillFiles({
        name: 'docx',
        description: 'Write documents',
        prompt: '# Legacy\nUse docs.',
      }),
    ).toEqual({
      skillMd:
        '---\nname: docx\ndescription: Write documents\n---\n# Legacy\nUse docs.',
      extraFiles: [],
    });

    expect(
      splitStandardSkillFiles({
        name: 'docx',
        description: 'Write documents',
        prompt: '',
        files: {
          'SKILL.md': '# Existing\n',
          'references/guide.md': 'guide',
        },
      }),
    ).toEqual({
      skillMd: '# Existing\n',
      extraFiles: [{ path: 'references/guide.md', content: 'guide' }],
    });
  });
});
