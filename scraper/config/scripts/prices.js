(function() {
    function parseNum(text) {
        if (!text) return NaN;
        // Убираем всё кроме цифр, точек и запятых, приводим к формату 0.00
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

    const h1 = document.querySelector('h1');
    const itemName = h1 ? h1.textContent.trim() : (document.title || '').split(' - ')[0].trim();
    const markets = [];

    // 1. Формат ПЛИТКИ (Revolution Case и подобные)
    const shareBoxes = document.querySelectorAll('a.share-box');
    if (shareBoxes.length > 0) {
        shareBoxes.forEach(box => {
            const labels = box.querySelectorAll('.share-label span');
            let marketName = '';

            // Название маркета обычно во втором span (например, "Steam Market")
            if (labels.length >= 2) {
                const marketStrong = labels[1].querySelector('strong');
                marketName = marketStrong ? marketStrong.textContent.trim() : labels[1].textContent.trim();
            }

            // Если не нашли в span, пробуем alt картинки
            if (!marketName) {
                const img = box.querySelector('img');
                marketName = img ? (img.getAttribute('alt') || '').replace(/ logo$/i, '').trim() : '';
            }

            // Цена обычно в первом span в теге strong
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

    // 2. Формат ТАБЛИЦЫ (Скины с износом)
    if (markets.length === 0) {
        const table = document.querySelector('.skin-price-table');
        if (table) {
            const rows = table.querySelectorAll('tbody tr');
            rows.forEach(row => {
                const th = row.querySelector('th, .market-col');
                if (!th) return;
                const marketName = th.textContent.trim().replace(/ logo$/i, '');
                const entry = { market: marketName, currency: 'USD', wearPrices: {}, urls: {} };

                const pills = row.querySelectorAll('.price-pill');
                if (pills.length === 1) {
                    const text = pills[0].textContent.trim();
                    entry.currency = detectCurrency(text);
                    entry.single = parseNum(text);
                    entry.url = pills[0].getAttribute('href') || '';
                } else {
                    const wears = ['FN', 'MW', 'FT', 'WW', 'BS'];
                    const cells = row.querySelectorAll('td');
                    cells.forEach((cell, i) => {
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
                if (entry.single || Object.keys(entry.wearPrices).length > 0) {
                    markets.push(entry);
                }
            });
        }
    }

    return {
        itemName: itemName,
        hasWear: markets.some(m => Object.keys(m.wearPrices).length > 0),
        markets: markets
    };
})();
