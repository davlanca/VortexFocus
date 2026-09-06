const express = require("express");
const path = require("path");

const app = express();
const PORT = Number(process.env.PORT || 3000);

app.use(express.json({ limit: "50mb" }));
app.use(express.static(__dirname));

/* =========================================================
   LOGGER SYSTEM
========================================================= */

function logInfo(tag, message) {
  const time = new Date().toLocaleTimeString("ru-RU");
  console.log(`\x1b[36m[${time}]\x1b[0m \x1b[32m[${tag}]\x1b[0m ${message}`);
}

function logWarn(tag, message) {
  const time = new Date().toLocaleTimeString("ru-RU");
  console.log(`\x1b[36m[${time}]\x1b[0m \x1b[33m[${tag}]\x1b[0m ${message}`);
}

function logError(tag, message) {
  const time = new Date().toLocaleTimeString("ru-RU");
  console.log(`\x1b[36m[${time}]\x1b[0m \x1b[31m[${tag}]\x1b[0m ${message}`);
}

/* =========================================================
   7 MARKETS & NATIVE CURRENCIES & FEES
========================================================= */

const MARKETS = [
  "MARKET.CSGO",
  "LIS-SKINS",
  "AVAN.MARKET",
  "STEAM",
  "LOOT.FARM",
  "CS.MONEY",
  "BUFF.163"
];

const MARKET_CURRENCIES = {
  "MARKET.CSGO": "RUB",
  "LIS-SKINS": "RUB",
  "AVAN.MARKET": "RUB",
  "STEAM": "RUB",
  "LOOT.FARM": "USD",
  "CS.MONEY": "USD",
  "BUFF.163": "USD"
};

const DEFAULT_FEES = {
  "MARKET.CSGO": { sell: 5.0, buy: 0, deposit: 0 },
  "LIS-SKINS": { sell: 0.0, buy: 0, deposit: 0 },
  "AVAN.MARKET": { sell: 0.0, buy: 0, deposit: 0 },
  "STEAM": { sell: 13.04, buy: 0, deposit: 0 },
  "LOOT.FARM": { sell: 5.0, buy: 0, deposit: 0 },
  "CS.MONEY": { sell: 7.0, buy: 0, deposit: 0 },
  "BUFF.163": { sell: 2.5, buy: 0, deposit: 0 }
};

let fees = JSON.parse(JSON.stringify(DEFAULT_FEES));

const BROWSER_HEADERS = {
  "User-Agent":
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
  "Accept": "application/json, text/plain, */*",
  "Accept-Language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
  "Cache-Control": "no-cache"
};
const PRICEMPIRE_API_KEY = process.env.PRICEMPIRE_API_KEY || "";

/* =========================================================
   CENTRAL PRICE DATABASE & DIAGNOSTICS LOGS
========================================================= */

// Map: market_hash_name -> { isLiquid: boolean, prices: { [market]: { value, currency, source } } }
const priceDatabase = new Map();
const endpointDiagnostics = {};
let refreshPromise = null;

/* =========================================================
   DIAGNOSTIC NETWORK INSPECTOR
========================================================= */

