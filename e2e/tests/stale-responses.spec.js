// @ts-check
const { test, expect } = require('@playwright/test');

// These tests answer the API themselves and hold one answer back until a
// later request has been answered, the way a slow server can.

function fakeEmails(count, start, prefix = 'fake') {
  return Array.from({ length: count }, (_, i) => ({
    id: `${prefix}-${start + i}`,
    subject: `Fake email ${start + i}`,
    from: [{ name: 'Sender', email: 'sender@example.com' }],
    preview: 'Preview text',
    receivedAt: new Date(Date.UTC(2026, 9, 1) - (start + i) * 60_000).toISOString(),
  }));
}

// A gate that holds an answer until opened.
function gate() {
  let open;
  const opened = new Promise((resolve) => { open = resolve; });
  return { open, opened };
}

const firstId = (page, list) => page.locator(`${list} .email-item`).first().getAttribute('data-email-id');

// Wait for the held answer to arrive, then give the page time to (wrongly)
// show it.
async function deliver(page, held, isHeldResponse) {
  const response = page.waitForResponse(isHeldResponse);
  held.open();
  await response;
  await page.waitForTimeout(250);
}

test('a slow answer for an earlier page does not replace a later page', async ({ page }) => {
  const page2 = gate();
  const offsetOf = (url) => Number(new URL(url).searchParams.get('offset'));
  await page.route((url) => url.pathname === '/api/emails', async (route) => {
    const offset = offsetOf(route.request().url());
    const limit = Number(new URL(route.request().url()).searchParams.get('limit'));
    if (offset === 50) await page2.opened;
    await route.fulfill({ json: { emails: fakeEmails(Math.min(limit, 150 - offset), offset), totalCount: 150 } });
  });

  await page.goto('/');
  await page.selectOption('#per-page-select', '50');
  await expect(page.locator('#total-pages')).toHaveText('3');

  await page.click('#next-page-btn'); // page 2, held back
  await page.click('#next-page-btn'); // page 3
  await expect(page.locator('#inbox-list .email-item').first()).toHaveAttribute('data-email-id', 'fake-100');

  await deliver(page, page2, (response) => response.url().includes('/api/emails') && offsetOf(response.url()) === 50);

  expect(await firstId(page, '#inbox-list')).toBe('fake-100');
  await expect(page.locator('#current-page')).toHaveText('3');
  await expect(page.locator('.pagination-info')).toHaveText('Showing 101-150 of 150');
});

// Answer /api/similar: a request that targets an email gets group "old" and
// is held back until opened; one without a target gets group "new" at once.
async function routeSimilar(page, held) {
  await page.route((url) => url.pathname === '/api/similar', async (route) => {
    if (route.request().postDataJSON().emailId) {
      await held.opened;
      await route.fulfill({ json: fakeEmails(2, 0, 'old') });
    } else {
      await route.fulfill({ json: fakeEmails(3, 0, 'new') });
    }
  });
}

const isTargetedSimilar = (response) => response.url().endsWith('/api/similar') && !!response.request().postDataJSON().emailId;

test('a slow Find Similar answer does not replace a later one', async ({ page }) => {
  const held = gate();
  await routeSimilar(page, held);
  await page.goto('/');
  const target = page.locator('#inbox-list .email-item').first();
  await expect(target).toBeVisible();

  await target.click(); // target an email
  await page.click('#find-similar-btn'); // held back
  await target.click(); // untarget it
  await page.click('#find-similar-btn');
  await expect(page.locator('#similar-list .email-item')).toHaveCount(3);

  await deliver(page, held, isTargetedSimilar);

  expect(await firstId(page, '#similar-list')).toBe('new-0');
  await expect(page.locator('#similar-list .email-item')).toHaveCount(3);
  await expect(page.locator('#similar-title')).toHaveText('Similar Emails (3)');
});

test('Clear Results drops a Find Similar answer still on its way', async ({ page }) => {
  const held = gate();
  await routeSimilar(page, held);
  await page.goto('/');
  const target = page.locator('#inbox-list .email-item').first();
  await expect(target).toBeVisible();

  await page.click('#find-similar-btn'); // answered at once
  await expect(page.locator('#similar-list .email-item')).toHaveCount(3);
  await target.click();
  await page.click('#find-similar-btn'); // held back
  await page.click('#clear-results-btn');
  await expect(page.locator('#similar-list .email-item')).toHaveCount(0);

  await deliver(page, held, isTargetedSimilar);

  await expect(page.locator('#similar-list .email-item')).toHaveCount(0);
  await expect(page.locator('#similar-list')).toContainText('Click "Find Similar Emails"');
});
