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

    const wearCodes = { 'Factory New': 'FN', 'Minimal Wear': 'MW', 'Field-Tested': 'FT', 'Well-Worn': 'WW', 'Battle-Scarred': 'BS' };

    function marketNameFromCell(th) {
        return th.querySelector('.sr-only')?.textContent.trim()
            || th.querySelector('img')?.getAttribute('alt')?.replace(/ logo$/i, '').trim()
            || th.textContent.trim().replace(/ logo$/i, '');
    }

    // Reads one .price-pill cell. Returns null for empty cells ("No listings").
    function readPill(pill) {
        if (!pill || pill.classList.contains('price-pill--empty')) return null;
        const text = pill.textContent.trim();
        const value = parseNum(text);
        if (isNaN(value)) return null;
        return {
            value: value,
            url: pill.getAttribute('href') || '',
            currency: detectCurrency(text),
            lowest: pill.classList.contains('price-pill--lowest')
        };
    }

    const normal = [];
    const stattrak = [];
    const souvenir = [];

    // 1. Разбираем все таблицы .skin-price-table (Оружие, Перчатки).
    //    На странице до трёх таблиц, вариант определяется по caption:
    //    "… Normal skin prices by marketplace", "… StatTrak™ …", "… Souvenir …".
    //    Колонки износа берём из thead (у Souvenir только Factory New).
    document.querySelectorAll('table.skin-price-table').forEach(table => {
        const caption = table.querySelector('caption')?.textContent.toLowerCase() || '';
        let bucket = normal;
        if (caption.includes('stattrak')) bucket = stattrak;
        else if (caption.includes('souvenir')) bucket = souvenir;

        // Маппинг: индекс колонки -> код износа (первая колонка — Marketplace)
        const colWear = [...table.querySelectorAll('thead th')].slice(1)
            .map(th => wearCodes[th.textContent.trim()] || th.textContent.trim());

        table.querySelectorAll('tbody tr').forEach(row => {
            const th = row.querySelector('th.market-col') || row.querySelector('th');
            if (!th) return;
            const marketName = marketNameFromCell(th);
            if (!marketName) return;

            const wearPrices = {}, urls = {}, lowest = {};
            let currency = 'USD', any = false;

            row.querySelectorAll('td').forEach((td, i) => {
                const info = readPill(td.querySelector('.price-pill'));
                if (!info) return;
                const wear = colWear[i] || 'FN';
                wearPrices[wear] = info.value;
                urls[wear] = info.url;
                currency = info.currency;
                if (info.lowest) lowest[wear] = true;
                any = true;
            });

            if (any) {
                bucket.push({
                    market: marketName,
                    currency: currency,
                    wearPrices: wearPrices,
                    urls: urls,
                    lowest: lowest
                });
            }
        });
    });

    // 2. Одиночная таблица (Агенты, Кейсы, Терминалы, Нашивки, Значки, Наклейки, Сувенирные паки)
    if (normal.length === 0 && stattrak.length === 0 && souvenir.length === 0) {
        const singleTable = document.querySelector('.single-price-table');
        if (singleTable) {
            singleTable.querySelectorAll('tbody tr').forEach(row => {
                const th = row.querySelector('th, .market-col');
                if (!th) return;
                const marketName = marketNameFromCell(th);

                const pill = row.querySelector('.price-pill:not(.price-pill--empty)');
                if (pill) {
                    const text = pill.textContent.trim();
                    const value = parseNum(text);
                    if (!isNaN(value)) {
                        const entry = {
                            market: marketName,
                            currency: detectCurrency(text),
                            single: value,
                            url: pill.getAttribute('href') || ''
                        };
                        if (pill.classList.contains('price-pill--lowest')) {
                            entry.lowest = { single: true };
                        }
                        normal.push(entry);
                    }
                }
            });
        }
    }

    // 3. Карточки как последний fallback (дублируют Normal-таблицу)
    if (normal.length === 0 && stattrak.length === 0 && souvenir.length === 0) {
        document.querySelectorAll('.skin-price-cards .skin-price-card').forEach(card => {
            const marketName = card.querySelector('.sr-only')?.textContent.trim()
                || card.querySelector('h3 img')?.getAttribute('alt')?.replace(/ logo$/i, '').trim();
            if (!marketName) return;

            card.querySelectorAll('.skin-price-card-row').forEach(row => {
                const wearName = row.querySelector('dt')?.textContent.trim() || '';
                const wear = wearCodes[wearName] || wearName;
                const info = readPill(row.querySelector('.price-pill'));
                if (info) {
                    normal.push({
                        market: marketName,
                        currency: info.currency,
                        wearPrices: { [wear]: info.value },
                        urls: { [wear]: info.url }
                    });
                }
            });
        });
    }

    const h1 = document.querySelector('h1');
    const itemName = h1 ? h1.textContent.trim() : (document.title || '').split(' - ')[0].trim();

    return {
        itemName: itemName,
        hasWear: !!document.querySelector('.skin-price-card-row dt') || !!document.querySelector('.skin-price-table thead th:nth-child(2)'),
        normal: normal,
        stattrak: stattrak,
        souvenir: souvenir
    };
})();
