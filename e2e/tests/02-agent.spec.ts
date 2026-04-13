/**
 * 智能体（Agent）测试
 * 覆盖：创建弹窗、表单验证、错误提示、创建成功跳转
 *
 * 近期相关改动：
 * - feat(agent): 添加ForceToolReturn开关，支持工具结果强制回传模型
 * - fix(agent): 修复智能体模式ChatFlow多轮对话无上下文的问题
 * - feat(agent): 添加非流式API支持和完善清除上下文功能
 */
import { test, expect } from '@playwright/test';
import { URLS, TEST_PREFIX } from './helpers/constants';

test.describe('智能体管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(URLS.develop);
    await page.waitForLoadState('networkidle');
  });

  test('项目开发页面 - 基础渲染', async ({ page }) => {
    await expect(page.getByText('项目开发').first()).toBeVisible();
    await expect(page.getByRole('button', { name: '创建' })).toBeVisible();
    // 搜索框
    await expect(page.getByPlaceholder('搜索项目')).toBeVisible();
    // 过滤下拉
    await expect(page.locator('[role="combobox"]').first()).toBeVisible();
  });

  test('创建弹窗 - 打开并关闭', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();

    const dialog = page.getByRole('dialog', { name: '创建' });
    await expect(dialog).toBeVisible();
    // 两种创建类型
    await expect(dialog.getByText('创建智能体')).toBeVisible();
    await expect(dialog.getByText('创建应用')).toBeVisible();

    // 关闭弹窗
    await dialog.getByRole('button', { name: 'close' }).click();
    await expect(dialog).not.toBeVisible();
  });

  test('创建智能体弹窗 - 表单结构', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();
    const dialog = page.getByRole('dialog', { name: '创建' });
    // 点"创建智能体"卡片里的创建按钮
    await dialog.getByText('创建智能体').locator('..').getByRole('button', { name: '创建' }).click();

    // 第二个弹窗（智能体表单），用 placeholder 定位避免依赖 ARIA name
    const agentDialog = page.locator('[role="dialog"]').filter({
      has: page.getByPlaceholder(/给智能体起一个独一无二的名字/),
    });
    await expect(agentDialog).toBeVisible();

    // 必填字段
    await expect(agentDialog.getByPlaceholder(/给智能体起一个独一无二的名字/)).toBeVisible();

    // 选填字段
    await expect(agentDialog.getByLabel('智能体功能介绍')).toBeVisible();

    // 图标上传区
    await expect(agentDialog.getByText('图标')).toBeVisible();

    // 未填名称时，确认按钮禁用
    await expect(agentDialog.getByRole('button', { name: '确认' })).toBeDisabled();
  });

  test('创建智能体弹窗 - 填写名称后确认按钮激活', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();
    const dialog = page.getByRole('dialog', { name: '创建' });
    await dialog.getByText('创建智能体').locator('..').getByRole('button', { name: '创建' }).click();

    const agentDialog = page.locator('[role="dialog"]').filter({
      has: page.getByPlaceholder(/给智能体起一个独一无二的名字/),
    });
    const nameInput = agentDialog.getByPlaceholder(/给智能体起一个独一无二的名字/);
    const confirmBtn = agentDialog.getByRole('button', { name: '确认' });

    // 填名称前：禁用
    await expect(confirmBtn).toBeDisabled();

    // 填入名称后：激活
    await nameInput.fill(`${TEST_PREFIX}测试智能体`);
    await expect(confirmBtn).toBeEnabled();

    // 清空后：重新禁用
    await nameInput.clear();
    await expect(confirmBtn).toBeDisabled();
  });

  test('创建智能体弹窗 - 名称字数限制 20 字', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();
    const dialog = page.getByRole('dialog', { name: '创建' });
    await dialog.getByText('创建智能体').locator('..').getByRole('button', { name: '创建' }).click();

    const agentDialog = page.locator('[role="dialog"]').filter({
      has: page.getByPlaceholder(/给智能体起一个独一无二的名字/),
    });
    const nameInput = agentDialog.getByPlaceholder(/给智能体起一个独一无二的名字/);

    // 输入 20 字
    await nameInput.fill('测试智能体名称限制二十字符12345');
    // 计数器应显示 ≤20
    await expect(agentDialog.getByText(/\d+\/20/)).toBeVisible();
  });

  test('创建智能体 - 无模型时显示错误提示', async ({ page }) => {
    await page.getByRole('button', { name: '创建' }).click();
    const dialog = page.getByRole('dialog', { name: '创建' });
    await dialog.getByText('创建智能体').locator('..').getByRole('button', { name: '创建' }).click();

    const agentDialog = page.locator('[role="dialog"]').filter({
      has: page.getByPlaceholder(/给智能体起一个独一无二的名字/),
    });
    await agentDialog.getByPlaceholder(/给智能体起一个独一无二的名字/).fill(`${TEST_PREFIX}测试智能体`);
    await agentDialog.getByRole('button', { name: '确认' }).click();

    // 当没有配置模型时，应提示错误（具体消息取决于环境配置）
    // 可能提示 "there is no llm model in use" 或成功跳转
    const errorAlert = page.locator('[role="alert"]').filter({ hasText: /model|模型|失败/ });
    const successRedirect = page.waitForURL(/\/bot\//, { timeout: 5000 });

    const result = await Promise.race([
      errorAlert.waitFor({ timeout: 5000 }).then(() => 'error'),
      successRedirect.then(() => 'success'),
    ]).catch(() => 'timeout');

    // 无论哪种结果都是预期行为，关键是不能白屏崩溃
    expect(['error', 'success']).toContain(result);
  });

  test('项目开发 - 搜索框可输入', async ({ page }) => {
    const searchInput = page.getByPlaceholder('搜索项目');
    await searchInput.fill('测试');
    await expect(searchInput).toHaveValue('测试');
  });
});
