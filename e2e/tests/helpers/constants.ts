export const BASE_URL = process.env.BASE_URL || 'http://10.10.10.220:9888';

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
