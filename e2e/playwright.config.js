// @ts-check
const { defineConfig } = require('@playwright/test');

const port = 18080;

module.exports = defineConfig({
  testDir: './tests',
  // The mock server keeps archive state in memory and is shared by every
  // test, so run tests one at a time.
  workers: 1,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: `http://localhost:${port}`,
    // Use the locally installed Google Chrome (also preinstalled on GitHub's
    // ubuntu runners) instead of downloading a Playwright browser build.
    channel: 'chrome',
    headless: true,
  },
  webServer: {
    command: 'go run main.go -config e2e/config-e2e.yaml',
    cwd: '..',
    url: `http://localhost:${port}`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
