(function() {
    const prices = [];
    const table = document.querySelector('.single-price-table');

    function parsePrice(text) {
        const value = Number.parseFloat(String(text || '').replace(/[^\d.,]/g, '').replace(',', '.'));
        return Number.isFinite(value) && value > 0 ? value : NaN;
    }

    if (table) {
        table.querySelectorAll('tbody tr').forEach(row => {
            const market = row.querySelector('.sr-only')?.textContent.trim()
                || row.querySelector('.market-col')?.textContent.trim();
            const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
            const price = parsePrice(pill?.textContent);
            if (!market || !pill || !Number.isFinite(price)) return;
            prices.push({ market, price, currency: 'USD', has_price: true, url: pill.getAttribute('href') || '' });
        });
        return prices;
    }

    document.querySelectorAll('.skin-price-cards .skin-price-card').forEach(card => {
        const market = card.querySelector('.sr-only')?.textContent.trim()
            || card.querySelector('h3 img')?.getAttribute('alt')?.replace(/ logo$/i, '').trim();
        if (!market) return;
        card.querySelectorAll('.skin-price-card-row').forEach(row => {
            const wear = row.querySelector('dt')?.textContent.trim() || '';
            const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
            const price = parsePrice(pill?.textContent);
            if (!pill || !Number.isFinite(price)) return;
            prices.push({ market, wear, price, currency: 'USD', has_price: true, url: pill.getAttribute('href') || '' });
        });
    });

    return prices;
})();
