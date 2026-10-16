// @ts-check
const { test, expect } = require('@playwright/test');

// The label in front of the slider shows its value. If the label changed
// width with the number, the slider would move under the pointer.
test('the similarity slider stays in place while its value changes', async ({ page }) => {
  await page.goto('/');
  const slider = page.locator('#similarity-slider');

  const positions = [];
  for (const value of ['0', '7', '42', '100']) {
    await slider.fill(value);
    await expect(page.locator('#similarity-value')).toHaveText(`${value}%`);
    positions.push((await slider.boundingBox())?.x);
  }

  expect(positions[0]).toBeDefined();
  expect(positions).toEqual(positions.map(() => positions[0]));
});
