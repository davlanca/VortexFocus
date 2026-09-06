(function() {
    const url = window.location.href;
    const path = window.location.pathname;
    const slugs = new Set();
    const items = [];

    function addItem(href, label) {
        if (!href) return;

        let slug = '';
        let type = '';

        // 1. Root Weapons page -> find individual weapons
        if (path.endsWith('/weapons') || path.endsWith('/weapons/')) {
            const m = href.match(/\/weapons\/([a-z0-9-]+)\/?$/);
            if (m) {
                slug = m[1];
                type = 'weapon';
            }
        }
        // 2. Individual Weapon page -> find skins
        else if (path.includes('/weapons/')) {
            const m = href.match(/\/skins\/([a-z0-9-]+)\/?$/);
            if (m) {
                slug = m[1];
                type = 'skin';
            }
        }

        if (slug && !slugs.has(slug) && slug !== 'page' && !/^\d+$/.test(slug)) {
            slugs.add(slug);
            items.push({
                slug: slug,
                name: (label || '').trim().slice(0, 200),
                url: href.startsWith('http') ? href : (window.location.origin + href),
                type: type
            });
        }
    }

    document.querySelectorAll('.item-box a, .weapon-box a, .skin-list a, a.share-box').forEach(a => {
        addItem(a.getAttribute('href'), a.textContent);
    });

    if (items.length === 0) {
        document.querySelectorAll('main a[href], #content a[href]').forEach(a => {
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