async function inspectedFetch(endpointName, url, options = {}) {
  const startTime = Date.now();
  const safeUrl = url.replace(/([?&]api_key=)[^&]+/i, "$1REDACTED");
  logInfo(`API REQ [${endpointName}]`, `GET ${safeUrl}`);

  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        ...BROWSER_HEADERS,
        ...(options.headers || {})
      }
    });

    const duration = Date.now() - startTime;
    const contentType = response.headers.get("content-type") || "unknown";
    const contentLength = response.headers.get("content-length") || "unknown";
    const rawText = await response.text();

    let parsedJson = null;
    let isJson = false;
    try {
      parsedJson = JSON.parse(rawText);
      isJson = true;
    } catch {
      isJson = false;
    }

    logInfo(
      `API RESP [${endpointName}]`,
      `HTTP ${response.status} (${duration}ms) | Content-Type: ${contentType} | Size: ${rawText.length} bytes`
    );

    endpointDiagnostics[endpointName] = {
      endpointName,
      url: safeUrl,
      status: response.status,
      statusText: response.statusText,
      duration: `${duration}ms`,
      contentType,
      contentLength,
      isJson,
      rawSnippet: rawText.slice(0, 400),
      timestamp: new Date().toISOString(),
      ok: response.ok
    };

    if (!response.ok) {
      logWarn(`API WARN [${endpointName}]`, `Код ${response.status}: ${rawText.slice(0, 200)}`);
      throw new Error(`HTTP ${response.status}: ${rawText.slice(0, 150)}`);
    }

    if (!isJson) {
      logError(`API ERR [${endpointName}]`, `Ответ не JSON: ${rawText.slice(0, 200)}`);
      throw new Error(`Сервер вернул не JSON (Content-Type: ${contentType})`);
    }

    return {
      status: response.status,
      duration,
      data: parsedJson
    };
  } catch (error) {
    const duration = Date.now() - startTime;
    logError(`API FAIL [${endpointName}]`, `${error.message} (${duration}ms)`);

    endpointDiagnostics[endpointName] = {
      endpointName,
      url: safeUrl,
      error: error.message,
      duration: `${duration}ms`,
      timestamp: new Date().toISOString(),
      ok: false
    };

    throw error;
  }
}

/* =========================================================
   TOP LIQUID CS2 CATALOG
========================================================= */

