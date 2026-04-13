import { defineConfig, devices } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://10.10.10.220:9888';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // 避免并发导致数据污染
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['html', { outputFolder: 'playwright-report' }], ['list']],

  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    // 登录 setup，只跑一次，不需要 storageState
    {
      name: 'setup',
      testMatch: /setup\/auth\.setup\.ts/,
    },
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // 使用已登录的 storageState，避免每次重新登录
        storageState: 'auth.json',
      },
      dependencies: ['setup'],
    },
  ],
});
