// @ts-check
const http = require('node:http');
const { test, expect } = require('@playwright/test');

// The attacker's site: a different site than the app (127.0.0.1 vs
// localhost), but also on loopback, so the browser's private network rules
// don't get in the way.
const attackerPort = 18081;

test('a page on another site cannot archive emails through the browser', async ({ page, context }) => {
  const attacker = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    res.end('<!doctype html><title>Some other site</title>');
  });
  await new Promise((resolve) => attacker.listen(attackerPort, '127.0.0.1', resolve));

  try {
    await page.goto('/');
    const target = page.locator('#inbox-list .email-item').first();
    await expect(target).toBeVisible();
    const targetId = await target.getAttribute('data-email-id');

    // From the other site, send the request a malicious page would: a
    // "simple" POST, which browsers send without a CORS preflight. Its
    // response is opaque to the page, but the server still receives it.
    const evil = await context.newPage();
    await evil.goto(`http://127.0.0.1:${attackerPort}/`);
    await evil.evaluate(async ({ url, id }) => {
      await fetch(url, {
        method: 'POST',
        mode: 'no-cors',
        headers: { 'Content-Type': 'text/plain' },
        body: JSON.stringify({ emailIds: [id] }),
      });
    }, { url: new URL('/api/archive', page.url()).href, id: targetId });

    const response = await page.request.get('/api/emails?limit=500');
    expect(response.ok()).toBe(true);
    const ids = (await response.json()).emails.map((email) => email.id);
    expect(ids).toContain(targetId);
  } finally {
    await new Promise((resolve) => attacker.close(resolve));
  }
});
