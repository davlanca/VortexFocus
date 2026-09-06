// Package worker is the top-level orchestrator.
package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/chromedp"

	"github.com/eovacius/csgodatabase-scraper/internal"
	"github.com/eovacius/csgodatabase-scraper/scraper/common"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/discovery"
)

// ScrapeSkins runs the scraper and returns skins and agents in the format expected by main.go.
func ScrapeSkins() ([]config.Skin, []config.Agent, error) {
	items, err := Scrape()
	if err != nil {
		return nil, nil, err
	}

	var skins []config.Skin
	var agents []config.Agent

	for _, it := range items {
		if it.Category == "agents" {
			agents = append(agents, internal.ConvertToAgent(it))
		} else {
			// All other categories treated as "skins" for the legacy output format
			skins = append(skins, internal.ConvertToSkin(it))
		}
	}

	return skins, agents, nil
}

// Scrape runs all configured category scrapers in parallel browser tabs.
func Scrape() ([]config.Item, error) {
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), config.Opts...)
	defer cancel()

	fmt.Println("\n[*] Creating allocator context and applying opts...")

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, config.DeadLine)
	defer cancel()

	fmt.Println("[*] Created allocator. Running pre-flight check...")
	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}
	fmt.Println("[*] Pre-flight check successful, Chrome is running.\n")

	var (
		mu  sync.Mutex
		all []config.Item
		wg  sync.WaitGroup
	)

	for _, cat := range config.AllCategories {
		wg.Add(1)
		go func(c config.Category) {
			defer wg.Done()
			items := runCategory(browserCtx, c)

			mu.Lock()
			all = append(all, items...)
			mu.Unlock()

			fmt.Printf("\033[32m[+]\033[0m Done %s (%d items)\n", c.DisplayName, len(items))
		}(cat)
	}

	wg.Wait()
	return all, nil
}

// runCategory scrapes a single category.
func runCategory(parent context.Context, cat config.Category) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	slugs := cat.SlugList
	if slugs == nil {
		fmt.Printf("\n\033[36m[*] Discovering slugs for /%s/ via pagination...\033[0m\n", cat.Slug)
		discovered, err := discovery.Discover(ictx, discovery.Options{
			Category: cat.Slug,
			OnProgress: func(page, found int) {
				fmt.Printf("\033[36m   [%s] page %d, total so far: %d\033[0m\n", cat.Slug, page, found)
			},
		})
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m Discovery failed for %s: %v\n", cat.Slug, err)
			return nil
		}
		for _, it := range discovered {
			slugs = append(slugs, it.Slug)
		}
		fmt.Printf("\033[36m   [%s] discovered %d items\033[0m\n", cat.Slug, len(slugs))
	}

	if len(slugs) == 0 {
		return nil
	}

	optsList := make([]common.FetchOptions, 0, len(slugs))
	for _, s := range slugs {
		optsList = append(optsList, common.FetchOptions{
			URL:      fmt.Sprintf("%s/%s/%s/", config.Target, cat.Slug, s),
			Category: cat.Slug,
			HasWear:  cat.HasWear,
			Slug:     s,
			Name:     humanize(s),
		})
	}

	return common.FetchMany(ictx, optsList)
}

func humanize(slug string) string {
	return internal.Humanize(slug)
}
