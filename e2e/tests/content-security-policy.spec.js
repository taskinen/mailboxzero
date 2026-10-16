// @ts-check
const { test, expect } = require('@playwright/test');

// Record every Content-Security-Policy violation the page reports.
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const violations = [];
    /** @type {any} */ (window).cspViolations = violations;
    document.addEventListener('securitypolicyviolation', (e) => {
      violations.push({ directive: e.effectiveDirective, blocked: e.blockedURI });
    });
  });
});

const violations = (page) => page.evaluate(() => /** @type {any} */ (window).cspViolations);

test('the page runs no script injected into it', async ({ page }) => {
  await page.goto('/');

  // Markup that slipped into the page, the way a bug in the preview once
  // let an email's HTML do: an image whose error handler runs script.
  await page.evaluate(() => new Promise((resolve) => {
    const image = document.createElement('img');
    image.setAttribute('onerror', 'window.injectedHandlerRan = true');
    image.addEventListener('error', resolve);
    image.src = '/no-such-image.png';
    document.body.appendChild(image);
  }));

  expect(await page.evaluate(() => /** @type {any} */ (window).injectedHandlerRan)).toBeUndefined();
  await expect.poll(() => violations(page)).toContainEqual({ directive: 'script-src-attr', blocked: 'inline' });
});

test('the app works within its own policy', async ({ page }) => {
  const response = await page.goto('/');
  expect(response?.headers()['content-security-policy']).toContain("default-src 'none'");
  await expect(page.locator('#inbox-list .email-item').first()).toBeVisible();
  await expect(page.locator('#archive-status')).toBeHidden();

  await page.click('#find-similar-btn');
  const similar = page.locator('#similar-list .email-item').first();
  await expect(similar).toBeVisible();

  // A preview marks up links and addresses with classes, not style
  // attributes, which the policy would block.
  const id = await similar.getAttribute('data-email-id');
  await page.route((url) => url.pathname === `/api/emails/${id}`, (route) => route.fulfill({
    json: {
      bodyValues: { 1: { value: 'See https://example.com/offer or write to help@example.com' } },
      textBody: [{ partId: '1', type: 'text/plain' }],
    },
  }));
  await similar.hover();
  const link = page.locator('#preview-body span', { hasText: 'https://example.com/offer' });
  await expect(link).toHaveCSS('color', 'rgb(52, 152, 219)');
  await expect(link).toHaveCSS('text-decoration-line', 'underline');
  await expect(page.locator('#preview-body span', { hasText: 'help@example.com' })).toHaveCSS('color', 'rgb(52, 152, 219)');
  await page.keyboard.press('Escape');

  // The status pill shows once something is archived. Undo puts the emails
  // back, since every test shares the mock inbox.
  await page.click('#archive-find-next-btn');
  await expect(page.locator('#archive-status')).toBeVisible();
  await expect(page.locator('#archive-status-text')).toHaveText(/^Successfully archived/);
  await page.click('#archive-undo-btn');
  await expect(page.locator('#archive-status-text')).toHaveText(/^Unarchived/);

  // An error message keeps its color.
  await page.route((url) => url.pathname === '/api/similar', (route) => route.fulfill({ status: 500, body: 'failed' }));
  await page.click('#find-similar-btn');
  const error = page.locator('#similar-list .empty-state p');
  await expect(error).toHaveText('Failed to find similar emails.');
  await expect(error).toHaveCSS('color', 'rgb(231, 76, 60)');

  expect(await violations(page)).toEqual([]);
});
