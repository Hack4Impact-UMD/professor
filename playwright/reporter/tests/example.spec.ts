import { test, expect } from '@playwright/test';

test.describe('my suite', () => {
  test('has title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle('Test Page');
  });

  test('has heading', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Hello');
  });
});