const LIQUID_CATALOG = [
  { name: "AK-47 | Redline (Field-Tested)", market_rub: 1650, lis_rub: 1520, avan_rub: 1510, steam_rub: 2180, lootfarm_usd: 16.80, csmoney_usd: 17.50, buff_usd: 16.90 },
  { name: "AK-47 | Slate (Field-Tested)", market_rub: 330, lis_rub: 300, avan_rub: 295, steam_rub: 435, lootfarm_usd: 3.30, csmoney_usd: 3.45, buff_usd: 3.30 },
  { name: "AK-47 | Asiimov (Field-Tested)", market_rub: 2520, lis_rub: 2310, avan_rub: 2290, steam_rub: 3350, lootfarm_usd: 25.20, csmoney_usd: 26.10, buff_usd: 25.50 },
  { name: "AK-47 | Ice Coaled (Factory New)", market_rub: 1540, lis_rub: 1420, avan_rub: 1400, steam_rub: 2050, lootfarm_usd: 15.50, csmoney_usd: 16.10, buff_usd: 15.70 },
  { name: "AK-47 | Vulcan (Field-Tested)", market_rub: 13800, lis_rub: 12600, avan_rub: 12500, steam_rub: 18000, lootfarm_usd: 136.00, csmoney_usd: 141.00, buff_usd: 138.00 },
  { name: "AWP | Asiimov (Field-Tested)", market_rub: 8750, lis_rub: 8050, avan_rub: 8000, steam_rub: 11500, lootfarm_usd: 87.00, csmoney_usd: 90.50, buff_usd: 88.00 },
  { name: "AWP | Neo-Noir (Factory New)", market_rub: 2840, lis_rub: 2600, avan_rub: 2580, steam_rub: 3750, lootfarm_usd: 28.20, csmoney_usd: 29.30, buff_usd: 28.50 },
  { name: "AWP | Atheris (Field-Tested)", market_rub: 230, lis_rub: 210, avan_rub: 205, steam_rub: 305, lootfarm_usd: 2.25, csmoney_usd: 2.40, buff_usd: 2.30 },
  { name: "M4A1-S | Printstream (Field-Tested)", market_rub: 10600, lis_rub: 9650, avan_rub: 9600, steam_rub: 14000, lootfarm_usd: 105.00, csmoney_usd: 109.50, buff_usd: 106.00 },
  { name: "M4A1-S | Decimator (Field-Tested)", market_rub: 1310, lis_rub: 1200, avan_rub: 1190, steam_rub: 1750, lootfarm_usd: 13.10, csmoney_usd: 13.70, buff_usd: 13.30 },
  { name: "M4A1-S | Hyper Beast (Field-Tested)", market_rub: 1960, lis_rub: 1790, avan_rub: 1770, steam_rub: 2600, lootfarm_usd: 19.50, csmoney_usd: 20.40, buff_usd: 19.80 },
  { name: "M4A4 | The Emperor (Field-Tested)", market_rub: 1130, lis_rub: 1030, avan_rub: 1020, steam_rub: 1500, lootfarm_usd: 11.20, csmoney_usd: 11.80, buff_usd: 11.40 },
  { name: "USP-S | Printstream (Field-Tested)", market_rub: 3550, lis_rub: 3250, avan_rub: 3220, steam_rub: 4700, lootfarm_usd: 35.20, csmoney_usd: 36.80, buff_usd: 35.80 },
  { name: "USP-S | Cortex (Factory New)", market_rub: 755, lis_rub: 690, avan_rub: 680, steam_rub: 995, lootfarm_usd: 7.45, csmoney_usd: 7.85, buff_usd: 7.60 },
  { name: "USP-S | Kill Confirmed (Field-Tested)", market_rub: 4650, lis_rub: 4250, avan_rub: 4220, steam_rub: 6150, lootfarm_usd: 46.20, csmoney_usd: 48.20, buff_usd: 46.80 },
  { name: "Desert Eagle | Printstream (Field-Tested)", market_rub: 3300, lis_rub: 3020, avan_rub: 2990, steam_rub: 4350, lootfarm_usd: 32.80, csmoney_usd: 34.20, buff_usd: 33.20 },
  { name: "★ Karambit | Doppler (Factory New)", market_rub: 81000, lis_rub: 74000, avan_rub: 73500, steam_rub: 106000, lootfarm_usd: 800.00, csmoney_usd: 840.00, buff_usd: 810.00 },
  { name: "★ Butterfly Knife | Doppler (Factory New)", market_rub: 139000, lis_rub: 127000, avan_rub: 126000, steam_rub: 182000, lootfarm_usd: 1370.00, csmoney_usd: 1430.00, buff_usd: 1390.00 },
  { name: "Recoil Case", market_rub: 25.0, lis_rub: 22.5, avan_rub: 22.0, steam_rub: 33.0, lootfarm_usd: 0.25, csmoney_usd: 0.26, buff_usd: 0.25 },
  { name: "Revolution Case", market_rub: 30.0, lis_rub: 27.0, avan_rub: 26.5, steam_rub: 40.0, lootfarm_usd: 0.30, csmoney_usd: 0.31, buff_usd: 0.30 },
  { name: "Dreams & Nightmares Case", market_rub: 85.0, lis_rub: 76.5, avan_rub: 75.5, steam_rub: 112.0, lootfarm_usd: 0.84, csmoney_usd: 0.88, buff_usd: 0.85 },
  { name: "Fracture Case", market_rub: 34.5, lis_rub: 31.0, avan_rub: 30.5, steam_rub: 45.5, lootfarm_usd: 0.34, csmoney_usd: 0.36, buff_usd: 0.35 },
  { name: "Clutch Case", market_rub: 63.0, lis_rub: 57.0, avan_rub: 56.0, steam_rub: 83.5, lootfarm_usd: 0.62, csmoney_usd: 0.65, buff_usd: 0.63 }
];

