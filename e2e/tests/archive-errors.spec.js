// @ts-check
const { test, expect } = require('@playwright/test');

const archiveError = 'Failed to archive emails: server rejected 1 email(s): email-0-0 (notFound: no such email)';
const unarchiveError = 'Failed to unarchive emails: server rejected 1 email(s): email-1-0 (forbidden)';

// Answer one API path with HTTP 500 and the message, the way the server
// reports emails that JMAP rejected.
async function failWith(page, path, message) {
  await page.route((url) => url.pathname === path, (route) =>
    route.fulfill({ status: 500, contentType: 'text/plain; charset=utf-8', body: `${message}\n` }));
}

// Count the inbox reloads (GET /api/emails) from now on.
function countInboxLoads(page) {
  const counter = { loads: 0 };
  page.on('request', (request) => {
    if (new URL(request.url()).pathname === '/api/emails') counter.loads++;
  });
  return counter;
}

async function findSimilar(page) {
  await page.goto('/');
  await expect(page.locator('#inbox-list .email-item').first()).toBeVisible();
  await page.click('#find-similar-btn');
  await expect(page.locator('#similar-list .email-item').first()).toBeVisible();
}

async function expectArchiveError(page, message) {
  await expect(page.locator('#archive-status')).toBeVisible();
  await expect(page.locator('#archive-status')).toHaveClass(/\berror\b/);
  await expect(page.locator('#archive-status-text')).toHaveText(message);
  await expect(page.locator('#archive-undo-btn')).toBeHidden();
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
  test(`${name} shows why the archive failed and reloads the inbox`, async ({ page }) => {
    await failWith(page, '/api/archive', archiveError);
    await findSimilar(page);
    const inbox = countInboxLoads(page);

    await archive(page);

    await expectArchiveError(page, archiveError);
    // Some emails may have moved before the server gave up.
    await expect.poll(() => inbox.loads).toBeGreaterThan(0);
  });
}

test('an archive that cannot reach the server says so', async ({ page }) => {
  await page.route((url) => url.pathname === '/api/archive', (route) => route.abort());
  await findSimilar(page);

  await page.click('#archive-find-next-btn');

  await expectArchiveError(page, 'Failed to archive emails: could not reach the server');
});

test('a failed undo shows why and reloads the inbox', async ({ page }) => {
  await findSimilar(page);
  // The archive itself goes through (mock mode) ...
  const nextSearch = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/similar');
  await page.click('#archive-find-next-btn');
  await expect(page.locator('#archive-status-text')).toHaveText(/^Successfully archived/);
  // The inbox reload after the archive comes before the next search.
  await nextSearch;

  // ... but the server rejects the undo.
  await failWith(page, '/api/unarchive', unarchiveError);
  const inbox = countInboxLoads(page);
  await page.click('#archive-undo-btn');

  await expectArchiveError(page, unarchiveError);
  // Some emails may have moved back before the server gave up.
  await expect.poll(() => inbox.loads).toBeGreaterThan(0);
});
