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

import { describe, expect, it, vi } from 'vitest';

vi.mock('../src/api/config', () => ({
  createAPI: (meta: unknown) => meta,
}));

describe('skill super-agent App Server APIs', () => {
  it('exports super-agent skill and marketplace endpoint metadata', async () => {
    const skillApi = await import('../src/idl/skill/skill');

    expect(skillApi.SuperAgentCreateSkill).toMatchObject({
      url: '/api/super-agent/skills/create',
      method: 'POST',
      reqMapping: {
        body: [
          'space_id',
          'name',
          'description',
          'prompt',
          'files',
          'icon_uri',
        ],
      },
    });
    expect(skillApi.SuperAgentImportSkillPackage).toMatchObject({
      url: '/api/super-agent/skills/import',
      method: 'POST',
      reqMapping: {
        body: ['space_id', 'filename', 'content', 'icon_uri'],
      },
    });
    expect(skillApi.SuperAgentImportRuntimeSkill).toMatchObject({
      url: '/api/super-agent/skills/import-runtime',
      method: 'POST',
      reqMapping: {
        body: [
          'agent_id',
          'bot_id',
          'connector_id',
          'name',
          'skill_id',
          'icon_uri',
          'publish_scope',
        ],
      },
    });
    expect(skillApi.SuperAgentValidateSkillPackage).toMatchObject({
      url: '/api/super-agent/skills/validate-package',
      method: 'POST',
      reqMapping: {
        body: ['filename', 'content'],
      },
    });
    expect(skillApi.SuperAgentExportSkillPackage).toMatchObject({
      url: '/api/super-agent/skills/export',
      method: 'POST',
      reqMapping: {
        body: ['space_id', 'skill_id'],
      },
    });
    expect(skillApi.SuperAgentListSkills).toMatchObject({
      url: '/api/super-agent/skills/list',
      method: 'GET',
      reqMapping: {
        query: ['space_id', 'page', 'page_size', 'keyword'],
      },
    });
    expect(skillApi.SuperAgentListSkillAssets).toMatchObject({
      url: '/api/super-agent/skills/assets/list',
      method: 'GET',
      reqMapping: {
        query: ['skill_id', 'space_id'],
      },
    });
    expect(skillApi.SuperAgentGetSkillAsset).toMatchObject({
      url: '/api/super-agent/skills/assets/get',
      method: 'GET',
      reqMapping: {
        query: ['skill_id', 'space_id', 'path'],
      },
    });
    expect(skillApi.SuperAgentDeleteSkillAsset).toMatchObject({
      url: '/api/super-agent/skills/assets/delete',
      method: 'POST',
      reqMapping: {
        body: ['skill_id', 'space_id', 'path'],
      },
    });
    expect(skillApi.SuperAgentMarketplaceListSkills).toMatchObject({
      url: '/api/super-agent/marketplace/list',
      method: 'GET',
      reqMapping: {
        query: ['space_id', 'scope', 'page', 'page_size', 'keyword'],
      },
    });
    expect(skillApi.SuperAgentMarketplaceGetSkill).toMatchObject({
      url: '/api/super-agent/marketplace/get',
      method: 'GET',
      reqMapping: {
        query: ['skill_id', 'space_id'],
      },
    });
    expect(skillApi.SuperAgentInstallMarketplaceSkill).toMatchObject({
      url: '/api/super-agent/marketplace/install',
      method: 'POST',
      reqMapping: {
        body: ['skill_id', 'space_id'],
      },
    });
  });
});
