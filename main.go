package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/worker"
)

func main() {
	// cli flags
	aggressive := flag.Bool("aggressive", false, "Run scraper aggressively")
	stealth := flag.Bool("stealth", false, "Run scraper in stealth mode")
	interactive := flag.Bool("interactive", false, "Pause for manual Cloudflare verification")
	delaySeconds := flag.Int("delay", int(config.Delay/time.Second), "Delay between page requests in seconds")
	workers := flag.Int("workers", config.Workers, "Number of concurrent item pages")

	// Limits
	maxAll := flag.Int("max", 0, "Universal limit for items per category")
	flag.IntVar(&config.MaxWeapons, "max-weapons", 0, "Limit weapon count")
	flag.IntVar(&config.MaxGloves, "max-gloves", 0, "Limit glove count")
	flag.IntVar(&config.MaxAgents, "max-agents", 0, "Limit agent count")
	flag.IntVar(&config.MaxCases, "max-cases", 0, "Limit case count")
	flag.IntVar(&config.MaxSouvenirs, "max-souvenirs", 0, "Limit souvenir count")
	flag.IntVar(&config.MaxPins, "max-pins", 0, "Limit pins count")
	flag.IntVar(&config.MaxPatches, "max-patches", 0, "Limit patches count")
	flag.IntVar(&config.MaxStickers, "max-stickers", 0, "Limit sticker capsules count")

	// Category Flags
	flag.BoolVar(&config.WeaponsOnly, "weapons-only", false, "Scrape only weapons")
	flag.BoolVar(&config.CasesOnly, "cases-only", false, "Scrape only cases")
	flag.BoolVar(&config.GlovesOnly, "gloves-only", false, "Scrape only gloves")
	flag.BoolVar(&config.AgentsOnly, "agents-only", false, "Scrape only agents")
	flag.BoolVar(&config.SouvenirsOnly, "souvenirs-only", false, "Scrape only souvenirs")
	flag.BoolVar(&config.PinsOnly, "pins-only", false, "Scrape only pins")
	flag.BoolVar(&config.PatchesOnly, "patches-only", false, "Scrape only patches")
	flag.BoolVar(&config.StickersOnly, "stickers-only", false, "Scrape only stickers and capsules")

	flag.Parse()

	if *delaySeconds >= 0 {
		config.Delay = time.Duration(*delaySeconds) * time.Second
	}
	if *workers > 0 {
		config.Workers = *workers
	}

	// Apply universal max
	if *maxAll > 0 {
		config.Max = *maxAll
		if config.MaxWeapons == 0 { config.MaxWeapons = config.Max }
		if config.MaxGloves == 0 { config.MaxGloves = config.Max }
		if config.MaxAgents == 0 { config.MaxAgents = config.Max }
		if config.MaxCases == 0 { config.MaxCases = config.Max }
		if config.MaxSouvenirs == 0 { config.MaxSouvenirs = config.Max }
		if config.MaxPins == 0 { config.MaxPins = config.Max }
		if config.MaxPatches == 0 { config.MaxPatches = config.Max }
		if config.MaxStickers == 0 { config.MaxStickers = config.Max }
	}

	if *aggressive { config.Delay = 0 }
	if *stealth { config.Delay += time.Duration(500+rand.Intn(1500)) * time.Millisecond }
	config.Interactive = *interactive

	fmt.Println("[*] Starting scraper...")

	var (
		dataMu sync.Mutex
		skins  []config.Skin
		agents []config.Agent
		misc   []config.Skin
	)

	savePartial := func(bSkins []config.Skin, bAgents []config.Agent, bMisc []config.Skin) {
		dataMu.Lock()
		defer dataMu.Unlock()
		skins = append(skins, bSkins...)
		agents = append(agents, bAgents...)
		misc = append(misc, bMisc...)
		_ = saveData(config.DataOutput{Skins: skins, Agents: agents, Misc: misc})
		fmt.Printf("\033[36m[*]\033[0m Progress saved. Skins: %d, Agents: %d, Misc: %d\n", len(skins), len(agents), len(misc))
	}

	resSkins, resAgents, resMisc, err := worker.ScrapeAll(savePartial)
	if err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error: %v", err)
	}

	if len(resSkins) == 0 && len(resAgents) == 0 && len(resMisc) == 0 {
		log.Fatalf("\033[31m[!]\033[0m No items were scraped!")
	}

	if err := saveData(config.DataOutput{Skins: resSkins, Agents: resAgents, Misc: resMisc}); err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error writing file: %v", err)
	}

	fmt.Println("\n\033[32m[+]\033[0m Done. Results in json/data.json")
}

func saveData(data config.DataOutput) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil { return err }
	_ = os.MkdirAll("json", 0755)
	return os.WriteFile("json/data.json", jsonData, 0644)
}
