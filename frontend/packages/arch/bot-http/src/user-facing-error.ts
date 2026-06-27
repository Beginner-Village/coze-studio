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

type MessagePair = readonly [source: string, localized: string];

const PASSWORD_POLICY_MESSAGES: readonly MessagePair[] = [
  ['length must be at least 8 characters', '密码长度不能少于 8 位'],
  ['length must not exceed 64 characters', '密码长度不能超过 64 位'],
  [
    'must contain at least one uppercase letter',
    '密码必须包含至少一个大写字母',
  ],
  [
    'must contain at least one lowercase letter',
    '密码必须包含至少一个小写字母',
  ],
  ['must contain at least one digit', '密码必须包含至少一个数字'],
  [
    'must contain at least one special character',
    '密码必须包含至少一个特殊字符',
  ],
  ['must not contain common weak patterns', '密码不能包含常见弱密码片段'],
];

const EXACT_MESSAGES: readonly MessagePair[] = [
  ['invalid email or password, please try again.', '邮箱或密码错误，请重试'],
  ['invalid email', '邮箱格式不正确'],
  ['internal server error', '服务器内部错误，请稍后重试'],
  ['binding key already exists', '绑定密钥已存在，请更换后重试'],
  ['network error', '网络异常，请检查网络后重试'],
  ['unauthorized', '未登录或登录状态已失效'],
  ['permission denied', '权限不足'],
  ['missing parameter', '缺少必要参数'],
  ['check permission failed', '权限校验失败'],
  ['check member permission failed', '成员权限校验失败'],
  ['not a member of this space', '你不是该空间成员'],
  ['not space member', '你不是该空间成员'],
  ['no permission to invite members', '无权限邀请成员'],
  ['search users failed', '搜索用户失败'],
  ['invite member failed', '邀请成员失败'],
  ['update member role failed', '更新成员角色失败'],
  ['remove member failed', '移除成员失败'],
  ['invalid user id', '用户 ID 不正确'],
  ['invalid user id format', '用户 ID 格式不正确'],
  ['only space owner can delete space', '只有空间所有者可以删除空间'],
  ['you can only delete spaces you own', '只能删除你拥有的空间'],
  ['delete space failed', '删除空间失败'],
  ['only space owner can transfer space', '只有空间所有者可以转让空间'],
  ['invalid new owner id format', '新所有者 ID 格式不正确'],
  ['cannot transfer space to yourself', '不能将空间转让给自己'],
  ['check new owner permission failed', '新所有者权限校验失败'],
  [
    'can only transfer space to existing space members',
    '只能转让给当前空间成员',
  ],
  ['transfer space failed', '转让空间失败'],
  ['only space owner can update space', '只有空间所有者可以更新空间'],
  ['you can only update spaces you own', '只能更新你拥有的空间'],
  ['get space failed', '获取空间失败'],
  ['update space failed', '更新空间失败'],
  ['create space failed', '创建空间失败'],
];

const normalizeMessage = (message: string) =>
  message.trim().replace(/\s+/g, ' ');

const findLocalizedMessage = (
  messages: readonly MessagePair[],
  lowerMessage: string,
) => messages.find(([source]) => source === lowerMessage)?.[1];

const localizeDetail = (detail: string) => {
  const normalizedDetail = normalizeMessage(detail);
  const lowerDetail = normalizedDetail.toLowerCase();

  return (
    findLocalizedMessage(PASSWORD_POLICY_MESSAGES, lowerDetail) ??
    findLocalizedMessage(EXACT_MESSAGES, lowerDetail) ??
    normalizedDetail
  );
};

type MessageMatcher = (
  normalizedMessage: string,
  lowerMessage: string,
) => string | undefined;

const MESSAGE_MATCHERS: readonly MessageMatcher[] = [
  (normalizedMessage, lowerMessage) => {
    const prefix = 'password does not meet the security policy: ';
    return lowerMessage.startsWith(prefix)
      ? localizeDetail(normalizedMessage.slice(prefix.length))
      : undefined;
  },
  (_normalizedMessage, lowerMessage) =>
    lowerMessage.startsWith('email already exist')
      ? '该邮箱已注册，请直接登录或更换邮箱'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    lowerMessage.startsWith('unique name already exist')
      ? '用户名已存在，请更换后重试'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    lowerMessage.startsWith('invalid space_id') ? '空间 ID 不正确' : undefined,
  (_normalizedMessage, lowerMessage) =>
    lowerMessage.startsWith('invalid embedding_id')
      ? 'Embedding 配置 ID 不正确'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    lowerMessage.startsWith('invalid rerank_id')
      ? 'Rerank 配置 ID 不正确'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    /^embedding configuration with name '.+' already exists( in this space)?$/.test(
      lowerMessage,
    )
      ? 'Embedding 配置名称已存在，请更换名称'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    /^rerank configuration with name '.+' already exists( in this space)?$/.test(
      lowerMessage,
    )
      ? 'Rerank 配置名称已存在，请更换名称'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    lowerMessage ===
    'the user registration has been disabled by the administrator. please contact the administrator!'
      ? '管理员已关闭用户注册，请联系管理员'
      : undefined,
  (normalizedMessage, lowerMessage) => {
    const prefix = 'invalid parameter : ';
    if (!lowerMessage.startsWith(prefix)) {
      return undefined;
    }
    const detail = normalizedMessage.slice(prefix.length);
    const localizedDetail = localizeDetail(detail);
    return localizedDetail === detail
      ? `参数不正确：${detail}`
      : localizedDetail;
  },
  (normalizedMessage, lowerMessage) => {
    const prefix = 'unauthorized access : ';
    if (!lowerMessage.startsWith(prefix)) {
      return undefined;
    }
    const detail = normalizedMessage.slice(prefix.length);
    const localizedDetail = localizeDetail(detail);
    return localizedDetail === detail
      ? `无权限访问：${detail}`
      : localizedDetail;
  },
  (_normalizedMessage, lowerMessage) =>
    /^request failed with status code \d+$/.test(lowerMessage)
      ? '请求失败，请稍后重试'
      : undefined,
  (_normalizedMessage, lowerMessage) =>
    /^timeout of \d+ms exceeded$/.test(lowerMessage)
      ? '请求超时，请稍后重试'
      : undefined,
];

export const getLocalizedErrorMessage = (message: unknown): string | undefined => {
  if (message === undefined || message === null) {
    return undefined;
  }

  if (typeof message !== 'string') {
    return String(message);
  }

  if (!message) {
    return message;
  }

  const normalizedMessage = normalizeMessage(message);
  const lowerMessage = normalizedMessage.toLowerCase();
  const exactMessage =
    findLocalizedMessage(PASSWORD_POLICY_MESSAGES, lowerMessage) ??
    findLocalizedMessage(EXACT_MESSAGES, lowerMessage);

  if (exactMessage) {
    return exactMessage;
  }

  for (const matcher of MESSAGE_MATCHERS) {
    const matchedMessage = matcher(normalizedMessage, lowerMessage);
    if (matchedMessage) {
      return matchedMessage;
    }
  }

  return message;
};
