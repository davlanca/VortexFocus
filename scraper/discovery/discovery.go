// Package discovery crawls a category page (e.g. /skins/, /weapons/) and
// returns all item slugs found, handling pagination automatically.
//
// Used by category scrapers that need to scrape every item, not just a
// hardcoded list.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// Item is a single discovered item.
type Item struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// PageResult is what discovery.js returns from one page.
type PageResult struct {
	Items       []Item `json:"items"`
	HasNextPage bool   `json:"hasNextPage"`
	NextPageURL string `json:"nextPageUrl"`
}

// Options configures a discovery crawl.
type Options struct {
	Category   string        // "skins", "cases", "weapons", ...
	BaseURL    string        // "https://www.csgodatabase.com"
	MaxPages   int           // hard cap; 0 = default 50
	PageDelay  time.Duration // sleep between page fetches; 0 = use config.Delay
	OnItem     func(Item)    // optional callback for streaming
	OnProgress func(page int, found int)
}

// Discover crawls /<category>/ across all pages and returns unique items.
// Stops when pagination ends or MaxPages is reached.
func Discover(ctx context.Context, opts Options) ([]Item, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = config.Target
	}
	if opts.MaxPages == 0 {
		opts.MaxPages = 50
	}
	if opts.PageDelay == 0 {
		opts.PageDelay = config.Delay
	}

	seen := make(map[string]bool)
	var all []Item
	pageURL := fmt.Sprintf("%s/%s/", opts.BaseURL, opts.Category)

	for page := 1; page <= opts.MaxPages; page++ {
		var jsonOut string
		err := chromedp.Run(ctx,
			chromedp.Navigate(pageURL),
			chromedp.Evaluate(string(scraper.ConfigJS), nil),
			chromedp.Sleep(opts.PageDelay),
			chromedp.Evaluate(string(scraper.DiscoveryJS), &jsonOut),
		)
		if err != nil {
			return all, fmt.Errorf("page %d: navigate failed: %w", page, err)
		}

		var res PageResult
		if jerr := json.Unmarshal([]byte(jsonOut), &res); jerr != nil {
			return all, fmt.Errorf("page %d: parse failed: %w", page, jerr)
		}

		newCount := 0
		for _, it := range res.Items {
			if !seen[it.Slug] {
				seen[it.Slug] = true
				all = append(all, it)
				newCount++
				if opts.OnItem != nil {
					opts.OnItem(it)
				}
			}
		}

		if opts.OnProgress != nil {
			opts.OnProgress(page, len(all))
		}

		if !res.HasNextPage || res.NextPageURL == "" {
			break
		}
		pageURL = res.NextPageURL
	}

	return all, nil
}

// QuickSlug returns a single discovery page worth of items, no pagination.
// Useful for small/fast categories.
func QuickSlugs(ctx context.Context, category string) ([]Item, error) {
	return Discover(ctx, Options{
		Category:  category,
		MaxPages:  1,
		PageDelay: config.Delay,
	})
}
