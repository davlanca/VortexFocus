package worker

// Package worker is the top-level scraper orchestrator.

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

// =============================================================
// SEEN URL MANAGEMENT
// =============================================================

func resetSeenURLs() {
	seenMu.Lock()
	defer seenMu.Unlock()

	seenURLs = make(map[string]bool)
}

func isSeen(url string) bool {
	if strings.TrimSpace(url) == "" {
		return true
	}

	seenMu.Lock()
	defer seenMu.Unlock()

	if seenURLs[url] {
		return true
	}

	seenURLs[url] = true

	return false
}

// =============================================================
// MAIN SCRAPER
// =============================================================

// ScrapeAll runs the full scraping cycle.
//
// The function returns three separate collections:
//   - skins
//   - agents
//   - misc
//
// The progress callback receives only the newly scraped batch.
// Final results are returned once the complete run finishes.
func ScrapeAll(
	progress func(
		[]config.Skin,
		[]config.Agent,
		[]config.Skin,
	),
) (
	[]config.Skin,
	[]config.Agent,
	[]config.Skin,
	error,
) {

	resetSeenURLs()

	fmt.Println("[*] Initializing Chrome allocator...")

	allocCtx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		config.GetOpts()...,
	)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(
		allocCtx,
	)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(
		browserCtx,
		config.DeadLine,
	)
	defer cancel()

	// =========================================================
	// PRE-FLIGHT CHECK
	// =========================================================

	fmt.Println("[*] Performing pre-flight check...")

	if err := chromedp.Run(
		browserCtx,

		chromedp.ActionFunc(
			func(ctx context.Context) error {
				_, err := page.AddScriptToEvaluateOnNewDocument(
					scraper.ConfigJS,
				).Do(ctx)

				return err
			},
		),

		chromedp.Navigate("about:blank"),
	); err != nil {

		return nil, nil, nil,
			fmt.Errorf(
				"failed to start Chrome: %w",
				err,
			)
	}

	// =========================================================
	// RESULT STORAGE
	// =========================================================

	var (
		allSkins  []config.Skin
		allAgents []config.Agent
		allMisc   []config.Skin
	)

	// =========================================================
	// BATCH HANDLER
	// =========================================================

	handleBatch := func(items []config.Item) {
		if len(items) == 0 {
			return
		}

		var (
			bSkins  []config.Skin
			bAgents []config.Agent
			bMisc   []config.Skin
		)

		for _, item := range items {

			category := strings.ToLower(
				strings.TrimSpace(item.Category),
			)

			// -------------------------------------------------
			// AGENTS
			// -------------------------------------------------

			if category == "agents" ||
				strings.Contains(category, "agent") {

				agent := internal.ConvertToAgent(item)

				allAgents = append(
					allAgents,
					agent,
				)

				bAgents = append(
					bAgents,
					agent,
				)

				continue
			}

			// -------------------------------------------------
			// WEAPONS / GLOVES / SKINS
			// -------------------------------------------------

			if category == "weapons" ||
				category == "gloves" ||
				strings.Contains(category, "skin") {

				skin := internal.ConvertToSkin(item)

				allSkins = append(
					allSkins,
					skin,
				)

				bSkins = append(
					bSkins,
					skin,
				)

				continue
			}

			// -------------------------------------------------
			// MISC
			// -------------------------------------------------

			misc := internal.ConvertToSkin(item)

			/*
			 * Keep the item even if its price information is
			 * empty.
			 *
			 * This is useful for diagnosing pages where
			 * discovery succeeded but price extraction failed.
			 */
			if len(misc.Prices) == 0 &&
				misc.Price.PriceString == "" {

				misc.Name = item.Name
				misc.URL = item.URL
				misc.Type = item.Type
			}

			allMisc = append(
				allMisc,
				misc,
			)

			bMisc = append(
				bMisc,
				misc,
			)
		}

		// -----------------------------------------------------
		// PROGRESS CALLBACK
		// -----------------------------------------------------

		if progress != nil &&
			(len(bSkins) > 0 ||
				len(bAgents) > 0 ||
				len(bMisc) > 0) {

			progress(
				bSkins,
				bAgents,
				bMisc,
			)
		}
	}

	// =========================================================
	// CATEGORY SELECTION
	// =========================================================

	onlySet := isAnyOnlyFlagSet()

	// =========================================================
	// WEAPONS
	// =========================================================

	if config.WeaponsOnly || !onlySet {
		handleBatch(
			runWeaponsFlow(browserCtx),
		)
	}

	// =========================================================
	// GLOVES
	// =========================================================

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

	// =========================================================
	// CASES
	// =========================================================

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

	// =========================================================
	// AGENTS
	// =========================================================

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

	// =========================================================
	// SOUVENIRS
	// =========================================================

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

	// =========================================================
	// PATCHES
	// =========================================================

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

	// =========================================================
	// PINS
	// =========================================================

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

	// =========================================================
	// STICKERS
	// =========================================================

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

	// =========================================================
	// SUMMARY
	// =========================================================

	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("SCRAPE SUMMARY")
	fmt.Println("========================================")
	fmt.Printf("Skins:  %d\n", len(allSkins))
	fmt.Printf("Agents: %d\n", len(allAgents))
	fmt.Printf("Misc:   %d\n", len(allMisc))
	fmt.Println("========================================")

	return allSkins, allAgents, allMisc, nil
}

