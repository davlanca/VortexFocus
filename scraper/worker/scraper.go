// Package worker is the top-level orchestrator.
package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/eovacius/csgodatabase-scraper/internal"
	"github.com/eovacius/csgodatabase-scraper/scraper"
	"github.com/eovacius/csgodatabase-scraper/scraper/common"
	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/discovery"
)

var (
	seenMu   sync.Mutex
	seenURLs = make(map[string]bool)
)

func resetSeenURLs() {
	seenMu.Lock()
	defer seenMu.Unlock()

	seenURLs = make(map[string]bool)
}

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
	resetSeenURLs()

	fmt.Println("[*] Initializing Chrome allocator...")

	allocCtx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		config.GetOpts()...,
	)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(
		browserCtx,
		config.DeadLine,
	)
	defer cancel()

	fmt.Println("[*] Performing pre-flight check...")

	if err := chromedp.Run(
		browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(
				scraper.ConfigJS,
			).Do(ctx)

			return err
		}),
		chromedp.Navigate("about:blank"),
	); err != nil {
		return nil, nil, nil,
			fmt.Errorf("failed to start Chrome: %w", err)
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

		var (
			bSkins  []config.Skin
			bAgents []config.Agent
			bMisc   []config.Skin
		)

		for _, it := range items {
			cat := strings.ToLower(it.Category)

			if cat == "agents" ||
				strings.Contains(cat, "agent") {

				a := internal.ConvertToAgent(it)

				allAgents = append(allAgents, a)
				bAgents = append(bAgents, a)

			} else if cat == "weapons" ||
				cat == "gloves" ||
				strings.Contains(cat, "skin") {

				s := internal.ConvertToSkin(it)

				allSkins = append(allSkins, s)
				bSkins = append(bSkins, s)

			} else {

				m := internal.ConvertToSkin(it)

				/*
				 * Keep the item even if prices are empty.
				 * This is useful for diagnosing pages where
				 * discovery succeeded but price scraping failed.
				 */
				if len(m.Prices) == 0 &&
					m.Price.PriceString == "" {

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
		handleBatch(
			runWeaponsFlow(browserCtx),
		)
	}

	if config.GlovesOnly || !onlySet {
		handleBatch(
			runGenericFlow(
				browserCtx,
				"gloves",
				"Glove",
				config.MaxGloves,
			),
		)
	}

	if config.CasesOnly || !onlySet {
		handleBatch(
			runGenericFlow(
				browserCtx,
				"cases",
				"Case",
				config.MaxCases,
			),
		)
	}

	if config.AgentsOnly || !onlySet {
		handleBatch(
			runGenericFlow(
				browserCtx,
				"agents",
				"Agent",
				config.MaxAgents,
			),
		)
	}

	if config.SouvenirsOnly || !onlySet {
		handleBatch(
			runGenericFlow(
				browserCtx,
				"souvenir-packages",
				"Souvenir",
				config.MaxSouvenirs,
			),
		)
	}

	if config.PatchesOnly || !onlySet {
		handleBatch(
			runGenericFlow(
				browserCtx,
				"patches",
				"Patch",
				config.MaxPatches,
			),
		)
	}

	if config.PinsOnly || !onlySet {
		handleBatch(
			runNestedFlow(
				browserCtx,
				"collectible-pins",
				"Pin",
				config.MaxPins,
			),
		)
	}

	if config.StickersOnly || !onlySet {
		handleBatch(
			runNestedFlow(
				browserCtx,
				"sticker-capsules",
				"Sticker",
				config.MaxStickers,
			),
		)
	}

	return allSkins, allAgents, allMisc, nil
}

func isAnyOnlyFlagSet() bool {
	return config.WeaponsOnly ||
		config.CasesOnly ||
		config.GlovesOnly ||
		config.AgentsOnly ||
		config.SouvenirsOnly ||
		config.PinsOnly ||
		config.PatchesOnly ||
		config.StickersOnly
}

func runWeaponsFlow(parent context.Context) []config.Item {
	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	weapons, err := discovery.Discover(
		ictx,
		discovery.Options{
			Category: "weapons",
		},
	)

	if err != nil {
		fmt.Printf(
			"\033[31m[!]\033[0m Weapon discovery error: %v\n",
			err,
		)
		return nil
	}

	fmt.Printf(
		"[*] Found %d weapon pages\n",
		len(weapons),
	)

	var out []config.Item

	for _, w := range weapons {

		if config.MaxWeapons > 0 &&
			len(out) >= config.MaxWeapons {
			break
		}

		fmt.Printf(
			"   -> Discovering skins for: %s\n",
			w.Name,
		)

		skins, err := discovery.Discover(
			ictx,
			discovery.Options{
				Category: w.URL,
			},
		)

		if err != nil {
			fmt.Printf(
				"\033[31m[!]\033[0m Skin discovery error for %s: %v\n",
				w.Name,
				err,
			)
			continue
		}

		fmt.Printf(
			"      [*] Found %d skin links for %s\n",
			len(skins),
			w.Name,
		)

		if len(skins) == 0 {
			fmt.Printf(
				"\033[33m[?]\033[0m No skins discovered for %s (%s)\n",
				w.Name,
				w.URL,
			)
			continue
		}

		var opts []common.FetchOptions

		for _, s := range skins {

			if config.MaxWeapons > 0 &&
				(len(out)+len(opts)) >= config.MaxWeapons {
				break
			}

			/*
			 * We only want actual skin pages here.
			 */
			if s.Type != "skin" {
				continue
			}

			if isSeen(s.URL) {
				continue
			}

			opts = append(
				opts,
				common.FetchOptions{
					URL:      s.URL,
					Name:     s.Name,
					Category: "weapons",
					HasWear:  true,
					Weapon:   w.Name,
				},
			)
		}

		if len(opts) == 0 {
			continue
		}

		results := common.FetchManyItems(
			parent,
			opts,
		)

		if len(results) == 0 {
			fmt.Printf(
				"\033[33m[?]\033[0m No prices scraped for %s\n",
				w.Name,
			)
			continue
		}

		out = append(out, results...)
	}

	fmt.Printf(
		"[*] Weapon scraping produced %d items\n",
		len(out),
	)

	return out
}

func runGenericFlow(
	parent context.Context,
	path string,
	itemType string,
	limit int,
) []config.Item {

	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	found, err := discovery.Discover(
		ictx,
		discovery.Options{
			Category: path,
		},
	)

	if err != nil {
		fmt.Printf(
			"\033[31m[!]\033[0m Discovery error for %s: %v\n",
			path,
			err,
		)
		return nil
	}

	fmt.Printf(
		"[*] %s discovery found %d items\n",
		path,
		len(found),
	)

	var opts []common.FetchOptions

	for _, it := range found {

		if limit > 0 &&
			len(opts) >= limit {
			break
		}

		if isSeen(it.URL) {
			continue
		}

		opts = append(
			opts,
			common.FetchOptions{
				URL:      it.URL,
				Name:     it.Name,
				Category: path,
				Type:     itemType,
				HasWear:  path == "gloves",
			},
		)
	}

	if len(opts) == 0 {
		fmt.Printf(
			"\033[33m[?]\033[0m No scrape targets for %s\n",
			path,
		)
		return nil
	}

	return common.FetchManyItems(
		parent,
		opts,
	)
}

func runNestedFlow(
	parent context.Context,
	path string,
	itemType string,
	limit int,
) []config.Item {

	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	capsules, err := discovery.Discover(
		ictx,
		discovery.Options{
			Category: path,
		},
	)

	if err != nil {
		fmt.Printf(
			"\033[31m[!]\033[0m Nested discovery error for %s: %v\n",
			path,
			err,
		)
		return nil
	}

	var all []config.Item

	for _, cap := range capsules {

		if limit > 0 &&
			len(all) >= limit {
			break
		}

		/*
		 * 1. Scrape the capsule itself.
		 */
		if !isSeen(cap.URL) {

			res, err := common.FetchItem(
				ictx,
				common.FetchOptions{
					URL:      cap.URL,
					Name:     cap.Name,
					Category: path,
					Type:     itemType + " Capsule",
				},
			)

			if err != nil {
				fmt.Printf(
					"\033[33m[?]\033[0m Failed to scrape %s: %v\n",
					cap.Name,
					err,
				)
			} else {
				all = append(all, res...)
			}
		}

		if limit > 0 &&
			len(all) >= limit {
			break
		}

		/*
		 * 2. Scrape items inside the capsule.
		 */
		inside, err := discovery.Discover(
			ictx,
			discovery.Options{
				Category: cap.URL,
			},
		)

		if err != nil {
			fmt.Printf(
				"\033[33m[?]\033[0m Nested discovery error for %s: %v\n",
				cap.Name,
				err,
			)
			continue
		}

		var opts []common.FetchOptions

		for _, it := range inside {

			if limit > 0 &&
				(len(all)+len(opts)) >= limit {
				break
			}

			if it.URL == cap.URL ||
				isSeen(it.URL) {
				continue
			}

			if path == "sticker-capsules" {

				if !strings.Contains(
					it.URL,
					"/sticker-capsules/",
				) &&
					!strings.Contains(
						it.URL,
						"/stickers/",
					) {
					continue
				}

			} else if path == "collectible-pins" {

				if !strings.Contains(
					it.URL,
					"/collectible-pins/",
				) {
					continue
				}
			}

			opts = append(
				opts,
				common.FetchOptions{
					URL:      it.URL,
					Name:     it.Name,
					Category: path,
					Type:     itemType,
				},
			)
		}

		if len(opts) > 0 {
			all = append(
				all,
				common.FetchManyItems(
					parent,
					opts,
				)...,
			)
		}
	}

	if limit > 0 &&
		len(all) > limit {
		all = all[:limit]
	}

	return all
}

// ScrapeSkinsWithProgress is kept for compatibility with main.go.
func ScrapeSkinsWithProgress(
	p func([]config.Skin),
) ([]config.Skin, []config.Agent, error) {

	s, a, m, err := ScrapeAll(
		func(
			skins []config.Skin,
			agents []config.Agent,
			misc []config.Skin,
		) {
			if p != nil {
				p(skins)
			}
		},
	)

	return append(s, m...), a, err
}
