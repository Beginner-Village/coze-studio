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

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import classNames from 'classnames';
import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { DeveloperApi } from '@coze-arch/bot-api';
import { Modal, Toast } from '@coze-arch/coze-design';

import {
  IcCheckList,
  IcClock,
  IcEdit,
  IcPlus,
  IcRefresh,
  IcTrash,
} from './icons';

import ss from './super-session-sidebar.module.less';

interface SuperAgentSession {
  session_id?: string;
  conversation_id?: string;
  title?: string;
  renamable?: boolean;
  created_at?: string | number;
  updated_at?: string | number;
  scene?: number;
}

const SESSION_PAGE_SIZE = 30;
const SUPER_AGENT_SESSION_SELECT_EVENT = 'coze:super-agent-session-select';
const superAgentSessionApi = DeveloperApi as typeof DeveloperApi & {
  SuperAgentDeleteSession: (params: {
    conversation_id: string;
  }) => Promise<unknown>;
};

const normalizeTimestamp = (value?: string | number) => {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) {
    return 0;
  }
  return n < 1_000_000_000_000 ? n * 1000 : n;
};

const formatSessionTime = (value?: string | number) => {
  const time = normalizeTimestamp(value);
  if (!time) {
    return '刚刚';
  }
  const d = new Date(time);
  const now = new Date();
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate();
  if (sameDay) {
    return d.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  }
  return d.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  });
};

const getSessionID = (session: SuperAgentSession) =>
  session.session_id || session.conversation_id || '';

// 与 chat-area provider 共用的持久化:按 bot 记住当前选中会话。
// 选中会话会让上层 provider 以 key={botId:conversationId:scene} 重挂载整棵子树(含本侧栏),
// 本地 useState 选中态因此丢失。这里与 provider 读写同一份 sessionStorage,使重挂载后
// 侧栏与 provider 恢复到同一个会话,既不回退也不产生二次重挂载/闪烁。
// 键名/格式必须与 chat-area-provider-adapter 保持一致。
const SELECTED_SESSION_KEY_PREFIX = 'super-agent:selected-session:';

// 模块级记录「每个 bot 最近一次已广播给 provider 的会话」(跨重挂载存活)。
// 用途:loadSessions 只在「目标会话 ≠ 已广播会话」时广播 —— 首次加载会广播(让 provider
// 加载并回显该会话历史),重挂载后因目标未变而不再广播(不触发二次重挂载/闪烁)。
const lastDispatchedByBot = new Map<string, string>();

const readPersistedSessionID = (botId: string): string => {
  if (!botId || typeof window === 'undefined') {
    return '';
  }
  try {
    const raw = window.sessionStorage.getItem(
      `${SELECTED_SESSION_KEY_PREFIX}${botId}`,
    );
    if (!raw) {
      return '';
    }
    return (JSON.parse(raw) as { conversationId?: string }).conversationId ?? '';
  } catch {
    return '';
  }
};

const writePersistedSession = (
  botId: string,
  sessionID: string,
  scene?: number,
) => {
  if (!botId || !sessionID || typeof window === 'undefined') {
    return;
  }
  try {
    window.sessionStorage.setItem(
      `${SELECTED_SESSION_KEY_PREFIX}${botId}`,
      JSON.stringify({ conversationId: sessionID, scene }),
    );
  } catch {
    /* sessionStorage 不可用时忽略 */
  }
};

const emitSessionSelection = (params: {
  botId: string;
  conversationId: string;
  scene?: number;
}) => {
  window.dispatchEvent(
    new CustomEvent(SUPER_AGENT_SESSION_SELECT_EVENT, {
      detail: params,
    }),
  );
};

interface SessionListItemProps {
  session: SuperAgentSession;
  sessionID: string;
  isActive: boolean;
  isEditing: boolean;
  draftTitle: string;
  onSelect: (sessionID: string, session: SuperAgentSession) => void;
  onStartRename: (session: SuperAgentSession) => void;
  onCommitRename: (session: SuperAgentSession) => void;
  onCancelRename: () => void;
  onDraftTitleChange: (title: string) => void;
  onDelete: (session: SuperAgentSession) => void;
  selectMode: boolean;
  isSelected: boolean;
  onToggleSelected: (sessionID: string) => void;
}

