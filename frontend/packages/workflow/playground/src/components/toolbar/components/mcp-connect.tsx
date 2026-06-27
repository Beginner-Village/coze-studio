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

import { useState } from 'react';

import { IconCozLink } from '@coze-arch/coze-design/icons';
import { Tooltip, IconButton } from '@coze-arch/coze-design';

import { ExternalMCPGuideModal } from '../../workflow-agent-panel/external-mcp-guide-modal';

/**
 * 工具栏入口:打开「接入外部AI(MCP)」说明弹窗,让外部 MCP 客户端
 * (Claude Code / Codex 等)知道如何连上本平台的工作流 MCP server 来搭建/修改工作流。
 * 复用 ExternalMCPGuideModal(同一份说明,AI 面板与此处共用)。
 */
export const MCPConnect = () => {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Tooltip content="接入外部AI(MCP):让 Claude Code / Codex 等外部 AI 连上来搭建工作流">
        <IconButton
          icon={<IconCozLink className="coz-fg-primary" />}
          color="secondary"
          data-testid="workflow.detail.controls.mcp-connect"
          onClick={() => setOpen(true)}
        />
      </Tooltip>
      <ExternalMCPGuideModal visible={open} onClose={() => setOpen(false)} />
    </>
  );
};
