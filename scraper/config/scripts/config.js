// prevents detection by ovverriding webdriver property.
// scraper might not work properly without this, so keep it.

Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
