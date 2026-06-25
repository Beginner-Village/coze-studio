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

import { useNavigate, useSearchParams } from 'react-router-dom';
import { useMemo } from 'react';

import { BuilderChat } from '@coze-studio/open-chat';

/**
 * EmployeeChatPage —— 虚拟员工独立聊天界面。
 *
 * 与编辑/编排页（/bot/:id/arrange）完全分离：招聘者只对话、不暴露编辑画布。
 * 复用 open-chat 的 BuilderChat，使用 internal（cookie 会话）鉴权，对话对象为
 * 招聘后生成的只读影子智能体（bot draft）。
 *
 * URL 参数：
 *   - agent_id: 影子智能体 ID（招聘返回的 shadow_agent_id）
 *   - name:     展示名称（虚拟员工名）
 */
export default function EmployeeChatPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const agentId = searchParams.get('agent_id') || '';
  const name = searchParams.get('name') || '虚拟员工';

  const workflow = useMemo(() => ({ id: undefined }), []);
  const project = useMemo(
    () => ({
      id: agentId,
      type: 'bot' as const,
      mode: 'draft' as const,
      name,
    }),
    [agentId, name],
  );
  const auth = useMemo(() => ({ type: 'internal' as const }), []);

  if (!agentId) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          color: '#86909c',
          fontSize: 14,
        }}
      >
        缺少 agent_id 参数
      </div>
    );
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        width: '100%',
        height: '100vh',
        overflow: 'hidden',
        background: '#f7f8fa',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          height: 52,
          flex: '0 0 52px',
          padding: '0 16px',
          borderBottom: '1px solid rgba(28, 31, 35, 0.08)',
          background: '#fff',
        }}
      >
        <button
          type="button"
          onClick={() => navigate(-1)}
          style={{
            cursor: 'pointer',
            border: 'none',
            background: 'transparent',
            fontSize: 14,
            color: '#4e5969',
            padding: '6px 10px',
            borderRadius: 6,
          }}
        >
          ← 返回
        </button>
        <span style={{ fontSize: 15, fontWeight: 600, color: '#1d2129' }}>
          {name}
        </span>
        <span style={{ fontSize: 12, color: '#86909c' }}>虚拟员工</span>
      </div>
      <div style={{ flex: 1, minHeight: 0 }}>
        <BuilderChat
          workflow={workflow}
          project={project}
          auth={auth}
          userInfo={undefined}
          areaUi={{
            header: { isShow: false },
            input: { isShow: true },
            uploadable: true,
          }}
          setting={{}}
        />
      </div>
    </div>
  );
}
