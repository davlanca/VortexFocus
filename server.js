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
   MARKETS & FEES
========================================================= */

const DEFAULT_MARKETS = [
  "MARKET.CSGO",
  "LIS-SKINS",
  "AVAN.MARKET",
  "STEAM",
  "LOOT.FARM",
  "CS.MONEY",
  "BUFF.163"
];

let MARKETS = [...DEFAULT_MARKETS];

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

function normalizeMarketName(name) {
  const rawMarket = String(name || "").trim();
  const aliases = {
    "MARKET.CSGO": "Market.CSGO",
    "STEAM MARKET": "Steam",
    "STEAM": "Steam",
    "SKINPORT.COM": "Skinport",
    "CS.MONEY": "CS.MONEY",
    "LIS-SKINS": "LIS-SKINS",
    "BUFF.163": "BUFF.163"
  };
  return aliases[rawMarket.toUpperCase()] || rawMarket;
}

const BROWSER_HEADERS = {
  "User-Agent":
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
  "Accept": "application/json, text/plain, */*",
  "Accept-Language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
  "Cache-Control": "no-cache"
};
const GITHUB_JSON_API = "https://api.github.com/repos/davlanca/VortexFocus/contents/json";

/* =========================================================
   CENTRAL PRICE DATABASE & DIAGNOSTICS LOGS
========================================================= */

// Map: market_hash_name -> { isLiquid: boolean, prices: { [market]: { value, source } } }
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
   LIQUID CATALOG
========================================================= */

const LIQUID_CATALOG = [
  { name: "AK-47 | Redline (Field-Tested)", usd: 16.80 },
  { name: "AK-47 | Slate (Field-Tested)", usd: 3.30 },
  { name: "AK-47 | Asiimov (Field-Tested)", usd: 25.20 },
  { name: "AK-47 | Ice Coaled (Factory New)", usd: 15.50 },
  { name: "AK-47 | Vulcan (Field-Tested)", usd: 136.00 },
  { name: "AWP | Asiimov (Field-Tested)", usd: 87.00 },
  { name: "AWP | Neo-Noir (Factory New)", usd: 28.20 },
  { name: "AWP | Atheris (Field-Tested)", usd: 2.25 },
  { name: "M4A1-S | Printstream (Field-Tested)", usd: 105.00 },
  { name: "M4A1-S | Decimator (Field-Tested)", usd: 13.10 },
  { name: "M4A1-S | Hyper Beast (Field-Tested)", usd: 19.50 },
  { name: "M4A4 | The Emperor (Field-Tested)", usd: 11.20 },
  { name: "USP-S | Printstream (Field-Tested)", usd: 35.20 },
  { name: "USP-S | Cortex (Factory New)", usd: 7.45 },
  { name: "USP-S | Kill Confirmed (Field-Tested)", usd: 46.20 },
  { name: "Desert Eagle | Printstream (Field-Tested)", usd: 32.80 },
  { name: "★ Karambit | Doppler (Factory New)", usd: 800.00 },
  { name: "★ Butterfly Knife | Doppler (Factory New)", usd: 1370.00 },
  { name: "Recoil Case", usd: 0.25 },
  { name: "Revolution Case", usd: 0.30 },
  { name: "Dreams & Nightmares Case", usd: 0.84 },
  { name: "Fracture Case", usd: 0.34 },
  { name: "Clutch Case", usd: 0.62 }
];

function initDatabase() {
  LIQUID_CATALOG.forEach(item => {
    priceDatabase.set(item.name, {
      isLiquid: true,
      prices: {}
    });
  });
  logInfo("INIT", `База инициализирована (${priceDatabase.size} ликвидных скинов)`);
}

function applyScraperJsonItem(item, source, updatedAt) {
  const weapon = String(item.weapon || "").trim();
  const name = String(item.name || "").trim();
  const prices = Array.isArray(item.prices) ? item.prices : [];
  if (!name || !prices.length) {
    return false;
  }

  const wearNames = {
    FN: "Factory New",
    MW: "Minimal Wear",
    FT: "Field-Tested",
    WW: "Well-Worn",
    BS: "Battle-Scarred"
  };
  const quality = item.type === "StatTrak" ? "StatTrak™ " : "";
  const baseName = name.includes("|") ? name : `${weapon} | ${name}`;

  prices.forEach(priceItem => {
    const value = Number(priceItem.price);
    if (!priceItem.has_price || !Number.isFinite(value) || value <= 0) return;

    const market = normalizeMarketName(priceItem.market);
    if (!market) return;
    const wear = wearNames[priceItem.wear] || priceItem.wear || "";
    const marketHashName = `${quality}${baseName}${wear ? ` (${wear})` : ""}`;
    const entry = priceDatabase.get(marketHashName) || { isLiquid: false, prices: {} };
    entry.prices[market] = {
      value: Number(value.toFixed(2)),
      currency: priceItem.currency || "USD",
      source: `${source} (${item.type || "Normal"})`,
      url: priceItem.url || "",
      isLive: true,
      fetchedAt: updatedAt || new Date().toISOString()
    };
    priceDatabase.set(marketHashName, entry);
  });

  return prices.some(priceItem => priceItem.has_price && Number(priceItem.price) > 0);
}

