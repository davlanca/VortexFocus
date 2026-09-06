// =========================================================================
// discovery.js — collect all item slugs from a category listing page
// =========================================================================
//
// Injected via chromedp.Evaluate. Returns: { items: [{slug, name, url}],
// hasNextPage: bool, nextPageUrl: string|null }.
//
// Strategy:
//   1. Primary selector: ".item-box a" (covers skins/cases/agents/etc.)
//   2. Fallback: any anchor whose href matches /category/<slug>/
//   3. Deduplicate by slug
//   4. Detect "next page" link using common pagination patterns

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
        // skip pure pagination / filter / sort links
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

    // Detect next-page link. Try several common patterns.
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
            // skip "#" or javascript: links
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
