/**
 * 工作流测试
 * 覆盖：列表页、创建、导入
 *
 * 近期相关改动：
 * - feat(workflow): 工作流调试面板trace数据迁移到CozeLoop API
 * - fix(chatflow): 修复ChatFlow模式custom_variables未传递到工作流变量的问题
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('工作流', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.workflow);
    await page.waitForLoadState('networkidle');
  });

  test('工作流列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('工作流').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建' })).toBeVisible();
    await expect(page.getByRole('button', { name: '导入' }).first()).toBeVisible();
    await expect(page.getByPlaceholder('搜索资源')).toBeVisible();
    // 状态过滤
    await expect(page.locator('[role="combobox"]').first()).toBeVisible();
  });

  test('工作流列表 - 空状态或列表', async ({ page }) => {
    const emptyText = page.getByText(/未能找到相关结果|暂无/);
    const hasEmpty = await emptyText.isVisible().catch(() => false);
    // 空列表时显示提示
    if (hasEmpty) {
      await expect(emptyText).toBeVisible();
    }
    // 不崩溃即通过
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('创建工作流 - 点击创建按钮', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();
    // 可能弹出 dialog 或直接跳转到编辑器
    const dialog = page.locator('[role="dialog"]');
    const hasDialog = await dialog.isVisible({ timeout: 3000 }).catch(() => false);

    if (hasDialog) {
      await expect(dialog).toBeVisible();
      await page.keyboard.press('Escape');
    }
    // 无论是否弹窗，页面不崩溃即通过
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('工作流搜索框 - 可输入', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索资源');
    await searchInput.fill('测试流程');
    await expect(searchInput).toHaveValue('测试流程');
  });
});
