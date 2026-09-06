(function() {
    const items = [];
    const seen = new Set();

    document.querySelectorAll('a[href*="/agents/"]').forEach(link => {
        const url = link.href.split('#')[0];
        const card = link.closest('.item-box');
        const image = card?.querySelector('img[alt]');
        const name = image?.getAttribute('alt')?.trim()
            || link.textContent.trim().replace(/\s+/g, ' ');
        if (!url || !name || seen.has(url) || url.endsWith('/agents/')) return;
        seen.add(url);
        items.push({ name, url, type: 'agents' });
    });

    return { items };
})();
