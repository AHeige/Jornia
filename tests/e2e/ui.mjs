// Shared navigation for the character-first desktop UI. No mutations are mocked here.
export async function settings(page){await page.locator('.settings-link').click();}
export async function companion(page){await page.locator('.app-nav [data-view="companion"]').click();}
export async function journey(page){await page.locator('.app-nav [data-view="journey"]').click();}
export async function openImport(page){await settings(page);if(await page.locator('#import-panel').getAttribute('open')===null)await page.locator('#import-panel > summary').click();}
