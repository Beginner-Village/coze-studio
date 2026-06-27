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

import { vi } from 'vitest';

import { axiosInstance } from '../src/axios';
import { DeveloperApi } from '../src/developer-api';

vi.mock('../src/axios', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

describe('DeveloperApi super-agent skills', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (axiosInstance.request as any).mockResolvedValue({ code: 0 });
  });

  it('routes skill and marketplace calls to the super-agent App Server endpoints', async () => {
    await (DeveloperApi as any).SuperAgentCreateSkill({
      space_id: 'space-1',
      name: 'pdf-tools',
      description: 'Handle PDFs',
      files: {
        'SKILL.md': '# PDF Tools\n',
        'scripts/run.py': 'print("ok")\n',
      },
    });

    await (DeveloperApi as any).SuperAgentListSkills({
      space_id: 'space-1',
      page: 1,
      page_size: 20,
      keyword: 'pdf',
    });

    await (DeveloperApi as any).SuperAgentMarketplaceListSkills({
      space_id: 'space-1',
      scope: 3,
      page: 1,
      page_size: 20,
      keyword: 'pdf',
    });

    await (DeveloperApi as any).SuperAgentMarketplaceGetSkill({
      space_id: 'space-1',
      skill_id: 'skill-1',
    });

    await (DeveloperApi as any).SuperAgentListSkillAssets({
      space_id: 'space-1',
      skill_id: 'skill-1',
    });

    await (DeveloperApi as any).SuperAgentGetSkillAsset({
      space_id: 'space-1',
      skill_id: 'skill-1',
      path: 'assets/logo.png',
    });

    await (DeveloperApi as any).SuperAgentUpsertSkillAsset({
      space_id: 'space-1',
      skill_id: 'skill-1',
      path: 'assets/logo.png',
      content: 'data:image/png;base64,iVBORw0KGgo=',
      mime: 'image/png',
    });

    await (DeveloperApi as any).SuperAgentDeleteSkillAsset({
      space_id: 'space-1',
      skill_id: 'skill-1',
      path: 'assets/logo.png',
    });

    await (DeveloperApi as any).SuperAgentImportSkillPackage({
      space_id: 'space-1',
      filename: 'pdf-tools.zip',
      content: 'data:application/zip;base64,UEs=',
    });

    await (DeveloperApi as any).SuperAgentImportRuntimeSkill({
      agent_id: 'agent-1',
      name: 'report-kit',
      skill_id: 'skill-1',
      publish_scope: 3,
    });

    await (DeveloperApi as any).SuperAgentValidateSkillPackage({
      filename: 'pdf-tools.zip',
      content: 'data:application/zip;base64,UEs=',
    });

    await (DeveloperApi as any).SuperAgentExportSkillPackage({
      space_id: 'space-1',
      skill_id: 'skill-1',
    });

    await (DeveloperApi as any).SuperAgentInstallMarketplaceSkill({
      space_id: 'space-1',
      skill_id: 'skill-1',
    });

    expect(axiosInstance.request).toHaveBeenNthCalledWith(1, {
      url: '/api/super-agent/skills/create',
      method: 'POST',
      data: {
        space_id: 'space-1',
        name: 'pdf-tools',
        description: 'Handle PDFs',
        prompt: undefined,
        files: {
          'SKILL.md': '# PDF Tools\n',
          'scripts/run.py': 'print("ok")\n',
        },
        icon_uri: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(2, {
      url: '/api/super-agent/skills/list',
      method: 'GET',
      params: {
        space_id: 'space-1',
        page: 1,
        page_size: 20,
        keyword: 'pdf',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(3, {
      url: '/api/super-agent/marketplace/list',
      method: 'GET',
      params: {
        space_id: 'space-1',
        scope: 3,
        page: 1,
        page_size: 20,
        keyword: 'pdf',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(4, {
      url: '/api/super-agent/marketplace/get',
      method: 'GET',
      params: {
        space_id: 'space-1',
        skill_id: 'skill-1',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(5, {
      url: '/api/super-agent/skills/assets/list',
      method: 'GET',
      params: {
        space_id: 'space-1',
        skill_id: 'skill-1',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(6, {
      url: '/api/super-agent/skills/assets/get',
      method: 'GET',
      params: {
        space_id: 'space-1',
        skill_id: 'skill-1',
        path: 'assets/logo.png',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(7, {
      url: '/api/super-agent/skills/assets/upsert',
      method: 'POST',
      data: {
        space_id: 'space-1',
        skill_id: 'skill-1',
        path: 'assets/logo.png',
        content: 'data:image/png;base64,iVBORw0KGgo=',
        mime: 'image/png',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(8, {
      url: '/api/super-agent/skills/assets/delete',
      method: 'POST',
      data: {
        space_id: 'space-1',
        skill_id: 'skill-1',
        path: 'assets/logo.png',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(9, {
      url: '/api/super-agent/skills/import',
      method: 'POST',
      data: {
        space_id: 'space-1',
        filename: 'pdf-tools.zip',
        content: 'data:application/zip;base64,UEs=',
        icon_uri: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(10, {
      url: '/api/super-agent/skills/import-runtime',
      method: 'POST',
      data: {
        agent_id: 'agent-1',
        bot_id: undefined,
        connector_id: undefined,
        name: 'report-kit',
        skill_id: 'skill-1',
        icon_uri: undefined,
        publish_scope: 3,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(11, {
      url: '/api/super-agent/skills/validate-package',
      method: 'POST',
      data: {
        filename: 'pdf-tools.zip',
        content: 'data:application/zip;base64,UEs=',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(12, {
      url: '/api/super-agent/skills/export',
      method: 'POST',
      data: {
        space_id: 'space-1',
        skill_id: 'skill-1',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(13, {
      url: '/api/super-agent/marketplace/install',
      method: 'POST',
      data: {
        space_id: 'space-1',
        skill_id: 'skill-1',
      },
    });
  });
});