function applyLegacySteamJsonItem(item, source, updatedAt) {
  const weapon = String(item.weapon || "").trim();
  const name = String(item.name || "").trim();
  const usd = Number(item.price?.min?.value);
  if (!weapon || !name || !Number.isFinite(usd) || usd <= 0) return false;
  const marketName = `${weapon} | ${name}`;
  const entry = priceDatabase.get(marketName) || { isLiquid: false, prices: {} };
  entry.prices.STEAM = {
    value: Number(usd.toFixed(2)), currency: "USD",
    source: `${source} (минимум)`, isLive: true,
    fetchedAt: updatedAt || new Date().toISOString()
  };
  priceDatabase.set(marketName, entry);
  return true;
}

function applyGithubItem(item, source, updatedAt) {
  if (Array.isArray(item.prices) && item.prices.length) {
    return applyScraperJsonItem(item, source, updatedAt);
  }
  return applyLegacySteamJsonItem(item, source, updatedAt);
}

async function parseLatestGithubSteamJson() {
  try {
    const listing = await inspectedFetch("GITHUB.JSON.INDEX", GITHUB_JSON_API, {
      headers: { Accept: "application/vnd.github+json" },
      signal: AbortSignal.timeout(10000)
    });
    const latestName = "data.json";
    const remoteFiles = Array.isArray(listing.data) ? listing.data : [];
    if (!remoteFiles.some(file => file.name === latestName)) {
      throw new Error("Файл data.json не найден в GitHub");
    }

    const latestFile = remoteFiles.find(file => file.name === latestName);
    const dataUrl = latestFile?.download_url || `https://raw.githubusercontent.com/davlanca/VortexFocus/main/json/${latestName}`;
    const dataResponse = await inspectedFetch("GITHUB.JSON.DATA", dataUrl, {
      signal: AbortSignal.timeout(30000)
    });
    const data = dataResponse.data;

    const items = Array.isArray(data?.skins) ? data.skins : [];
    let count = 0;
    const availableMarkets = new Set();
    items.forEach(item => {
      if (applyGithubItem(item, `VortexFocus ${latestName}`, item.price?.updated_at)) {
        count++;
      }
      for (const price of item.prices || []) {
        const market = normalizeMarketName(price.market);
        if (market && price.has_price && Number(price.price) > 0) {
          availableMarkets.add(market);
        }
      }
    });
    MARKETS = [...availableMarkets];
    logInfo("PARSER: STEAM.JSON", `Файл ${latestName}: ${count} предметов в USD`);
    return true;
  } catch (error) {
    logWarn("PARSER: STEAM.JSON", `GitHub недоступен: ${error.message}`);
    return false;
  }
}

// 5. STEAM (Точечный запрос в USD: currency=1)
async function fetchSteamLivePrice(marketHashName) {
  const url = `https://steamcommunity.com/market/priceoverview/?appid=730&currency=1&market_hash_name=${encodeURIComponent(
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
      const numUsd = Number(clean);
      if (numUsd > 0) {
        logInfo("STEAM LIVE", `"${marketHashName}" = $${numUsd}`);
        return {
          value: Number(numUsd.toFixed(2)),
          currency: "USD",
          source: "Steam Live API (Обычный, отображение в USD)",
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
  logInfo("INIT", "=== Загрузка готового файла цен из GitHub ===");
  await parseLatestGithubSteamJson();
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

function calculateTrade({ buyPrice, sellPrice, buyMarket, sellMarket }) {
  const buyFee = getFee(buyMarket, "buy");
  const depositFee = getFee(buyMarket, "deposit");
  const sellFee = getFee(sellMarket, "sell");

  const buyFeeVal = buyPrice * (buyFee / 100);
  const depositFeeVal = buyPrice * (depositFee / 100);
  const buyTotal = buyPrice + buyFeeVal + depositFeeVal;

  const sellFeeVal = sellPrice * (sellFee / 100);
  const sellNet = sellPrice - sellFeeVal;

  const profitVal = sellNet - buyTotal;
  const profitPercent = buyTotal > 0 ? (profitVal / buyTotal) * 100 : 0;

  return {
    buyPrice: Number(buyPrice.toFixed(2)),
    buyCurrency: "USD",
    buyFeeVal: Number(buyFeeVal.toFixed(2)),
    depositFeeVal: Number(depositFeeVal.toFixed(2)),
    buyTotal: Number(buyTotal.toFixed(2)),
    sellPrice: Number(sellPrice.toFixed(2)),
    sellCurrency: "USD",
    sellFeeVal: Number(sellFeeVal.toFixed(2)),
    sellNet: Number(sellNet.toFixed(2)),
    profitVal: Number(profitVal.toFixed(2)),
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
    const buyMarket = normalizeMarketName(body.buyMarket);
    const sellMarket = normalizeMarketName(body.sellMarket);
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
        sellPrice: sellEntry.value,
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
        buyUrl: buyEntry.url || "",
        sellUrl: sellEntry.url || "",
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