// =============================================================
// ONLY-FLAG DETECTION
// =============================================================

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

// =============================================================
// WEAPONS FLOW
// =============================================================

func runWeaponsFlow(
	parent context.Context,
) []config.Item {

	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	// ---------------------------------------------------------
	// Discover weapon pages
	// ---------------------------------------------------------

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

	// ---------------------------------------------------------
	// Process every weapon page
	// ---------------------------------------------------------

	for _, weapon := range weapons {

		/*
		 * MaxWeapons historically limits the number of scraped
		 * skin items, not the number of weapon category pages.
		 */
		if config.MaxWeapons > 0 &&
			len(out) >= config.MaxWeapons {
			break
		}

		fmt.Printf(
			"   -> Discovering skins for: %s\n",
			weapon.Name,
		)

		skins, err := discovery.Discover(
			ictx,
			discovery.Options{
				Category: weapon.URL,
			},
		)

		if err != nil {
			fmt.Printf(
				"\033[31m[!]\033[0m Skin discovery error for %s: %v\n",
				weapon.Name,
				err,
			)

			continue
		}

		fmt.Printf(
			"      [*] Found %d skin links for %s\n",
			len(skins),
			weapon.Name,
		)

		if len(skins) == 0 {
			fmt.Printf(
				"\033[33m[?]\033[0m No skins discovered for %s (%s)\n",
				weapon.Name,
				weapon.URL,
			)

			continue
		}

		var opts []common.FetchOptions

		// -----------------------------------------------------
		// Build fetch batch
		// -----------------------------------------------------

		for _, skin := range skins {

			if config.MaxWeapons > 0 &&
				(len(out)+len(opts)) >= config.MaxWeapons {
				break
			}

			// Only actual skin pages.
			if skin.Type != "skin" {
				continue
			}

			if isSeen(skin.URL) {
				continue
			}

			opts = append(
				opts,
				common.FetchOptions{
					URL:      skin.URL,
					Name:     skin.Name,
					Category: "weapons",
					HasWear:  true,
					Weapon:   weapon.Name,
				},
			)
		}

		if len(opts) == 0 {
			continue
		}

		// -----------------------------------------------------
		// Fetch prices
		// -----------------------------------------------------

		results := common.FetchManyItems(
			parent,
			opts,
		)

		if len(results) == 0 {
			fmt.Printf(
				"\033[33m[?]\033[0m No prices scraped for %s\n",
				weapon.Name,
			)

			continue
		}

		out = append(
			out,
			results...,
		)

		fmt.Printf(
			"      [+] Scraped %d items for %s\n",
			len(results),
			weapon.Name,
		)
	}

	fmt.Printf(
		"[*] Weapon scraping produced %d items\n",
		len(out),
	)

	return out
}

