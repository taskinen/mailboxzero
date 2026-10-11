// @ts-check
const { test, expect } = require('@playwright/test');

// These tests answer the inbox listing themselves. The predicate matches only
// /api/emails, not other API paths.
const isInboxListing = (url) => url.pathname === '/api/emails';

function fakeEmails(count, start) {
  return Array.from({ length: count }, (_, i) => ({
    id: `fake-${start + i}`,
    subject: `Fake email ${start + i}`,
    from: [{ name: 'Sender', email: 'sender@example.com' }],
    preview: 'Preview text',
    receivedAt: new Date(Date.UTC(2026, 9, 1) - (start + i) * 60_000).toISOString(),
  }));
}

test('an empty inbox shows a message instead of a blank list', async ({ page }) => {
  await page.route(isInboxListing, (route) => route.fulfill({ json: { emails: [], totalCount: 0 } }));
  await page.goto('/');

  await expect(page.locator('#inbox-list')).toContainText('Inbox zero! There are no emails left in your inbox.');
  await expect(page.locator('#inbox-title')).toHaveText('Inbox (0)');
  await expect(page.locator('.pagination-info')).toHaveText('Showing 0-0 of 0');
  await expect(page.locator('#current-page')).toHaveText('1');
  await expect(page.locator('#total-pages')).toHaveText('1');
  await expect(page.locator('#next-page-btn')).toBeDisabled();
});

test('emptying the last page moves back to the new last page', async ({ page }) => {
  // The inbox has 60 emails until page 2 is requested. By then the 10 emails
  // on it have been archived (say, in another tab), leaving 50.
  let total = 60;
  await page.route(isInboxListing, (route) => {
    const url = new URL(route.request().url());
    const limit = Number(url.searchParams.get('limit'));
    const offset = Number(url.searchParams.get('offset'));
    if (offset > 0) total = 50;
    const count = Math.max(0, Math.min(limit, total - offset));
    route.fulfill({ json: { emails: fakeEmails(count, offset), totalCount: total } });
  });

  await page.goto('/');
  await page.selectOption('#per-page-select', '50');
  await expect(page.locator('#total-pages')).toHaveText('2');

  await page.click('#next-page-btn');

  // "of 1" only appears once page 2 has been answered, so check it first.
  await expect(page.locator('#total-pages')).toHaveText('1');
  await expect(page.locator('#current-page')).toHaveText('1');
  await expect(page.locator('#inbox-list .email-item')).toHaveCount(50);
  await expect(page.locator('.pagination-info')).toHaveText('Showing 1-50 of 50');
});
