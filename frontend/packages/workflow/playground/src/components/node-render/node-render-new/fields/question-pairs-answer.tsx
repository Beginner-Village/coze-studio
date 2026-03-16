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

import { get } from 'lodash-es';
import classnames from 'classnames';
import { I18n } from '@coze-arch/i18n';

import { LabelWithTooltip } from './label-with-tooltip';
import { FieldEmpty } from './field-empty';

import styles from './question-pairs.module.less';

// 提取实际的内容值，处理对象和字符串两种情况
const extractContent = (content: unknown): string | null => {
  if (content === null || content === undefined) {
    return null;
  }
  if (typeof content === 'string') {
    // 保留空格等空白字符，只有真正的空字符串才返回 null
    return content === '' ? null : content;
  }
  if (typeof content === 'object') {
    // 尝试从对象中提取 content.value.content 或 content.content
    const nestedContent =
      get(content, 'value.content') ?? get(content, 'content');
    if (typeof nestedContent === 'string') {
      return nestedContent === '' ? null : nestedContent;
    }
  }
  return null;
};

export const AnswerItem = ({
  label,
  content,
  showLabel = true,
  maxWidth = 148,
}) => {
  const displayContent = extractContent(content);

  return (
    <div className={'flex items-center w-full h-[20px]'}>
      {showLabel ? (
        <div
          className={classnames(
            styles.tagItem,
            'px-1 py-0.5 gap-0.5 w-[50px] mr-[6px] flex items-center justify-center',
          )}
          style={{
            flex: '0 0 50px',
          }}
        >
          <span className={styles.tagItemLabel}>{label ?? ''}</span>
        </div>
      ) : null}
      {displayContent === null ? (
        <FieldEmpty
          fieldName={I18n.t(
            'workflow_ques_ans_type_option_content',
            {},
            '内容',
          )}
        />
      ) : (
        <LabelWithTooltip
          customClassName={styles.question_pairs_content}
          maxWidth={maxWidth}
          content={displayContent}
        />
      )}
    </div>
  );
};
