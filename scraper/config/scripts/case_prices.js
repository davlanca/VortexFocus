(function() {
    const prices = [];
    const table = document.querySelector('.single-price-table');
    if (!table) return prices;

    table.querySelectorAll('tbody tr').forEach(row => {
        const market = row.querySelector('.sr-only')?.textContent.trim()
            || row.querySelector('.market-col')?.textContent.trim();
        const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
        if (!market || !pill) return;

        const text = pill.textContent.trim();
        const normalized = text.replace(/[^\d.,]/g, '').replace(',', '.');
        const price = Number.parseFloat(normalized);
        if (!Number.isFinite(price) || price <= 0) return;

        prices.push({
            market,
            price,
            currency: 'USD',
            has_price: true,
            url: pill.getAttribute('href') || ''
        });
    });

    return prices;
})();
