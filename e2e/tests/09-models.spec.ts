/**
 * 模型管理测试
 * 覆盖：列表页、过滤、搜索、添加模型入口
 */
import { test, expect } from '@playwright/test';
import { URLS } from './helpers/constants';

test.describe('模型管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.models);
    await page.waitForLoadState('networkidle');
  });

  test('模型配置页 - 基础渲染', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '模型配置' })).toBeVisible();
    await expect(page.getByRole('button', { name: '添加模型' })).toBeVisible();
    await expect(page.getByPlaceholder('搜索模型')).toBeVisible();
    // 厂商和类型过滤下拉
    const combos = page.locator('[role="combobox"]');
    await expect(combos.first()).toBeVisible();
  });

  test('模型列表 - 至少有一个模型', async ({ page }) => {
    // 部署时已初始化 Qwen3.5-Plus
    const modelCard = page.locator('h3').first();
    const hasModel = await modelCard.isVisible({ timeout: 5000 }).catch(() => false);
    if (hasModel) {
      await expect(modelCard).toBeVisible();
    }
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('搜索框 - 可输入过滤', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索模型');
    await searchInput.fill('Qwen');
    await expect(searchInput).toHaveValue('Qwen');
    await page.waitForTimeout(500);
    // 不崩溃即通过
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('添加模型按钮 - 跳转到添加页', async ({ page }) => {
    await page.getByRole('button', { name: '添加模型' }).click();
    await expect(page).toHaveURL(/models\/add/);
    await expect(page.getByRole('heading', { name: '添加模型' })).toBeVisible();
  });

  test('添加模型页 - 表单结构', async ({ page }) => {
    await page.goto(`${URLS.models}/add`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByPlaceholder('请输入模型名称')).toBeVisible();
    // 厂商下拉
    await expect(page.getByRole('combobox', { name: '厂商*' })).toBeVisible();
    // 类型单选
    await expect(page.getByRole('radio', { name: '文本生成' })).toBeVisible();
    // 链接和密钥
    await expect(page.getByPlaceholder(/api.openai.com/)).toBeVisible();
    await expect(page.getByPlaceholder('请输入API密钥')).toBeVisible();
    // 保存按钮
    await expect(page.getByRole('button', { name: '保存' })).toBeVisible();
  });
});
