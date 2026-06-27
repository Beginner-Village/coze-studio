const STORAGE_PREFIX = 'coze.workflowAgentPanel.open';

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

const memoryState = new Map<string, boolean>();

const getStorage = (): StorageLike | undefined => {
  if (typeof window === 'undefined') {
    return undefined;
  }
  return window.localStorage;
};

export const getWorkflowAgentPanelOpenStateKey = (
  workflowId?: string,
): string => `${STORAGE_PREFIX}:${workflowId || 'default'}`;

export const readWorkflowAgentPanelOpenState = (
  workflowId?: string,
  storage = getStorage(),
): boolean => {
  const key = getWorkflowAgentPanelOpenStateKey(workflowId);
  if (memoryState.has(key)) {
    return Boolean(memoryState.get(key));
  }
  try {
    return storage?.getItem(key) === '1';
  } catch {
    return false;
  }
};

export const writeWorkflowAgentPanelOpenState = (
  workflowId: string | undefined,
  open: boolean,
  storage = getStorage(),
): void => {
  const key = getWorkflowAgentPanelOpenStateKey(workflowId);
  memoryState.set(key, open);
  try {
    if (open) {
      storage?.setItem(key, '1');
    } else {
      storage?.removeItem(key);
    }
  } catch {
    // The in-memory value still preserves state across React remounts.
  }
};

export const clearWorkflowAgentPanelOpenStateForTest = (): void => {
  memoryState.clear();
};
