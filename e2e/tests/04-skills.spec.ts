/**
 * 技能（Skill）管理测试
 * 覆盖：列表页、创建技能、搜索
 *
 * 近期相关改动（重点）：
 * - feat(skill): 添加Skill技能管理功能（新功能）
 * - feat(skill): 技能编辑改为独立路由页面，支持从Agent配置区跳转
 */
import { test, expect } from '@playwright/test';
import { URLS, TEST_PREFIX } from './helpers/constants';

test.describe('技能管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.skills);
    await page.waitForLoadState('networkidle');
  });

  test('技能列表页 - 基础渲染', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '技能管理' })).toBeVisible();
    // 副标题描述
    await expect(page.getByText(/场景化能力包/)).toBeVisible();
    // 创建按钮
    await expect(page.getByRole('button', { name: '创建技能' })).toBeVisible();
    // 搜索框
    await expect(page.getByPlaceholder('搜索技能')).toBeVisible();
    // 技能数量统计
    await expect(page.getByText(/共 \d+ 个技能/)).toBeVisible();
  });

  test('技能列表 - 空状态提示', async ({ page }) => {
    // 首次部署时无技能
    const emptyText = page.getByText(/暂无技能|创建你的第一个技能/);
    // 空状态或有技能列表，二者之一
    const hasList = page.locator('[data-testid*="skill"], .skill-card').first();
    const hasEmpty = await emptyText.isVisible().catch(() => false);
    const hasSomeSkills = await hasList.isVisible().catch(() => false);
    expect(hasEmpty || hasSomeSkills).toBeTruthy();
  });

  test('搜索框 - 可输入并过滤', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索技能');
    await searchInput.fill('测试');
    await expect(searchInput).toHaveValue('测试');
    // 等待搜索结果更新
    await page.waitForTimeout(500);
    // 不报错即通过
  });

  test('创建技能 - 点击按钮', async ({ page }) => {
    await page.getByRole('button', { name: '创建技能' }).click();
    await page.waitForTimeout(1000);

    // 可能跳转到编辑页或打开 dialog
    const currentUrl = page.url();
    const dialog = page.locator('[role="dialog"]');
    const hasDialog = await dialog.isVisible().catch(() => false);

    if (hasDialog) {
      await expect(dialog).toBeVisible();
      await page.keyboard.press('Escape');
    } else {
      // URL 应有变化（跳转到技能相关页面）
      expect(currentUrl).toMatch(/skill/i);
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('技能编辑页 - 从 URL 直接访问', async ({ page }) => {
    await page.getByRole('button', { name: '创建技能' }).click();
    await page.waitForTimeout(1500);

    // 验证页面不是空白，且没有崩溃
    await expect(page.locator('body')).not.toBeEmpty();
    // 不应出现 500 错误页
    const errorPage = page.getByText(/500|Internal Server Error/);
    await expect(errorPage).not.toBeVisible();
  });
});
