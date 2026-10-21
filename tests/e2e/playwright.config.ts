import { defineConfig, devices } from '@playwright/test';

const port = process.env.GRAFANA_PORT ?? '3300';

export default defineConfig({
  testDir: './specs',
  timeout: 90_000,
  expect: { timeout: 15_000 },
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.GRAFANA_URL ?? `http://localhost:${port}`,
    viewport: { width: 1600, height: 1000 },
    // Grafana's number formatting rejects POSIX-style locale tags some
    // environments pass through, so pin a plain one.
    locale: 'en-US',
    timezoneId: 'UTC',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1600, height: 1000 } } }],
});
