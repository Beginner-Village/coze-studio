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

import type { bot_open_api } from '@coze-arch/idl/bot_open_api';

describe('BotOpenApi super-agent manifest types', () => {
  it('types the external App Server discovery contract', () => {
    const manifest: bot_open_api.SuperAgentManifestResponse = {
      code: 0,
      msg: 'success',
      data: {
        protocol_version: 'super-agent.app-server.v1',
        capabilities: ['sandbox', 'workspace', 'skills', 'harness'],
        auth: {
          type: 'bearer',
          header: 'Authorization',
          scheme: 'Bearer',
        },
        stream: {
          transport: 'sse',
          content_type: 'text/event-stream',
          events: ['conversation.run.created'],
          done_event: 'conversation.stream.done',
          error_event: 'conversation.error',
        },
        trace: {
          events: ['conversation.run.created'],
          max_page_size: 200,
        },
        artifacts: {
          root: '/outputs',
          list_route: 'POST /api/super-agent/artifacts/list',
          download_route: 'POST /api/super-agent/artifacts/download',
          delete_route: 'POST /api/super-agent/artifacts/delete',
          move_route: 'POST /api/super-agent/artifacts/move',
          metadata_fields: ['artifact_id', 'path'],
          previewable_mimes: ['text/markdown'],
          max_list_items: 200,
          request_schemas: {
            list: {
              required: [],
              optional: ['agent_id', 'bot_id'],
            },
          },
        },
        skills: {
          entry_file: 'SKILL.md',
          file_roots: ['SKILL.md', 'scripts/', 'references/'],
          publish_scopes: {
            private: 1,
            space: 2,
            global: 3,
          },
          routes: {
            list: 'GET /api/super-agent/skills/list',
          },
          request_schemas: {
            list: {
              required: ['space_id'],
              optional: ['page', 'page_size', 'keyword'],
            },
          },
          assets: {
            root: 'assets/',
            upsert_route: 'POST /api/super-agent/skills/assets/upsert',
            allowed_mimes: ['image/png'],
            content_encoding: 'utf8-or-data-url',
          },
          agent_tool: {
            name: 'skill_manage',
            actions: ['create', 'read'],
          },
        },
        skill_publish_scopes: {
          private: 1,
          space: 2,
          global: 3,
        },
        workspace_roots: [
          {
            path: '/workspace',
            label: '工作区',
            readonly: false,
          },
        ],
        workspace: {
          identifier_fields: ['agent_id', 'bot_id'],
          readable_roots: ['/workspace', '/skills'],
          writable_roots: ['/workspace'],
          read_max_bytes: 26214400,
          binary_encoding: 'base64',
          write_route: 'POST /api/super-agent/workspace/write',
          move_route: 'POST /api/super-agent/workspace/move',
          mkdir_route: 'POST /api/super-agent/workspace/mkdir',
          stat_route: 'POST /api/super-agent/workspace/stat',
          grep_route: 'POST /api/super-agent/workspace/grep',
          glob_route: 'POST /api/super-agent/workspace/glob',
          edit_route: 'POST /api/super-agent/workspace/edit',
          patch_route: 'POST /api/super-agent/workspace/patch',
          request_schemas: {
            patch: {
              required: ['patch'],
              optional: ['agent_id', 'bot_id', 'workdir', 'work_dir'],
              aliases: {
                workdir: ['work_dir'],
              },
            },
          },
        },
        sandbox: {
          exec_route: 'POST /api/super-agent/sandbox/exec',
          default_workdir: '/workspace',
          max_timeout_sec: 300,
          output_max_bytes: 65536,
          request_schemas: {
            exec: {
              required: ['command'],
              optional: ['agent_id', 'bot_id', 'workdir', 'work_dir'],
              aliases: {
                workdir: ['work_dir'],
              },
            },
          },
        },
        harness: {
          deliverable_root: '/outputs',
          plan_path: '/workspace/.plan.json',
          tool_output_root: '/workspace/.agent/tooloutputs',
          state_route: 'POST /api/super-agent/harness/state',
          state_fields: ['plan', 'tool_outputs', 'runtime_skills'],
          plan_update_route: 'POST /api/super-agent/harness/plan',
          plan_route: 'POST /api/super-agent/workspace/read',
          tool_outputs_route: 'POST /api/super-agent/workspace/list',
          skill_runtime_root: '/skills',
          skill_entry_file: 'SKILL.md',
          skill_file_roots: ['SKILL.md', 'scripts/'],
          tools: [
            {
              name: 'run_bash',
              category: 'sandbox',
              available: 'super_agent',
              mutates: true,
            },
          ],
        },
        external_api: {
          base_path: '/api/super-agent',
          protocol_version: 'super-agent.app-server.v1',
          auth: {
            type: 'bearer',
            header: 'Authorization',
            scheme: 'Bearer',
          },
          session_auth_supported: true,
          transports: ['json', 'sse'],
          identifier_fields: ['agent_id', 'bot_id'],
          capabilities: ['sandbox', 'workspace', 'skills', 'harness'],
          entry_routes: {
            create: 'POST /api/super-agent/runs/create',
            stream: 'POST /api/super-agent/runs/stream',
          },
          request_schemas: {
            create: {
              required: [],
              required_one_of: [['agent_id', 'bot_id']],
              optional: ['additional_messages', 'client_id'],
            },
          },
          client_metadata: {
            app_server_flag: 'super_agent_app_server',
            capabilities_param: 'super_agent_capabilities',
            capabilities_value: 'sandbox,workspace,skills,harness',
          },
        },
        routes: {
          manifest: 'GET /api/super-agent/manifest',
        },
      },
    };

    expect(manifest.data?.external_api.entry_routes.create).toBe(
      'POST /api/super-agent/runs/create',
    );
    expect(
      manifest.data?.external_api.request_schemas.create.required_one_of?.[0],
    ).toEqual(['agent_id', 'bot_id']);
  });
});
