// =========================================================================
// discovery.js — collect all item slugs from a category listing page
// =========================================================================

(function() {
    const url = window.location.href;
    const m = url.match(/\/(agents|cases|collections|terminals|gloves|patches|collectible-pins|skins|souvenir-packages|stickers|sticker-capsules|weapons)\/?/);
    if (!m) return { items: [], hasNextPage: false, nextPageUrl: null };
    const category = m[1];

    const slugs = new Set();
    const items = [];

    function addItem(href, label) {
        if (!href) return;
        const re = new RegExp(`/${category}/([a-z0-9-]+)/?$`);
        const mm = href.match(re);
        if (!mm) return;
        const slug = mm[1];
        if (slugs.has(slug)) return;
        if (slug === 'page' || /^\d+$/.test(slug)) return;
        slugs.add(slug);
        items.push({
            slug: slug,
            name: (label || '').trim().slice(0, 200),
            url: href.startsWith('http') ? href : (window.location.origin + href)
        });
    }

    document.querySelectorAll('.item-box a').forEach(a => {
        addItem(a.getAttribute('href'), a.textContent);
    });

    if (items.length === 0) {
        document.querySelectorAll('main a[href], .content a[href], .item-list a[href]').forEach(a => {
            addItem(a.getAttribute('href'), a.textContent);
        });
    }

    let nextPageUrl = null;
    const nextCandidates = [
        'a.next[href]',
        'a[rel="next"][href]',
        'a.pagination__next[href]',
        'a[aria-label*="Next" i][href]',
        'a[title*="Next" i][href]',
        '.pagination a:last-child[href]',
        '.page-numbers a.next[href]'
    ];
    for (const sel of nextCandidates) {
        const el = document.querySelector(sel);
        if (el && el.getAttribute('href')) {
            const href = el.getAttribute('href');
            if (href && !href.startsWith('#') && !href.toLowerCase().startsWith('javascript')) {
                nextPageUrl = href.startsWith('http') ? href : (window.location.origin + href);
                break;
            }
        }
    }

    return {
        items: items,
        hasNextPage: !!nextPageUrl,
        nextPageUrl: nextPageUrl
    };
})();
