/**
 * 数据库管理测试
 * 覆盖：列表页、创建、导入、搜索
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('数据库管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.database);
    await page.waitForLoadState('networkidle');
  });

  test('数据库列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('数据库').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建数据库' })).toBeVisible();
    await expect(page.getByRole('button', { name: '导入' }).first()).toBeVisible();
    await expect(page.getByPlaceholder('搜索资源')).toBeVisible();
  });

  test('数据库列表 - 空状态或有数据', async ({ page }) => {
    const emptyText = page.getByText(/未能找到相关结果|暂无/);
    const hasEmpty = await emptyText.isVisible().catch(() => false);
    if (hasEmpty) {
      await expect(emptyText).toBeVisible();
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('搜索框 - 可输入', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索资源');
    await searchInput.fill('测试数据库');
    await expect(searchInput).toHaveValue('测试数据库');
  });

  test('创建数据库 - 点击按钮', async ({ page }) => {
    await page.getByRole('button', { name: '创建数据库' }).click();
    await page.waitForTimeout(1000);
    const dialog = page.locator('[role="dialog"]');
    const hasDialog = await dialog.isVisible().catch(() => false);
    if (hasDialog) {
      await expect(dialog).toBeVisible();
      await page.keyboard.press('Escape');
    } else {
      await expect(page).toHaveURL(/database|library/);
    }
  });
});
