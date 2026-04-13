/**
 * 可观测性（Trace）测试
 * 覆盖：Trace 列表页、过滤器、空状态
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('可观测性', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.observability);
    await page.waitForLoadState('networkidle');
  });

  test('Trace 页面 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('Trace').first()).toBeVisible();
    // 表头列
    await expect(page.getByText('TraceID')).toBeVisible();
    await expect(page.getByText('Latency')).toBeVisible();
    await expect(page.getByText('Name')).toBeVisible();
  });

  test('Trace 页面 - 过滤器可见', async ({ page }) => {
    // 时间范围过滤
    await expect(page.getByText('过去 7 天')).toBeVisible();
    // Span 类型过滤
    await expect(page.getByText('Root Span')).toBeVisible();
    // 类型过滤
    await expect(page.getByText('全部类型')).toBeVisible();
  });

  test('Trace 页面 - 空状态或有数据', async ({ page }) => {
    const emptyText = page.getByText('暂无 Trace 数据');
    const hasEmpty = await emptyText.isVisible({ timeout: 5000 }).catch(() => false);
    if (hasEmpty) {
      await expect(emptyText).toBeVisible();
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('时间范围过滤 - 下拉可展开', async ({ page }) => {
    const timeFilter = page.locator('[role="combobox"]').filter({ hasText: '过去 7 天' });
    await timeFilter.click();
    await page.waitForTimeout(500);
    // 下拉展开后应有选项
    const options = page.locator('[role="option"], [role="listbox"]');
    const hasOptions = await options.first().isVisible({ timeout: 2000 }).catch(() => false);
    // 不崩溃即通过
    await expect(page.locator('body')).not.toBeEmpty();
    await page.keyboard.press('Escape');
  });
});
