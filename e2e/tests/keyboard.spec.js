// @ts-check
const { test, expect } = require('@playwright/test');

async function openInbox(page) {
  await page.goto('/');
  await expect(page.locator('#inbox-list .email-item').first()).toBeVisible();
}

test('F, A and U find, archive and undo without the mouse', async ({ page }) => {
  await openInbox(page);

  await page.keyboard.press('f');
  const similar = page.locator('#similar-list .email-item');
  await expect(similar.first()).toBeVisible();
  const count = await page.locator('#similar-list .email-checkbox:checked').count();

  await page.keyboard.press('a');
  await expect(page.locator('#archive-status-text')).toHaveText(`Successfully archived ${count} e-mail(s)`);

  await page.keyboard.press('u');
  await expect(page.locator('#archive-status-text')).toHaveText(`Unarchived ${count} e-mail(s)`);

  // C clears the results.
  await expect(similar.first()).toBeVisible();
  await page.keyboard.press('c');
  await expect(similar).toHaveCount(0);
});

test('inbox emails can be chosen with the arrow keys and Enter', async ({ page }) => {
  await openInbox(page);
  const rows = page.locator('#inbox-list .email-item');

  await rows.first().focus();
  await page.keyboard.press('ArrowDown');
  await expect(rows.nth(1)).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('ArrowUp');
  await expect(rows.nth(1)).toBeFocused();

  await page.keyboard.press('Enter');
  await expect(rows.nth(1)).toHaveClass(/\bselected\b/);
  await expect(rows.nth(1)).toHaveAttribute('aria-pressed', 'true');
  const targetId = await rows.nth(1).getAttribute('data-email-id');

  // F now finds the emails similar to the chosen one.
  await page.keyboard.press('f');
  await expect(page.locator(`#similar-list [data-email-id="${targetId}"]`)).toHaveCount(1);

  // Space chooses it again, which clears the choice.
  await rows.nth(1).focus();
  await page.keyboard.press(' ');
  await expect(rows.nth(1)).not.toHaveClass(/\bselected\b/);
  await expect(rows.nth(1)).toHaveAttribute('aria-pressed', 'false');
});

test('arrow keys move between the checkboxes of the similar emails', async ({ page }) => {
  await openInbox(page);
  await page.keyboard.press('f');
  const checkboxes = page.locator('#similar-list .email-checkbox');
  await expect(checkboxes.nth(1)).toBeVisible();

  await checkboxes.first().focus();
  await page.keyboard.press('ArrowDown');
  await expect(checkboxes.nth(1)).toBeFocused();
  await page.keyboard.press(' ');
  await expect(checkboxes.nth(1)).not.toBeChecked();
});

test('letters typed in a drop-down are not shortcuts', async ({ page }) => {
  await openInbox(page);
  let searches = 0;
  page.on('request', (request) => {
    if (new URL(request.url()).pathname === '/api/similar') searches++;
  });

  await page.focus('#inbox-sort');
  await page.keyboard.press('f');
  await page.focus('#per-page-select');
  await page.keyboard.press('f');

  await page.waitForTimeout(250);
  expect(searches).toBe(0);
  await expect(page.locator('#similar-list .email-item')).toHaveCount(0);
});

test('Escape closes the archive dialog, and shortcuts wait while it is open', async ({ page }) => {
  await openInbox(page);
  await page.keyboard.press('f');
  await expect(page.locator('#similar-list .email-item').first()).toBeVisible();

  await page.click('#archive-btn');
  await expect(page.locator('#archive-modal')).toBeVisible();
  await page.keyboard.press('c');
  await expect(page.locator('#similar-list .email-item').first()).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(page.locator('#archive-modal')).toBeHidden();
});

test('the buttons show their shortcuts', async ({ page }) => {
  await openInbox(page);
  for (const [id, key] of [['#find-similar-btn', 'F'], ['#archive-find-next-btn', 'A'], ['#archive-undo-btn', 'U'], ['#refresh-btn', 'R'], ['#clear-results-btn', 'C']]) {
    await expect(page.locator(id)).toHaveAttribute('aria-keyshortcuts', key);
    await expect(page.locator(id)).toHaveAttribute('title', new RegExp(`\\(${key}\\)$`));
  }
});
