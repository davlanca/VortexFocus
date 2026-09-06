// =========================================================================
// prices.js — extract price table for a single item
// =========================================================================
// Returns JSON-encoded string with { itemName, hasWear, markets[] }.
// Reads from .skin-price-table (desktop) or .skin-price-card (mobile).

(function() {
    const WEARS = ['FN', 'MW', 'FT', 'WW', 'BS'];
    const WEAR_FULL = {
        FN: 'Factory New',
        MW: 'Minimal Wear',
        FT: 'Field-Tested',
        WW: 'Well-Worn',
        BS: 'Battle-Scarred'
    };
    const WEAR_HEADERS = Object.values(WEAR_FULL);

    function parseNum(text) {
        if (!text) return NaN;
        return parseFloat(text.replace(/[^\d.]/g, ''));
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

    const headers = Array.from(document.querySelectorAll('.skin-price-table thead th, .skin-price-table thead td'));
    const headerTexts = headers.map(h => h.textContent.trim());
    const hasWear = WEAR_HEADERS.every(w => headerTexts.some(ht => ht.includes(w)));

    const markets = [];
    const rows = document.querySelectorAll('.skin-price-table tbody tr');
    rows.forEach(row => {
        const th = row.querySelector('th, .market-col');
        if (!th) return;
        const img = th.querySelector('img');
        const alt = img ? img.getAttribute('alt') : (th.querySelector('.sr-only') ? th.querySelector('.sr-only').textContent : '');
        const marketName = (alt || '').replace(/ logo$/i, '').trim();
        if (!marketName) return;

        const cells = Array.from(row.querySelectorAll('td'));
        const entry = { market: marketName, currency: 'USD', wearPrices: {}, urls: {} };

        if (hasWear) {
            cells.forEach((cell, i) => {
                if (i >= WEARS.length) return;
                const wear = WEARS[i];
                const pill = cell.querySelector('.price-pill');
                const empty = cell.querySelector('.price-pill--empty');
                if (empty || !pill) {
                    entry.wearPrices[wear] = null;
                    entry.urls[wear] = null;
                    return;
                }
                const text = pill.textContent.trim();
                const href = pill.getAttribute('href') || '';
                entry.currency = detectCurrency(text);
                const num = parseNum(text);
                if (!isNaN(num)) {
                    entry.wearPrices[wear] = num;
                    entry.urls[wear] = href;
                } else {
                    entry.wearPrices[wear] = null;
                    entry.urls[wear] = null;
                }
            });
        } else {
            for (const cell of cells) {
                const pill = cell.querySelector('.price-pill');
                const empty = cell.querySelector('.price-pill--empty');
                if (empty || !pill) continue;
                const text = pill.textContent.trim();
                entry.currency = detectCurrency(text);
                const num = parseNum(text);
                if (!isNaN(num)) {
                    entry.single = num;
                    entry.url = pill.getAttribute('href') || '';
                    break;
                }
            }
        }
        markets.push(entry);
    });

    return JSON.stringify({
        itemName: itemName,
        hasWear: hasWear,
        markets: markets
    });
})();
