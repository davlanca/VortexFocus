// Package common provides shared scraping helpers.
package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

type PriceTableResult struct {
	ItemName    string             `json:"itemName"`
	HasWear     bool               `json:"hasWear"`
	HasSouvenir bool               `json:"hasSouvenir"`
	Normal      []MarketPriceEntry `json:"normal"`
	Souvenir    []MarketPriceEntry `json:"souvenir"`
}

type MarketPriceEntry struct {
	Market     string              `json:"market"`
	Currency   string              `json:"currency"`
	Single     *float64            `json:"single,omitempty"`
	URL        string              `json:"url,omitempty"`
	WearPrices map[string]*float64 `json:"wearPrices"`
	URLs       map[string]string   `json:"urls"`
}

type FetchOptions struct {
	URL        string
	Category   string
	Weapon     string
	Rarity     string
	Collection string
	Type       string
	HasWear    bool
	Slug       string
	Name       string
}

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
		var pageTitle string
		var res PriceTableResult

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
			return nil, fmt.Errorf("404")
		}
		if strings.Contains(lower, "verify") || strings.Contains(lower, "human") {
			fmt.Printf("\033[31m[!]\033[0m Detection on %s, retry %d\n", opts.Slug, attempt)
			continue
		}

		// Try to get souvenir prices if they exist but weren't captured initially
		if res.HasSouvenir && len(res.Souvenir) == 0 {
			var souvenirRes []MarketPriceEntry
			err = chromedp.Run(ctx,
				chromedp.Click(`.price-type-tab[data-type="souvenir"], button[data-filter="souvenir"], .tab-link[href*="souvenir"]`, chromedp.ByQuery),
				chromedp.Sleep(1000*time.Millisecond),
				chromedp.Evaluate(`window.extractPrices()`, &souvenirRes),
			)
			if err == nil {
				res.Souvenir = souvenirRes
			}
		}

		var items []config.Item
		if len(res.Normal) > 0 {
			it := baseItem
			it.Name = res.ItemName
			if it.Name == "" { it.Name = opts.Name }
			it.Type = "Normal"
			it.Prices = processMarketEntries(res.Normal, it.HasWear || res.HasWear)
			items = append(items, it)
		}
		if len(res.Souvenir) > 0 {
			it := baseItem
			it.Name = res.ItemName + " (Souvenir)"
			if res.ItemName == "" { it.Name = opts.Name + " (Souvenir)" }
			it.Type = "Souvenir"
			it.Prices = processMarketEntries(res.Souvenir, it.HasWear || res.HasWear)
			items = append(items, it)
		}

		if len(items) > 0 {
			fmt.Printf("\033[32m[+]\033[0m Scraped: %s\n", opts.Slug)
			return items, nil
		}
		fmt.Printf("\033[33m[?]\033[0m No prices for %s\n", opts.Slug)
	}
	return nil, fmt.Errorf("failed")
}

func processMarketEntries(entries []MarketPriceEntry, hasWear bool) []config.MarketPrice {
	var out []config.MarketPrice
	for _, m := range entries {
		if hasWear && len(m.WearPrices) > 0 {
			for _, wear := range config.AllWearConditions {
				key := string(wear)
				if val, ok := m.WearPrices[key]; ok && val != nil {
					out = append(out, config.MarketPrice{
						Market: m.Market, Wear: key, Currency: m.Currency,
						Price: *val, URL: m.URLs[key], HasPrice: true,
					})
				}
			}
		} else if m.Single != nil {
			out = append(out, config.MarketPrice{
				Market: m.Market, Currency: m.Currency,
				Price: *m.Single, URL: m.URL, HasPrice: true,
			})
		}
	}
	return out
}

func FetchManyItems(parent context.Context, optsList []FetchOptions) []config.Item {
	var (
		mu  sync.Mutex
		out []config.Item
		wg  sync.WaitGroup
		sem = make(chan struct{}, config.Workers)
	)

	for _, o := range optsList {
		wg.Add(1)
		go func(opts FetchOptions) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := chromedp.NewContext(parent)
			defer cancel()

			items, err := FetchItem(ctx, opts)
			if err == nil {
				mu.Lock()
				out = append(out, items...)
				mu.Unlock()
			}
		}(o)
	}
	wg.Wait()
	return out
}