function initDatabase() {
  LIQUID_CATALOG.forEach(item => {
    priceDatabase.set(item.name, {
      isLiquid: true,
      prices: {
        "MARKET.CSGO": { value: item.market_rub, currency: "RUB", source: "Market.CSGO Резерв", isLive: false },
        "LIS-SKINS": { value: item.lis_rub, currency: "RUB", source: "Lis-Skins Резерв", isLive: false },
        "AVAN.MARKET": { value: item.avan_rub, currency: "RUB", source: "Avan.Market Резерв", isLive: false },
        "STEAM": { value: item.steam_rub, currency: "RUB", source: "Steam Резерв", isLive: false },
        "LOOT.FARM": { value: item.lootfarm_usd, currency: "USD", source: "Loot.Farm Резерв", isLive: false },
        "CS.MONEY": { value: item.csmoney_usd, currency: "USD", source: "CS.Money Резерв", isLive: false },
        "BUFF.163": { value: item.buff_usd, currency: "USD", source: "Buff.163 Резерв", isLive: false }
      }
    });
  });
  logInfo("INIT", `База инициализирована (${priceDatabase.size} ликвидных скинов для 7 магазинов)`);
}

/* =========================================================
   PARSERS
========================================================= */

// 1. MARKET.CSGO (Рублевый дамп)
async function parseMarketCSGO() {
  const url = "https://market.csgo.com/api/v2/prices/RUB.json";
  try {
    const res = await inspectedFetch("MARKET.CSGO", url, { signal: AbortSignal.timeout(12000) });
    let count = 0;

    if (res.data?.items && Array.isArray(res.data.items)) {
      res.data.items.forEach(item => {
        const name = item.market_hash_name;
        const rub = Number(item.price);
        if (name && rub > 0.5) {
          const entry = priceDatabase.get(name) || { isLiquid: false, prices: {} };
          entry.prices["MARKET.CSGO"] = {
            value: rub,
            currency: "RUB",
            source: "Market.CSGO Live API",
            isLive: true,
            fetchedAt: new Date().toISOString()
          };
          priceDatabase.set(name, entry);
          count++;
        }
      });
    }
    logInfo("PARSER: MARKET.CSGO", `Успешно спарсено ${count} цен в рублях`);
    return true;
  } catch (err) {
    return false;
  }
}

// 2. LOOT.FARM (USD центы дамп)
async function parseLootFarm() {
  const url = "https://loot.farm/fullprice.json";
  try {
    const res = await inspectedFetch("LOOT.FARM", url, { signal: AbortSignal.timeout(12000) });
    let count = 0;

    if (Array.isArray(res.data)) {
      res.data.forEach(item => {
        const name = item.name;
        const usd = Number((item.price / 100).toFixed(2));
        if (name && usd > 0.05) {
          const entry = priceDatabase.get(name) || { isLiquid: false, prices: {} };
          entry.prices["LOOT.FARM"] = {
            value: usd,
            currency: "USD",
            source: "Loot.Farm Live API",
            isLive: true,
            fetchedAt: new Date().toISOString()
          };
          priceDatabase.set(name, entry);
          count++;
        }
      });
    }
    logInfo("PARSER: LOOT.FARM", `Успешно спарсено ${count} цен в USD`);
    return true;
  } catch (err) {
    return false;
  }
}

// 3. PRICEEMPIRE (Buff.163 и skins.com)
async function parsePriceEmpire() {
  if (!PRICEMPIRE_API_KEY) {
    logWarn("PARSER: PRICEEMPIRE", "PRICEMPIRE_API_KEY не задан, импорт пропущен");
    return false;
  }

  const query = new URLSearchParams({
    api_key: PRICEMPIRE_API_KEY,
    app_id: "730",
    sources: "buff163,skins",
    currency: "USD"
  });
  const url = `https://api.pricempire.com/v4/trader/items/prices?${query}`;

  try {
    const res = await inspectedFetch("PRICEEMPIRE", url, { signal: AbortSignal.timeout(30000) });
    let count = 0;
    let skipped = 0;

    if (Array.isArray(res.data)) {
      res.data.forEach(item => {
        const name = item.market_hash_name;
        const buffPrice = item.prices?.find(price => price.provider_key === "buff163");
        const usd = Number(buffPrice?.price) / 100;

        if (!name || !Number.isFinite(usd) || usd <= 0) {
          skipped++;
          return;
        }

        const entry = priceDatabase.get(name) || { isLiquid: false, prices: {} };
        entry.prices["BUFF.163"] = {
          value: usd,
          currency: "USD",
          source: "Pricempire / Buff.163 Live API",
          isLive: true,
          fetchedAt: buffPrice.updated_at || new Date().toISOString()
        };
        priceDatabase.set(name, entry);
        count++;
      });
    }

    logInfo("PARSER: PRICEEMPIRE", `Buff.163: ${count} цен, пропущено: ${skipped}; skins.com оставлен отдельным источником`);
    return true;
  } catch (err) {
    return false;
  }
}

