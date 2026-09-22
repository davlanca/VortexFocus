(function() {

    function parseNum(text) {
        if (!text) {
            return NaN;
        }

        let cleaned = String(text)
            .replace(/\u00a0/g, ' ')
            .replace(/[^\d.,-]/g, '')
            .trim();

        if (!cleaned) {
            return NaN;
        }

        const lastComma = cleaned.lastIndexOf(',');
        const lastDot = cleaned.lastIndexOf('.');

        if (lastComma >= 0 && lastDot >= 0) {
            if (lastComma > lastDot) {
                cleaned = cleaned
                    .replace(/\./g, '')
                    .replace(',', '.');
            } else {
                cleaned = cleaned.replace(/,/g, '');
            }
        } else if (lastComma >= 0) {
            if (/,\d{1,2}$/.test(cleaned)) {
                cleaned = cleaned.replace(',', '.');
            } else {
                cleaned = cleaned.replace(/,/g, '');
            }
        }

        const value = parseFloat(cleaned);

        return Number.isFinite(value) ? value : NaN;
    }

    function detectCurrency(text) {
        if (!text) {
            return 'USD';
        }

        const t = String(text).toLowerCase();

        if (t.includes('€')) {
            return 'EUR';
        }

        if (t.includes('£')) {
            return 'GBP';
        }

        if (t.includes('₽')) {
            return 'RUB';
        }

        if (t.includes('¥') || t.includes('cny')) {
            return 'CNY';
        }

        if (t.includes('$') || t.includes('usd')) {
            return 'USD';
        }

        return 'USD';
    }

    const wearCodes = {
        'factory new': 'FN',
        'minimal wear': 'MW',
        'field-tested': 'FT',
        'field tested': 'FT',
        'well-worn': 'WW',
        'well worn': 'WW',
        'battle-scarred': 'BS',
        'battle scarred': 'BS'
    };

    const wearOrder = ['FN', 'MW', 'FT', 'WW', 'BS'];

    function normalizeText(value) {
        return String(value || '')
            .replace(/\u00a0/g, ' ')
            .replace(/\s+/g, ' ')
            .trim();
    }

    function getCellText(cell) {
        if (!cell) {
            return '';
        }

        return normalizeText(cell.textContent);
    }

    function getPriceFromCell(cell) {
        if (!cell) {
            return null;
        }

        const text = getCellText(cell);

        if (!text) {
            return null;
        }

        /*
         * Explicitly ignore unavailable listings.
         */
        if (
            /no listings?/i.test(text) ||
            /not available/i.test(text) ||
            /^[-—–]+$/.test(text)
        ) {
            return null;
        }

        const value = parseNum(text);

        if (!Number.isFinite(value) || value <= 0) {
            return null;
        }

        const anchor = cell.querySelector('a[href]');

        return {
            price: value,
            currency: detectCurrency(text),
            url: anchor ? anchor.href : ''
        };
    }

    function getMarketName(cell) {
        if (!cell) {
            return '';
        }

        /*
         * Prefer logo alt text because it is much cleaner than
         * textContent when the cell contains images.
         */
        const img = cell.querySelector('img[alt]');

        if (img) {
            const alt = normalizeText(img.getAttribute('alt'));

            if (alt) {
                return alt
                    .replace(/\s+logo$/i, '')
                    .trim();
            }
        }

        return getCellText(cell)
            .replace(/\s+logo$/i, '')
            .trim();
    }

    function getHeaderMap(table) {
        const rows = Array.from(table.querySelectorAll('tr'));

        for (const row of rows) {
            const cells = Array.from(
                row.querySelectorAll('th, td')
            );

            if (cells.length < 2) {
                continue;
            }

            const headers = cells.map(cell =>
                normalizeText(cell.textContent).toLowerCase()
            );

            const map = {};

            headers.forEach((header, index) => {
                if (wearCodes[header]) {
                    map[wearCodes[header]] = index;
                }
            });

            /*
             * Current CSGO Database marketplace tables have:
             *
             * Marketplace
             * Factory New
             * Minimal Wear
             * Field-Tested
             * Well-Worn
             * Battle-Scarred
             */
            if (
                Object.keys(map).length >= 1 &&
                (
                    headers[0] === 'marketplace' ||
                    headers[0].includes('marketplace')
                )
            ) {
                return {
                    row: row,
                    map: map,
                    headers: headers
                };
            }
        }

        return null;
    }

    function parseMarketplaceTable(table) {
        const headerInfo = getHeaderMap(table);

        if (!headerInfo) {
            return [];
        }

        const rows = Array.from(table.querySelectorAll('tr'));

        const result = [];

        for (const row of rows) {
            if (row === headerInfo.row) {
                continue;
            }

            const cells = Array.from(
                row.querySelectorAll('td, th')
            );

            if (cells.length < 2) {
                continue;
            }

            const market = getMarketName(cells[0]);

            if (!market) {
                continue;
            }

            /*
             * Ignore rows that are clearly not marketplace rows.
             */
            if (
                /^(marketplace|variant|price|wear)$/i.test(market)
            ) {
                continue;
            }

            const wearPrices = {};
            const urls = {};

            for (const wear of wearOrder) {
                const index = headerInfo.map[wear];

                if (index === undefined || !cells[index]) {
                    continue;
                }

                const data = getPriceFromCell(cells[index]);

                if (!data) {
                    continue;
                }

                wearPrices[wear] = data.price;
                urls[wear] = data.url;
            }

            if (Object.keys(wearPrices).length === 0) {
                continue;
            }

            /*
             * Currency is normally USD on CSGO Database.
             * Use the first available price to determine it.
             */
            let currency = 'USD';

            for (const wear of wearOrder) {
                const index = headerInfo.map[wear];

                if (index === undefined || !cells[index]) {
                    continue;
                }

                const data = getPriceFromCell(cells[index]);

                if (data) {
                    currency = data.currency;
                    break;
                }
            }

            result.push({
                market: market,
                currency: currency,
                wearPrices: wearPrices,
                urls: urls
            });
        }

        return result;
    }

    /*
     * Current CSGO Database has two marketplace tables:
     *
     * 1. Normal
     * 2. StatTrak
     *
     * We identify them by their actual table headers instead
     * of relying on fragile CSS classes.
     */
    const marketplaceTables = [];

    document.querySelectorAll('table').forEach(table => {
        const parsed = parseMarketplaceTable(table);

        if (parsed.length > 0) {
            marketplaceTables.push(parsed);
        }
    });

    let normal = [];
    let stattrak = [];

    if (marketplaceTables.length >= 1) {
        normal = marketplaceTables[0];
    }

    if (marketplaceTables.length >= 2) {
        stattrak = marketplaceTables[1];
    }

    /*
     * Fallback for pages that still use the old single-price-table.
     */
    if (
        normal.length === 0 &&
        stattrak.length === 0
    ) {
        const singleTable =
            document.querySelector('.single-price-table');

        if (singleTable) {
            singleTable.querySelectorAll('tbody tr').forEach(row => {
                const cells = Array.from(
                    row.querySelectorAll('th, td')
                );

                if (cells.length < 2) {
                    return;
                }

                const market = getMarketName(cells[0]);

                if (!market) {
                    return;
                }

                const data = getPriceFromCell(cells[1]);

                if (!data) {
                    return;
                }

                normal.push({
                    market: market,
                    currency: data.currency,
                    single: data.price,
                    url: data.url
                });
            });
        }
    }

    /*
     * Another fallback for older skin-price-card pages.
     */
    if (
        normal.length === 0 &&
        stattrak.length === 0
    ) {
        document
            .querySelectorAll('.skin-price-cards .skin-price-card')
            .forEach(card => {

                const market =
                    card.querySelector('.sr-only')?.textContent?.trim() ||
                    card.querySelector('h3 img')?.getAttribute('alt')?.replace(/\s+logo$/i, '').trim();

                if (!market) {
                    return;
                }

                const wearPrices = {};
                const urls = {};
                let currency = 'USD';

                card.querySelectorAll('.skin-price-card-row').forEach(row => {
                    const wearName = normalizeText(
                        row.querySelector('dt')?.textContent
                    ).toLowerCase();

                    const wear = wearCodes[wearName];

                    if (!wear) {
                        return;
                    }

                    const cell =
                        row.querySelector('dd') ||
                        row;

                    const data = getPriceFromCell(cell);

                    if (!data) {
                        return;
                    }

                    wearPrices[wear] = data.price;
                    urls[wear] = data.url;
                    currency = data.currency;
                });

                if (Object.keys(wearPrices).length > 0) {
                    normal.push({
                        market: market,
                        currency: currency,
                        wearPrices: wearPrices,
                        urls: urls
                    });
                }
            });
    }

    const h1 = document.querySelector('h1');

    const itemName = h1
        ? normalizeText(h1.textContent)
        : normalizeText(
            document.title
                .split(' - ')[0]
        );

    const hasWear =
        normal.some(x =>
            x.wearPrices &&
            Object.keys(x.wearPrices).length > 0
        ) ||
        stattrak.some(x =>
            x.wearPrices &&
            Object.keys(x.wearPrices).length > 0
        );

    return {
        itemName: itemName,
        hasWear: hasWear,
        normal: normal,
        stattrak: stattrak
    };
})();
