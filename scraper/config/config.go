// main configuration file where you can set options for scraper.
package config

import (
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// Scraper-wide settings.
var (
	Target   = "https://www.csgodatabase.com"
	DeadLine = 60 * time.Minute
	Delay    = 2500 * time.Millisecond
	Workers  = 2
	Headless = true
)

// GetOpts returns optimized chromedp allocator options for CI/Docker environments.
func GetOpts() []chromedp.ExecAllocatorOption {
	// Create a unique temp directory for each run to avoid permission issues in CI
	tmpDir, _ := os.MkdirTemp("", "chrome-profile-*")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("single-process", true),
		chromedp.Flag("user-data-dir", tmpDir),
		chromedp.Flag("window-size", "1280,1080"),
	)

	// If CHROME_PATH is explicitly set
	if path := os.Getenv("CHROME_PATH"); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	} else {
		// Fallback for standard Ubuntu GHA runner paths
		lookIn := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
		}
		for _, p := range lookIn {
			if _, err := os.Stat(p); err == nil {
				opts = append(opts, chromedp.ExecPath(p))
				break
			}
		}
	}

	return opts
}

type Category struct {
	Slug        string
	DisplayName string
	HasWear     bool
	HasWeapon   bool
	SlugList    []string
}

var AllCategories = []Category{
	{
		Slug:        "weapons",
		DisplayName: "Weapons",
		HasWear:     true,
		HasWeapon:   true,
		SlugList:    nil,
	},
}

func CategoryBySlug(slug string) (Category, bool) {
	for _, c := range AllCategories {
		if c.Slug == slug {
			return c, true
		}
	}
	return Category{}, false
}
