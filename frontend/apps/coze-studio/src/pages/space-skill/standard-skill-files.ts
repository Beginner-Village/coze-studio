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

export interface StandardSkillFileDraft {
  path: string;
  content: string;
}

export interface StandardSkillForm {
  skillMd: string;
  extraFiles: StandardSkillFileDraft[];
}

const frontmatterPattern = /^---\n([\s\S]*?)\n---\n?/;
const allowedExtraFileRoots = [
  'scripts/',
  'references/',
  'templates/',
  'assets/',
];

const normalizeNewlines = (value: string) => value.replace(/\r\n/g, '\n');

const normalizeSkillFilePath = (path: string) => {
  const normalized = path
    .trim()
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+/g, '/');

  if (
    !normalized ||
    normalized === 'SKILL.md' ||
    normalized.split('/').some(part => part === '..' || part === '.') ||
    !allowedExtraFileRoots.some(root => normalized.startsWith(root))
  ) {
    return '';
  }

  return normalized;
};

const defaultSkillMdBody = (name: string) =>
  `# ${name.trim() || '技能名'}\n\n用途说明...\n\n## 工作流\n1. ...`;

export const upsertSkillMdFrontmatter = ({
  name,
  description,
  skillMd,
}: {
  name: string;
  description: string;
  skillMd: string;
}) => {
  const body = normalizeNewlines(skillMd).trim()
    ? normalizeNewlines(skillMd)
    : defaultSkillMdBody(name);
  const headerLines = [
    `name: ${name.trim()}`,
    `description: ${description.trim()}`,
  ];
  const match = body.match(frontmatterPattern);

  if (!match) {
    return `---\n${headerLines.join('\n')}\n---\n${body}`;
  }

  const customFrontmatter = match[1]
    .split('\n')
    .filter(line => {
      const key = line.split(':')[0]?.trim().toLowerCase();
      return key !== 'name' && key !== 'description' && line.trim();
    });
  const content = body.slice(match[0].length);

  return `---\n${[...headerLines, ...customFrontmatter].join(
    '\n',
  )}\n---\n${content}`;
};

export const buildStandardSkillFiles = ({
  name,
  description,
  skillMd,
  extraFiles,
}: {
  name: string;
  description: string;
  skillMd: string;
  extraFiles: StandardSkillFileDraft[];
}) => {
  const files: Record<string, string> = {
    'SKILL.md': upsertSkillMdFrontmatter({ name, description, skillMd }),
  };

  extraFiles.forEach(file => {
    const path = normalizeSkillFilePath(file.path);
    if (path && file.content.trim()) {
      files[path] = normalizeNewlines(file.content);
    }
  });

  return files;
};

export const splitStandardSkillFiles = ({
  name,
  description,
  prompt,
  files,
}: {
  name: string;
  description: string;
  prompt: string;
  files?: Record<string, string>;
}): StandardSkillForm => {
  if (files?.['SKILL.md']) {
    return {
      skillMd: files['SKILL.md'],
      extraFiles: Object.entries(files)
        .filter(([path]) => path !== 'SKILL.md')
        .map(([path, content]) => ({ path, content })),
    };
  }

  return {
    skillMd: upsertSkillMdFrontmatter({
      name,
      description,
      skillMd: prompt,
    }),
    extraFiles: [],
  };
};
