package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/eovacius/csgodatabase-scraper/scraper/config"
	"github.com/eovacius/csgodatabase-scraper/scraper/worker"
)

func main() {
	// cli flags
	aggressive := flag.Bool("aggressive", false, "Run scraper aggressively (less delay)")
	stealth := flag.Bool("stealth", false, "Run scraper in stealth mode (randomized and more human like delay)")
	interactive := flag.Bool("interactive", false, "Pause for manual Cloudflare verification in visible Chrome")
	delaySeconds := flag.Int("delay", int(config.Delay/time.Second), "Delay between page requests in seconds")
	workers := flag.Int("workers", config.Workers, "Number of concurrent item pages")
	maxWeapons := flag.Int("max-weapons", config.MaxWeapons, "Limit weapon count for a partial test run; 0 means all")

	flag.Parse()

	if *delaySeconds >= 0 {
		config.Delay = time.Duration(*delaySeconds) * time.Second
	}
	if *workers > 0 {
		config.Workers = *workers
	}
	if *maxWeapons >= 0 {
		config.MaxWeapons = *maxWeapons
	}
	if *aggressive {
		config.Delay = 0
		fmt.Println("Aggressive mode. Delay:", config.Delay)
	}
	if *stealth {
		randomMs := 500 + rand.Intn(1500)
		config.Delay += time.Duration(randomMs) * time.Millisecond
	}
	config.Interactive = *interactive

	fmt.Println("[*] Starting scraper...")

	skins, agents, err := worker.ScrapeSkins()
	if err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error during scraping: %v", err)
	}

	combinedData := config.DataOutput{
		Skins:  skins,
		Agents: agents,
	}

	if len(skins) == 0 && len(agents) == 0 {
		log.Fatalf("\033[31m[!]\033[0m No items were scraped! Exiting...")
	}

	jsonData, err := json.MarshalIndent(combinedData, "", "  ")
	if err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error marshaling JSON: %v", err)
	}

	filename := "json/data.json"

	// Ensure json directory exists
	if err := os.MkdirAll("json", 0755); err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error creating json folder: %v", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Fatalf("\033[31m[!]\033[0m Error writing to file: %v", err)
	}

	fmt.Println("\n\033[32m[+]\033[0m Done. See files inside json folder")
	config.Interactive = *interactive
}
