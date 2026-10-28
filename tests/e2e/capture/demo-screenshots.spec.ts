import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { expect, test, type Page } from '@playwright/test';

const repoRoot = path.resolve(__dirname, '../../..');
const materials = path.join(repoRoot, 'materials');

type Step = { n: number; lang: string; query: string; image: string };

/** The demo script's steps, each with its query block and screenshot path. */
function demoSteps(): Step[] {
  const text = readFileSync(path.join(materials, 'demo-script.md'), 'utf8');
  return text
    .split(/^## Step /m)
    .slice(1)
    .map((s: string) => {
      const block = s.match(/^```(logql|promql)\n([\s\S]*?)^```$/m);
      const image = s.match(/!\[[^\]]*\]\(([^)]+\.png)\)/);
      if (!block || !image) throw new Error(`step without a query or screenshot: ${s.slice(0, 40)}`);
      return { n: parseInt(s, 10), lang: block[1], query: block[2].trim(), image: image[1] };
    });
}

function incident(arg: string) {
  execFileSync(path.join(repoRoot, 'incident.sh'), [arg], { cwd: repoRoot, stdio: 'inherit' });
}

/** A deadline error's request_id whose inventory line has arrived. */
async function deadlineRequestID(page: Page, from: number): Promise<string> {
  const api = '/api/datasources/proxy/uid/loki/loki/api/v1/query_range';
  const range = async (query: string) => {
    const res = await page.request.get(api, {
      params: { query, start: `${from}000000`, end: `${Date.now()}000000`, limit: '50' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    return body.data.result.flatMap((s: { values: [string, string][] }) => s.values.map((v) => JSON.parse(v[1])));
  };
  for (const line of await range('{service="orders", level="ERROR"} |= "context deadline exceeded"')) {
    const both = await range(`{env="lab"} | json | request_id="${line.request_id}"`);
    if (both.length >= 2) return line.request_id;
  }
  throw new Error('no deadline error with an inventory line yet');
}

function exploreURL(query: string, from: number, to: number): string {
  const panes = {
    a: {
      datasource: 'loki',
      // A topk() reads best as a table of counts, which an instant query gives.
      queries: [
        {
          refId: 'A',
          expr: query,
          queryType: query.startsWith('topk(') ? 'instant' : 'range',
          datasource: { type: 'loki', uid: 'loki' },
        },
      ],
      range: { from: String(from), to: String(to) },
    },
  };
  return `/explore?schemaVersion=1&orgId=1&theme=light&panes=${encodeURIComponent(JSON.stringify(panes))}`;
}

async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
  await expect(page.getByText(/No data|No logs found/)).toHaveCount(0);
  await page.waitForTimeout(1_500);
}

// Regenerates materials/demo/*.png. Mode B is kept running (each run of
// ./incident.sh B restarts its timer), so this works with any
// INCIDENT_DURATION.
test('demo script screenshots', async ({ page }) => {
  const steps = demoSteps();
  incident('clear');
  let keepAlive: ReturnType<typeof setInterval> | undefined;
  let started = 0;
  try {
    for (const step of steps) {
      if (step.lang === 'promql') {
        // The PromQL step starts mode B and shows the dashboard once it has
        // caught up; the steps after it look at the logs since then.
        if (!keepAlive) {
          started = Date.now();
          incident('B');
          keepAlive = setInterval(() => incident('B'), 10_000);
          await page.waitForTimeout(45_000);
        }
        await page.goto(`/d/lab-overview/lab-overview?from=now-15m&to=now&refresh=&theme=light`);
        await expect(page.locator('[data-viz-panel-key="panel-1"] .u-over')).toBeVisible();
      } else {
        const from = started ? started - 60_000 : Date.now() - 5 * 60_000;
        let query = step.query;
        if (query.includes('<request_id>')) query = query.replace('<request_id>', await deadlineRequestID(page, from));
        await page.goto(exploreURL(query, from, Date.now()));
      }
      await settle(page);
      await page.screenshot({ path: path.join(materials, step.image) });
    }
  } finally {
    if (keepAlive) clearInterval(keepAlive);
    incident('clear');
  }
});
