(function() {
    function parseNum(text) {
        if (!text) return NaN;
        let cleaned = text.replace(/[^\d.,]/g, '').trim();
        const lastComma = cleaned.lastIndexOf(',');
        const lastDot = cleaned.lastIndexOf('.');
        if (lastComma >= 0 && lastDot >= 0) {
            cleaned = lastComma > lastDot ? cleaned.replace(/\./g, '').replace(',', '.') : cleaned.replace(/,/g, '');
        } else if (lastComma >= 0) {
            cleaned = /,\d{1,2}$/.test(cleaned) ? cleaned.replace(',', '.') : cleaned.replace(/,/g, '');
        }
        return parseFloat(cleaned);
    }

    function detectCurrency(text) {
        if (!text) return 'USD';
        const t = text.toLowerCase();
        if (t.includes('€')) return 'EUR';
        if (t.includes('£')) return 'GBP';
        if (t.includes('₽')) return 'RUB';
        return 'USD';
    }

    const markets = [];
    const wearCodes = { 'Factory New': 'FN', 'Minimal Wear': 'MW', 'Field-Tested': 'FT', 'Well-Worn': 'WW', 'Battle-Scarred': 'BS' };

    // 1. Парсим single-price-table (Агенты, Кейсы, Нашивки, Значки, Наклейки)
    const singleTable = document.querySelector('.single-price-table');
    if (singleTable) {
        singleTable.querySelectorAll('tbody tr').forEach(row => {
            const th = row.querySelector('th, .market-col');
            if (!th) return;
            const marketName = th.querySelector('.sr-only')?.textContent.trim()
                || th.querySelector('img')?.getAttribute('alt')?.replace(/ logo$/i, '').trim()
                || th.textContent.trim().replace(/ logo$/i, '');

            const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
            if (pill) {
                const text = pill.textContent.trim();
                const value = parseNum(text);
                if (!isNaN(value)) {
                    markets.push({
                        market: marketName,
                        currency: detectCurrency(text),
                        single: value,
                        url: pill.getAttribute('href') || ''
                    });
                }
            }
        });
    }

    // 2. Если таблица пуста, пробуем карточки (Оружие, Перчатки)
    if (markets.length === 0) {
        document.querySelectorAll('.skin-price-cards .skin-price-card').forEach(card => {
            const marketName = card.querySelector('.sr-only')?.textContent.trim()
                || card.querySelector('h3 img')?.getAttribute('alt')?.replace(/ logo$/i, '').trim();
            if (!marketName) return;

            card.querySelectorAll('.skin-price-card-row').forEach(row => {
                const wearName = row.querySelector('dt')?.textContent.trim() || '';
                const wear = wearCodes[wearName] || wearName;
                const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
                if (pill) {
                    const text = pill.textContent.trim();
                    const value = parseNum(text);
                    if (!isNaN(value)) {
                        markets.push({
                            market: marketName,
                            currency: detectCurrency(text),
                            wearPrices: { [wear]: value },
                            urls: { [wear]: pill.getAttribute('href') || '' }
                        });
                    }
                }
            });
        });
    }

    const h1 = document.querySelector('h1');
    const itemName = h1 ? h1.textContent.trim() : (document.title || '').split(' - ')[0].trim();

    return {
        itemName: itemName,
        hasWear: !!document.querySelector('.skin-price-card-row dt') || !!document.querySelector('.skin-price-table thead th:nth-child(2)'),
        normal: markets,
        stattrak: []
    };
})();
