// Package common provides shared scraping helpers used by all category
// scrapers. The main entry point is FetchItem: it opens an item page, runs
// the anti-detection JS, waits for the price table to render, and extracts
// the prices into a config.Item ready to be saved as JSON.
package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// PriceTableResult is the JSON shape produced by prices.js.
type PriceTableResult struct {
	ItemName string             `json:"itemName"`
	HasWear  bool               `json:"hasWear"`
	Markets  []MarketPriceEntry `json:"markets"`
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
// the price table. Returns a config.Item with all prices populated.
func FetchItem(ctx context.Context, opts FetchOptions) (config.Item, error) {
	item := config.Item{
		URL:        opts.URL,
		Slug:       opts.Slug,
		Category:   opts.Category,
		Weapon:     opts.Weapon,
		Rarity:     opts.Rarity,
		Collection: opts.Collection,
		Type:       opts.Type,
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
		var jsonOut string

		err := chromedp.Run(ctx,
			chromedp.Navigate(opts.URL),
			chromedp.Evaluate(string(scraper.ConfigJS), nil),
			chromedp.Sleep(config.Delay),
			chromedp.Title(&pageTitle),
			chromedp.Evaluate(string(scraper.PricesJS), &jsonOut),
		)
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m chromedp error: %v\n", err)
			break
		}

		lower := strings.ToLower(pageTitle)
		if strings.Contains(lower, "page not found") {
			fmt.Printf("\033[33m[?]\033[0m 404: %s\n", opts.URL)
			return item, fmt.Errorf("404 not found")
		}
		if strings.Contains(lower, "verify") ||
			strings.Contains(lower, "human") ||
			strings.Contains(lower, "just a moment") {
			fmt.Printf("\033[31m[!]\033[0m Detection triggered, retrying...\n")
			continue
		}

		var pr PriceTableResult
		if jerr := json.Unmarshal([]byte(jsonOut), &pr); jerr != nil {
			fmt.Printf("\033[31m[!]\033[0m Failed to parse prices JSON: %v\n", jerr)
			continue
		}

		name := pr.ItemName
		if name == "" {
			name = opts.Name
		}
		if name == "" {
			name = strings.ReplaceAll(opts.Slug, "-", " ")
		}
		item.Name = name

		if opts.HasWear {
			item.HasWear = true
		} else {
			item.HasWear = pr.HasWear
		}

		for _, m := range pr.Markets {
			if item.HasWear {
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
					item.Prices = append(item.Prices, mp)
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
				item.Prices = append(item.Prices, mp)
			}
		}

		if len(item.Prices) == 0 {
			fmt.Printf("\033[31m[!]\033[0m No prices found for %s\n", opts.URL)
			continue
		}

		return item, nil
	}

	return item, fmt.Errorf("failed to scrape: %s", opts.URL)
}

// FetchMany scrapes a list of items sequentially using the same browser
// context. Returns the successful items (failures are logged and skipped).
func FetchMany(ctx context.Context, optsList []FetchOptions) []config.Item {
	var out []config.Item
	for _, o := range optsList {
		it, err := FetchItem(ctx, o)
		if err != nil {
			continue
		}
		out = append(out, it)
	}
	return out
}
