/**
 * 插件管理测试
 * 覆盖：列表页、创建、导入、搜索
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('插件管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.plugin);
    await page.waitForLoadState('networkidle');
  });

  test('插件列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('插件').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建插件' })).toBeVisible();
    await expect(page.getByRole('button', { name: '导入' }).first()).toBeVisible();
    await expect(page.getByPlaceholder('搜索资源')).toBeVisible();
    // 状态过滤下拉
    await expect(page.locator('[role="combobox"]').first()).toBeVisible();
  });

  test('插件列表 - 空状态或有数据', async ({ page }) => {
    const emptyText = page.getByText(/未能找到相关结果|暂无/);
    const hasEmpty = await emptyText.isVisible().catch(() => false);
    if (hasEmpty) {
      await expect(emptyText).toBeVisible();
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('搜索框 - 可输入', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索资源');
    await searchInput.fill('测试插件');
    await expect(searchInput).toHaveValue('测试插件');
  });

  test('创建插件按钮 - 可点击', async ({ page }) => {
    await page.getByRole('button', { name: '创建插件' }).click();
    // 可能弹出 dialog 或跳转到创建页
    await page.waitForTimeout(1000);
    const dialog = page.locator('[role="dialog"]');
    const hasDialog = await dialog.isVisible().catch(() => false);
    if (hasDialog) {
      await expect(dialog).toBeVisible();
      await page.keyboard.press('Escape');
    } else {
      // 跳转到创建页
      await expect(page).toHaveURL(/plugin|library/);
    }
  });

  test('导入按钮 - 可点击不崩溃', async ({ page }) => {
    const importBtn = page.getByRole('button', { name: '导入' }).first();
    await expect(importBtn).toBeEnabled();
    await importBtn.click();
    await page.waitForTimeout(500);
    const errorMsg = page.locator('[role="alert"]').filter({ hasText: /error|错误|失败/ });
    await expect(errorMsg).not.toBeVisible();
  });
});
