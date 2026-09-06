// main configuration file where you can set options for scraper.
package config

import (
	"time"

	"github.com/chromedp/chromedp"
)

// Scraper-wide settings.
var (
	Target   = "https://www.csgodatabase.com"
	DeadLine = 60 * time.Minute // Increased for deep weapon/skin scraping
	Delay    = 1500 * time.Millisecond
	Workers  = 4
	Headless = true
)

// Opts are the chromedp allocator flags.
var Opts = append(chromedp.DefaultExecAllocatorOptions[:],
	chromedp.NoSandbox,
	chromedp.DisableGPU,
	chromedp.Flag("disable-setuid-sandbox", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("disable-blink-features", "AutomationControlled"),
	chromedp.Flag("blink-settings", "imagesEnabled=false"),
	chromedp.Flag("exclude-switches", "enable-automation"),
	chromedp.Flag("disable-extensions", true),
	chromedp.Flag("start-maximized", false),
	chromedp.Flag("window-size", "800,600"),
	chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"),
)

// Category describes a scraper category.
type Category struct {
	Slug        string
	DisplayName string
	HasWear     bool
	HasWeapon   bool
	SlugList    []string
}

// AllCategories now focuses exclusively on Weapons.
var AllCategories = []Category{
	{
		Slug:        "weapons",
		DisplayName: "Weapons",
		HasWear:     true,
		HasWeapon:   true,
		SlugList:    nil,
	},
}

// CategoryBySlug returns the Category struct for the given URL slug.
func CategoryBySlug(slug string) (Category, bool) {
	for _, c := range AllCategories {
		if c.Slug == slug {
			return c, true
		}
	}
	return Category{}, false
}
