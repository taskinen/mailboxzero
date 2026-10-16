// @ts-check
const { test, expect } = require('@playwright/test');

// The header names the running version: the release (v1.2.3), the commit of
// a development build (abc1234, -dirty with uncommitted changes), or "dev"
// for go run, which records neither. Hovering shows the full description.
test('the header shows the version', async ({ page }) => {
  await page.goto('/');

  const label = page.locator('header h1 .app-version');
  await expect(label).toBeVisible();
  await expect(label).toHaveText(/^(v\d+\.\d+\.\d+\S*|[0-9a-f]{7}(-dirty)?|dev)$/);
  await expect(label).toHaveAttribute('title', /^mailboxzero \S+ \(go\S+, \w+\/\w+\)$/);
});
