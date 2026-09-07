// Package worker is the top-level orchestrator.
package worker

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/eovacius/csgodatabase-scraper/internal"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/common"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/discovery"
)

// randomDelay sleeps for a random duration between minSecs and maxSecs (inclusive).
func randomDelay(minSecs, maxSecs int) {
	d := time.Duration(minSecs*1000+rand.Intn((maxSecs-minSecs+1)*1000)) * time.Millisecond
	time.Sleep(d)
}

// randomDelayMs sleeps for a random duration between minMs and maxMs milliseconds.
func randomDelayMs(minMs, maxMs int) {
	d := time.Duration(minMs+rand.Intn(maxMs-minMs+1)) * time.Millisecond
	time.Sleep(d)
}

var (
	seenMu   sync.Mutex
	seenURLs = make(map[string]bool)
)

func isSeen(url string) bool {
	seenMu.Lock()
	defer seenMu.Unlock()
	if seenURLs[url] {
		return true
	}
	seenURLs[url] = true
	return false
}

// ScrapeAll runs the full scraping cycle for all categories.
func ScrapeAll(progress func([]config.Skin, []config.Agent, []config.Skin)) ([]config.Skin, []config.Agent, []config.Skin, error) {
	fmt.Println("[*] Initializing Chrome allocator...")
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), config.GetOpts()...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, config.DeadLine)
	defer cancel()

	fmt.Println("[*] Performing pre-flight check...")
	if err := chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(scraper.ConfigJS).Do(ctx)
			return err
		}),
		chromedp.Navigate("about:blank"),
	); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	var (
		allSkins  []config.Skin
		allAgents []config.Agent
		allMisc   []config.Skin
	)

	handleBatch := func(items []config.Item) {
		if len(items) == 0 {
			return
		}
		var bSkins []config.Skin
		var bAgents []config.Agent
		var bMisc []config.Skin

		for _, it := range items {
			cat := strings.ToLower(it.Category)
			// Strictly route items to correct slices based on category
			if cat == "agents" || strings.Contains(cat, "agent") {
				a := internal.ConvertToAgent(it)
				allAgents = append(allAgents, a)
				bAgents = append(bAgents, a)
			} else if cat == "weapons" || cat == "gloves" || strings.Contains(cat, "skin") {
				s := internal.ConvertToSkin(it)
				allSkins = append(allSkins, s)
				bSkins = append(bSkins, s)
			} else {
				// Cases, Stickers, Souvenirs, Pins, Patches go to misc
				m := internal.ConvertToSkin(it)
				// If no prices were scraped, still save the item with basic info
				if len(m.Prices) == 0 && m.Price.PriceString == "" {
					m.Name = it.Name
					m.URL = it.URL
					m.Type = it.Type
				}
				allMisc = append(allMisc, m)
				bMisc = append(bMisc, m)
			}
		}
		if progress != nil {
			progress(bSkins, bAgents, bMisc)
		}
	}

	onlySet := isAnyOnlyFlagSet()

	if config.WeaponsOnly || !onlySet {
		handleBatch(runWeaponsFlow(browserCtx))
	}
	if config.GlovesOnly || !onlySet {
		handleBatch(runGenericFlow(browserCtx, "gloves", "Glove", config.MaxGloves))
	}
	if config.CasesOnly || !onlySet {
		handleBatch(runGenericFlow(browserCtx, "cases", "Case", config.MaxCases))
	}
	if config.AgentsOnly || !onlySet {
		handleBatch(runGenericFlow(browserCtx, "agents", "Agent", config.MaxAgents))
	}
	if config.SouvenirsOnly || !onlySet {
		handleBatch(runGenericFlow(browserCtx, "souvenir-packages", "Souvenir", config.MaxSouvenirs))
	}
	if config.PatchesOnly || !onlySet {
		handleBatch(runGenericFlow(browserCtx, "patches", "Patch", config.MaxPatches))
	}
	if config.PinsOnly || !onlySet {
		handleBatch(runNestedFlow(browserCtx, "collectible-pins", "Pin", config.MaxPins))
	}
	if config.StickersOnly || !onlySet {
		handleBatch(runNestedFlow(browserCtx, "sticker-capsules", "Sticker", config.MaxStickers))
	}

	return allSkins, allAgents, allMisc, nil
}

func isAnyOnlyFlagSet() bool {
	return config.WeaponsOnly || config.CasesOnly || config.GlovesOnly || config.AgentsOnly || config.SouvenirsOnly || config.PinsOnly || config.PatchesOnly || config.StickersOnly
}

