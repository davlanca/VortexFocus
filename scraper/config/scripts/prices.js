(function () {
'use strict';

// ============================================================
// NUMBER PARSER
// ============================================================

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

    // Example:
    // 1,234.56  -> 1234.56
    // 1.234,56  -> 1234.56
    if (lastComma >= 0 && lastDot >= 0) {
        if (lastComma > lastDot) {
            cleaned = cleaned
                .replace(/\./g, '')
                .replace(',', '.');
        } else {
            cleaned = cleaned.replace(/,/g, '');
        }
    }

    // Example:
    // 1234,56 -> 1234.56
    // 1,234   -> 1234
    else if (lastComma >= 0) {
        if (/,\d{1,2}$/.test(cleaned)) {
            cleaned = cleaned.replace(',', '.');
        } else {
            cleaned = cleaned.replace(/,/g, '');
        }
    }

    const value = parseFloat(cleaned);

    return Number.isFinite(value) ? value : NaN;
}

// ============================================================
// CURRENCY
// ============================================================

function detectCurrency(text) {
    if (!text) {
        return 'USD';
    }

    const value = String(text).toLowerCase();

    if (value.includes('€')) {
        return 'EUR';
    }

    if (value.includes('£')) {
        return 'GBP';
    }

    if (value.includes('₽')) {
        return 'RUB';
    }

    if (
        value.includes('¥') ||
        value.includes('cny') ||
        value.includes('rmb')
    ) {
        return 'CNY';
    }

    if (
        value.includes('$') ||
        value.includes('usd')
    ) {
        return 'USD';
    }

    return 'USD';
}

// ============================================================
// WEAR
// ============================================================

const wearCodes = {
    'factory new': 'FN',
    'factory-new': 'FN',

    'minimal wear': 'MW',
    'minimal-wear': 'MW',

    'field-tested': 'FT',
    'field tested': 'FT',

    'well-worn': 'WW',
    'well worn': 'WW',

    'battle-scarred': 'BS',
    'battle scarred': 'BS'
};

const wearOrder = [
    'FN',
    'MW',
    'FT',
    'WW',
    'BS'
];

// ============================================================
// TEXT HELPERS
// ============================================================

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

// ============================================================
// PRICE CELL
// ============================================================

function getPriceFromCell(cell) {
    if (!cell) {
        return null;
    }

    const text = getCellText(cell);

    if (!text) {
        return null;
    }

    // Ignore unavailable listings.
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

// ============================================================
// MARKET NAME
// ============================================================

function getMarketName(cell) {
    if (!cell) {
        return '';
    }

    // Prefer image alt because marketplace cells often
    // contain logos and additional visual elements.
    const image = cell.querySelector('img[alt]');

    if (image) {
        const alt = normalizeText(
            image.getAttribute('alt')
        );

        if (alt) {
            return alt
                .replace(/\s+logo$/i, '')
                .trim();
        }
    }

    // Some pages use aria-label.
    const labelled = cell.querySelector(
        '[aria-label]'
    );

    if (labelled) {
        const label = normalizeText(
            labelled.getAttribute('aria-label')
        );

        if (label) {
            return label
                .replace(/\s+logo$/i, '')
                .trim();
        }
    }

    return getCellText(cell)
        .replace(/\s+logo$/i, '')
        .trim();
}

// ============================================================
// TABLE HEADER
// ============================================================

function getHeaderMap(table) {
    const rows = Array.from(
        table.querySelectorAll('tr')
    );

    for (const row of rows) {
        const cells = Array.from(
            row.querySelectorAll('th, td')
        );

        if (cells.length < 2) {
            continue;
        }

        const headers = cells.map(cell =>
            normalizeText(
                cell.textContent
            ).toLowerCase()
        );

        const map = {};

        headers.forEach((header, index) => {
            if (wearCodes[header]) {
                map[wearCodes[header]] = index;
            }
        });

        // Current CSGO Database marketplace tables
        // normally start with "Marketplace".
        const firstHeader = headers[0] || '';

        if (
            Object.keys(map).length >= 1 &&
            (
                firstHeader === 'marketplace' ||
                firstHeader.includes('marketplace')
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

// ============================================================
// MARKETPLACE TABLE PARSER
// ============================================================

function parseMarketplaceTable(table) {
    const headerInfo = getHeaderMap(table);

    if (!headerInfo) {
        return [];
    }

    const rows = Array.from(
        table.querySelectorAll('tr')
    );

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

        // Ignore rows that are not marketplace entries.
        if (
            /^(marketplace|variant|price|wear)$/i.test(
                market
            )
        ) {
            continue;
        }

        const wearPrices = {};
        const urls = {};

        let currency = 'USD';

        for (const wear of wearOrder) {
            const index = headerInfo.map[wear];

            if (
                index === undefined ||
                !cells[index]
            ) {
                continue;
            }

            const data = getPriceFromCell(
                cells[index]
            );

            if (!data) {
                continue;
            }

            wearPrices[wear] = data.price;
            urls[wear] = data.url;

            if (currency === 'USD') {
                currency = data.currency;
            }
        }

        if (
            Object.keys(wearPrices).length === 0
        ) {
            continue;
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

// ============================================================
// DETECT TABLE TYPE
// ============================================================
//
// We do not blindly assume that table #1 is Normal and
// table #2 is StatTrak.
//
// We inspect nearby text where possible and otherwise
// fall back to the current CSGO Database order.
// ============================================================

function detectTableType(table, index) {
    // Check immediately preceding elements.
    let node = table.previousElementSibling;

    for (let i = 0; node && i < 8; i++) {
        const text = normalizeText(
            node.textContent
        ).toLowerCase();

        if (text.includes('stattrak')) {
            return 'stattrak';
        }

        if (
            text === 'normal' ||
            /normal/.test(text)
        ) {
            return 'normal';
        }

        node = node.previousElementSibling;
    }

    // Check nearby headings.
    const parent = table.parentElement;

    if (parent) {
        const headings = Array.from(
            parent.querySelectorAll(
                'h1, h2, h3, h4, h5, h6'
            )
        );

        for (const heading of headings) {
            const text = normalizeText(
                heading.textContent
            ).toLowerCase();

            if (text.includes('stattrak')) {
                return 'stattrak';
            }

            if (/normal/.test(text)) {
                return 'normal';
            }
        }
    }

    // Final fallback:
    // CSGO Database: first marketplace table = Normal,
    // second marketplace table = StatTrak.
    if (index === 0) {
        return 'normal';
    }

    if (index === 1) {
        return 'stattrak';
    }

    return '';
}

// ============================================================
// CURRENT MARKETPLACE TABLES
// ============================================================

const marketplaceTables = [];

document
    .querySelectorAll('table')
    .forEach(table => {
        const parsed = parseMarketplaceTable(
            table
        );

        if (parsed.length > 0) {
            marketplaceTables.push({
                table: table,
                data: parsed
            });
        }
    });

let normal = [];
let stattrak = [];

marketplaceTables.forEach((entry, index) => {
    const type = detectTableType(
        entry.table,
        index
    );

    if (type === 'normal') {
        if (normal.length === 0) {
            normal = entry.data;
        }
    }

    if (type === 'stattrak') {
        if (stattrak.length === 0) {
            stattrak = entry.data;
        }
    }
});

// Safety fallback in case table detection could not
// identify the type.
if (
    normal.length === 0 &&
    marketplaceTables.length >= 1
) {
    normal = marketplaceTables[0].data;
}

if (
    stattrak.length === 0 &&
    marketplaceTables.length >= 2
) {
    stattrak = marketplaceTables[1].data;
}

// ============================================================
// OLD SINGLE PRICE TABLE FALLBACK
// ============================================================

if (
    normal.length === 0 &&
    stattrak.length === 0
) {
    const singleTable =
        document.querySelector(
            '.single-price-table'
        );

    if (singleTable) {
        singleTable
            .querySelectorAll('tbody tr')
            .forEach(row => {
                const cells = Array.from(
                    row.querySelectorAll(
                        'th, td'
                    )
                );

                if (cells.length < 2) {
                    return;
                }

                const market =
                    getMarketName(cells[0]);

                if (!market) {
                    return;
                }

                const data =
                    getPriceFromCell(cells[1]);

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

// ============================================================
// OLD SKIN PRICE CARD FALLBACK
// ============================================================

if (
    normal.length === 0 &&
    stattrak.length === 0
) {
    document
        .querySelectorAll(
            '.skin-price-cards .skin-price-card'
        )
        .forEach(card => {
            let market = '';

            const screenReader =
                card.querySelector('.sr-only');

            if (screenReader) {
                market = normalizeText(
                    screenReader.textContent
                );
            }

            if (!market) {
                const logo =
                    card.querySelector(
                        'h3 img[alt]'
                    );

                if (logo) {
                    market = normalizeText(
                        logo.getAttribute('alt')
                    )
                        .replace(
                            /\s+logo$/i,
                            ''
                        )
                        .trim();
                }
            }

            if (!market) {
                return;
            }

            const wearPrices = {};
            const urls = {};

            let currency = 'USD';

            card
                .querySelectorAll(
                    '.skin-price-card-row'
                )
                .forEach(row => {
                    const wearName =
                        normalizeText(
                            row
                                .querySelector('dt')
                                ?.textContent
                        ).toLowerCase();

                    const wear =
                        wearCodes[wearName];

                    if (!wear) {
                        return;
                    }

                    const cell =
                        row.querySelector('dd') ||
                        row;

                    const data =
                        getPriceFromCell(cell);

                    if (!data) {
                        return;
                    }

                    wearPrices[wear] =
                        data.price;

                    urls[wear] =
                        data.url;

                    currency =
                        data.currency;
                });

            if (
                Object.keys(wearPrices)
                    .length > 0
            ) {
                normal.push({
                    market: market,
                    currency: currency,
                    wearPrices: wearPrices,
                    urls: urls
                });
            }
        });
}

// ============================================================
// ITEM NAME
// ============================================================

const h1 = document.querySelector('h1');

const itemName = h1
    ? normalizeText(h1.textContent)
    : normalizeText(
        document.title
            .split(' - ')[0]
    );

// ============================================================
// HAS WEAR
// ============================================================

const hasWear =
    normal.some(item =>
        item.wearPrices &&
        Object.keys(
            item.wearPrices
        ).length > 0
    ) ||
    stattrak.some(item =>
        item.wearPrices &&
        Object.keys(
            item.wearPrices
        ).length > 0
    );

// ============================================================
// RESULT
// ============================================================

return {
    itemName: itemName,
    hasWear: hasWear,
    normal: normal,
    stattrak: stattrak
};

})();
