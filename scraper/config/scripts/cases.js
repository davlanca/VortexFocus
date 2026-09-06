(function() {
    const cases = [];
    const seen = new Set();

    document.querySelectorAll('.case-card[data-case-card] a.case-card__title[href]').forEach(link => {
        const url = link.href;
        const name = link.textContent.trim();
        if (url && name && !seen.has(url)) {
            seen.add(url);
            cases.push({ name, url, type: 'case' });
        }
    });

    return { items: cases };
})();
