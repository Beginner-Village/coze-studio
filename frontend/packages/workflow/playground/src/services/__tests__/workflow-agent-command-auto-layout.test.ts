import { describe, expect, it, vi } from 'vitest';

import { autoLayoutAfterAgentCommandIfNeeded } from '../workflow-agent-command-auto-layout';

describe('workflow-agent-command-auto-layout', () => {
  it('auto layouts after each successful mutating streamed command', async () => {
    const autoLayout = vi.fn(() => Promise.resolve());

    await expect(
      autoLayoutAfterAgentCommandIfNeeded(
        { op: 'addNode' },
        { ok: true, nodeId: 'node-1' },
        autoLayout,
      ),
    ).resolves.toEqual({ ok: true, nodeId: 'node-1' });

    expect(autoLayout).toHaveBeenCalledTimes(1);
  });

  it('does not auto layout after failed commands, explicit layout, or test run', async () => {
    const autoLayout = vi.fn(() => Promise.resolve());

    await autoLayoutAfterAgentCommandIfNeeded(
      { op: 'addNode' },
      { ok: false },
      autoLayout,
    );
    await autoLayoutAfterAgentCommandIfNeeded(
      { op: 'autoLayout' },
      { ok: true },
      autoLayout,
    );
    await autoLayoutAfterAgentCommandIfNeeded(
      { op: 'testRun' },
      { ok: true },
      autoLayout,
    );

    expect(autoLayout).not.toHaveBeenCalled();
  });
});
