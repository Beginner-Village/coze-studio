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

import { getLocalizedErrorMessage } from '../src/user-facing-error';

describe('getLocalizedErrorMessage', () => {
  it.each([
    [
      'password does not meet the security policy: length must be at least 8 characters',
      '密码长度不能少于 8 位',
    ],
    [
      'password does not meet the security policy: must contain at least one uppercase letter',
      '密码必须包含至少一个大写字母',
    ],
    [
      'password does not meet the security policy: must contain at least one lowercase letter',
      '密码必须包含至少一个小写字母',
    ],
    [
      'password does not meet the security policy: must contain at least one digit',
      '密码必须包含至少一个数字',
    ],
    [
      'password does not meet the security policy: must contain at least one special character',
      '密码必须包含至少一个特殊字符',
    ],
    ['invalid email or password, please try again.', '邮箱或密码错误，请重试'],
    [
      'email already exist : foo@example.com',
      '该邮箱已注册，请直接登录或更换邮箱',
    ],
    ['invalid parameter : Invalid email', '邮箱格式不正确'],
    ['internal server error', '服务器内部错误，请稍后重试'],
    ['permission denied', '权限不足'],
    ['invalid space_id: strconv.ParseInt failed', '空间 ID 不正确'],
    [
      "embedding configuration with name 'test' already exists in this space",
      'Embedding 配置名称已存在，请更换名称',
    ],
    [
      "rerank configuration with name 'test' already exists",
      'Rerank 配置名称已存在，请更换名称',
    ],
    ['Binding key already exists', '绑定密钥已存在，请更换后重试'],
    ['Network Error', '网络异常，请检查网络后重试'],
  ])('localizes %s', (message, expected) => {
    expect(getLocalizedErrorMessage(message)).toBe(expected);
  });

  it('keeps unknown backend messages unchanged', () => {
    expect(getLocalizedErrorMessage('custom backend message')).toBe(
      'custom backend message',
    );
  });
});
