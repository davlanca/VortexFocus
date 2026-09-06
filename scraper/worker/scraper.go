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

// ScrapeSkins runs the scraper and returns skins in the format expected by main.go.
func ScrapeSkins() ([]config.Skin, []config.Agent, error) {
	items, err := Scrape()
	if err != nil {
		return nil, nil, err
	}

	var skins []config.Skin
	// Agents are empty in this focused version
	var agents []config.Agent

	for _, it := range items {
		skins = append(skins, internal.ConvertToSkin(it))
	}

	return skins, agents, nil
}

// Scrape runs the focused weapons scraper.
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

	// In this version, we only care about Weapons
	cat := config.AllCategories[0] // Weapons

	wg.Add(1)
	go func(c config.Category) {
		defer wg.Done()
		items := runWeaponsFlow(browserCtx, c)

		mu.Lock()
		all = append(all, items...)
		mu.Unlock()

		fmt.Printf("\033[32m[+]\033[0m Done Weapons flow (%d total items)\n", len(items))
	}(cat)

	wg.Wait()
	return all, nil
}

// runWeaponsFlow implements: /weapons/ -> Weapon -> Skins -> Prices
func runWeaponsFlow(parent context.Context, cat config.Category) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	fmt.Printf("\n\033[36m[*] Step 1: Discovering all Weapons from /weapons/...\033[0m\n")
	weapons, err := discovery.Discover(ictx, discovery.Options{
		Category: "weapons",
	})
	if err != nil {
		fmt.Printf("\033[31m[!]\033[0m Weapon discovery failed: %v\n", err)
		return nil
	}

	var allItems []config.Item
	for _, w := range weapons {
		fmt.Printf("\033[36m[*] Step 2: Discovering all Skins for weapon: %s\033[0m\n", w.Name)

		// Each weapon page contains a list of skins
		skinSlugs, err := discovery.Discover(ictx, discovery.Options{
			Category: w.URL,
		})
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m Skin discovery failed for %s: %v\n", w.Name, err)
			continue
		}

		fmt.Printf("\033[36m[*] Step 3: Scraping prices for %d skins of %s...\033[0m\n", len(skinSlugs), w.Name)

		optsList := make([]common.FetchOptions, 0, len(skinSlugs))
		for _, s := range skinSlugs {
			optsList = append(optsList, common.FetchOptions{
				URL:      s.URL,
				Category: "weapons",
				HasWear:  true,
				Slug:     s.Slug,
				Name:     s.Name,
				Weapon:   w.Name,
			})
		}

		// Use a local context for fetching many items to avoid session bloat
		fetchCtx, fCancel := chromedp.NewContext(ictx)
		items := common.FetchManyItems(fetchCtx, optsList)
		fCancel()

		allItems = append(allItems, items...)
	}

	return allItems
}
