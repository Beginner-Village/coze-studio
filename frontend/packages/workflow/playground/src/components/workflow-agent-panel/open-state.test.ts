import { afterEach, describe, expect, it } from 'vitest';

import {
  clearWorkflowAgentPanelOpenStateForTest,
  readWorkflowAgentPanelOpenState,
  writeWorkflowAgentPanelOpenState,
} from './open-state';

const createStorage = () => {
  const values = new Map<string, string>();
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    removeItem: (key: string) => {
      values.delete(key);
    },
  };
};

describe('workflow-agent-panel open-state', () => {
  afterEach(() => {
    clearWorkflowAgentPanelOpenStateForTest();
  });

  it('restores an open panel for the same workflow after remount', () => {
    const storage = createStorage();

    writeWorkflowAgentPanelOpenState('workflow-1', true, storage);

    expect(readWorkflowAgentPanelOpenState('workflow-1', storage)).toBe(true);
    expect(readWorkflowAgentPanelOpenState('workflow-2', storage)).toBe(false);
  });

  it('clears the restored state when the user closes the panel', () => {
    const storage = createStorage();

    writeWorkflowAgentPanelOpenState('workflow-1', true, storage);
    writeWorkflowAgentPanelOpenState('workflow-1', false, storage);

    expect(readWorkflowAgentPanelOpenState('workflow-1', storage)).toBe(false);
  });
});
