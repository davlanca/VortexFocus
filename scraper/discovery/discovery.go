// Package discovery crawls a category page (e.g. /skins/, /weapons/) and
// returns all item slugs found, handling pagination automatically.
package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// Item is a single discovered item.
type Item struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	IsCollection bool   `json:"isCollection"`
}

// PageResult is what discovery.js returns from one page.
type PageResult struct {
	Items       []Item `json:"items"`
	HasNextPage bool   `json:"hasNextPage"`
	NextPageURL string `json:"nextPageUrl"`
}

// Options configures a discovery crawl.
type Options struct {
	Category   string
	BaseURL    string
	MaxPages   int
	PageDelay  time.Duration
	OnItem     func(Item)
	OnProgress func(page int, found int)
}

// Discover crawls /<category>/ across all pages and returns unique items.
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

	// If Category is already a full URL, use it. Otherwise build from BaseURL.
	pageURL := opts.Category
	if !contains(pageURL, "http") {
		pageURL = fmt.Sprintf("%s/%s/", opts.BaseURL, opts.Category)
	}

	for page := 1; page <= opts.MaxPages; page++ {
		var res PageResult
		err := chromedp.Run(ctx,
			chromedp.Navigate(pageURL),
			chromedp.Evaluate(string(scraper.ConfigJS), nil),
			chromedp.Sleep(opts.PageDelay),
			chromedp.Evaluate(string(scraper.DiscoveryJS), &res),
		)
		if err != nil {
			return all, fmt.Errorf("page %d: %w", page, err)
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))
}
