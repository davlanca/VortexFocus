// Package discovery crawls a category page and returns all item slugs found.
package discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	cdpage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

type Item struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

type PageResult struct {
	Items       []Item `json:"items"`
	HasNextPage bool   `json:"hasNextPage"`
	NextPageURL string `json:"nextPageUrl"`
}

type Options struct {
	Category   string
	BaseURL    string
	MaxPages   int
	PageDelay  time.Duration
	OnItem     func(Item)
	OnProgress func(page int, found int)
}

// scrollDownAction scrolls the page to the bottom once to trigger lazy loading.
func scrollDownAction() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil).Do(ctx)
	})
}

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

	pageURL := opts.Category
	if !strings.Contains(pageURL, "http") {
		// Remove leading slash if present to avoid double slashes in URL
		category := strings.TrimPrefix(opts.Category, "/")
		pageURL = fmt.Sprintf("%s/%s/", opts.BaseURL, category)
	}

	for page := 1; page <= opts.MaxPages; page++ {
		var res PageResult
		var title string

		fmt.Printf("   -> [Page %d] Discovery navigating to: %s\n", page, pageURL)

		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, _, _, err := cdpage.Navigate(pageURL).Do(ctx)
				return err
			}),
			chromedp.Title(&title),
			chromedp.WaitReady(`body`, chromedp.ByQuery),
			// Wait exactly 1 second for dynamic content
			chromedp.Sleep(1 * time.Second),
			// Scroll to bottom to trigger lazy loading
			scrollDownAction(),
			// Wait 500ms for scroll to complete
			chromedp.Sleep(500 * time.Millisecond),
			chromedp.Evaluate(string(scraper.DiscoveryJS), &res),
		)
		if err != nil {
			return all, fmt.Errorf("page %d: %w", page, err)
		}

		fmt.Printf("      [Title: %s] Found %d potential links\n", title, len(res.Items))

		lowerTitle := strings.ToLower(title)
		blocked := strings.Contains(lowerTitle, "cloudflare") ||
			strings.Contains(lowerTitle, "access denied") ||
			strings.Contains(lowerTitle, "attention required") ||
			strings.Contains(lowerTitle, "just a moment")
		if config.Interactive && blocked {
			fmt.Println("      [!] Cloudflare verification is open in Chrome. Complete it manually, then press Enter here.")
			_, _ = fmt.Scanln()
			if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, _, _, err := cdpage.Navigate(pageURL).Do(ctx)
				return err
			}), chromedp.WaitReady(`body`, chromedp.ByQuery), chromedp.Sleep(1*time.Second), scrollDownAction(), chromedp.Evaluate(string(scraper.DiscoveryJS), &res)); err != nil {
				return all, fmt.Errorf("page %d after manual verification: %w", page, err)
			}
		}

		// Check if we hit a verification page - full retry with 10 sec pause
		if strings.Contains(strings.ToLower(title), "just a moment") || strings.Contains(strings.ToLower(title), "verify") {
			fmt.Printf("\033[31m[!]\033[0m Cloudflare on %s - waiting 10s before retry...\n", pageURL)
			time.Sleep(10 * time.Second)
			if err := chromedp.Run(ctx,
				chromedp.ActionFunc(func(ctx context.Context) error {
					_, _, _, _, err := cdpage.Navigate(pageURL).Do(ctx)
					return err
				}),
				chromedp.WaitReady(`body`, chromedp.ByQuery),
				chromedp.Sleep(1*time.Second),
				scrollDownAction(),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.Evaluate(string(scraper.DiscoveryJS), &res),
			); err != nil {
				fmt.Printf("\033[31m[!]\033[0m Cloudflare retry failed on %s: %v\n", pageURL, err)
			}
		}

		// No items found - retry once with 3 sec pause
		if len(res.Items) == 0 {
			fmt.Printf("\033[33m[~]\033[0m No items found on %s - waiting 3s before retry...\n", pageURL)
			time.Sleep(3 * time.Second)
			_ = chromedp.Run(ctx,
				chromedp.WaitReady(`body`, chromedp.ByQuery),
				chromedp.Sleep(1*time.Second),
				scrollDownAction(),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.Evaluate(string(scraper.DiscoveryJS), &res),
			)
			if len(res.Items) == 0 {
				fmt.Printf("\033[33m[!]\033[0m Skipping page %s after 2 failed attempts\n", pageURL)
			}
		}

		for _, it := range res.Items {
			if !seen[it.URL] {
				seen[it.URL] = true
				all = append(all, it)
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
