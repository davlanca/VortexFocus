(function() {
    const items = [];
    const seen = new Set();
    const loc = window.location;
    const targetPaths = ['/weapons/', '/skins/', '/gloves/', '/cases/', '/agents/', '/souvenir-packages/', '/collectible-pins/', '/patches/', '/sticker-capsules/', '/stickers/'];

    // Normalize current path for accurate category detection
    let currentPath = loc.pathname.toLowerCase();
    if (!currentPath.endsWith('/')) currentPath += '/';

    // Determine the active category based on the URL
    // We look for the most specific match among targetPaths
    const activeCategory = targetPaths
        .filter(p => currentPath.startsWith(p))
        .sort((a, b) => b.length - a.length)[0] || '/';

    document.querySelectorAll('a[href]').forEach(a => {
        try {
            const url = new URL(a.href, loc.origin);
            if (url.origin !== loc.origin) return;

            let path = url.pathname.toLowerCase();
            if (!path.endsWith('/')) path += '/';

            // Skip common non-item pages
            if (path.includes('/page/') || path === '/' || path.includes('/compare/')) return;

            // Avoid scraping the category root page itself as an item
            if (targetPaths.includes(path)) return;

            // STRICT FILTERING: Link must be inside the active category path
            // Exception: from sticker capsules we allow stickers, from pins we allow specific pins
            const isNestedAllowed = (activeCategory === '/sticker-capsules/' && path.startsWith('/stickers/')) ||
                                    (activeCategory === '/collectible-pins/' && path.startsWith('/collectible-pins/') && path !== activeCategory);

            if (!path.startsWith(activeCategory) && !isNestedAllowed) return;

            if (targetPaths.some(p => path.startsWith(p))) {
                if (!seen.has(path)) {
                    seen.add(path);
                    const card = a.closest('.item-box') || a.closest('.skin-box');
                    let name = card?.querySelector('img[alt]')?.getAttribute('alt') ||
                               card?.querySelector('.item-name')?.textContent ||
                               a.textContent.trim();

                    // Cleanup name
                    if (name) name = name.replace(/\s+/g, ' ').trim();

                    // Ignore generic category names
                    if (!name || /^(gloves|cases|stickers|patches|agents|pins|weapons|souvenir packages)$/i.test(name)) return;

                    items.push({
                        slug: path.split('/').filter(Boolean).pop(),
                        name: name,
                        url: url.origin + path,
                        type: path.split('/')[1]
                    });
                }
            }
        } catch (e) {}
    });
    return { items: items, hasNextPage: false, nextPageUrl: null };
})();