const SessionListItem: React.FC<SessionListItemProps> = ({
  session,
  sessionID,
  isActive,
  isEditing,
  draftTitle,
  onSelect,
  onStartRename,
  onCommitRename,
  onCancelRename,
  onDraftTitleChange,
  onDelete,
  selectMode,
  isSelected,
  onToggleSelected,
}) => {
  const selectable = selectMode && Boolean(session.renamable);
  const handleActivate = () => {
    if (selectable) {
      onToggleSelected(sessionID);
      return;
    }
    if (selectMode) {
      return;
    }
    onSelect(sessionID, session);
  };
  return (
  <div
    role="button"
    tabIndex={0}
    className={classNames(
      ss.sessionItem,
      isActive && !selectMode && ss.active,
      selectable && isSelected && ss.selected,
    )}
    onClick={handleActivate}
    onKeyDown={e => {
      if (e.key === 'Enter') {
        handleActivate();
      }
    }}
  >
    {selectMode ? (
      <span
        className={classNames(
          ss.checkbox,
          isSelected && ss.checkboxChecked,
          !session.renamable && ss.checkboxDisabled,
        )}
      />
    ) : null}
    <div className={ss.sessionMain}>
      {isEditing ? (
        <input
          autoFocus
          className={ss.renameInput}
          value={draftTitle}
          onChange={e => onDraftTitleChange(e.target.value)}
          onBlur={() => onCommitRename(session)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              onCommitRename(session);
            }
            if (e.key === 'Escape') {
              onCancelRename();
            }
          }}
        />
      ) : (
        <span className={ss.sessionTitle}>
          {session.title || '未命名会话'}
        </span>
      )}
      <span className={ss.sessionMeta}>
        <IcClock size={13} />
        {formatSessionTime(session.updated_at || session.created_at)}
      </span>
    </div>
    {session.renamable && !isEditing && !selectMode ? (
      <span className={ss.sessionActions}>
        <span
          role="button"
          tabIndex={0}
          className={ss.sessionActionButton}
          aria-label="重命名会话"
          onClick={e => {
            e.stopPropagation();
            onStartRename(session);
          }}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.stopPropagation();
              onStartRename(session);
            }
          }}
        >
          <IcEdit size={14} />
        </span>
        <span
          role="button"
          tabIndex={0}
          className={classNames(ss.sessionActionButton, ss.deleteButton)}
          aria-label="删除会话"
          onClick={e => {
            e.stopPropagation();
            onDelete(session);
          }}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.stopPropagation();
              onDelete(session);
            }
          }}
        >
          <IcTrash size={14} />
        </span>
      </span>
    ) : null}
  </div>
  );
};

const confirmDeleteSession = ({
  activeSessionID,
  session,
  setActiveSessionID,
  setSessions,
}: {
  activeSessionID: string;
  session: SuperAgentSession;
  setActiveSessionID: React.Dispatch<React.SetStateAction<string>>;
  setSessions: React.Dispatch<React.SetStateAction<SuperAgentSession[]>>;
}) => {
  const sessionID = getSessionID(session);
  if (!sessionID || !session.renamable) {
    return;
  }
  Modal.confirm({
    title: '删除会话',
    content: `确定要删除「${session.title || '未命名会话'}」吗？`,
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    onOk: async () => {
      try {
        await superAgentSessionApi.SuperAgentDeleteSession({
          conversation_id: sessionID,
        });
        setSessions(prev => {
          const nextSessions = prev.filter(
            item => getSessionID(item) !== sessionID,
          );
          if (activeSessionID === sessionID) {
            const nextActive = getSessionID(nextSessions[0] ?? {});
            setActiveSessionID(nextActive);
          }
          return nextSessions;
        });
        Toast.success('会话已删除');
      } catch {
        Toast.error('会话删除失败');
      }
    },
  });
};

