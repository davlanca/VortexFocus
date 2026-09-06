(function() {
    const url = window.location.href;
    const path = window.location.pathname;

    // Determine the current category from URL
    const m = url.match(/\/(agents|cases|collections|terminals|gloves|patches|collectible-pins|skins|souvenir-packages|stickers|sticker-capsules|weapons)\/?/);
    const category = m ? m[1] : '';

    const slugs = new Set();
    const items = [];

    function addItem(href, label) {
        if (!href) return;

        // Logic for nested discovery:
        // 1. If at /skins/ root -> we want to discover Collections
        // 2. If at /collections/ page -> we want to discover Skins
        // 3. Otherwise -> we want to discover items of the current category

        let targetCat = category;
        const isSkinsRoot = path.endsWith('/skins') || path.endsWith('/skins/');
        const isCollectionPage = path.includes('/collections/');

        if (isSkinsRoot) {
            targetCat = 'collections';
        } else if (isCollectionPage) {
            targetCat = 'skins';
        }

        if (!targetCat) return;

        const re = new RegExp(`/${targetCat}/([a-z0-9-]+)/?$`);
        const mm = href.match(re);
        if (!mm) return;

        const slug = mm[1];
        // Ignore pagination and noise
        if (slugs.has(slug) || slug === 'page' || /^\d+$/.test(slug)) return;

        slugs.add(slug);
        items.push({
            slug: slug,
            name: (label || '').trim().slice(0, 200),
            url: href.startsWith('http') ? href : (window.location.origin + href),
            isCollection: isSkinsRoot
        });
    }

    // Capture links from item boxes, collection boxes, etc.
    document.querySelectorAll('.item-box a, .collection-box a, a.share-box, .skin-list a').forEach(a => {
        addItem(a.getAttribute('href'), a.textContent);
    });

    // Fallback: search main content area
    if (items.length === 0) {
        document.querySelectorAll('main a[href], #content a[href], .container a[href]').forEach(a => {
            addItem(a.getAttribute('href'), a.textContent);
        });
    }

    let nextPageUrl = null;
    const nextEl = document.querySelector('a.next, a[rel="next"], .pagination a:last-child');
    if (nextEl && nextEl.getAttribute('href')) {
        const href = nextEl.getAttribute('href');
        if (href && !href.startsWith('#') && !href.toLowerCase().startsWith('javascript')) {
            nextPageUrl = href.startsWith('http') ? href : (window.location.origin + href);
        }
    }

    return {
        items: items,
        hasNextPage: !!nextPageUrl,
        nextPageUrl: nextPageUrl
    };
})();
