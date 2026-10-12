// @ts-check
const { test, expect } = require('@playwright/test');

// The e2e server really archives (dry_run: false), so these tests answer
// /api/archive the way the server does in dry-run mode.
async function archiveAsDryRun(page) {
  await page.route((url) => url.pathname === '/api/archive', async (route) => {
    const { emailIds } = route.request().postDataJSON();
    await route.fulfill({
      json: { success: true, message: `Successfully archived ${emailIds.length} emails`, dryRun: true },
    });
  });
}

const archiveButtons = [
  { name: 'Archive & Find Next', archive: (page) => page.click('#archive-find-next-btn') },
  {
    name: 'Archive Selected',
    archive: async (page) => {
      await page.click('#archive-btn');
      await page.click('#confirm-archive-btn');
    },
  },
];

for (const { name, archive } of archiveButtons) {
  test(`${name} in a dry run says nothing was archived and offers no undo`, async ({ page }) => {
    await archiveAsDryRun(page);
    await page.goto('/');
    await expect(page.locator('#inbox-list .email-item').first()).toBeVisible();
    await page.click('#find-similar-btn');
    const selected = page.locator('#similar-list .email-checkbox:checked');
    await expect(selected.first()).toBeVisible();
    const count = await selected.count();

    await archive(page);

    await expect(page.locator('#archive-status')).toBeVisible();
    await expect(page.locator('#archive-status-text')).toHaveText(`Dry run: would have archived ${count} e-mail(s)`);
    await expect(page.locator('#archive-status')).toHaveClass(/\bdry-run\b/);
    await expect(page.locator('#archive-undo-btn')).toBeHidden();
  });
}