const useSuperAgentSessions = (botId: string, spaceId: string) => {
  const [sessions, setSessions] = useState<SuperAgentSession[]>([]);
  // 用持久化的选中会话初始化(重挂载后保持高亮);注意:绝不在挂载/重挂载时广播,
  // 广播只发生在用户点击或首次加载,否则会触发上层重挂载→再广播的死循环。
  const [activeSessionID, setActiveSessionID] = useState(() =>
    readPersistedSessionID(botId),
  );
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [editingSessionID, setEditingSessionID] = useState('');
  const [draftTitle, setDraftTitle] = useState('');
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
  const [batchDeleting, setBatchDeleting] = useState(false);
  const lastSelectionKeyRef = useRef('');

  const activeSession = useMemo(
    () => sessions.find(item => getSessionID(item) === activeSessionID),
    [activeSessionID, sessions],
  );

  // 跟踪当前选中,供 loadSessions 同步计算下一个选中而无需把 activeSessionID 加入依赖。
  const activeSessionIDRef = useRef(activeSessionID);
  activeSessionIDRef.current = activeSessionID;

  const emitActiveSessionSelection = useCallback(
    (sessionID: string, session?: SuperAgentSession) => {
      if (!botId || !sessionID) {
        return;
      }
      // 持久化当前选中(与 provider 共用),供重挂载后恢复。
      writePersistedSession(botId, sessionID, session?.scene);
      // 记录最近广播的会话(模块级,跨重挂载存活),供 loadSessions 判断是否需要广播。
      lastDispatchedByBot.set(botId, sessionID);
      const selectionKey = `${botId}:${sessionID}:${session?.scene ?? ''}`;
      if (lastSelectionKeyRef.current === selectionKey) {
        return;
      }
      lastSelectionKeyRef.current = selectionKey;
      emitSessionSelection({
        botId,
        conversationId: sessionID,
        scene: session?.scene,
      });
    },
    [botId],
  );

  const loadSessions = useCallback(async () => {
    if (!botId) {
      setSessions([]);
      return;
    }
    setLoading(true);
    try {
      const resp = await DeveloperApi.SuperAgentListSessions({
        space_id: spaceId,
        bot_id: botId,
        page: 1,
        page_size: SESSION_PAGE_SIZE,
      });
      const nextSessions =
        (resp?.data?.sessions as SuperAgentSession[] | undefined) ?? [];
      setSessions(nextSessions);
      const exists = (id: string) =>
        Boolean(id) && nextSessions.some(item => getSessionID(item) === id);
      const persisted = readPersistedSessionID(botId);
      // 解析下一个选中:① 当前仍在列表→保持;② 否则恢复持久化;③ 兜底第一个。
      let nextActive = activeSessionIDRef.current;
      if (!exists(nextActive)) {
        nextActive = exists(persisted)
          ? persisted
          : getSessionID(nextSessions[0] ?? {});
      }
      setActiveSessionID(nextActive);
      // 仅当"目标会话 ≠ 最近广播给 provider 的会话"时才广播:
      //   - 首次加载:map 无记录 → 广播 → provider 加载并回显该会话历史;
      //   - 重挂载后:nextActive 已是最近广播的会话 → 不广播 → 不触发二次重挂载/闪烁。
      if (nextActive && lastDispatchedByBot.get(botId) !== nextActive) {
        const session = nextSessions.find(
          item => getSessionID(item) === nextActive,
        );
        emitActiveSessionSelection(nextActive, session);
      }
    } catch {
      Toast.error('会话列表加载失败');
    } finally {
      setLoading(false);
    }
  }, [botId, spaceId, emitActiveSessionSelection]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  const startRename = (session: SuperAgentSession) => {
    if (!session.renamable) {
      return;
    }
    setEditingSessionID(getSessionID(session));
    setDraftTitle(session.title || '未命名会话');
  };

  const cancelRename = () => {
    setEditingSessionID('');
    setDraftTitle('');
  };

  const commitRename = async (session: SuperAgentSession) => {
    const sessionID = getSessionID(session);
    const title = draftTitle.trim();
    if (!sessionID || !title || title === session.title) {
      cancelRename();
      return;
    }
    try {
      await DeveloperApi.SuperAgentRenameSession({
        conversation_id: sessionID,
        title,
      });
      setSessions(prev =>
        prev.map(item =>
          getSessionID(item) === sessionID ? { ...item, title } : item,
        ),
      );
      Toast.success('会话已重命名');
    } catch {
      Toast.error('会话重命名失败');
    } finally {
      cancelRename();
    }
  };

  const deleteSession = (session: SuperAgentSession) => {
    confirmDeleteSession({
      activeSessionID,
      session,
      setActiveSessionID,
      setSessions,
    });
  };

  const deletableSessions = useMemo(
    () => sessions.filter(item => item.renamable && getSessionID(item)),
    [sessions],
  );

  const toggleSelectMode = () => {
    setSelectMode(prev => {
      if (prev) {
        setSelectedIDs(new Set());
      }
      return !prev;
    });
  };

  const toggleSelected = (sessionID: string) => {
    setSelectedIDs(prev => {
      const next = new Set(prev);
      if (next.has(sessionID)) {
        next.delete(sessionID);
      } else {
        next.add(sessionID);
      }
      return next;
    });
  };

  const allSelected =
    deletableSessions.length > 0 &&
    deletableSessions.every(item => selectedIDs.has(getSessionID(item)));

  const toggleSelectAll = () => {
    setSelectedIDs(
      allSelected
        ? new Set()
        : new Set(deletableSessions.map(item => getSessionID(item))),
    );
  };

  const batchDelete = () => {
    const ids = Array.from(selectedIDs);
    if (ids.length === 0 || batchDeleting) {
      return;
    }
    Modal.confirm({
      title: '批量删除会话',
      content: `确定要删除选中的 ${ids.length} 个会话吗？`,
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        setBatchDeleting(true);
        const results = await Promise.allSettled(
          ids.map(id =>
            superAgentSessionApi.SuperAgentDeleteSession({
              conversation_id: id,
            }),
          ),
        );
        const failed = results.filter(r => r.status === 'rejected').length;
        const deleted = new Set(
          ids.filter((_, i) => results[i].status === 'fulfilled'),
        );
        setSessions(prev => {
          const nextSessions = prev.filter(
            item => !deleted.has(getSessionID(item)),
          );
          if (deleted.has(activeSessionID)) {
            setActiveSessionID(getSessionID(nextSessions[0] ?? {}));
          }
          return nextSessions;
        });
        setSelectedIDs(new Set());
        setSelectMode(false);
        setBatchDeleting(false);
        if (failed > 0) {
          Toast.error(`已删除 ${deleted.size} 个，${failed} 个失败`);
        } else {
          Toast.success(`已删除 ${deleted.size} 个会话`);
        }
      },
    });
  };

  const selectSession = (sessionID: string, session?: SuperAgentSession) => {
    setActiveSessionID(sessionID);
    emitActiveSessionSelection(sessionID, session);
  };

  const createSession = async () => {
    if (!botId || creating) {
      return;
    }
    setCreating(true);
    try {
      const resp = await DeveloperApi.SuperAgentCreateSession({
        space_id: spaceId,
        bot_id: botId,
        title: '新会话',
      });
      const session = resp?.data?.session as SuperAgentSession | undefined;
      const sessionID = getSessionID(session ?? {});
      if (!session || !sessionID) {
        throw new Error('missing session');
      }
      setSessions(prev => [
        session,
        ...prev.filter(item => getSessionID(item) !== sessionID),
      ]);
      selectSession(sessionID, session);
    } catch {
      Toast.error('新会话创建失败');
    } finally {
      setCreating(false);
    }
  };

  return {
    activeSession,
    activeSessionID,
    cancelRename,
    commitRename,
    createSession,
    creating,
    deleteSession,
    draftTitle,
    editingSessionID,
    loadSessions,
    loading,
    selectSession,
    sessions,
    setDraftTitle,
    startRename,
    selectMode,
    selectedIDs,
    batchDeleting,
    toggleSelectMode,
    toggleSelected,
    allSelected,
    toggleSelectAll,
    batchDelete,
    deletableCount: deletableSessions.length,
  };
};