// 4. STEAM (Точечный запрос в RUB: currency=5)
async function fetchSteamLivePrice(marketHashName) {
  const url = `https://steamcommunity.com/market/priceoverview/?appid=730&currency=5&market_hash_name=${encodeURIComponent(
    marketHashName
  )}`;
  try {
    const res = await inspectedFetch("STEAM", url, { signal: AbortSignal.timeout(5000) });
    if (res.data?.lowest_price) {
      const clean = res.data.lowest_price
        .replace(/\s/g, "")
        .replace("pуб.", "")
        .replace("руб.", "")
        .replace(",", ".");
      const numRub = Number(clean);
      if (numRub > 0) {
        logInfo("STEAM LIVE", `"${marketHashName}" = ${numRub} ₽`);
        return {
          value: numRub,
          currency: "RUB",
          source: "Steam Live API (Обычный)",
          isLive: true,
          fetchedAt: new Date().toISOString()
        };
      }
    }
    return null;
  } catch (err) {
    return null;
  }
}

// 5. CS.MONEY (Точечный запрос в USD)
async function fetchCSMoneyLivePrice(marketHashName) {
  const url = `https://cs.money/1.0/market/sell-orders?limit=1&name=${encodeURIComponent(
    marketHashName
  )}`;
  try {
    const res = await inspectedFetch("CS.MONEY", url, { signal: AbortSignal.timeout(5000) });
    if (res.data?.items && res.data.items[0]?.pricing?.computed) {
      const numUsd = Number(res.data.items[0].pricing.computed.toFixed(2));
      logInfo("CS.MONEY LIVE", `"${marketHashName}" = $${numUsd}`);
      return {
        value: numUsd,
        currency: "USD",
        source: "CS.Money Live API",
        isLive: true,
        fetchedAt: new Date().toISOString()
      };
    }
    return null;
  } catch (err) {
    return null;
  }
}

async function masterInit() {
  logInfo("INIT", "=== Старт фонового сбора данных по 7 магазинам ===");
  for (const entry of priceDatabase.values()) {
    for (const market of ["MARKET.CSGO", "LOOT.FARM", "BUFF.163"]) {
      if (entry.prices[market]?.isLive) {
        delete entry.prices[market];
      }
    }
  }
  await Promise.allSettled([
    parseMarketCSGO(),
    parseLootFarm(),
    parsePriceEmpire()
  ]);
  logInfo("INIT COMPLETE", `Всего скинов в базе: ${priceDatabase.size}`);
}

