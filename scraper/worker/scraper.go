// Package worker is the top-level orchestrator.
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/eovacius/csgodatabase-scraper/internal"
	"github.com/eovacius/csgodatabase-scraper/scraper"
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
	fmt.Println("[*] Initializing Chrome allocator...")
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), config.GetOpts()...)
	defer cancel()

	fmt.Println("[*] Creating browser context...")
	// Log Chrome output for debugging in GitHub Actions
	browserCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, config.DeadLine)
	defer cancel()

	fmt.Println("[*] Performing pre-flight check (navigating to about:blank)...")
	if err := chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(scraper.ConfigJS).Do(ctx)
			return err
		}),
		chromedp.Navigate("about:blank"),
	); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}
	fmt.Println("[*] Pre-flight check successful, Chrome is active.")

	var (
		mu  sync.Mutex
		all []config.Item
		wg  sync.WaitGroup
	)

	// Focus on Weapons
	if len(config.AllCategories) == 0 {
		return nil, fmt.Errorf("no categories configured")
	}
	cat := config.AllCategories[0]

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
	for weaponIndex, w := range weapons {
		if config.MaxWeapons > 0 && weaponIndex >= config.MaxWeapons {
			break
		}
		fmt.Printf("\033[36m[*] Step 2: Discovering all Skins for weapon: %s\033[0m\n", w.Name)

		skinSlugs, err := discovery.Discover(ictx, discovery.Options{
			Category: w.URL,
		})
		if err != nil {
			fmt.Printf("\033[31m[!]\033[0m Skin discovery failed for %s: %v\n", w.Name, err)
			continue
		}

		skinSlugs = filterDiscoveryItems(skinSlugs, "skin")

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

		// Use a local context for fetching many items
		fetchCtx, fCancel := chromedp.NewContext(ictx)
		items := common.FetchManyItems(fetchCtx, optsList)
		fCancel()

		allItems = append(allItems, items...)
	}

	return allItems
}

func filterDiscoveryItems(items []discovery.Item, itemType string) []discovery.Item {
	filtered := make([]discovery.Item, 0, len(items))
	for _, item := range items {
		if item.Type == itemType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
