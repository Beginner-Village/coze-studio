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

import { type AgentCanvasCommand } from './workflow-agent-command-service';

export interface WorkflowAgentTestRunRequirement {
  required: boolean;
  input?: Record<string, string>;
}

export const WORKFLOW_AGENT_AUTO_TEST_RUN_DELAY_MS = 1800;

const TEST_RUN_INTENT_RE =
  /试运行|调试|运行|验证|测试|跑一下|run|debug|test/i;

const extractExplicitStartInput = (
  rawQuery: string,
): string | undefined =>
  rawQuery
    .match(/输入(?:是|为|:|：)\s*([^。；;，,\n]+)/)?.[1]
    ?.trim();

export const detectWorkflowAgentTestRunRequirement = (
  rawQuery: string,
): WorkflowAgentTestRunRequirement => {
  const required = TEST_RUN_INTENT_RE.test(rawQuery);
  if (!required) {
    return { required: false, input: undefined };
  }

  return {
    required: true,
    input: { input: extractExplicitStartInput(rawQuery) || '示例输入' },
  };
};

const delay = (ms: number): Promise<void> =>
  new Promise(resolve => setTimeout(resolve, ms));

export const waitForWorkflowAgentSavingIdle = async ({
  isSaving,
  timeoutMs = 8000,
  intervalMs = 200,
  wait = delay,
}: {
  isSaving: () => boolean;
  timeoutMs?: number;
  intervalMs?: number;
  wait?: (ms: number) => Promise<void>;
}): Promise<boolean> => {
  const start = Date.now();
  while (isSaving()) {
    if (Date.now() - start >= timeoutMs) {
      return false;
    }
    await wait(intervalMs);
  }
  return true;
};

export class WorkflowAgentTestRunGuard {
  private pending: WorkflowAgentTestRunRequirement | null = null;

  arm(requirement: WorkflowAgentTestRunRequirement): void {
    this.pending = requirement.required ? requirement : null;
  }

  clear(): void {
    this.pending = null;
  }

  peek(): WorkflowAgentTestRunRequirement | null {
    return this.pending;
  }

  noteCommand(cmd: Pick<AgentCanvasCommand, 'op'>): void {
    if (cmd.op === 'testRun') {
      this.clear();
    }
  }

  async runAfterAutoLayoutIfNeeded(
    cmd: Pick<AgentCanvasCommand, 'op'>,
    runTestRun: (input?: Record<string, string>) => Promise<unknown>,
    delayMs = WORKFLOW_AGENT_AUTO_TEST_RUN_DELAY_MS,
  ): Promise<boolean> {
    if (cmd.op !== 'autoLayout') {
      return false;
    }
    const pending = this.pending;
    if (!pending?.required) {
      return false;
    }

    await delay(delayMs);
    if (this.pending !== pending) {
      return false;
    }

    this.clear();
    await runTestRun(pending.input);
    return true;
  }
}
