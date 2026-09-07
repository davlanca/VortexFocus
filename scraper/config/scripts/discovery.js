(function() {
    // DEBUG: Log current page info
    console.log('[DEBUG] Page location:', window.location.pathname);
    
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

    // Get the current weapon slug (e.g., "cz75-auto" from "/weapons/cz75-auto/")
    // Filter out empty strings from split
    const pathParts = currentPath.split('/').filter(Boolean);
    const currentWeaponSlug = (activeCategory === '/weapons/' && pathParts.length > 1) 
        ? pathParts[1] 
        : null;

    console.log('[DEBUG] activeCategory:', activeCategory);
    console.log('[DEBUG] currentWeaponSlug:', currentWeaponSlug);

    // NOTE: Scrolling to trigger lazy-loaded content is handled on the Go side
    // (discovery.go) before this script runs, because an `await` inside this
    // non-async IIFE is a syntax error that breaks the whole script.
    let linksChecked = 0;
    let linksMatched = 0;
    let linksRejected = { noMatch: 0, wrongWeapon: 0, notTarget: 0, duplicate: 0};

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
            // From weapons pages, allow ONLY skins that match the current weapon
            let isNestedAllowed = (activeCategory === '/sticker-capsules/' && path.startsWith('/stickers/')) ||
                                  (activeCategory === '/collectible-pins/' && path.startsWith('/collectible-pins/') && path !== activeCategory);

            if (activeCategory === '/weapons/' && currentWeaponSlug && path.startsWith('/skins/')) {
                const skinWeaponPart = path.split('/')[2]; // e.g., "cz75-auto-victoria"
                // Accept if skin weapon part starts with current weapon slug
                // e.g., "cz75-auto-victoria" starts with "cz75-auto" = true
                isNestedAllowed = skinWeaponPart && (
                    skinWeaponPart.startsWith(currentWeaponSlug + '-') ||
                    skinWeaponPart === currentWeaponSlug
                );
                if (isNestedAllowed) {
                    console.log('[DEBUG] ✓ Accepted skin:', path, 'skinPart:', skinWeaponPart, 'weaponSlug:', currentWeaponSlug);
                }
            }

            if (!path.startsWith(activeCategory) && !isNestedAllowed) {
                // Detailed logging for rejected links
                if (path.startsWith('/skins/')) {
                    console.log('[DEBUG] ✗ Rejected skin (not current weapon):', path, 'activeCategory:', activeCategory, 'weaponSlug:', currentWeaponSlug);
                }
                linksRejected.notTarget++;
                return;
            }

            linksChecked++;
            if (targetPaths.some(p => path.startsWith(p))) {
                if (seen.has(path)) {
                    linksRejected.duplicate++;
                    return;
                }
                seen.add(path);
                linksMatched++;

                const card = a.closest('.item-box') || a.closest('.skin-box');
                let name = card?.querySelector('img[alt]')?.getAttribute('alt') ||
                           card?.querySelector('.item-name')?.textContent ||
                           a.textContent.trim();

                // Cleanup name
                if (name) name = name.replace(/\s+/g, ' ').trim();

                // Ignore generic category names
                if (!name || /^(gloves|cases|stickers|patches|agents|pins|weapons|souvenir packages)$/i.test(name)) {
                    console.log('[DEBUG] Rejected item (generic name):', path, 'name:', name);
                    linksRejected.noMatch++;
                    return;
                }

                items.push({
                    slug: path.split('/').filter(Boolean).pop(),
                    name: name,
                    url: url.origin + path,
                    type: path.split('/')[1]
                });
            }
        } catch (e) {}
    });
    
    console.log('[DEBUG] Links checked:', linksChecked, 'matched:', linksMatched);
    console.log('[DEBUG] Links rejected:', JSON.stringify(linksRejected));
    console.log('[DEBUG] Items found:', items.length);
    if (items.length > 0) {
        console.log('[DEBUG] First 5 items:', items.slice(0, 5).map(i => i.url).join(', '));
    }
    
    return { 
        items: items, 
        hasNextPage: false, 
        nextPageUrl: null
    };
})();
