// @ts-check
const { test, expect } = require('@playwright/test');

const apiPath = (response) => new URL(response.url()).pathname;

test('listings carry no bodies, and hovering an email loads its full body', async ({ page }) => {
  const inboxResponse = page.waitForResponse((response) => apiPath(response) === '/api/emails');
  await page.goto('/');
  const inbox = await (await inboxResponse).json();
  expect(inbox.emails.length).toBeGreaterThan(0);
  for (const email of inbox.emails) {
    expect(email.bodyValues).toBeNull();
  }

  const similarResponse = page.waitForResponse((response) => apiPath(response) === '/api/similar');
  await page.click('#find-similar-btn');
  const similar = await (await similarResponse).json();
  expect(similar.length).toBeGreaterThan(1);
  for (const email of similar) {
    expect(email.bodyValues).toBeNull();
  }

  // Hovering a result fetches that one email with its body. The sample body
  // ends with a sentence that is not part of the preview text.
  const target = page.locator('#similar-list .email-item').first();
  const targetId = await target.getAttribute('data-email-id');
  const bodyRequest = page.waitForRequest((request) => apiPath(request) === `/api/emails/${targetId}`);
  await target.hover();
  await bodyRequest;

  await expect(page.locator('#email-preview-popup')).toBeVisible();
  await expect(page.locator('#preview-body')).toContainText('This is additional content for the email body.');
});
