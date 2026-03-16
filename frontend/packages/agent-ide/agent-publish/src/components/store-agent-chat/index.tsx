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

import React from 'react';

// 商店智能体聊天组件的占位符
// 实际功能将通过ConditionalAgentLayout来处理
export const StoreAgentChat: React.FC = () => {
  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      backgroundColor: '#f5f5f5'
    }}>
      <div style={{
        fontSize: '16px',
        color: '#666',
        textAlign: 'center'
      }}>
        <div style={{ marginBottom: '12px' }}>🤖</div>
        <div>商店智能体聊天界面加载中...</div>
        <div style={{ fontSize: '12px', marginTop: '8px', color: '#999' }}>
          这个组件将被ConditionalAgentLayout替换
        </div>
      </div>
    </div>
  );
};