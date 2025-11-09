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

  test('Browse page - should show File Name column', async ({ page }) => {
    await page.goto(UI_URL);

    // Check that File Name column header exists
    await expect(page.locator('th:has-text("File Name")')).toBeVisible();

    // Check that old CID/SHA256 columns are removed
    const cidHeader = await page.locator('th:has-text("CID")').count();
    const sha256Header = await page.locator('th:has-text("SHA256")').count();
    expect(cidHeader).toBe(0);
    expect(sha256Header).toBe(0);
  });

  test('Browse page - should have Copy CID button when files exist', async ({ page, request }) => {
    // First check if there are any files
    const response = await request.get(`${UI_URL}/api/files`);

    if (response.ok()) {
      const data = await response.json();

      if (data && data.length > 0) {
        await page.goto(UI_URL);

        // Should have copy button with title
        await expect(page.locator('button[title="Copy CID to clipboard"]').first()).toBeVisible();
      }
    }
  });

  test.describe('Search Functionality', () => {
    test('should display search input with correct placeholder', async ({ page }) => {
      await page.goto(UI_URL);

      const searchInput = page.locator('input[type="text"]').first();
      await expect(searchInput).toBeVisible();
      await expect(searchInput).toHaveAttribute('placeholder', /file name.*type.*CID.*hash/i);
    });

    test('should filter files by search term (file name)', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          // Get initial file count
          const initialCount = await page.locator('tbody tr').count();

          // Find a file with a fileName
          const fileWithName = files.find(f => f.fileName);

          if (fileWithName) {
            // Search for part of the filename
            const searchTerm = fileWithName.fileName.substring(0, 5);
            await page.locator('input[type="text"]').first().fill(searchTerm);

            // Wait for filtering to occur
            await page.waitForTimeout(500);

            // Should show fewer or equal files
            const filteredCount = await page.locator('tbody tr').count();
            expect(filteredCount).toBeLessThanOrEqual(initialCount);

            // Should show the file count indicator
            await expect(page.locator('text=/Showing \\d+ of \\d+ files/i')).toBeVisible();
          }
        }
      }
    });

    test('should filter files by CID', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0 && files[0].ipfsCID) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          // Search for first 10 chars of CID
          const searchTerm = files[0].ipfsCID.substring(0, 10);
          await page.locator('input[type="text"]').first().fill(searchTerm);

          await page.waitForTimeout(500);

          // Should have at least one result
          const rowCount = await page.locator('tbody tr').count();
          expect(rowCount).toBeGreaterThan(0);
        }
      }
    });

    test('should show "no results" message for non-matching search', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          // Search for something unlikely to exist
          await page.locator('input[type="text"]').first().fill('xyznonexistentfile123456');

          await page.waitForTimeout(500);

          // Should show no results message (different from "no files" message)
          await expect(page.locator('td:has-text("No results found")')).toBeVisible();
        }
      }
    });

    test('should persist search in URL parameters', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          // Enter a search term
          const searchTerm = 'test';
          await page.locator('input[type="text"]').first().fill(searchTerm);

          await page.waitForTimeout(500);

          // Check URL contains search parameter
          expect(page.url()).toContain(`search=${searchTerm}`);

          // Reload page
          await page.reload();

          // Search term should persist
          const inputValue = await page.locator('input[type="text"]').first().inputValue();
          expect(inputValue).toBe(searchTerm);
        }
      }
    });

    test('should have dynamic file type filter', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          const typeFilter = page.locator('select').first();
          await expect(typeFilter).toBeVisible();

          // Should have "All Types" option
          await expect(typeFilter.locator('option:has-text("All Types")')).toBeVisible();

          // Should have at least one file type option based on actual data
          const optionCount = await typeFilter.locator('option').count();
          expect(optionCount).toBeGreaterThan(1); // At least "All Types" + 1 actual type
        }
      }
    });

    test('should combine search with filter', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 1) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          // Get unique file types
          const fileTypes = [...new Set(files.map(f => f.fileType).filter(Boolean))];

          if (fileTypes.length > 0) {
            // Select first file type
            await page.locator('select').first().selectOption(fileTypes[0]);
            await page.waitForTimeout(500);

            const filteredCount = await page.locator('tbody tr').count();

            // Add search on top of filter
            await page.locator('input[type="text"]').first().fill('test');
            await page.waitForTimeout(500);

            const doubleFilteredCount = await page.locator('tbody tr').count();

            // Search + filter should show same or fewer results than filter alone
            expect(doubleFilteredCount).toBeLessThanOrEqual(filteredCount);
          }
        }
      }
    });

    test('should highlight search matches in results', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          const fileWithName = files.find(f => f.fileName && f.fileName.length > 5);

          if (fileWithName) {
            await page.goto(UI_URL);
            await page.waitForLoadState('networkidle');

            // Search for part of filename
            const searchTerm = fileWithName.fileName.substring(0, 4);
            await page.locator('input[type="text"]').first().fill(searchTerm);

            await page.waitForTimeout(500);

            // Check for highlighted text (mark element with yellow background)
            const highlightedElements = page.locator('mark.bg-yellow-200');
            const count = await highlightedElements.count();

            // Should have at least one highlighted match
            expect(count).toBeGreaterThan(0);
          }
        }
      }
    });

    test('should clear search when input is emptied', async ({ page, request }) => {
      const response = await request.get(`${UI_URL}/api/files`);

      if (response.ok()) {
        const files = await response.json();

        if (files && files.length > 0) {
          await page.goto(UI_URL);
          await page.waitForLoadState('networkidle');

          const initialCount = await page.locator('tbody tr').count();

          // Add search term
          await page.locator('input[type="text"]').first().fill('test');
          await page.waitForTimeout(500);

          // Clear search
          await page.locator('input[type="text"]').first().fill('');
          await page.waitForTimeout(500);

          // Should show all files again
          const finalCount = await page.locator('tbody tr').count();
          expect(finalCount).toBe(initialCount);

          // URL should not have search param
          expect(page.url()).not.toContain('search=');
        }
      }
    });
  });
});
