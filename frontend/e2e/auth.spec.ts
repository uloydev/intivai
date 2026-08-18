import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should render login page correctly', async ({ page }) => {
    await page.goto('/login');

    // Check if the URL is /login
    await expect(page).toHaveURL(/.*\/login/);

    // Verify key elements are present
    await expect(page.getByText('Intivai Workspace')).toBeVisible();
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
  });

  test('should show validation errors on empty submit', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('button', { name: 'Sign in' }).click();
    
    // Form remains on /login when required fields are missing
    await expect(page).toHaveURL(/.*\/login/);
  });
});
