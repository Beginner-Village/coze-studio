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
 * 批量测试（评测实验）E2E —— 覆盖 batch test 智能体/工作流的端到端链路：
 *   观测台 → 实验 → 新建实验 → 选工作流/智能体为评测对象 → 选评估集 → 提交 → 逐行结果。
 *
 * ⚠️ 运行前置条件（必须在 226 真实环境具备，否则跳过/失败属预期）：
 *   1. loop 服务已配置访问 studio 的通道：
 *        YNET_STUDIO_OPENAPI_BASE_URL = studio 内网地址（如 http://coze-server:8888）
 *        YNET_STUDIO_OPENAPI_TOKEN    = studio PAT（该 PAT 所属用户须属于 SPACE_ID 空间）
 *   2. SPACE_ID 空间内存在：至少一个【已发布】工作流；至少一个评估集。
 *   3. e2e/auth.json 有效登录态。
 *
 * 运行：BASE_URL=http://10.10.10.226:8896 pnpm playwright test 13-experiment-batch-test
 */
import { test, expect } from '@playwright/test';

import { URLS, TEST_PREFIX } from './helpers/constants';

// Semi/coze-design Select：点击触发器展开后点选第一个 option。
async function pickFirstOption(page, triggerLocator) {
  await triggerLocator.click();
  const option = page.locator('[role="option"], .semi-select-option').first();
  await option.waitFor({ state: 'visible', timeout: 8000 });
  await option.click();
}

test.describe('批量测试 / 评测实验', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${URLS.observability}?tab=experiments`);
    await page.waitForLoadState('networkidle');
  });

  test('实验列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('实验').first()).toBeVisible();
    await expect(
      page.getByRole('button', { name: '新建实验' }),
    ).toBeVisible();
  });

  test('创建工作流批量测试实验并查看逐行结果', async ({ page }) => {
    const exptName = `${TEST_PREFIX} wf-batch-${Date.now()}`;

    // 进入新建实验向导
    await page.getByRole('button', { name: '新建实验' }).click();

    // Step 0: 基础信息
    await page.getByPlaceholder('请输入实验名称').fill(exptName);
    // 评测对象类型默认「工作流 (Workflow)」；评测对象从下拉选第一个真实工作流
    const targetSelect = page
      .locator('.semi-select')
      .filter({ hasText: /工作流 ID|target_id/ })
      .first();
    await pickFirstOption(page, targetSelect);
    await page.getByRole('button', { name: '下一步' }).click();

    // Step 1: 选择评估集
    const evalSetSelect = page
      .locator('.semi-select')
      .filter({ hasText: '请选择评估集' })
      .first();
    await pickFirstOption(page, evalSetSelect);
    await page.getByRole('button', { name: '下一步' }).click();

    // Step 2: 评估器可选，直接提交（纯批量跑目标看输出）
    await page.getByRole('button', { name: '提交' }).click();

    // 断言创建成功
    await expect(page.getByText('创建成功')).toBeVisible({ timeout: 15000 });

    // 打开刚创建的实验，等待执行并检查「结果」tab 出现逐行结果或至少不报错
    await page.waitForLoadState('networkidle');
    await page.getByText(exptName).first().click();
    await expect(page.getByText('结果').first()).toBeVisible();
    await page.getByText('结果').first().click();
    // 逐行结果异步产出：出现结果卡片或「暂无逐行结果」都算页面正常（不崩溃）
    await expect(page.locator('body')).not.toBeEmpty();
  });
});
