(function() {
    window.extractPrices = function() {
        function parseNum(text) {
            if (!text) return NaN;
            const cleaned = text.replace(/[^\d.,]/g, '').replace(',', '.');
            return parseFloat(cleaned);
        }

        function detectCurrency(text) {
            if (!text) return 'USD';
            if (text.includes('€')) return 'EUR';
            if (text.includes('£')) return 'GBP';
            if (text.includes('₽')) return 'RUB';
            return 'USD';
        }

        const markets = [];
        const shareBoxes = document.querySelectorAll('a.share-box');
        if (shareBoxes.length > 0) {
            shareBoxes.forEach(box => {
                const labels = box.querySelectorAll('.share-label span');
                let marketName = '';
                if (labels.length >= 2) {
                    const marketStrong = labels[1].querySelector('strong');
                    marketName = marketStrong ? marketStrong.textContent.trim() : labels[1].textContent.trim();
                }
                if (!marketName) {
                    const img = box.querySelector('img');
                    marketName = img ? (img.getAttribute('alt') || '').replace(/ logo$/i, '').trim() : '';
                }

                const priceStrong = box.querySelector('.share-label strong');
                if (priceStrong && marketName) {
                    const text = priceStrong.textContent.trim();
                    const num = parseNum(text);
                    if (!isNaN(num)) {
                        markets.push({
                            market: marketName,
                            currency: detectCurrency(text),
                            single: num,
                            url: box.getAttribute('href') || '',
                            wearPrices: {},
                            urls: {}
                        });
                    }
                }
            });
        }

        if (markets.length === 0) {
            const table = document.querySelector('.skin-price-table');
            if (table) {
                table.querySelectorAll('tbody tr').forEach(row => {
                    const th = row.querySelector('th, .market-col');
                    if (!th) return;
                    const marketName = th.textContent.trim().replace(/ logo$/i, '');
                    const entry = { market: marketName, currency: 'USD', wearPrices: {}, urls: {} };
                    const pills = row.querySelectorAll('.price-pill');
                    if (pills.length === 1) {
                        const text = pills[0].textContent.trim();
                        entry.single = parseNum(text);
                        entry.url = pills[0].getAttribute('href') || '';
                        entry.currency = detectCurrency(text);
                    } else {
                        const wears = ['FN', 'MW', 'FT', 'WW', 'BS'];
                        row.querySelectorAll('td').forEach((cell, i) => {
                            const pill = cell.querySelector('.price-pill');
                            if (pill && i < wears.length) {
                                const text = pill.textContent.trim();
                                const val = parseNum(text);
                                if (!isNaN(val)) {
                                    entry.wearPrices[wears[i]] = val;
                                    entry.urls[wears[i]] = pill.getAttribute('href') || '';
                                    entry.currency = detectCurrency(text);
                                }
                            }
                        });
                    }
                    if (entry.single || Object.keys(entry.wearPrices).length > 0) markets.push(entry);
                });
            }
        }
        return markets;
    };

    const h1 = document.querySelector('h1');
    const itemName = h1 ? h1.textContent.trim() : (document.title || '').split(' - ')[0].trim();
    const souvenirTab = document.querySelector('.price-type-tab[data-type="souvenir"], a[href*="souvenir"]');
    const hasWear = !!document.querySelector('.skin-price-table thead th:nth-child(2)');

    return {
        itemName: itemName,
        hasWear: hasWear,
        normal: window.extractPrices(),
        hasSouvenir: !!souvenirTab,
        souvenir: null
    };
})();
