// @ts-check
const { test, expect } = require('@playwright/test');

// An email's HTML as an attacker would write it: an image whose error
// handler runs script (from another site, and from the app's own origin,
// which a Content-Security-Policy allows), a script, and style and title
// text that is not content.
const html = `<!doctype html>
<html>
<head><title>Head title</title><style>.head-style { color: red }</style></head>
<body>
<style>.body-style { color: red }</style>
<p>Visible text</p>
<img src="https://tracker.example/open.gif" onerror="window.previewHandlerRan = true">
<img src="/open.gif" onerror="window.previewHandlerRan = true">
<script>window.previewScriptRan = true;</script>
</body>
</html>`;

const bodies = [
  {
    name: 'an HTML part next to an empty text part',
    body: {
      bodyValues: { 1: { value: '' }, 2: { value: html } },
      textBody: [{ partId: '1', type: 'text/plain' }],
      htmlBody: [{ partId: '2', type: 'text/html' }],
    },
  },
  {
    name: 'HTML body values without body parts',
    body: { bodyValues: { 1: { value: html } } },
  },
];

for (const { name, body } of bodies) {
  test(`the preview of ${name} shows its text without loading or running anything`, async ({ page }) => {
    let imageRequests = 0;
    await page.route('**/open.gif', (route) => {
      imageRequests++;
      return route.fulfill({ status: 404 });
    });

    await page.goto('/');
    const email = page.locator('#inbox-list .email-item').first();
    await expect(email).toBeVisible();
    const id = await email.getAttribute('data-email-id');
    // The preview fetches the body on the first hover and keeps it, so
    // answer that fetch with the attacker's HTML.
    await page.route((url) => url.pathname === `/api/emails/${id}`, (route) => route.fulfill({ json: body }));

    await email.hover();
    const preview = page.locator('#preview-body');
    await expect(preview).toContainText('Visible text');
    // Give the images time to (wrongly) load and their handlers to run.
    await page.waitForTimeout(250);

    expect(imageRequests).toBe(0);
    expect(await page.evaluate(() => /** @type {any} */ (window).previewHandlerRan)).toBeUndefined();
    expect(await page.evaluate(() => /** @type {any} */ (window).previewScriptRan)).toBeUndefined();
    for (const text of ['Head title', 'head-style', 'body-style', 'previewScriptRan']) {
      await expect(preview).not.toContainText(text);
    }
  });
}
