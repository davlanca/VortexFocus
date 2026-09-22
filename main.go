package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/worker"
)

func main() {
	// =========================================================
	// CLI FLAGS
	// =========================================================

	aggressive := flag.Bool(
		"aggressive",
		false,
		"Run scraper aggressively",
	)

	stealth := flag.Bool(
		"stealth",
		false,
		"Run scraper in stealth mode",
	)

	interactive := flag.Bool(
		"interactive",
		false,
		"Pause for manual Cloudflare verification",
	)

	delaySeconds := flag.Int(
		"delay",
		int(config.Delay/time.Second),
		"Delay between page requests in seconds",
	)

	workers := flag.Int(
		"workers",
		config.Workers,
		"Number of concurrent item pages",
	)

	// =========================================================
	// LIMITS
	// =========================================================

	maxAll := flag.Int(
		"max",
		0,
		"Universal limit for items per category",
	)

	flag.IntVar(
		&config.MaxWeapons,
		"max-weapons",
		0,
		"Limit weapon count",
	)

	flag.IntVar(
		&config.MaxGloves,
		"max-gloves",
		0,
		"Limit glove count",
	)

	flag.IntVar(
		&config.MaxAgents,
		"max-agents",
		0,
		"Limit agent count",
	)

	flag.IntVar(
		&config.MaxCases,
		"max-cases",
		0,
		"Limit case count",
	)

	flag.IntVar(
		&config.MaxSouvenirs,
		"max-souvenirs",
		0,
		"Limit souvenir count",
	)

	flag.IntVar(
		&config.MaxPins,
		"max-pins",
		0,
		"Limit pins count",
	)

	flag.IntVar(
		&config.MaxPatches,
		"max-patches",
		0,
		"Limit patch count",
	)

	flag.IntVar(
		&config.MaxStickers,
		"max-stickers",
		0,
		"Limit sticker capsules count",
	)

	// =========================================================
	// CATEGORY FLAGS
	// =========================================================

	flag.BoolVar(
		&config.WeaponsOnly,
		"weapons-only",
		false,
		"Scrape only weapons",
	)

	flag.BoolVar(
		&config.CasesOnly,
		"cases-only",
		false,
		"Scrape only cases",
	)

	flag.BoolVar(
		&config.GlovesOnly,
		"gloves-only",
		false,
		"Scrape only gloves",
	)

	flag.BoolVar(
		&config.AgentsOnly,
		"agents-only",
		false,
		"Scrape only agents",
	)

	flag.BoolVar(
		&config.SouvenirsOnly,
		"souvenirs-only",
		false,
		"Scrape only souvenirs",
	)

	flag.BoolVar(
		&config.PinsOnly,
		"pins-only",
		false,
		"Scrape only pins",
	)

	flag.BoolVar(
		&config.PatchesOnly,
		"patches-only",
		false,
		"Scrape only patches",
	)

	flag.BoolVar(
		&config.StickersOnly,
		"stickers-only",
		false,
		"Scrape only stickers and capsules",
	)

	flag.Parse()

	// =========================================================
	// APPLY CLI SETTINGS
	// =========================================================

	if *delaySeconds >= 0 {
		config.Delay = time.Duration(*delaySeconds) * time.Second
	}

	if *workers > 0 {
		config.Workers = *workers
	}

	// =========================================================
	// UNIVERSAL MAX
	// =========================================================

	if *maxAll > 0 {
		config.Max = *maxAll

		if config.MaxWeapons == 0 {
			config.MaxWeapons = config.Max
		}

		if config.MaxGloves == 0 {
			config.MaxGloves = config.Max
		}

		if config.MaxAgents == 0 {
			config.MaxAgents = config.Max
		}

		if config.MaxCases == 0 {
			config.MaxCases = config.Max
		}

		if config.MaxSouvenirs == 0 {
			config.MaxSouvenirs = config.Max
		}

		if config.MaxPins == 0 {
			config.MaxPins = config.Max
		}

		if config.MaxPatches == 0 {
			config.MaxPatches = config.Max
		}

		if config.MaxStickers == 0 {
			config.MaxStickers = config.Max
		}
	}

	// =========================================================
	// MODES
	// =========================================================

	if *aggressive {
		config.Delay = 0
	}

	if *stealth {
		config.Delay += time.Duration(
			500+rand.Intn(1500),
		) * time.Millisecond
	}

	config.Interactive = *interactive

	fmt.Println("[*] Starting scraper...")

	// =========================================================
	// SHARED RESULT STORAGE
	// =========================================================

	var (
		dataMu sync.Mutex

		skins  []config.Skin
		agents []config.Agent
		misc   []config.Skin
	)

	// =========================================================
	// PARTIAL SAVE
	// =========================================================

	savePartial := func(
		bSkins []config.Skin,
		bAgents []config.Agent,
		bMisc []config.Skin,
	) {
		dataMu.Lock()
		defer dataMu.Unlock()

		skins = append(skins, bSkins...)
		agents = append(agents, bAgents...)
		misc = append(misc, bMisc...)

		data := config.DataOutput{
			Skins:  skins,
			Agents: agents,
			Misc:   misc,
		}

		if err := saveData(data); err != nil {
			log.Printf(
				"\033[31m[!] Failed to save progress: %v\033[0m",
				err,
			)
			return
		}

		fmt.Printf(
			"\033[36m[*]\033[0m Progress saved. Skins: %d, Agents: %d, Misc: %d\n",
			len(skins),
			len(agents),
			len(misc),
		)
	}

	// =========================================================
	// RUN SCRAPER
	// =========================================================

	resSkins, resAgents, resMisc, err := worker.ScrapeAll(savePartial)

	if err != nil {
		log.Fatalf(
			"\033[31m[!]\033[0m Error: %v",
			err,
		)
	}

	// =========================================================
	// EMPTY RESULT CHECK
	// =========================================================

	if len(resSkins) == 0 &&
		len(resAgents) == 0 &&
		len(resMisc) == 0 {

		log.Fatalf(
			"\033[31m[!]\033[0m No items were scraped!",
		)
	}

	// =========================================================
	// FINAL SAVE
	// =========================================================

	finalData := config.DataOutput{
		Skins:  resSkins,
		Agents: resAgents,
		Misc:   resMisc,
	}

	if err := saveData(finalData); err != nil {
		log.Fatalf(
			"\033[31m[!]\033[0m Error writing file: %v",
			err,
		)
	}

	fmt.Println("")
	fmt.Println("\033[32m[+]\033[0m Done.")
	fmt.Println("\033[36m    -> json/data.json\033[0m")
	fmt.Println("\033[36m    -> json/data.js\033[0m")

}

