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

// FetchItem is the unified way to scrape any item page.
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

		// Per-item timeout prevents one broken page from hanging
		// the entire scraper.
		runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		err := chromedp.Run(
			runCtx,

			chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, _, _, err := page.Navigate(opts.URL).Do(ctx)
				return err
			}),

			chromedp.WaitReady(`body`, chromedp.ByQuery),

			// Site configuration / helper JS.
			chromedp.Evaluate(string(scraper.ConfigJS), nil),

			chromedp.Sleep(config.NextDelay()),

			chromedp.Title(&pageTitle),

			// Parse marketplace tables.
			chromedp.Evaluate(string(scraper.PricesJS), &res),
		)

		cancel()

		if err != nil {
			fmt.Printf(
				"\033[31m[!]\033[0m [%s] attempt %d error: %v\n",
				opts.Name,
				attempt+1,
				err,
			)

			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}

			continue
		}

		// ------------------------------------------------------------
		// Basic page validation
		// ------------------------------------------------------------

		lowerTitle := strings.ToLower(pageTitle)

		if strings.Contains(lowerTitle, "page not found") {
			return nil, fmt.Errorf("404")
		}

		// Cloudflare / anti-bot / verification page.
		if strings.Contains(lowerTitle, "verify") ||
			strings.Contains(lowerTitle, "human") ||
			strings.Contains(lowerTitle, "just a moment") ||
			strings.Contains(lowerTitle, "attention required") {

			fmt.Printf(
				"\033[31m[!]\033[0m Detection on %s, retry %d\n",
				opts.Name,
				attempt+1,
			)

			if config.Interactive {
				fmt.Println(
					"      [!] Cloudflare verification needed. " +
						"Complete it in browser, then press Enter.",
				)

				var dummy string
				fmt.Scanln(&dummy)

				// Repeat the same attempt after manual verification.
				attempt--
				continue
			}

			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}

			continue
		}

		// ------------------------------------------------------------
		// Resolve item name
		// ------------------------------------------------------------

		finalName := strings.TrimSpace(res.ItemName)

		if finalName == "" {
			finalName = strings.TrimSpace(opts.Name)
		}

		if finalName == "" {
			finalName = "Unknown Item"
		}

		var items []config.Item

		// ------------------------------------------------------------
		// Normal / regular item
		// ------------------------------------------------------------

		if len(res.Normal) > 0 {
			it := baseItem

			it.Name = finalName
			it.Type = opts.Type

			// If the caller did not explicitly specify a type,
			// infer it from the category.
			if it.Type == "" || it.Type == "Normal" {
				switch strings.ToLower(opts.Category) {
				case "gloves":
					it.Type = "Gloves"

				case "cases":
					it.Type = "Case"

				default:
					it.Type = "Normal"
				}
			}

			// prices.js can detect wear information itself.
			hasWear := it.HasWear || res.HasWear

			it.Prices = processMarketEntries(
				res.Normal,
				hasWear,
			)

			items = append(items, it)
		}

		// ------------------------------------------------------------
		// StatTrak
		// ------------------------------------------------------------

		if len(res.StatTrak) > 0 {
			it := baseItem

			it.Name = finalName
			it.Type = "StatTrak"

			hasWear := it.HasWear || res.HasWear

			it.Prices = processMarketEntries(
				res.StatTrak,
				hasWear,
			)

			items = append(items, it)
		}

		// ------------------------------------------------------------
		// Successful scrape
		// ------------------------------------------------------------

		if len(items) > 0 {
			totalPrices := 0

			for _, item := range items {
				totalPrices += len(item.Prices)
			}

			if totalPrices == 0 {
				fmt.Printf(
					"\033[33m[?]\033[0m %s parsed but contains no usable prices (attempt %d)\n",
					finalName,
					attempt+1,
				)

				if attempt < maxRetries {
					time.Sleep(retryDelay)
				}

				continue
			}

			fmt.Printf(
				"\033[32m[+]\033[0m Scraped: %s\n",
				finalName,
			)

			return items, nil
		}

		fmt.Printf(
			"\033[33m[?]\033[0m No prices for %s (attempt %d)\n",
			opts.Name,
			attempt+1,
		)

		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf(
		"failed to scrape %s after retries",
		opts.Name,
	)
}

// FetchCase is a compatibility helper for case scraping.
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

// FetchGlove is a compatibility helper for glove scraping.
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

// processMarketEntries converts the result returned by prices.js
// into the unified config.MarketPrice structure used by data.json.
func processMarketEntries(
	entries []MarketPriceEntry,
	hasWear bool,
) []config.MarketPrice {

	var out []config.MarketPrice

	for _, m := range entries {
		marketName := strings.TrimSpace(m.Market)

		if marketName == "" {
			continue
		}

		// ------------------------------------------------------------
		// Wear-based item
		// ------------------------------------------------------------

		if hasWear && len(m.WearPrices) > 0 {
			for _, wear := range config.AllWearConditions {
				key := string(wear)

				val, ok := m.WearPrices[key]

				if !ok || val == nil {
					continue
				}

				if *val <= 0 {
					continue
				}

				url := ""

				if m.URLs != nil {
					url = strings.TrimSpace(m.URLs[key])
				}

				out = append(out, config.MarketPrice{
					Market:   marketName,
					Wear:     key,
					Price:    *val,
					Currency: m.Currency,
					HasPrice: true,
					URL:      url,
				})
			}

			continue
		}

		// ------------------------------------------------------------
		// Single-price item
		// Cases / agents / other non-wear items
		// ------------------------------------------------------------

		if m.Single != nil && *m.Single > 0 {
			out = append(out, config.MarketPrice{
				Market:   marketName,
				Price:    *m.Single,
				Currency: m.Currency,
				HasPrice: true,
				URL:      strings.TrimSpace(m.URL),
			})
		}
	}

	return out
}

// FetchManyItems fetches multiple pages concurrently while respecting
// the configured worker limit.
func FetchManyItems(
	parent context.Context,
	optsList []FetchOptions,
) []config.Item {

	var (
		mu  sync.Mutex
		out []config.Item
		wg  sync.WaitGroup
	)

	workers := config.Workers

	if workers < 1 {
		workers = 1
	}

	sem := make(chan struct{}, workers)

	for _, o := range optsList {
		opts := o

		wg.Add(1)

		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
			}()

			// Each item gets its own chromedp context/tab.
			// This prevents pages from interfering with one another.
			ctx, cancel := chromedp.NewContext(parent)
			defer cancel()

			items, err := FetchItem(ctx, opts)

			if err != nil {
				fmt.Printf(
					"\033[31m[!]\033[0m Failed: %s — %v\n",
					opts.Name,
					err,
				)
				return
			}

			if len(items) == 0 {
				return
			}

			mu.Lock()
			out = append(out, items...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	return out
}
