// @ts-check
const { test, expect } = require('@playwright/test');

// In the mock inbox, email-1-1 (one of four Stripe emails) is flagged, and the
// newest email from each sender (email-1-3 for Stripe) is unread.
const inboxItem = (page, id) => page.locator(`#inbox-list [data-email-id="${id}"]`);
const similarItem = (page, id) => page.locator(`#similar-list [data-email-id="${id}"]`);

test('the inbox marks unread and flagged emails', async ({ page }) => {
  await page.goto('/');

  await expect(inboxItem(page, 'email-1-3')).toHaveClass(/\bunread\b/);
  await expect(inboxItem(page, 'email-1-3').locator('.unread-marker')).toBeVisible();
  await expect(inboxItem(page, 'email-1-0')).not.toHaveClass(/\bunread\b/);
  await expect(inboxItem(page, 'email-1-0').locator('.unread-marker')).toHaveCount(0);

  await expect(inboxItem(page, 'email-1-1')).toHaveClass(/\bflagged\b/);
  await expect(inboxItem(page, 'email-1-1').locator('.flag-marker')).toBeVisible();
  await expect(inboxItem(page, 'email-1-0').locator('.flag-marker')).toHaveCount(0);
});

test('a flagged email in a group of similar emails starts unselected', async ({ page }) => {
  await page.goto('/');
  await inboxItem(page, 'email-1-0').click();
  await page.click('#find-similar-btn');

  await expect(similarItem(page, 'email-1-1')).toBeVisible();
  await expect(similarItem(page, 'email-1-1').locator('.flag-marker')).toBeVisible();
  await expect(similarItem(page, 'email-1-1').locator('.email-checkbox')).not.toBeChecked();
  for (const id of ['email-1-0', 'email-1-2', 'email-1-3']) {
    await expect(similarItem(page, id).locator('.email-checkbox')).toBeChecked();
  }
  // Some, not all, are selected.
  await expect(page.locator('#select-all-checkbox')).toHaveJSProperty('indeterminate', true);

  // Archive Selected leaves the flagged email out.
  await page.click('#archive-btn');
  await expect(page.locator('#archive-count')).toHaveText('3');
  await page.click('#cancel-archive-btn');

  // Select All still includes it when asked to.
  await page.check('#select-all-checkbox');
  await expect(similarItem(page, 'email-1-1').locator('.email-checkbox')).toBeChecked();
});
