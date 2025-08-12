/*
 * Copyright 2025 coze-dev Authors
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

import {
  CardSelectorModal,
  type CardItem,
} from '@/components/card-selector-modal';

export interface UseCardSelectorModalProps {
  closeCallback?: () => void;
  openModeCallback?: (
    selectedCards: CardItem[],
  ) => Promise<boolean | undefined>;
}

export const useCardSelectorModal = (props?: UseCardSelectorModalProps) => {
  const { closeCallback, openModeCallback } = props || {};
  const [visible, setVisible] = useState(false);

  const open = () => {
    setVisible(true);
  };

  const close = () => {
    setVisible(false);
    closeCallback?.();
  };

  const handleConfirm = async (selectedCards: CardItem[]) => {
    if (openModeCallback) {
      try {
        const result = await openModeCallback(selectedCards);
        // 如果回调返回false，不关闭弹框
        if (result !== false) {
          setVisible(false);
          closeCallback?.();
        }
      } catch (error) {
        console.error('Card selector callback error:', error);
        // 出错时也关闭弹框
        setVisible(false);
        closeCallback?.();
      }
    } else {
      setVisible(false);
      closeCallback?.();
    }
  };

  const node = visible ? (
    <CardSelectorModal
      visible={visible}
      onCancel={close}
      onConfirm={handleConfirm}
      title="选择猎鹰卡片"
    />
  ) : null;

  return {
    node,
    open,
    close,
  };
};