func runWeaponsFlow(ctx context.Context) []config.Item {
	// First discover all weapons from the weapons page
	weapons, err := discovery.Discover(ctx, discovery.Options{Category: "weapons"})
	if err != nil {
		fmt.Printf("      [!] Error discovering weapons: %v\n", err)
	}

	// Random delay after discovering weapon list
	randomDelayMs(300, 600)

	var allItems []config.Item
	weaponsProcessed := 0
	for _, w := range weapons {
		// MaxWeapons limits the number of weapon categories checked
		if config.MaxWeapons > 0 && weaponsProcessed >= config.MaxWeapons { break }

		// Random delay before scraping each weapon
		randomDelayMs(300, 600)

		// Navigate to each weapon page and discover skins
		skins, err := discovery.Discover(ctx, discovery.Options{Category: w.URL})
		if err != nil {
			fmt.Printf("      [!] Error discovering skins for %s: %v\n", w.Name, err)
			continue
		}

		var opts []common.FetchOptions
		for _, s := range skins {
			if isSeen(s.URL) || (s.Type != "skin" && s.Type != "skins") { continue }
			opts = append(opts, common.FetchOptions{URL: s.URL, Name: s.Name, Category: "weapons", HasWear: true, Weapon: w.Name})
		}

		if len(opts) > 0 {
			fetched := common.FetchManyItems(ctx, opts)
			allItems = append(allItems, fetched...)
		}

		// Random delay after finishing this weapon (before next)
		randomDelayMs(300, 600)
		weaponsProcessed++
	}

	return allItems
}

func runGenericFlow(parent context.Context, path, itemType string, limit int) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()
	found, _ := discovery.Discover(ictx, discovery.Options{Category: path})

	// Random delay after discovering items list
	randomDelayMs(300, 600)

	var opts []common.FetchOptions
	for _, it := range found {
		if limit > 0 && len(opts) >= limit { break }
		if isSeen(it.URL) { continue }
		opts = append(opts, common.FetchOptions{URL: it.URL, Name: it.Name, Category: path, Type: itemType, HasWear: path == "gloves"})
	}
	if len(opts) == 0 { return nil }

	result := common.FetchManyItems(parent, opts)

	// Random delay after fetching
	randomDelayMs(300, 600)
	return result
}

func runNestedFlow(parent context.Context, path, itemType string, limit int) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()
	capsules, _ := discovery.Discover(ictx, discovery.Options{Category: path})

	// Random delay after discovering capsules
	randomDelayMs(300, 600)

	var all []config.Item
	for _, cap := range capsules {
		// Respect limit for capsules (for stickers, pins, etc.)
		if limit > 0 && len(all) >= limit { break }

		// Random delay before scraping capsule
		randomDelayMs(300, 600)

		// 1. Scrape the capsule itself
		if !isSeen(cap.URL) {
			res, err := common.FetchItem(ictx, common.FetchOptions{URL: cap.URL, Name: cap.Name, Category: path, Type: itemType + " Capsule"})
			if err == nil && len(res) > 0 {
				all = append(all, res...)
			}
		}

		if limit > 0 && len(all) >= limit { break }

		// Random delay after capsule fetch
		randomDelayMs(300, 600)

		// 2. Scrape items inside
		inside, _ := discovery.Discover(ictx, discovery.Options{Category: cap.URL})
		var opts []common.FetchOptions
		for _, it := range inside {
			if limit > 0 && (len(all)+len(opts)) >= limit { break }
			if it.URL == cap.URL || isSeen(it.URL) { continue }

			// Strict filter to ensure we stay within the category
			if path == "sticker-capsules" {
				if !strings.Contains(it.URL, "/sticker-capsules/") && !strings.Contains(it.URL, "/stickers/") {
					continue
				}
			} else if path == "collectible-pins" {
				if !strings.Contains(it.URL, "/collectible-pins/") {
					continue
				}
			}

			opts = append(opts, common.FetchOptions{URL: it.URL, Name: it.Name, Category: path, Type: itemType})
		}

		if len(opts) > 0 {
			all = append(all, common.FetchManyItems(parent, opts)...)
		}

		// Random delay between capsules
		randomDelayMs(300, 600)
	}
	// Final trim to be absolutely sure about the limit
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// ScrapeSkins (compatibility with main.go progress check)
func ScrapeSkinsWithProgress(p func([]config.Skin)) ([]config.Skin, []config.Agent, error) {
	s, a, m, err := ScrapeAll(func(skins []config.Skin, agents []config.Agent, misc []config.Skin) {
		if p != nil { p(skins) }
	})
	return append(s, m...), a, err
}
