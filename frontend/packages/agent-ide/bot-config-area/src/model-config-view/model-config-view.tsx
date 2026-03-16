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

import { I18n } from '@coze-arch/i18n';
import { BotMode } from '@coze-arch/bot-api/playground_api';

import { SingleAgentModelView } from './single-agent-model-view';
import { DialogueConfigView } from './dialogue-config-view';

export const ModelConfigView: React.FC<{
  mode: BotMode;
  modelListExtraHeaderSlot?: React.ReactNode;
}> = ({ mode, modelListExtraHeaderSlot }) => {
  // Note: Always render SingleAgentModelView for SingleMode
  // The component handles fallback to first available model internally
  // Do not check currentModel?.model_type here as it would hide the selector
  // when the model is not found (e.g., after import)
  if (mode === BotMode.SingleMode) {
    return (
      <SingleAgentModelView
        modelListExtraHeaderSlot={modelListExtraHeaderSlot}
      />
    );
  }
  if (mode === BotMode.MultiMode || mode === BotMode.WorkflowMode) {
    return (
      <DialogueConfigView
        tips={
          mode === BotMode.WorkflowMode
            ? I18n.t('workflow_agent_dialog_set_desc')
            : null
        }
      />
    );
  }
  return null;
};
