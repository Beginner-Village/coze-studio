import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  detectWorkflowAgentTestRunRequirement,
  waitForWorkflowAgentSavingIdle,
  WorkflowAgentTestRunGuard,
} from '../workflow-agent-test-run-guard';

describe('workflow-agent-test-run-guard', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('detects explicit test-run intent and extracts start input', () => {
    expect(
      detectWorkflowAgentTestRunRequirement(
        '帮我搭一个工作流并完成试运行。输入是 hello。',
      ),
    ).toEqual({
      required: true,
      input: { input: 'hello' },
    });
  });

  it('does not require test run for plain build requests', () => {
    expect(
      detectWorkflowAgentTestRunRequirement('帮我搭一个客服工作流'),
    ).toEqual({
      required: false,
      input: undefined,
    });
  });

  it('does not treat automatic validation feedback as another test request', () => {
    expect(
      detectWorkflowAgentTestRunRequirement(
        '【工作流自动校验反馈】状态：failed。请继续修复错误节点。',
      ),
    ).toEqual({
      required: false,
      input: undefined,
    });
  });

  it('runs pending test run once after auto layout', async () => {
    vi.useFakeTimers();
    const guard = new WorkflowAgentTestRunGuard();
    const calls: Array<Record<string, string> | undefined> = [];

    guard.arm({
      required: true,
      input: { input: 'hello' },
    });
    const fallback = guard.runAfterAutoLayoutIfNeeded(
      { op: 'autoLayout' },
      input => {
        calls.push(input);
        return Promise.resolve();
      },
      1000,
    );

    await vi.advanceTimersByTimeAsync(999);
    expect(calls).toEqual([]);

    await vi.advanceTimersByTimeAsync(1);
    await expect(fallback).resolves.toBe(true);
    expect(calls).toEqual([{ input: 'hello' }]);
    expect(guard.peek()).toBeNull();
  });

  it('does not duplicate the model test_run tool call', async () => {
    vi.useFakeTimers();
    const guard = new WorkflowAgentTestRunGuard();
    const calls: Array<Record<string, string> | undefined> = [];

    guard.arm({
      required: true,
      input: { input: 'hello' },
    });
    const fallback = guard.runAfterAutoLayoutIfNeeded(
      { op: 'autoLayout' },
      input => {
        calls.push(input);
        return Promise.resolve();
      },
      1000,
    );

    guard.noteCommand({ op: 'testRun', args: { input: { input: 'hello' } } });
    await vi.advanceTimersByTimeAsync(1000);

    await expect(fallback).resolves.toBe(false);
    expect(calls).toEqual([]);
    expect(guard.peek()).toBeNull();
  });

  it('waits for workflow saving to become idle before test run', async () => {
    vi.useFakeTimers();
    let saving = true;

    const wait = waitForWorkflowAgentSavingIdle({
      isSaving: () => saving,
      timeoutMs: 1000,
      intervalMs: 100,
    });

    await vi.advanceTimersByTimeAsync(300);
    saving = false;
    await vi.advanceTimersByTimeAsync(100);

    await expect(wait).resolves.toBe(true);
  });

  it('stops waiting when workflow saving never becomes idle', async () => {
    vi.useFakeTimers();

    const wait = waitForWorkflowAgentSavingIdle({
      isSaving: () => true,
      timeoutMs: 300,
      intervalMs: 100,
    });

    await vi.advanceTimersByTimeAsync(300);

    await expect(wait).resolves.toBe(false);
  });
});
