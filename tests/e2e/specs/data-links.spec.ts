import { expect, test } from '@playwright/test';
import { explorePane, followDataLink, forceOrders5xx, openOverview } from './helpers';

// T3.4-2 (M3 exit)
test('request rate panel links to the service logs in Explore', async ({ page }) => {
  const to = Date.now();
  const from = to - 5 * 60 * 1000;
  await openOverview(page, from, to);
  await followDataLink(page, 'Request rate by service', 'orders', /Logs for orders/);

  const pane = explorePane(page.url());
  expect(pane.datasource).toBe('loki');
  expect(pane.queries[0].expr).toBe('{service="orders"}');
  expect(pane.range).toEqual({ from: String(from), to: String(to) });
  await expectLogRows(page);
});

// T3.4-3
test('error rate panel links to the logs of the failing service', async ({ page }) => {
  await forceOrders5xx();
  // Wait for a scrape plus the rate window to pick it up.
  await page.waitForTimeout(15_000);
  const to = Date.now();
  const from = to - 5 * 60 * 1000;
  await openOverview(page, from, to);
  await followDataLink(page, 'Error rate (5xx) by endpoint', 'orders POST /orders', /Logs for orders/);

  const pane = explorePane(page.url());
  expect(pane.datasource).toBe('loki');
  expect(pane.queries[0].expr).toBe('{service="orders"}');
  expect(pane.range).toEqual({ from: String(from), to: String(to) });
  await expectLogRows(page);
});

async function expectLogRows(page: import('@playwright/test').Page) {
  await expect(page.getByText('"service":"orders"').first()).toBeVisible();
}
