// main configuration file where you can set options for scraper.
package config

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"
)

var (
	Target          = "https://www.csgodatabase.com"
	DeadLine        = 120 * time.Minute
	Delay           = 2 * time.Second
	Workers         = 2

	// Limits
	Max          = 0 // Universal limit
	MaxWeapons   = 0
	MaxGloves    = 0
	MaxAgents    = 0
	MaxCases     = 0
	MaxSouvenirs = 0
	MaxPins      = 0
	MaxPatches   = 0
	MaxStickers  = 0

	// Category Flags
	WeaponsOnly   = false
	CasesOnly     = false
	GlovesOnly    = false
	AgentsOnly    = false
	SouvenirsOnly = false
	PinsOnly      = false
	PatchesOnly   = false
	StickersOnly  = false

	Headless    = false
	Interactive = false
)

func NextDelay() time.Duration {
	if Delay <= 0 {
		return 0
	}
	extra := time.Duration(rand.Intn(2000)) * time.Millisecond
	return Delay + extra
}

func GetOpts() []chromedp.ExecAllocatorOption {
	profileDir := os.Getenv("CHROME_USER_DATA_DIR")
	if profileDir == "" && Interactive {
		profileDir = "chrome-profile"
	}
	if profileDir == "" {
		profileDir, _ = os.MkdirTemp("", "chrome-profile-*")
	}
	if absoluteDir, err := filepath.Abs(profileDir); err == nil {
		profileDir = absoluteDir
	}
	if Interactive {
		_ = os.MkdirAll(profileDir, 0755)
	}

	headless := Headless || os.Getenv("CI") == "true"
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.Flag("exclude-switches", "enable-automation"),
		chromedp.Flag("disable-extensions", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-background-mode", true),
		chromedp.Flag("new-window", true),
		chromedp.Flag("start-maximized", true),
		chromedp.Flag("headless", headless),
		chromedp.Flag("user-data-dir", profileDir),
		chromedp.Flag("window-size", "1280,1024"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"),
	)
	if proxy := os.Getenv("CSGO_PROXY"); proxy != "" {
		opts = append(opts, chromedp.ProxyServer(proxy))
	}
	if path := os.Getenv("CHROME_PATH"); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	} else {
		lookIn := []string{"/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium-browser", "/usr/bin/chromium"}
		if runtime.GOOS == "windows" {
			lookIn = []string{
				`C:\Program Files\Google\Chrome\Application\chrome.exe`,
				`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
				`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			}
		}
		for _, path := range lookIn {
			if _, err := os.Stat(path); err == nil {
				opts = append(opts, chromedp.ExecPath(path))
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
	},
}

func CategoryBySlug(slug string) (Category, bool) {
	for _, category := range AllCategories {
		if category.Slug == slug {
			return category, true
		}
	}
	return Category{}, false
}
