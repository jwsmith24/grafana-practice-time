import { expect, test } from '@playwright/test';
import { openOverview } from './helpers';

// T3.5-4
test('deploy annotation marker shows the version', async ({ page }) => {
  const to = Date.now();
  const from = to - 15 * 60 * 1000;
  await openOverview(page, from, to);
  const marker = page.locator('[data-viz-panel-key="panel-1"]').getByTestId('data-testid annotation-marker').first();
  await expect(marker).toBeVisible();
  await expect(page.getByText('1.4.2')).toHaveCount(0);
  await marker.hover();
  await expect(page.getByText('1.4.2').first()).toBeVisible();
});
