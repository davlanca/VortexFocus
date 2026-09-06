(function() {
    const items = [];
    const seen = new Set();

    document.querySelectorAll('a[href*="/gloves/"]').forEach(link => {
        const url = link.href.split('#')[0];
        const card = link.closest('.item-box');
        const image = card?.querySelector('img[alt]');
        const name = image?.getAttribute('alt')?.trim()
            || link.textContent.trim().replace(/\s+/g, ' ');
        if (!url || !name || seen.has(url)) return;
        seen.add(url);
        items.push({ name, url, type: 'gloves' });
    });

    return { items };
})();
