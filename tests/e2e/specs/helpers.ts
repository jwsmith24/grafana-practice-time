import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import { expect, type Locator, type Page } from '@playwright/test';

const ordersURL = process.env.ORDERS_URL ?? 'http://localhost:8080';
const repoRoot = path.resolve(__dirname, '../../..');

/** POST an order and return the response status. */
export async function postOrder(itemID: string, requestID = randomUUID()): Promise<number> {
  const res = await fetch(`${ordersURL}/orders`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Request-ID': requestID },
    body: JSON.stringify({ item_id: itemID, quantity: 1 }),
  });
  return res.status;
}

function compose(...args: string[]) {
  execFileSync('docker', ['compose', '-f', 'docker-compose.yml', '-f', 'docker-compose.test.yml', ...args], {
    cwd: repoRoot,
    stdio: 'inherit',
  });
}

/**
 * Force orders 5xx by taking inventory away for a moment. It stays down
 * across at least two Prometheus scrapes (5 s apart), so loadgen's traffic
 * keeps increasing the new status="502" series and rate() rises above 0.
 */
export async function forceOrders5xx() {
  compose('stop', 'inventory');
  try {
    let status = 0;
    for (let i = 0; i < 3 && status < 500; i++) status = await postOrder('SKU-1001');
    expect(status).toBeGreaterThanOrEqual(500);
    await new Promise((resolve) => setTimeout(resolve, 12_000));
    expect(await postOrder('SKU-1001')).toBeGreaterThanOrEqual(500);
  } finally {
    compose('up', '-d', '--wait', 'inventory');
  }
}

/** Open Lab Overview on a fixed absolute range with auto-refresh off. */
export async function openOverview(page: Page, from: number, to: number) {
  await page.goto(`/d/lab-overview/lab-overview?from=${from}&to=${to}&refresh=`);
  await expect(page.locator('[data-viz-panel-key="panel-1"] .u-over')).toBeVisible();
}

/**
 * Isolate one series in a timeseries panel via its legend, click on its line
 * to pin the tooltip, and follow the named data link. Grafana only offers
 * data links for the series under the cursor, so the click has to land on
 * the line: the helper finds the line's highest point by scanning the
 * panel canvas for the series' (saturated) stroke colour.
 */
export async function followDataLink(page: Page, panelTitle: string, legendLabel: string, linkName: RegExp) {
  const panel = page.locator('[data-viz-panel-key]').filter({ has: page.getByText(panelTitle, { exact: true }) });
  await expect(panel).toHaveCount(1);
  await panel.getByText(legendLabel, { exact: true }).first().click();
  const link = page.getByRole('link', { name: linkName });
  for (let attempt = 0; attempt < 5; attempt++) {
    await page.waitForTimeout(500);
    const point = await linePeak(panel);
    if (point) {
      await page.mouse.move(point.x, point.y);
      await page.waitForTimeout(300);
      await page.mouse.click(point.x, point.y);
      const shown = await link
        .first()
        .waitFor({ state: 'visible', timeout: 2_000 })
        .then(() => true)
        .catch(() => false);
      if (shown) break;
    }
    await page.keyboard.press('Escape');
  }
  await link.first().click();
  await page.waitForURL(/\/explore/);
}

/** Find the topmost point of the (only visible) series line in a panel. */
async function linePeak(panel: Locator): Promise<{ x: number; y: number } | null> {
  const over = await panel.locator('.u-over').boundingBox();
  if (!over) return null;
  return panel
    .locator('canvas')
    .first()
    .evaluate((c: HTMLCanvasElement, box) => {
      const r = c.getBoundingClientRect();
      const scale = c.width / r.width;
      const img = c.getContext('2d')!.getImageData(0, 0, c.width, c.height).data;
      const x0 = Math.round((box.x - r.left) * scale);
      const x1 = Math.round((box.x + box.width - r.left) * scale);
      const y0 = Math.round((box.y - r.top) * scale);
      const y1 = Math.round((box.y + box.height - r.top) * scale);
      let best: { x: number; y: number } | null = null;
      for (let x = x0 + 4; x < x1 - 4; x += 3) {
        for (let y = y0; y < y1; y++) {
          const i = (y * c.width + x) * 4;
          const [R, G, B, A] = [img[i], img[i + 1], img[i + 2], img[i + 3]];
          if (A > 200 && Math.max(R, G, B) - Math.min(R, G, B) > 60 && Math.max(R, G, B) > 120) {
            if (!best || y < best.y) best = { x, y };
            break;
          }
        }
      }
      return best && { x: r.left + best.x / scale, y: r.top + best.y / scale };
    }, over);
}

export type Pane = {
  datasource: string;
  queries: { expr: string; datasource?: { uid: string } }[];
  range: { from: string; to: string };
};

/** Decode the first Explore pane from the current URL. */
export function explorePane(url: string): Pane {
  const panes = new URL(url).searchParams.get('panes');
  if (!panes) throw new Error(`no panes in ${url}`);
  return Object.values(JSON.parse(panes))[0] as Pane;
}
