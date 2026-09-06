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
		Rarity:     item.Rarity,
		Collection: item.Collection,
		URL:        item.URL,
		Price:      SummarizePrices(item.Prices),
	}
}

// ConvertToAgent maps a universal Item to the legacy Agent struct.
func ConvertToAgent(item config.Item) config.Agent {
	side := "Unknown"
	if strings.Contains(item.Name, "FBI") || strings.Contains(item.Name, "Seal") || strings.Contains(item.Name, "SWAT") {
		side = "CT"
	} else if strings.Contains(item.Name, "Guerrilla") || strings.Contains(item.Name, "Professional") || strings.Contains(item.Name, "Phoenix") {
		side = "T"
	}

	return config.Agent{
		Name:       item.Name,
		Side:       side,
		Rarity:     item.Rarity,
		Collection: item.Collection,
		URL:        item.URL,
		Price:      SummarizePriceSimple(item.Prices),
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
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}

// SummarizePriceSimple builds a single PriceSimple entry.
func SummarizePriceSimple(mps []config.MarketPrice) config.PriceSimple {
	var val float64
	var currency string = "USD"
	for _, mp := range mps {
		if mp.HasPrice && mp.Price > 0 {
			val = mp.Price
			currency = mp.Currency
			break
		}
	}
	return config.PriceSimple{
		PriceString: fmt.Sprintf("$%.2f", val),
		Currency:    currency,
		From: config.PriceValue{
			Value: val,
			Unit:  currency,
		},
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}