// =============================================================
// SAVE DATABASE
// =============================================================
//
// Creates:
//
//   json/data.json
//   json/data.js
//
// data.js is intentionally generated as a global JavaScript
// variable so index.html can be opened directly through
// file:// without fetch/CORS problems.
//
// =============================================================

func saveData(data config.DataOutput) error {
	// ---------------------------------------------------------
	// Ensure output directory exists
	// ---------------------------------------------------------

	if err := os.MkdirAll("json", 0755); err != nil {
		return fmt.Errorf("create json directory: %w", err)
	}

	// ---------------------------------------------------------
	// Marshal JSON
	// ---------------------------------------------------------

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	// ---------------------------------------------------------
	// Save JSON database
	// ---------------------------------------------------------

	if err := os.WriteFile(
		"json/data.json",
		jsonData,
		0644,
	); err != nil {
		return fmt.Errorf("write json/data.json: %w", err)
	}

	// ---------------------------------------------------------
	// Prepare JavaScript database
	// ---------------------------------------------------------
	//
	// encoding/json already escapes characters such as "<",
	// ">" and "&" when producing JSON.
	//
	// NewReplacer below additionally guarantees that the data
	// cannot accidentally form a closing </script> sequence
	// when embedded into a browser page.
	// ---------------------------------------------------------

	safeJSData := strings.NewReplacer(
		"<", "\\u003c",
		">", "\\u003e",
		"&", "\\u0026",
	).Replace(string(jsonData))

	jsData := "window.CS2_DATA = " + safeJSData + ";\n"

	// ---------------------------------------------------------
	// Save JavaScript database
	// ---------------------------------------------------------

	if err := os.WriteFile(
		"json/data.js",
		[]byte(jsData),
		0644,
	); err != nil {
		return fmt.Errorf("write json/data.js: %w", err)
	}

	return nil

}
