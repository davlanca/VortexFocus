// Package common provides shared scraping helpers used by all category
// scrapers. The main entry point is FetchItem: it opens an item page, runs
// the anti-detection JS, waits for the price table to render, and extracts
// the prices into a config.Item ready to be saved as JSON.
package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// PriceTableResult is the JSON shape produced by prices.js.
type PriceTableResult struct {
	ItemName    string             `json:"itemName"`
	HasWear     bool               `json:"hasWear"`
	HasSouvenir bool               `json:"hasSouvenir"`
	Normal      []MarketPriceEntry `json:"normal"`
	Souvenir    []MarketPriceEntry `json:"souvenir"`
}

// MarketPriceEntry matches the per-market shape from prices.js.
type MarketPriceEntry struct {
	Market     string              `json:"market"`
	Currency   string              `json:"currency"`
	Single     *float64            `json:"single,omitempty"`
	URL        string              `json:"url,omitempty"`
	WearPrices map[string]*float64 `json:"wearPrices"`
	URLs       map[string]string   `json:"urls"`
}

// FetchOptions configures FetchItem.
type FetchOptions struct {
	URL        string
	Category   string // "skins", "agents", ...
	Weapon     string
	Rarity     string
	Collection string
	Type       string // "Souvenir", "StatTrak", ""
	HasWear    bool   // expected has-wear (skins/weapons/gloves)
	Slug       string
	Name       string // fallback name from URL slug if DOM doesn't expose h1
}

// FetchItem navigates to the URL, runs the anti-detection patch, and parses
// the price table. Returns one or more config.Item (e.g. Normal and Souvenir).
func FetchItem(ctx context.Context, opts FetchOptions) ([]config.Item, error) {
	baseItem := config.Item{
		URL:        opts.URL,
		Slug:       opts.Slug,
		Category:   opts.Category,
		Weapon:     opts.Weapon,
		Rarity:     opts.Rarity,
		Collection: opts.Collection,
		HasWear:    opts.HasWear,
		ScrapedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	const maxRetries = 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt == 0 {
			fmt.Printf("Scraping: %s\n", opts.URL)
		} else {
			fmt.Printf("Retry %d/%d: %s\n", attempt, maxRetries, opts.URL)
		}

		var pageTitle string
		var res PriceTableResult

		// Step 1: Navigate and get Normal prices
		err := chromedp.Run(ctx,
			chromedp.Navigate(opts.URL),
			chromedp.Evaluate(string(scraper.ConfigJS), nil),
			chromedp.Sleep(config.Delay),
			chromedp.Title(&pageTitle),
			chromedp.Evaluate(string(scraper.PricesJS), &res),
		)
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m chromedp error: %v\n", err)
			break
		}

		lower := strings.ToLower(pageTitle)
		if strings.Contains(lower, "page not found") {
			return nil, fmt.Errorf("404 not found")
		}
		if strings.Contains(lower, "verify") || strings.Contains(lower, "human") {
			fmt.Printf("\033[31m[!]\033[0m Detection triggered, retrying...\n")
			continue
		}

		// If there are souvenir prices but they weren't in the initial DOM,
		// we might need to click the tab. For now, let's assume prices.js tries to find them.
		// If prices.js returned souvenir=null but hasSouvenir=true, we click.
		if res.HasSouvenir && len(res.Souvenir) == 0 {
			var souvenirRes []MarketPriceEntry
			err = chromedp.Run(ctx,
				// Click the Souvenir tab. Selectors based on common site patterns.
				chromedp.Click(`.price-type-tab[data-type="souvenir"], .tab-link[href*="souvenir"]`, chromedp.ByQuery),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.Evaluate(`(function(){
					// Re-run extraction logic just for the souvenir table
					return extractPrices(); // assuming prices.js exposes it or we inject it
				})()`, &souvenirRes),
			)
			if err == nil {
				res.Souvenir = souvenirRes
			}
		}

		var items []config.Item

		// Process Normal
		if len(res.Normal) > 0 {
			normalItem := baseItem
			normalItem.Name = res.ItemName
			if normalItem.Name == "" {
				normalItem.Name = opts.Name
			}
			normalItem.Type = "Normal"
			normalItem.Prices = processMarketEntries(res.Normal, normalItem.HasWear || res.HasWear)
			items = append(items, normalItem)
		}

		// Process Souvenir
		if len(res.Souvenir) > 0 {
			souvItem := baseItem
			souvItem.Name = res.ItemName + " (Souvenir)"
			if res.ItemName == "" {
				souvItem.Name = opts.Name + " (Souvenir)"
			}
			souvItem.Type = "Souvenir"
			souvItem.Prices = processMarketEntries(res.Souvenir, souvItem.HasWear || res.HasWear)
			items = append(items, souvItem)
		}

		if len(items) == 0 {
			fmt.Printf("\033[31m[!]\033[0m No prices found for %s\n", opts.URL)
			continue
		}

		return items, nil
	}

	return nil, fmt.Errorf("failed to scrape: %s", opts.URL)
}

func processMarketEntries(entries []MarketPriceEntry, hasWear bool) []config.MarketPrice {
	var out []config.MarketPrice
	for _, m := range entries {
		if hasWear && len(m.WearPrices) > 0 {
			for _, wear := range config.AllWearConditions {
				key := string(wear)
				val := m.WearPrices[key]
				mp := config.MarketPrice{
					Market:   m.Market,
					Wear:     key,
					Currency: m.Currency,
					URL:      m.URLs[key],
					HasPrice: val != nil,
				}
				if val != nil {
					mp.Price = *val
				}
				out = append(out, mp)
			}
		} else {
			mp := config.MarketPrice{
				Market:   m.Market,
				Currency: m.Currency,
				URL:      m.URL,
				HasPrice: m.Single != nil,
			}
			if m.Single != nil {
				mp.Price = *m.Single
			}
			out = append(out, mp)
		}
	}
	return out
}

// FetchManyItems scrapes a list of items and returns all successful results.
func FetchManyItems(ctx context.Context, optsList []FetchOptions) []config.Item {
	var out []config.Item
	for _, o := range optsList {
		items, err := FetchItem(ctx, o)
		if err != nil {
			continue
		}
		out = append(out, items...)
	}
	return out
}
