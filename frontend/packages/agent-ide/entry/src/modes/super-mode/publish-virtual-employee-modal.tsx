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

import { useCallback, useEffect, useRef, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { Input, Modal, Toast } from '@coze-arch/coze-design';
import { axiosInstance } from '@coze-arch/bot-api';

// ─── API helpers ──────────────────────────────────────────────────────────────

interface PublishResult {
  code: number;
  msg: string;
  data: { product_id: string; version: string };
}

interface BuildStatusResult {
  code: number;
  msg: string;
  data: { product_id: string; version: string; build_status: string };
}

const postApi = <T,>(url: string, data: Record<string, unknown>): Promise<T> =>
  axiosInstance.request({
    url,
    method: 'POST',
    data,
    withCredentials: true,
  }) as unknown as Promise<T>;

export const agentAppApi = {
  publish: (p: {
    bot_id: string;
    space_id: string;
    name: string;
    version: string;
  }): Promise<PublishResult> =>
    postApi<PublishResult>('/api/super-agent/agent-app/publish', {
      agent_id: p.bot_id,
      space_id: p.space_id,
      name: p.name,
      version: p.version,
    }),

  buildStatus: (p: {
    product_id: string;
    space_id?: string;
    version?: string;
  }): Promise<BuildStatusResult> =>
    postApi<BuildStatusResult>('/api/super-agent/agent-app/build-status', {
      product_id: p.product_id,
      space_id: p.space_id,
      version: p.version,
    }),
};

// ─── Types & constants ────────────────────────────────────────────────────────

type BuildPhase = 'idle' | 'building' | 'ready' | 'failed';

const PHASE_LABEL: Record<BuildPhase, string> = {
  idle: '',
  building: '构建中',
  ready: '已就绪',
  failed: '失败',
};

const PHASE_COLOR: Record<
  Exclude<BuildPhase, 'idle'>,
  { bg: string; fg: string }
> = {
  building: { bg: 'rgba(46,115,255,0.10)', fg: 'rgb(46,115,255)' },
  ready: { bg: 'rgba(5,150,105,0.10)', fg: 'rgb(5,150,105)' },
  failed: { bg: 'rgba(239,68,68,0.10)', fg: 'rgb(220,38,38)' },
};

const POLL_MS = 2000;
const POLL_MAX = 60;

// ─── Polling hook ─────────────────────────────────────────────────────────────

const usePublishFlow = (spaceId: string) => {
  const [phase, setPhase] = useState<BuildPhase>('idle');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const countRef = useRef(0);

  const stopPoll = useCallback(() => {
    if (pollRef.current !== null) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  useEffect(() => () => stopPoll(), [stopPoll]);

  const reset = useCallback(() => {
    stopPoll();
    setPhase('idle');
  }, [stopPoll]);

  const startPoll = useCallback(
    (productId: string, ver: string) => {
      countRef.current = 0;
      setPhase('building');
      pollRef.current = setInterval(() => {
        countRef.current += 1;
        if (countRef.current > POLL_MAX) {
          stopPoll();
          setPhase('failed');
          Toast.error('构建超时，请稍后重试');
          return;
        }
        agentAppApi
          .buildStatus({
            product_id: productId,
            space_id: spaceId,
            version: ver,
          })
          .then(res => {
            if (res.code !== 0) {
              stopPoll();
              setPhase('failed');
              Toast.error(res.msg || '构建状态查询失败');
              return;
            }
            const s = res.data?.build_status;
            if (s === 'ready') {
              stopPoll();
              setPhase('ready');
              Toast.success('虚拟员工已就绪');
            } else if (s === 'failed') {
              stopPoll();
              setPhase('failed');
              Toast.error('构建失败，请重试');
            }
          })
          .catch((err: unknown) => {
            // network error: keep polling; log for observability
            console.warn('[publish-virtual-employee] poll error:', err);
          });
      }, POLL_MS);
    },
    [spaceId, stopPoll],
  );

  return { phase, reset, startPoll };
};

// ─── Modal ────────────────────────────────────────────────────────────────────

export interface PublishVirtualEmployeeModalProps {
  visible: boolean;
  onClose: () => void;
}

export const PublishVirtualEmployeeModal: React.FC<
  PublishVirtualEmployeeModalProps
> = ({ visible, onClose }) => {
  const botId = useBotInfoStore((s: { botId: string }) => s.botId);
  const spaceId = useBotInfoStore((s: { space_id: string }) => s.space_id);

  const [name, setName] = useState('');
  const [version, setVersion] = useState('v1.0.0');
  const [submitting, setSubmitting] = useState(false);
  const { phase, reset, startPoll } = usePublishFlow(spaceId);

  const isBuilding = phase === 'building';
  const isDone = phase === 'ready' || phase === 'failed';

  const handleClose = useCallback(() => {
    reset();
    setSubmitting(false);
    onClose();
  }, [onClose, reset]);

  const handleSubmit = useCallback(async () => {
    if (!botId || !spaceId) {
      Toast.error('智能体信息未就绪，请稍后重试');
      return;
    }
    const trimmedName = name.trim();
    if (!trimmedName) {
      Toast.error('请填写虚拟员工名称');
      return;
    }
    setSubmitting(true);
    try {
      const res = await agentAppApi.publish({
        bot_id: botId,
        space_id: spaceId,
        name: trimmedName,
        version: version.trim() || 'v1.0.0',
      });
      if (res.code !== 0) {
        Toast.error(res.msg || '发布失败');
        return;
      }
      startPoll(res.data?.product_id, res.data?.version);
    } catch (err: unknown) {
      Toast.error('发布请求失败，请重试');
      console.warn('[publish-virtual-employee] publish error:', err);
    } finally {
      setSubmitting(false);
    }
  }, [botId, name, spaceId, startPoll, version]);

  const fieldDisabled = isBuilding || submitting || isDone;
  const phaseColors = phase !== 'idle' ? PHASE_COLOR[phase] : null;

  return (
    <Modal
      visible={visible}
      title="发布为虚拟员工"
      onCancel={handleClose}
      onOk={isDone ? handleClose : handleSubmit}
      okText={isDone ? '关闭' : isBuilding ? '构建中…' : '发布'}
      cancelText="取消"
      okButtonProps={{ disabled: isBuilding || submitting }}
      cancelButtonProps={{ disabled: isBuilding || submitting }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div>
          <div
            style={{ marginBottom: 6, fontSize: 13, fontWeight: 500 }}
            data-testid="name-label"
          >
            虚拟员工名称
            <span
              style={{ color: 'var(--coz-fg-danger, #f5222d)', marginLeft: 2 }}
            >
              *
            </span>
          </div>
          <Input
            placeholder="为这位虚拟员工起个名字"
            value={name}
            onChange={(v: string) => setName(v)}
            disabled={fieldDisabled}
            data-testid="name-input"
          />
        </div>
        <div>
          <div
            style={{ marginBottom: 6, fontSize: 13, fontWeight: 500 }}
            data-testid="version-label"
          >
            版本号
          </div>
          <Input
            placeholder="v1.0.0"
            value={version}
            onChange={(v: string) => setVersion(v)}
            disabled={fieldDisabled}
            data-testid="version-input"
          />
        </div>
        {phaseColors ? (
          <div
            data-testid="build-status"
            style={{
              padding: '8px 12px',
              borderRadius: 6,
              fontSize: 13,
              background: phaseColors.bg,
              color: phaseColors.fg,
            }}
          >
            {PHASE_LABEL[phase]}
          </div>
        ) : null}
      </div>
    </Modal>
  );
};
