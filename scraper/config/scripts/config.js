// Prevents detection by overriding the webdriver property.
// On modern Chrome, `navigator.webdriver` is a NON-CONFIGURABLE accessor, so a
// bare Object.defineProperty(navigator, 'webdriver', ...) throws
// "Cannot redefine property: webdriver". That throw aborts this injected
// script (and, because it runs in every evaluated context, breaks the whole
// scraper). We guard every override so the script never throws, and fall back
// to shadowing on Navigator.prototype when the own property is not configurable.

try {
  Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
} catch (e) {
  // Own property is non-configurable (Chrome 118+). Try the prototype instead.
  try {
    const desc = Object.getOwnPropertyDescriptor(Navigator.prototype, 'webdriver');
    Object.defineProperty(Navigator.prototype, 'webdriver', {
      set: undefined,
      enumerable: desc ? desc.enumerable : false,
      configurable: true,
      get: () => undefined,
    });
  } catch (e2) { /* best effort only */ }
}

try {
  Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
} catch (e) {
  try {
    const desc = Object.getOwnPropertyDescriptor(Navigator.prototype, 'plugins');
    Object.defineProperty(Navigator.prototype, 'plugins', {
      set: undefined,
      enumerable: desc ? desc.enumerable : false,
      configurable: true,
      get: () => [1, 2, 3, 4, 5],
    });
  } catch (e2) { /* best effort only */ }
}

