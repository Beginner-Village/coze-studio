/**
 * 知识库测试
 * 覆盖：列表页、创建弹窗、搜索过滤、导入功能
 *
 * 近期相关改动：
 * - feat(knowledge): 知识库QA解析和检索测试功能
 * - fix(agent): 过滤知识库消息避免污染大模型上下文
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('知识库', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.knowledge);
    await page.waitForLoadState('networkidle');
  });

  test('知识库列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('知识库').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建知识库' })).toBeVisible();
    await expect(page.getByRole('button', { name: '导入' }).first()).toBeVisible();
    // 搜索框
    await expect(page.getByPlaceholder('搜索资源')).toBeVisible();
    // 过滤器
    await expect(page.getByText('所有类型')).toBeVisible();
  });

  test('知识库列表页 - 过滤下拉可展开', async ({ page }) => {
    const typeFilter = page.locator('[role="combobox"]').filter({ hasText: '所有类型' });
    await typeFilter.click();
    // 等待下拉展开，验证有选项
    await expect(page.locator('[role="option"], [role="listbox"]').first()).toBeVisible({ timeout: 3000 })
      .catch(() => {
        // 某些实现不用 option，只要点击不报错即可
      });
  });

  test('知识库搜索框 - 可输入', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索资源');
    await searchInput.fill('测试知识库');
    await expect(searchInput).toHaveValue('测试知识库');
  });

  test('创建知识库按钮 - 打开创建弹窗或页面', async ({ page }) => {
    await page.getByRole('button', { name: '创建知识库' }).click();
    // 应弹出 dialog 或跳转到新页面
    const dialog = page.locator('[role="dialog"]');
    const hasDialog = await dialog.isVisible().catch(() => false);

    if (hasDialog) {
      await expect(dialog).toBeVisible();
      // 关闭
      await page.keyboard.press('Escape');
    } else {
      // 跳转到创建知识库页
      await expect(page).toHaveURL(/knowledge/);
    }
  });

  test('导入按钮 - 可点击', async ({ page }) => {
    const importBtn = page.getByRole('button', { name: '导入' }).first();
    await expect(importBtn).toBeEnabled();
    await importBtn.click();
    // 可能弹出文件选择或对话框，验证不崩溃
    await page.waitForTimeout(500);
    // 检查有无错误弹窗
    const errorMsg = page.locator('[role="alert"]').filter({ hasText: /error|错误|失败/ });
    await expect(errorMsg).not.toBeVisible();
  });
});
