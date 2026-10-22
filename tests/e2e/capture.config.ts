import { defineConfig } from '@playwright/test';
import base from './playwright.config';

// Regenerates the demo script's screenshots (materials/demo/*.png) against a
// running stack. Not part of the e2e suite: run it by hand after a dashboard
// or Grafana change.
//
//   npx playwright test -c capture.config.ts
export default defineConfig({
  ...base,
  testDir: './capture',
  timeout: 5 * 60_000,
  retries: 0,
  reporter: 'list',
  use: { ...base.use, viewport: { width: 1440, height: 900 }, colorScheme: 'light' },
  projects: undefined,
});
