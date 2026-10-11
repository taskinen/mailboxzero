// @ts-check
const { test, expect } = require('@playwright/test');

test('Archive & Find Next after targeting an email moves on to the next group', async ({ page }) => {
  await page.goto('/');

  const inboxItems = page.locator('#inbox-list .email-item');
  const similarItems = page.locator('#similar-list .email-item');
  const idsOf = (items) => items.evaluateAll((els) => els.map((el) => el.dataset.emailId));

  // Target the first inbox email and find the emails similar to it.
  const target = inboxItems.first();
  await expect(target).toBeVisible();
  const targetId = await target.getAttribute('data-email-id');
  await target.click();
  await expect(target).toHaveClass(/selected/);

  await page.click('#find-similar-btn');
  await expect(page.locator(`#similar-list [data-email-id="${targetId}"]`)).toHaveCount(1);
  const archivedIds = await idsOf(similarItems);
  expect(archivedIds.length).toBeGreaterThan(1);

  // Archive the whole group (all results are selected by default) and let
  // the app search again.
  await page.click('#archive-find-next-btn');
  await expect(page.locator('#archive-status-text')).toHaveText(`Successfully archived ${archivedIds.length} e-mail(s)`);

  // The follow-up search must not target the archived email any more: it
  // should show the next group instead of an error.
  await expect
    .poll(async () => {
      const ids = await idsOf(similarItems);
      return ids.length > 0 && !ids.some((id) => archivedIds.includes(id));
    }, { message: 'next group of similar emails is shown' })
    .toBe(true);
  await expect(page.locator('#similar-list')).not.toContainText('Failed to find similar emails');

  await expect(page.locator(`#inbox-list [data-email-id="${targetId}"]`)).toHaveCount(0);
  await expect(page.locator('#inbox-list .email-item.selected')).toHaveCount(0);
});
