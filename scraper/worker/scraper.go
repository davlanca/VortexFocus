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
			skins = append(skins, internal.ConvertToSkin(it))
		}
	}

	return skins, agents, nil
}

// Scrape runs all configured category scrapers in parallel browser tabs.
func Scrape() ([]config.Item, error) {
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), config.Opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, config.DeadLine)
	defer cancel()

	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	var (
		mu  sync.Mutex
		all []config.Item
		wg  sync.WaitGroup
	)

	for _, cat := range config.AllCategories {
		wg.Add(1)
		go func(c config.Category) {
			defer wg.Done()
			var items []config.Item

			if c.Slug == "skins" {
				items = runSkinsCategory(browserCtx, c)
			} else {
				items = runCategory(browserCtx, c)
			}

			mu.Lock()
			all = append(all, items...)
			mu.Unlock()

			fmt.Printf("\033[32m[+]\033[0m Done %s (%d items)\n", c.DisplayName, len(items))
		}(cat)
	}

	wg.Wait()
	return all, nil
}

// runSkinsCategory implements the nested flow: /skins/ -> Collections -> Skins
func runSkinsCategory(parent context.Context, cat config.Category) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	fmt.Printf("\n\033[36m[*] Discovering Collections for /skins/...\033[0m\n")
	colls, err := discovery.Discover(ictx, discovery.Options{
		Category: "skins",
	})
	if err != nil {
		fmt.Printf("\033[31m[!]\033[0m Collection discovery failed: %v\n", err)
		return nil
	}

	var allSkins []config.Item
	for _, col := range colls {
		fmt.Printf("\033[36m   -> Collection: %s\033[0m\n", col.Name)
		skinSlugs, err := discovery.Discover(ictx, discovery.Options{
			Category: col.URL, // Use the full URL discovered
		})
		if err != nil {
			continue
		}

		optsList := make([]common.FetchOptions, 0, len(skinSlugs))
		for _, s := range skinSlugs {
			optsList = append(optsList, common.FetchOptions{
				URL:        s.URL,
				Category:   "skins",
				HasWear:    true,
				Slug:       s.Slug,
				Name:       s.Name,
				Collection: col.Name,
			})
		}
		items := common.FetchManyItems(ictx, optsList)
		allSkins = append(allSkins, items...)
	}

	return allSkins
}

// runCategory scrapes a single category.
func runCategory(parent context.Context, cat config.Category) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	slugs := cat.SlugList
	if slugs == nil {
		fmt.Printf("\n\033[36m[*] Discovering items for /%s/ via pagination...\033[0m\n", cat.Slug)
		discovered, err := discovery.Discover(ictx, discovery.Options{
			Category: cat.Slug,
		})
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m Discovery failed for %s: %v\n", cat.Slug, err)
			return nil
		}

		optsList := make([]common.FetchOptions, 0, len(discovered))
		for _, it := range discovered {
			optsList = append(optsList, common.FetchOptions{
				URL:      it.URL,
				Category: cat.Slug,
				HasWear:  cat.HasWear,
				Slug:     it.Slug,
				Name:     it.Name,
			})
		}
		return common.FetchManyItems(ictx, optsList)
	}

	optsList := make([]common.FetchOptions, 0, len(slugs))
	for _, s := range slugs {
		optsList = append(optsList, common.FetchOptions{
			URL:      fmt.Sprintf("%s/%s/%s/", config.Target, cat.Slug, s),
			Category: cat.Slug,
			HasWear:  cat.HasWear,
			Slug:     s,
			Name:     internal.Humanize(s),
		})
	}

	return common.FetchManyItems(ictx, optsList)
}
