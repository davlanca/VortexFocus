// Package common provides shared scraping helpers.
package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

type PriceTableResult struct {
	ItemName string             `json:"itemName"`
	HasWear  bool               `json:"hasWear"`
	Normal   []MarketPriceEntry `json:"normal"`
	StatTrak []MarketPriceEntry `json:"stattrak"`
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

// FetchItem is the unified way to scrape any item page (weapon, glove, case).
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
	const retryDelay = 10 * time.Second
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var pageTitle string
		var res PriceTableResult

		// Per-item timeout to prevent hanging the whole process
		runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		err := chromedp.Run(runCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, _, _, err := page.Navigate(opts.URL).Do(ctx)
				return err
			}),
			chromedp.WaitReady(`body`, chromedp.ByQuery),
			chromedp.Evaluate(string(scraper.ConfigJS), nil),
			chromedp.Sleep(config.NextDelay()),
			chromedp.Title(&pageTitle),
			chromedp.Evaluate(string(scraper.PricesJS), &res),
		)
		cancel()

		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m [%s] attempt %d error: %v\n", opts.Name, attempt+1, err)
			continue
		}

		lower := strings.ToLower(pageTitle)
		if strings.Contains(lower, "page not found") {
			return nil, fmt.Errorf("404")
		}
		if strings.Contains(lower, "verify") || strings.Contains(lower, "human") || strings.Contains(lower, "just a moment") || strings.Contains(lower, "attention required") {
			fmt.Printf("\033[31m[!]\033[0m Detection on %s, retry %d\n", opts.Name, attempt+1)
			if config.Interactive {
				fmt.Println("      [!] Cloudflare verification needed. Complete it in browser, then press Enter.")
				var dummy string
				fmt.Scanln(&dummy)
				attempt--
			}
			continue
		}

		var items []config.Item
		finalName := res.ItemName
		if finalName == "" {
			finalName = opts.Name
		}

		// Handle Normal / Gloves / Cases
		if len(res.Normal) > 0 {
			it := baseItem
			it.Name = finalName
			it.Type = opts.Type
			if it.Type == "" || it.Type == "Normal" {
				if opts.Category == "gloves" {
					it.Type = "Gloves"
				} else if opts.Category == "cases" {
					it.Type = "Case"
				} else {
					it.Type = "Normal"
				}
			}
			it.Prices = processMarketEntries(res.Normal, it.HasWear || res.HasWear)
			items = append(items, it)
		}

		// Handle StatTrak (mostly for weapons)
		if len(res.StatTrak) > 0 {
			it := baseItem
			it.Name = finalName
			it.Type = "StatTrak"
			it.Prices = processMarketEntries(res.StatTrak, it.HasWear || res.HasWear)
			items = append(items, it)
		}

		if len(items) > 0 {
			fmt.Printf("\033[32m[+]\033[0m Scraped: %s\n", finalName)
			return items, nil
		}

		fmt.Printf("\033[33m[?]\033[0m No prices for %s (attempt %d)\n", opts.Name, attempt+1)
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("failed to scrape %s after retries", opts.Name)
}

func FetchCase(ctx context.Context, url, name string) (config.Item, error) {
	items, err := FetchItem(ctx, FetchOptions{
		URL:      url,
		Name:     name,
		Category: "cases",
		Type:     "Case",
		HasWear:  false,
	})
	if err != nil || len(items) == 0 {
		return config.Item{}, err
	}
	return items[0], nil
}

func FetchGlove(ctx context.Context, url, name string) (config.Item, error) {
	items, err := FetchItem(ctx, FetchOptions{
		URL:      url,
		Name:     name,
		Category: "gloves",
		Type:     "Gloves",
		HasWear:  true,
	})
	if err != nil || len(items) == 0 {
		return config.Item{}, err
	}
	return items[0], nil
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

			// Each item gets a fresh context/tab to avoid cross-contamination and hangs
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