function runRefresh() {
  if (!refreshPromise) {
    refreshPromise = masterInit().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

/* =========================================================
   TRADE CALCULATOR
========================================================= */

function getFee(market, type) {
  const data = fees[market] || { sell: 0, buy: 0, deposit: 0 };
  return Math.max(0, Number(data[type]) || 0);
}

function calculateTrade({ buyPrice, sellPrice, buyCurrency, sellCurrency, buyMarket, sellMarket }) {
  const buyFee = getFee(buyMarket, "buy");
  const depositFee = getFee(buyMarket, "deposit");
  const sellFee = getFee(sellMarket, "sell");

  const buyFeeVal = buyPrice * (buyFee / 100);
  const depositFeeVal = buyPrice * (depositFee / 100);
  const buyTotal = buyPrice + buyFeeVal + depositFeeVal;

  const sellFeeVal = sellPrice * (sellFee / 100);
  const sellNet = sellPrice - sellFeeVal;

  let profitVal = sellNet - buyTotal;
  let profitPercent = 0;

  if (buyCurrency === sellCurrency) {
    profitPercent = buyTotal > 0 ? (profitVal / buyTotal) * 100 : 0;
  } else {
    // Внутренний срез кросс-курса для расчета процентов
    const rubToUsd = buyCurrency === "RUB" ? 1 / 95 : 95;
    const normBuy = buyCurrency === "RUB" ? buyTotal * rubToUsd : buyTotal;
    const normSell = sellCurrency === "RUB" ? sellNet * (1 / 95) : sellNet;
    profitVal = sellNet - (buyCurrency === "RUB" && sellCurrency === "USD" ? buyTotal / 95 : buyTotal * 95);
    profitPercent = normBuy > 0 ? ((normSell - normBuy) / normBuy) * 100 : 0;
  }

  return {
    buyPrice,
    buyCurrency,
    buyFeeVal,
    depositFeeVal,
    buyTotal,
    sellPrice,
    sellCurrency,
    sellFeeVal,
    sellNet,
    profitVal,
    profitPercent
  };
}

/* =========================================================
   API ROUTES
========================================================= */

app.get("/api/config", (req, res) => {
  res.json({
    success: true,
    markets: MARKETS,
    currencies: MARKET_CURRENCIES,
    fees,
    totalItems: priceDatabase.size
  });
});

app.get("/api/diagnostic", (req, res) => {
  res.json({
    success: true,
    diagnostics: endpointDiagnostics,
    totalItems: priceDatabase.size,
    sampleSkins: [...priceDatabase.entries()].slice(0, 5).map(([name, data]) => ({
      name,
      prices: data.prices
    }))
  });
});

app.post("/api/refresh", async (req, res) => {
  try {
    await runRefresh();
    res.json({ success: true, itemsCount: priceDatabase.size });
  } catch (err) {
    res.status(500).json({ success: false, error: err.message });
  }
});

app.post("/api/fees", (req, res) => {
  try {
    const incoming = req.body?.fees;
    if (!incoming || typeof incoming !== "object") {
      return res.status(400).json({ success: false, error: "Invalid fees object." });
    }

    for (const market of MARKETS) {
      if (!incoming[market]) continue;
      fees[market] = {
        sell: Math.max(0, Number(incoming[market].sell) || 0),
        buy: Math.max(0, Number(incoming[market].buy) || 0),
        deposit: Math.max(0, Number(incoming[market].deposit) || 0)
      };
    }
    res.json({ success: true, fees });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

app.post("/api/arbitrage", async (req, res) => {
  const scanStart = Date.now();
  try {
    if (refreshPromise) {
      await refreshPromise;
    }
    const body = req.body || {};
    const buyMarket = String(body.buyMarket || "").trim();
    const sellMarket = String(body.sellMarket || "").trim();
    const onlyLiquid = Boolean(body.onlyLiquid);

    const minProfit =
      body.minProfit === undefined || body.minProfit === null || body.minProfit === ""
        ? -Infinity
        : Number(body.minProfit);

    const maxProfit =
      body.maxProfit === undefined || body.maxProfit === null || body.maxProfit === ""
        ? Infinity
        : Number(body.maxProfit);

    const search = String(body.search || "").trim().toLowerCase();
    const sortBy = body.sortBy || "profitPercent";
    const sortDirection = body.sortDirection === "asc" ? 1 : -1;

    logInfo("SCAN", `[${buyMarket}] ➔ [${sellMarket}] | Поиск: "${search || 'ВСЕ'}" | Ликвид: ${onlyLiquid}`);

    if (!buyMarket || !sellMarket) {
      return res.status(400).json({ success: false, error: "Выберите площадки покупки и продажи." });
    }

    if (buyMarket === sellMarket) {
      return res.status(400).json({ success: false, error: "Площадки покупки и продажи должны отличаться." });
    }

    // Живой опрос Steam / CSMoney при поиске
    if (search && search.length >= 3) {
      const matchKey = [...priceDatabase.keys()].find(k => k.toLowerCase().includes(search));
      const skinNameToQuery = matchKey || search;

      const [steamLive, csmoneyLive] = await Promise.all([
        buyMarket === "STEAM" || sellMarket === "STEAM" ? fetchSteamLivePrice(skinNameToQuery) : null,
        buyMarket === "CS.MONEY" || sellMarket === "CS.MONEY" ? fetchCSMoneyLivePrice(skinNameToQuery) : null
      ]);

      const entry = priceDatabase.get(skinNameToQuery) || { isLiquid: true, prices: {} };
      if (steamLive) entry.prices["STEAM"] = steamLive;
      if (csmoneyLive) entry.prices["CS.MONEY"] = csmoneyLive;
      priceDatabase.set(skinNameToQuery, entry);
    }

    const results = [];
    let matchedBothCount = 0;

    for (const [skinName, itemData] of priceDatabase.entries()) {
      if (search && !skinName.toLowerCase().includes(search)) {
        continue;
      }

      if (onlyLiquid && !itemData.isLiquid) {
        continue;
      }

      const buyEntry = itemData.prices ? itemData.prices[buyMarket] : null;
      const sellEntry = itemData.prices ? itemData.prices[sellMarket] : null;

      if (
        !buyEntry ||
        !sellEntry ||
        buyEntry.isLive !== true ||
        sellEntry.isLive !== true ||
        buyEntry.value <= 0 ||
        sellEntry.value <= 0
      ) {
        continue;
      }

      matchedBothCount++;

      const calc = calculateTrade({
        buyPrice: buyEntry.value,
        buyCurrency: buyEntry.currency,
        sellPrice: sellEntry.value,
        sellCurrency: sellEntry.currency,
        buyMarket,
        sellMarket
      });

      if (calc.profitPercent < minProfit || calc.profitPercent > maxProfit) {
        continue;
      }

      results.push({
        market_name: skinName,
        buyMarket,
        sellMarket,
        buySource: buyEntry.source,
        sellSource: sellEntry.source,
        isLiquid: Boolean(itemData.isLiquid),
        ...calc
      });
    }

    results.sort((a, b) => {
      let av, bv;
      switch (sortBy) {
        case "buyPrice":
          av = a.buyTotal;
          bv = b.buyTotal;
          break;
        case "sellPrice":
          av = a.sellPrice;
          bv = b.sellPrice;
          break;
        case "profitVal":
          av = a.profitVal;
          bv = b.profitVal;
          break;
        default:
          av = a.profitPercent;
          bv = b.profitPercent;
      }
      return (av - bv) * sortDirection;
    });

    const totalTime = Date.now() - scanStart;
    logInfo("SCAN SUCCESS", `Связок найдено: ${results.length} (${totalTime}ms)`);

    res.json({
      success: true,
      count: results.length,
      matchedBothCount,
      totalItems: priceDatabase.size,
      buyMarket,
      sellMarket,
      results
    });
  } catch (error) {
    logError("SCAN FAIL", `${error.message} (${Date.now() - scanStart}ms)`);
    res.status(500).json({ success: false, error: error.message, results: [] });
  }
});

/* =========================================================
   FALLBACK & SERVER START
========================================================= */

app.use((req, res) => {
  res.sendFile(path.join(__dirname, "index.html"));
});

initDatabase();

app.listen(PORT, () => {
  console.log("");
  console.log("\x1b[36m========================================\x1b[0m");
  console.log("\x1b[32m CS2 7-MARKET ARBITRAGE SCANNER (NATIVE CURRENCY)\x1b[0m");
  console.log("\x1b[36m========================================\x1b[0m");
  console.log(` \x1b[33mhttp://localhost:${PORT}\x1b[0m`);
  console.log(` \x1b[35mДиагностика 7 API: http://localhost:${PORT}/api/diagnostic\x1b[0m`);
  console.log("");
  runRefresh();
});