export const SuperSessionSidebar: React.FC = () => {
  const botId = useBotInfoStore((state: { botId: string }) => state.botId);
  const spaceId = useBotInfoStore(
    (state: { space_id: string }) => state.space_id,
  );
  const {
    activeSession,
    activeSessionID,
    cancelRename,
    commitRename,
    createSession,
    creating,
    deleteSession,
    draftTitle,
    editingSessionID,
    loadSessions,
    loading,
    selectSession,
    sessions,
    setDraftTitle,
    startRename,
    selectMode,
    selectedIDs,
    batchDeleting,
    toggleSelectMode,
    toggleSelected,
    allSelected,
    toggleSelectAll,
    batchDelete,
    deletableCount,
  } = useSuperAgentSessions(botId, spaceId);

  return (
    <aside className={ss.sidebar}>
      <div className={ss.header}>
        <div>
          <div className={ss.title}>会话</div>
          <div className={ss.subtitle}>
            {activeSession?.title || '新的工作会话'}
          </div>
        </div>
        <div className={ss.headerActions}>
          <button
            type="button"
            className={classNames(ss.iconButton, selectMode && ss.active)}
            aria-label={selectMode ? '退出多选' : '多选'}
            disabled={deletableCount === 0}
            onClick={toggleSelectMode}
          >
            <IcCheckList size={15} />
          </button>
          <button
            type="button"
            className={ss.iconButton}
            aria-label="刷新会话"
            onClick={loadSessions}
          >
            <IcRefresh size={15} />
          </button>
          <button
            type="button"
            className={ss.iconButtonPrimary}
            aria-label="新会话"
            disabled={creating}
            onClick={createSession}
          >
            <IcPlus size={15} />
          </button>
        </div>
      </div>

      {selectMode ? (
        <div className={ss.batchBar}>
          <button
            type="button"
            className={ss.batchSelectAll}
            onClick={toggleSelectAll}
          >
            <span
              className={classNames(
                ss.checkbox,
                allSelected && ss.checkboxChecked,
              )}
            />
            {allSelected ? '取消全选' : '全选'}
          </button>
          <span className={ss.batchCount}>已选 {selectedIDs.size}</span>
          <button
            type="button"
            className={ss.batchDeleteButton}
            disabled={selectedIDs.size === 0 || batchDeleting}
            onClick={batchDelete}
          >
            <IcTrash size={13} />
            删除
          </button>
        </div>
      ) : null}

      <div className={ss.currentCard}>
        <div className={ss.currentLabel}>当前</div>
        <div className={ss.currentTitle}>
          {activeSession?.title || '新会话'}
        </div>
      </div>

      <div className={ss.listHeader}>
        <span>最近会话</span>
        <span>{loading ? '同步中' : `${sessions.length}`}</span>
      </div>

      <div className={ss.list}>
        {sessions.map(session => {
          const sessionID = getSessionID(session);
          return (
            <SessionListItem
              key={sessionID}
              session={session}
              sessionID={sessionID}
              isActive={sessionID === activeSessionID}
              isEditing={sessionID === editingSessionID}
              draftTitle={draftTitle}
              onSelect={selectSession}
              onStartRename={startRename}
              onCommitRename={commitRename}
              onCancelRename={cancelRename}
              onDraftTitleChange={setDraftTitle}
              onDelete={deleteSession}
              selectMode={selectMode}
              isSelected={selectedIDs.has(sessionID)}
              onToggleSelected={toggleSelected}
            />
          );
        })}
        {!loading && sessions.length === 0 ? (
          <div className={ss.empty}>暂无会话</div>
        ) : null}
      </div>
    </aside>
  );
};