// =============================================================
// GENERIC CATEGORY FLOW
// =============================================================

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

	for _, item := range found {

		if limit > 0 &&
			len(opts) >= limit {
			break
		}

		if isSeen(item.URL) {
			continue
		}

		opts = append(
			opts,
			common.FetchOptions{
				URL:      item.URL,
				Name:     item.Name,
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

	results := common.FetchManyItems(
		parent,
		opts,
	)

	fmt.Printf(
		"[*] %s scraping produced %d items\n",
		path,
		len(results),
	)

	return results
}

// =============================================================
// NESTED CATEGORY FLOW
// =============================================================
//
// Used for:
//   - collectible pins
//   - sticker capsules
//
// The category contains containers/capsules, and the actual
// collectible items are discovered inside those containers.
// =============================================================

func runNestedFlow(
	parent context.Context,
	path string,
	itemType string,
	limit int,
) []config.Item {

	ictx, cancel := chromedp.NewContext(parent)
	defer cancel()

	containers, err := discovery.Discover(
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

	fmt.Printf(
		"[*] %s discovery found %d containers\n",
		path,
		len(containers),
	)

	var all []config.Item

	// ---------------------------------------------------------
	// Process containers
	// ---------------------------------------------------------

	for _, container := range containers {

		if limit > 0 &&
			len(all) >= limit {
			break
		}

		// =====================================================
		// 1. SCRAPE CONTAINER ITSELF
		// =====================================================

		if !isSeen(container.URL) {

			results, fetchErr := common.FetchItem(
				ictx,
				common.FetchOptions{
					URL:      container.URL,
					Name:     container.Name,
					Category: path,
					Type:     itemType + " Capsule",
				},
			)

			if fetchErr != nil {
				fmt.Printf(
					"\033[33m[?]\033[0m Failed to scrape %s: %v\n",
					container.Name,
					fetchErr,
				)
			} else {
				all = append(
					all,
					results...,
				)
			}
		}

		if limit > 0 &&
			len(all) >= limit {
			break
		}

		// =====================================================
		// 2. DISCOVER ITEMS INSIDE CONTAINER
		// =====================================================

		inside, discoverErr := discovery.Discover(
			ictx,
			discovery.Options{
				Category: container.URL,
			},
		)

		if discoverErr != nil {
			fmt.Printf(
				"\033[33m[?]\033[0m Nested discovery error for %s\n",
				container.Name,
			)

			continue
		}

		var opts []common.FetchOptions

		for _, item := range inside {

			if limit > 0 &&
				(len(all)+len(opts)) >= limit {
				break
			}

			if item.URL == container.URL {
				continue
			}

			if isSeen(item.URL) {
				continue
			}

			// -------------------------------------------------
			// Sticker validation
			// -------------------------------------------------

			if path == "sticker-capsules" {

				if !strings.Contains(
					item.URL,
					"/sticker-capsules/",
				) &&
					!strings.Contains(
						item.URL,
						"/stickers/",
					) {

					continue
				}
			}

			// -------------------------------------------------
			// Pin validation
			// -------------------------------------------------

			if path == "collectible-pins" {

				if !strings.Contains(
					item.URL,
					"/collectible-pins/",
				) {

					continue
				}
			}

			opts = append(
				opts,
				common.FetchOptions{
					URL:      item.URL,
					Name:     item.Name,
					Category: path,
					Type:     itemType,
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

		all = append(
			all,
			results...,
		)

		fmt.Printf(
			"      [+] %s: scraped %d items from %s\n",
			path,
			len(results),
			container.Name,
		)
	}

	// ---------------------------------------------------------
	// Final safety limit
	// ---------------------------------------------------------

	if limit > 0 &&
		len(all) > limit {

		all = all[:limit]
	}

	fmt.Printf(
		"[*] %s scraping produced %d items\n",
		path,
		len(all),
	)

	return all
}

// =============================================================
// LEGACY COMPATIBILITY
// =============================================================

// ScrapeSkinsWithProgress is kept for compatibility with
// older code that expects the original API.
func ScrapeSkinsWithProgress(
	p func([]config.Skin),
) ([]config.Skin, []config.Agent, error) {

	skins, agents, misc, err := ScrapeAll(
		func(
			batchSkins []config.Skin,
			_ []config.Agent,
			_ []config.Skin,
		) {
			if p != nil {
				p(batchSkins)
			}
		},
	)

	return append(skins, misc...), agents, err
}
