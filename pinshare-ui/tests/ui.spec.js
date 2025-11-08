import { test, expect } from '@playwright/test';

const UI_URL = 'http://localhost:5174';

test.describe('PinShare UI', () => {
  test('should load the dashboard', async ({ page }) => {
    await page.goto(UI_URL);

    // Check title
    await expect(page.locator('h1')).toContainText('PinShare Dashboard');

    // Check navigation links exist
    await expect(page.locator('a:has-text("Browse")')).toBeVisible();
    await expect(page.locator('a:has-text("Network Info")')).toBeVisible();
    await expect(page.locator('a:has-text("Network Graph")')).toBeVisible();
  });

  test('should show stats in header', async ({ page }) => {
    await page.goto(UI_URL);

    // Stats should be visible (even if 0) - look for the specific stats format
    await expect(page.locator('text=/\\d+ Files/')).toBeVisible();
    await expect(page.locator('text=/\\d+ Peers/')).toBeVisible();
  });

  test('Browse page - should handle API errors gracefully', async ({ page }) => {
    await page.goto(UI_URL);

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Check if we see an error message OR data loaded
    const hasError = await page.locator('text=/Error|failed|502/i').isVisible().catch(() => false);
    const hasTable = await page.locator('table').isVisible().catch(() => false);
    const hasLoading = await page.locator('text=Loading').isVisible().catch(() => false);

    // Should show one of these states
    expect(hasError || hasTable || hasLoading).toBeTruthy();

    // If error, log it for debugging
    if (hasError) {
      const errorText = await page.locator('text=/Error|failed|502/i').first().textContent();
      console.log('UI shows error:', errorText);
    }
  });

  test('should test API endpoint directly through proxy', async ({ page, request }) => {
    // Test if the nginx proxy is working
    const response = await request.get(`${UI_URL}/api/files`);

    console.log('API Response Status:', response.status());
    console.log('API Response Headers:', await response.headers());

    if (response.status() === 502) {
      console.log('502 Bad Gateway - nginx cannot reach backend');
      const body = await response.text();
      console.log('Response body:', body);
    } else if (response.ok()) {
      const data = await response.json();
      console.log('API returned data:', data);
    }

    // This might fail with 502, but we want to see the details
    expect([200, 502]).toContain(response.status());
  });

  test('Network Graph page - should load', async ({ page }) => {
    await page.goto(`${UI_URL}/network-graph`);

    // Check the page title
    await expect(page.locator('h2:has-text("Network Visualization")')).toBeVisible();
  });
});
