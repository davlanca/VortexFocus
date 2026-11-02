// Package common provides shared scraping helpers.
package common
import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"strings"
	"time"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)
// randomDelay sleeps for a random duration between min and max seconds.
func randomDelayMs(minMs, maxMs int) {
	d := time.Duration(minMs+rand.Intn(maxMs-minMs+1)) * time.Millisecond
	time.Sleep(d)
}

func randomDelay(minSecs, maxSecs int) {
	d := time.Duration(minSecs*1000+rand.Intn((maxSecs-minSecs+1)*1000)) * time.Millisecond
	time.Sleep(d)
}
type PriceTableResult struct {
	ItemName string             `json:"itemName"`
	HasWear  bool               `json:"hasWear"`
	Normal   []MarketPriceEntry `json:"normal"`
	StatTrak []MarketPriceEntry `json:"stattrak"`
	Souvenir []MarketPriceEntry `json:"souvenir"`
}
type MarketPriceEntry struct {
	Market     string              `json:"market"`
	Currency   string              `json:"currency"`
	Single     *float64            `json:"single,omitempty"`
	URL        string              `json:"url,omitempty"`
	WearPrices map[string]*float64 `json:"wearPrices"`
	URLs       map[string]string   `json:"urls"`
	Lowest     map[string]bool     `json:"lowest,omitempty"`
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
	const (
	maxRetries     = 2
	retryPause     = 10 * time.Second
)
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Random delay before each attempt
		randomDelayMs(100, 300)

		var pageTitle string
		var res PriceTableResult
		// Per-item timeout to prevent hanging the whole process
		runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Navigation logging for debugging
		fmt.Printf("\033[36m[~]\033[0m Navigating: %s\n", opts.URL)

		err := chromedp.Run(runCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, _, _, err := page.Navigate(opts.URL).Do(ctx)
				return err
			}),
			chromedp.WaitReady(`body`, chromedp.ByQuery),
			chromedp.Sleep(1*time.Second),
			chromedp.Title(&pageTitle),
			chromedp.Evaluate(string(scraper.PricesJS), &res),
		)

		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m [%s] attempt %d/%d error: %v\n", opts.Name, attempt+1, maxRetries+1, err)
			if attempt < maxRetries {
				fmt.Printf("\033[33m[~]\033[0m Waiting %ds before retry...\n", int(retryPause.Seconds()))
				time.Sleep(retryPause)
			}
			continue
		}
		lower := strings.ToLower(pageTitle)
		if strings.Contains(lower, "page not found") {
			fmt.Printf("\033[33m[404]\033[0m %s  ->  page not found\n", opts.Name)
			return nil, nil
		}
		if strings.Contains(lower, "verify") || strings.Contains(lower, "human") ||
			strings.Contains(lower, "just a moment") || strings.Contains(lower, "attention required") ||
			strings.Contains(lower, "cloudflare") {
			fmt.Printf("\033[31m[!]\033[0m Cloudflare on %s (attempt %d/%d)\n", opts.Name, attempt+1, maxRetries+1)
			if config.Interactive {
				fmt.Println("      [!] Cloudflare verification needed. Complete it in browser, then press Enter.")
				var dummy string
				fmt.Scanln(&dummy)
				attempt--
			}
			if attempt < maxRetries {
				fmt.Printf("\033[33m[~]\033[0m Waiting %ds before retry...\n", int(retryPause.Seconds()))
				time.Sleep(retryPause)
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
		// Handle Souvenir (weapon/glove souvenir variants and souvenir packages)
		if len(res.Souvenir) > 0 {
			it := baseItem
			it.Name = finalName
			it.Type = "Souvenir"
			it.Prices = processMarketEntries(res.Souvenir, it.HasWear || res.HasWear)
			items = append(items, it)
		}
		if len(items) > 0 {
			priceCount := 0
			for _, it := range items {
				priceCount += len(it.Prices)
			}
			fmt.Printf("\033[32m[+] OK\033[0m   %s  ->  %d prices\n", finalName, priceCount)
			return items, nil
		}
		fmt.Printf("\033[33m[?]\033[0m No prices for %s (attempt %d/%d)\n", opts.Name, attempt+1, maxRetries+1)
		if attempt < maxRetries {
			time.Sleep(retryPause)
		}
	}
	name := opts.Name
	if name == "" {
		name = opts.URL
	}
	fmt.Printf("\033[33m[!] SKIP\033[0m  %s  ->  %d attempts failed\n", name, maxRetries+1)
	return nil, nil // skip, do not error
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
						Lowest: m.Lowest[key],
					})
				}
			}
		} else if m.Single != nil {
			out = append(out, config.MarketPrice{
				Market: m.Market, Currency: m.Currency,
				Price: *m.Single, URL: m.URL, HasPrice: true,
				Lowest: m.Lowest["single"],
			})
		}
	}
	return out
}
// FetchManyItems scrapes multiple items in parallel, opening a new browser tab per worker.
func FetchManyItems(parent context.Context, optsList []FetchOptions) []config.Item {
	if len(optsList) == 0 {
		return nil
	}

	workers := config.Workers
	if workers < 1 {
		workers = 2
	}
	sem := make(chan struct{}, workers)

	var mu sync.Mutex
	var out []config.Item
	var wg sync.WaitGroup

	for _, opts := range optsList {
		wg.Add(1)
		go func(o FetchOptions) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Each goroutine gets its own NEW browser tab
			ctx, cancel := chromedp.NewContext(parent)
			defer cancel()

			// Fixed 2.5s delay between opening tabs to avoid rate limiting
			time.Sleep(2500 * time.Millisecond)

			items, err := FetchItem(ctx, o)
			if err == nil && len(items) > 0 {
				mu.Lock()
				out = append(out, items...)
				mu.Unlock()
			}
		}(opts)
	}

	wg.Wait()
	return out
}
