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

/**
 * 登录 setup：获取登录态并保存到 auth.json
 * 只需跑一次，后续所有测试复用
 *
 * 运行：npx playwright test setup/auth.setup.ts
 */
import { test as setup, expect } from '@playwright/test';
import path from 'path';

const AUTH_FILE = path.join(__dirname, '../../auth.json');

const BASE_URL = process.env.BASE_URL || 'http://10.10.10.226:8896';
const EMAIL = process.env.TEST_EMAIL || 'admin@ynet.com';
const PASSWORD = process.env.TEST_PASSWORD || 'Admin@2026';

setup('登录并保存登录态', async ({ page }) => {
  await page.goto(`${BASE_URL}/sign`);

  // 等待登录页面加载
  await page.waitForLoadState('networkidle');

  // 填写邮箱和密码（placeholder: 请输入邮箱 / 请输入密码）
  const emailInput = page.getByPlaceholder('请输入邮箱');
  const passwordInput = page.getByPlaceholder('请输入密码');

  await emailInput.fill(EMAIL);
  await passwordInput.fill(PASSWORD);

  // 提交登录
  await page.getByRole('button', { name: '登录' }).click();

  // 等待跳转到 workspace
  await page.waitForURL(/\/space/, { timeout: 15000 });
  await expect(page.getByText(/工作空间|Personal Space/).first()).toBeVisible({ timeout: 10000 });

  // 保存登录态
  await page.context().storageState({ path: AUTH_FILE });
  console.log(`✅ 登录成功，state 已保存到 ${AUTH_FILE}`);
});
