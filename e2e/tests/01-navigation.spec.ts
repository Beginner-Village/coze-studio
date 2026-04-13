/**
 * 导航测试
 * 验证：左侧菜单所有入口可正常访问，页面标题/核心元素正确渲染
 */
import { test, expect } from '@playwright/test';
import { URLS, SPACE_ID } from './helpers/constants';

test.describe('导航 - 左侧菜单', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.space);
    // 等待自动跳转到 develop 页
    await page.waitForURL(`**/space/${SPACE_ID}/develop`, { timeout: 10000 });
  });

  test('工作空间首页加载正常', async ({ page }) => {
    await expect(page).toHaveTitle(/猎鹰/);
    // 主内容区标题和创建按钮
    await expect(page.getByText('项目开发').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建' })).toBeVisible();
    // 左侧导航存在
    await expect(page.getByText('商店').first()).toBeVisible();
    await expect(page.getByText('模板').first()).toBeVisible();
  });

  test('左侧菜单 - 项目开发', async ({ page }) => {
    // sidebar 中"项目开发"文字（用 test-id 或第一个出现的文本）
    await page.getByText('项目开发').first().click();
    await page.waitForURL(`**/space/${SPACE_ID}/develop`);
    await expect(page.getByText('项目开发').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建' })).toBeVisible();
  });

  test('左侧菜单 - 知识库', async ({ page }) => {
    await page.getByText('知识库').click();
    await page.waitForURL(`**/library/4`);
    await expect(page.getByText('知识库').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建知识库' })).toBeVisible();
  });

  test('左侧菜单 - 工作流', async ({ page }) => {
    await page.getByText('工作流').click();
    await page.waitForURL(`**/library/2`);
    await expect(page.getByText('工作流').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建' })).toBeVisible();
  });

  test('左侧菜单 - 技能', async ({ page }) => {
    await page.getByText('技能').click();
    await page.waitForURL(`**/skills`);
    await expect(page.getByRole('heading', { name: '技能管理' })).toBeVisible();
    await expect(page.getByRole('button', { name: '创建技能' })).toBeVisible();
  });

  test('左侧菜单 - 模型管理', async ({ page }) => {
    await page.getByText('模型管理').click();
    await page.waitForURL(`**/models`);
    await expect(page.getByRole('heading', { name: '模型配置' })).toBeVisible();
    await expect(page.getByRole('button', { name: '添加模型' })).toBeVisible();
  });

  test('左侧菜单 - 可观测性', async ({ page }) => {
    await page.getByText('可观测性').click();
    await page.waitForURL(`**/observability`);
    // Trace 页面核心元素（列头为 generic 元素，用 getByText）
    await expect(page.getByText('TraceID').first()).toBeVisible();
    await expect(page.getByText('Latency').first()).toBeVisible();
  });

  test('顶部导航 - 商店', async ({ page }) => {
    await page.locator('a[href="/explore"]').click();
    // SPA 路由，检查 URL 包含 /explore
    await page.waitForURL(/\/explore/, { timeout: 10000 });
    await expect(page).toHaveURL(/\/explore/);
  });

  test('顶部导航 - 模板', async ({ page }) => {
    await page.locator('a[href="/template"]').click();
    await page.waitForURL(/\/template/, { timeout: 10000 });
    await expect(page).toHaveURL(/\/template/);
  });
});
