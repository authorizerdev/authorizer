import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testIgnore: ['**/node_modules/**', '**/mocks/**'],
  timeout: 30_000,
  retries: 0,
  reporter: [['html', { outputFolder: 'playwright-report', open: 'never' }]],
  use: {
    baseURL: process.env.AUTHORIZER_BASE_URL || 'http://localhost:8080',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'mfa-off',
      testIgnore: [/mfa-routing-matrix\.spec\.ts/, '**/mocks/**', '**/node_modules/**'],
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'mfa-on',
      testMatch: /mfa-routing-matrix\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
