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

import { expect, request as requestFactory, test, type APIRequestContext, type TestInfo } from '@playwright/test';

type JsonResponse = {
  code?: number;
  msg?: string;
  data?: any;
};

const requiredEnv = (name: string, fallback?: string): string => {
  const value = process.env[name] || fallback;
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
};

const safeName = (value: string): string => value.replace(/[^a-zA-Z0-9_.-]+/g, '-').slice(0, 80);

async function attachText(testInfo: TestInfo, name: string, body: string, contentType = 'application/json') {
  await testInfo.attach(safeName(name), {
    body,
    contentType,
  });
}

async function postJSON(
  request: APIRequestContext,
  testInfo: TestInfo,
  path: string,
  body: Record<string, unknown>,
  expectBusinessOK = true,
): Promise<JsonResponse> {
  const response = await request.post(path, { data: body });
  const text = await response.text();
  await attachText(testInfo, `${path}.json`, text);

  expect(response.status(), `${path} HTTP status`).toBeLessThan(500);
  const json = JSON.parse(text) as JsonResponse;
  if (expectBusinessOK) {
    expect(json.code, `${path} business code: ${json.msg || ''}`).toBe(0);
  }
  return json;
}

test.describe('Studio super-agent Sandbox/BashTool on 226', () => {
  test.setTimeout(120_000);

  test.beforeEach(async ({}, testInfo) => {
    test.skip(testInfo.project.name !== 'super-agent-api', 'API-only super-agent test runs in the super-agent-api project');
    test.skip(!process.env.SUPER_AGENT_API_KEY, 'SUPER_AGENT_API_KEY is required for 226 super-agent API acceptance');
  });

  test('executes bash, edits workspace, downloads artifact, records harness state, and rejects unsafe paths', async ({ baseURL }, testInfo) => {
    const apiKey = requiredEnv('SUPER_AGENT_API_KEY');
    const agentID = requiredEnv('SUPER_AGENT_ID', '7652617174313336832');
    const spaceID = requiredEnv('SUPER_AGENT_SPACE_ID', '7652614054615187456');
    const marker = `e2e-${Date.now()}`;
    const workspaceDir = `/workspace/${marker}`;
    const notePath = `${workspaceDir}/note.txt`;
    const artifactPath = `/outputs/${marker}-artifact.txt`;

    const request = await requestFactory.newContext({
      baseURL,
      extraHTTPHeaders: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
    });

    try {
      await attachText(testInfo, 'run-context.json', JSON.stringify({ baseURL, agentID, spaceID, marker, workspaceDir, artifactPath }, null, 2));

      const session = await postJSON(request, testInfo, '/api/super-agent/sessions/create', {
        agent_id: agentID,
        space_id: spaceID,
        title: `Sandbox ${marker}`,
      });
      const conversationID = session.data?.session?.conversation_id;
      expect(conversationID, 'conversation id').toBeTruthy();

      const sessionGet = await postJSON(request, testInfo, '/api/super-agent/sessions/get', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
        include_snapshot: true,
      });
      expect(sessionGet.data?.session?.conversation_id).toBe(conversationID);

      const sessionList = await postJSON(request, testInfo, '/api/super-agent/sessions/list', {
        agent_id: agentID,
        space_id: spaceID,
        page: 1,
        page_size: 20,
      });
      expect(JSON.stringify(sessionList.data?.sessions || [])).toContain(conversationID);

      const renamed = await postJSON(request, testInfo, '/api/super-agent/sessions/rename', {
        conversation_id: conversationID,
        title: `Sandbox renamed ${marker}`,
      });
      expect(renamed.data?.session?.title).toContain('renamed');

      const bash = await postJSON(request, testInfo, '/api/super-agent/sandbox/exec', {
        agent_id: agentID,
        space_id: spaceID,
        command: `pwd && echo ${marker}-bash-ok`,
        workdir: '/workspace',
        timeout_sec: 10,
      });
      expect(bash.data?.exit_code).toBe(0);
      expect(bash.data?.stdout).toContain(`${marker}-bash-ok`);

      await postJSON(request, testInfo, '/api/super-agent/workspace/mkdir', {
        agent_id: agentID,
        space_id: spaceID,
        path: workspaceDir,
      });

      await postJSON(request, testInfo, '/api/super-agent/workspace/list', {
        agent_id: agentID,
        space_id: spaceID,
        path: '/workspace',
      });

      await postJSON(request, testInfo, '/api/super-agent/workspace/write', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
        content: 'alpha\n',
      });

      const beforePatch = await postJSON(request, testInfo, '/api/super-agent/workspace/read', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
      });
      expect(beforePatch.data?.content).toBe('alpha\n');

      const patch = `*** Begin Patch
*** Update File: note.txt
@@
-alpha
+alpha
+patched
*** End Patch
`;
      const patchResult = await postJSON(request, testInfo, '/api/super-agent/workspace/patch', {
        agent_id: agentID,
        space_id: spaceID,
        workdir: workspaceDir,
        patch,
      });
      expect(patchResult.data?.changed_files).toBeGreaterThan(0);

      const editResult = await postJSON(request, testInfo, '/api/super-agent/workspace/edit', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
        old_string: 'patched',
        new_string: 'patched\nedited',
      });
      expect(editResult.data?.replacements).toBeGreaterThan(0);

      const afterPatch = await postJSON(request, testInfo, '/api/super-agent/workspace/read', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
      });
      expect(afterPatch.data?.content).toContain('edited');

      const noteDownload = await request.post('/api/super-agent/workspace/download', {
        data: {
          agent_id: agentID,
          space_id: spaceID,
          path: notePath,
        },
      });
      const noteBody = await noteDownload.text();
      await attachText(testInfo, 'workspace-download.txt', noteBody, 'text/plain');
      expect(noteDownload.status()).toBe(200);
      expect(noteBody).toContain('edited');

      const stat = await postJSON(request, testInfo, '/api/super-agent/workspace/stat', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
      });
      expect(stat.data?.exists).toBe(true);

      const grep = await postJSON(request, testInfo, '/api/super-agent/workspace/grep', {
        agent_id: agentID,
        space_id: spaceID,
        path: workspaceDir,
        pattern: 'edited',
      });
      expect(grep.data?.output).toContain('edited');

      const glob = await postJSON(request, testInfo, '/api/super-agent/workspace/glob', {
        agent_id: agentID,
        space_id: spaceID,
        path: workspaceDir,
        pattern: '*.txt',
        limit: 20,
      });
      expect(JSON.stringify(glob.data?.matches || [])).toContain('note.txt');

      await postJSON(request, testInfo, '/api/super-agent/workspace/upload', {
        agent_id: agentID,
        space_id: spaceID,
        path: `${workspaceDir}/upload.txt`,
        content: 'upload-ok\n',
      });

      const move = await postJSON(request, testInfo, '/api/super-agent/workspace/move', {
        agent_id: agentID,
        space_id: spaceID,
        path: notePath,
        target_path: `${workspaceDir}/moved.txt`,
      });
      expect(move.data?.path).toContain('moved.txt');

      await postJSON(request, testInfo, '/api/super-agent/workspace/delete', {
        agent_id: agentID,
        space_id: spaceID,
        path: `${workspaceDir}/upload.txt`,
      });


      await postJSON(request, testInfo, '/api/super-agent/sandbox/exec', {
        agent_id: agentID,
        space_id: spaceID,
        command: `mkdir -p /outputs && printf 'artifact-ok ${marker}\\n' > ${artifactPath}`,
        workdir: '/workspace',
        timeout_sec: 10,
      });

      const artifacts = await postJSON(request, testInfo, '/api/super-agent/artifacts/list', {
        agent_id: agentID,
        path: '/outputs',
        limit: 100,
      });
      expect(JSON.stringify(artifacts.data?.artifacts || [])).toContain(artifactPath);

      const download = await request.post('/api/super-agent/artifacts/download', {
        data: {
          agent_id: agentID,
          path: artifactPath,
        },
      });
      const artifactBody = await download.text();
      await attachText(testInfo, 'artifact-download.txt', artifactBody, 'text/plain');
      expect(download.status()).toBe(200);
      expect(artifactBody).toContain(`artifact-ok ${marker}`);

      const movedArtifactPath = `/outputs/${marker}-artifact-moved.txt`;
      const artifactMove = await postJSON(request, testInfo, '/api/super-agent/artifacts/move', {
        agent_id: agentID,
        path: artifactPath,
        target_path: movedArtifactPath,
      });
      expect(artifactMove.data?.path || '').toContain('artifact-moved');

      const movedDownload = await request.post('/api/super-agent/artifacts/download', {
        data: {
          agent_id: agentID,
          path: movedArtifactPath,
        },
      });
      const movedArtifactBody = await movedDownload.text();
      await attachText(testInfo, 'artifact-moved-download.txt', movedArtifactBody, 'text/plain');
      expect(movedDownload.status()).toBe(200);
      expect(movedArtifactBody).toContain(`artifact-ok ${marker}`);

      const denied = await postJSON(
        request,
        testInfo,
        '/api/super-agent/workspace/read',
        {
          agent_id: agentID,
          space_id: spaceID,
          path: '/etc/passwd',
        },
        false,
      );
      expect(denied.code).not.toBe(0);
      expect(denied.msg || '').toMatch(/path|workspace|outputs|uploads|skills/i);

      const harnessState = await postJSON(request, testInfo, '/api/super-agent/harness/state', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
      });
      expect(harnessState.data).toBeTruthy();

      await postJSON(request, testInfo, '/api/super-agent/harness/plan', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
        items: [
          { content: `${marker} sandbox/bash acceptance`, status: 'completed' },
        ],
      });

      await postJSON(request, testInfo, '/api/super-agent/harness/tool-outputs', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
      });

      const snapshot = await postJSON(request, testInfo, '/api/super-agent/harness/snapshot', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
        include_harness: true,
        include_messages: true,
        include_runs: true,
        include_workspace: true,
        include_artifacts: true,
        include_trace: true,
        workspace_path: workspaceDir,
      });
      expect(snapshot.data?.components || []).toContain('harness');
      expect(snapshot.data?.components || []).toContain('messages');
      expect(snapshot.data?.components || []).toContain('runs');
      expect(snapshot.data?.components || []).toContain('workspace');
      expect(snapshot.data?.components || []).toContain('trace');
      expect(JSON.stringify(snapshot.data)).toContain(marker);

      const messages = await postJSON(request, testInfo, '/api/super-agent/messages/list', {
        conversation_id: conversationID,
        limit: 20,
      });
      expect(messages.data?.conversation_id).toBe(conversationID);

      const runs = await postJSON(request, testInfo, '/api/super-agent/runs/list', {
        conversation_id: conversationID,
        limit: 20,
      });
      expect(runs.data?.conversation_id).toBe(conversationID);

      const trace = await postJSON(request, testInfo, '/api/super-agent/traces/get', {
        conversation_id: conversationID,
        limit: 20,
      });
      expect(trace.data?.conversation_id).toBe(conversationID);
      expect(JSON.stringify(trace.data?.events || [])).toContain('plan.updated');

      const resume = await postJSON(request, testInfo, '/api/super-agent/harness/resume', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
      });
      expect(resume.data?.conversation_id).toBe(conversationID);

      await postJSON(request, testInfo, '/api/super-agent/harness/context/clear', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
      });

      await postJSON(request, testInfo, '/api/super-agent/harness/cleanup', {
        agent_id: agentID,
        space_id: spaceID,
        conversation_id: conversationID,
        keep_latest: 1,
        dry_run: true,
      });

      await postJSON(request, testInfo, '/api/super-agent/artifacts/delete', {
        agent_id: agentID,
        path: movedArtifactPath,
      });

      const deletedSession = await postJSON(request, testInfo, '/api/super-agent/sessions/delete', {
        conversation_id: conversationID,
      });
      expect(deletedSession.data?.conversation_id).toBe(conversationID);
    } finally {
      await request.dispose();
    }
  });
});
