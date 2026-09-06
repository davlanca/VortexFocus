package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// Humanize converts a-slug-style-string to A Slug Style String.
func Humanize(slug string) string {
	out := make([]byte, 0, len(slug)+5)
	upper := true
	for i := 0; i < len(slug); i++ {
		c := slug[i]
		if c == '-' {
			out = append(out, ' ')
			upper = true
			continue
		}
		if upper && c >= 'a' && c <= 'z' {
			out = append(out, c-32)
			upper = false
			continue
		}
		out = append(out, c)
		upper = false
	}
	return string(out)
}

// ConvertToSkin maps a universal Item to the legacy Skin struct.
func ConvertToSkin(item config.Item) config.Skin {
	return config.Skin{
		Name:       item.Name,
		Weapon:     item.Weapon,
		Type:       item.Type,
		Rarity:     item.Rarity,
		Collection: item.Collection,
		URL:        item.URL,
		Price:      SummarizePrices(item.Prices),
		Prices:     item.Prices,
	}
}

// ConvertToAgent maps a universal Item to the legacy Agent struct.
func ConvertToAgent(item config.Item) config.Agent {
	side := "Unknown"
	lowerName := strings.ToLower(item.Name)

	// Better side detection
	ctKeywords := []string{"fbi", "seal", "swat", "sas", "gendarmerie", "ksk", "nswc", "usaf"}
	tKeywords := []string{"guerrilla", "professional", "phoenix", "balkan", "sabotage", "elite crew", "reapers"}

	for _, k := range ctKeywords {
		if strings.Contains(lowerName, k) {
			side = "CT"
			break
		}
	}
	if side == "Unknown" {
		for _, k := range tKeywords {
			if strings.Contains(lowerName, k) {
				side = "T"
				break
			}
		}
	}

	return config.Agent{
		Name:        item.Name,
		Side:        side,
		Affiliation: item.Weapon, // Use weapon field for affiliation if provided
		Rarity:      item.Rarity,
		Collection:  item.Collection,
		URL:         item.URL,
		Price:       SummarizePriceSimple(item.Prices),
		Prices:      item.Prices,
	}
}

// SummarizePrices builds a Price range from multiple market entries.
func SummarizePrices(mps []config.MarketPrice) config.Price {
	var min, max float64
	var currency string = "USD"
	first := true

	for _, mp := range mps {
		if !mp.HasPrice || mp.Price <= 0 {
			continue
		}
		if first {
			min = mp.Price
			max = mp.Price
			currency = mp.Currency
			first = false
			continue
		}
		if mp.Price < min {
			min = mp.Price
		}
		if mp.Price > max {
			max = mp.Price
		}
	}

	return config.Price{
		PriceString: fmt.Sprintf("$%.2f - $%.2f", min, max),
		Currency:    currency,
		Min: config.PriceValue{
			Value: min,
			Unit:  currency,
		},
		Max: config.PriceValue{
			Value: max,
			Unit:  currency,
		},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// SummarizePriceSimple builds a single PriceSimple entry using the lowest price found.
func SummarizePriceSimple(mps []config.MarketPrice) config.PriceSimple {
	var lowest float64
	var currency string = "USD"
	first := true

	for _, mp := range mps {
		if mp.HasPrice && mp.Price > 0 {
			if first || mp.Price < lowest {
				lowest = mp.Price
				currency = mp.Currency
				first = false
			}
		}
	}

	return config.PriceSimple{
		PriceString: fmt.Sprintf("$%.2f", lowest),
		Currency:    currency,
		From: config.PriceValue{
			Value: lowest,
			Unit:  currency,
		},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
