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

export const BASE_URL = process.env.BASE_URL || 'http://10.10.10.226:8896';

// 已知 space id（Personal Space）
export const SPACE_ID = '7619233545302573056';

export const URLS = {
  space: `${BASE_URL}/space`,
  develop: `${BASE_URL}/space/${SPACE_ID}/develop`,
  knowledge: `${BASE_URL}/space/${SPACE_ID}/library/4`,
  workflow: `${BASE_URL}/space/${SPACE_ID}/library/2`,
  plugin: `${BASE_URL}/space/${SPACE_ID}/library/1`,
  prompt: `${BASE_URL}/space/${SPACE_ID}/library/6`,
  database: `${BASE_URL}/space/${SPACE_ID}/library/7`,
  skills: `${BASE_URL}/space/${SPACE_ID}/skills`,
  models: `${BASE_URL}/space/${SPACE_ID}/models`,
  embedding: `${BASE_URL}/space/${SPACE_ID}/embedding-config`,
  observability: `${BASE_URL}/space/${SPACE_ID}/observability`,
  members: `${BASE_URL}/space/${SPACE_ID}/members`,
  explore: `${BASE_URL}/explore`,
  template: `${BASE_URL}/template`,
};

// 测试数据前缀（方便识别和清理）
export const TEST_PREFIX = '[E2E]';
