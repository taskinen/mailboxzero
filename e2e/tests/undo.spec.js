// @ts-check
const { test, expect } = require('@playwright/test');

test('Undo reloads the inbox: the count and the emails come back', async ({ page }) => {
  await page.goto('/');
  const title = page.locator('#inbox-title');
  await expect(page.locator('#inbox-list .email-item').first()).toBeVisible();
  // The mock server's state is shared by every test, so count from here.
  const before = Number((await title.textContent())?.match(/Inbox \((\d+)\)/)?.[1]);
  expect(before).toBeGreaterThan(0);

  await page.click('#find-similar-btn');
  const selected = page.locator('#similar-list .email-item:has(.email-checkbox:checked)');
  await expect(selected.first()).toBeVisible();
  const archivedIds = await selected.evaluateAll((els) => els.map((el) => el.dataset.emailId));

  await page.click('#archive-find-next-btn');
  await expect(page.locator('#archive-status-text')).toHaveText(`Successfully archived ${archivedIds.length} e-mail(s)`);
  await expect(title).toHaveText(`Inbox (${before - archivedIds.length})`);
  for (const id of archivedIds) {
    await expect(page.locator(`#inbox-list [data-email-id="${id}"]`)).toHaveCount(0);
  }

  await page.click('#archive-undo-btn');
  await expect(page.locator('#archive-status-text')).toHaveText(`Unarchived ${archivedIds.length} e-mail(s)`);
  await expect(title).toHaveText(`Inbox (${before})`);
  for (const id of archivedIds) {
    await expect(page.locator(`#inbox-list [data-email-id="${id}"]`)).toHaveCount(1);
  }
});
