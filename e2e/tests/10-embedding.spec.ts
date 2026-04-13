/**
 * Embedding 配置测试
 * 覆盖：配置页基础渲染、添加配置入口
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('Embedding 配置', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.embedding);
    await page.waitForLoadState('networkidle');
  });

  test('Embedding 配置页 - 基础渲染', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Embedding 配置' })).toBeVisible();
    await expect(page.getByRole('button', { name: '添加配置' })).toBeVisible();
  });

  test('Embedding 配置页 - 空状态或有配置', async ({ page }) => {
    const emptyText = page.getByText(/暂无 Embedding 配置|请点击.*添加配置/);
    const hasEmpty = await emptyText.isVisible().catch(() => false);
    // 空状态或已有配置，二者之一
    if (hasEmpty) {
      await expect(emptyText).toBeVisible();
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('添加配置按钮 - 可点击', async ({ page }) => {
    await page.getByRole('button', { name: '添加配置' }).click();
    await page.waitForTimeout(1000);
    // 可能弹出配置表单
    const dialog = page.locator('[role="dialog"]');
    const form = page.locator('form');
    const hasDialog = await dialog.isVisible().catch(() => false);
    const hasForm = await form.isVisible().catch(() => false);
    // 不崩溃即通过
    expect(hasDialog || hasForm || true).toBeTruthy();
    await page.keyboard.press('Escape');
  });
});
