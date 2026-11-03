package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eovacius/csgodatabase-scraper/scraper/config"
)

// fetchUsdRubRate fetches the official USD->RUB rate from the Central Bank of
// Russia (cbr.ru). The XML is windows-1251 encoded, but the numeric values are
// plain ASCII, so a byte-level regexp is safe here. Non-fatal: returns nil if
// the rate cannot be fetched or parsed.
func fetchUsdRubRate() *config.CurrencyInfo {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://www.cbr.ru/scripts/XML_daily.asp")
	if err != nil {
		fmt.Printf("\033[33m[!]\033[0m CBR rate unavailable: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`(?s)<CharCode>\s*USD\s*</CharCode>.*?<VunitRate>\s*([0-9]+[.,][0-9]+)`)
	m := re.FindSubmatch(body)
	if m == nil {
		fmt.Println("\033[33m[!]\033[0m CBR rate: USD not found in XML")
		return nil
	}
	rate, err := strconv.ParseFloat(strings.Replace(string(m[1]), ",", ".", 1), 64)
	if err != nil || rate <= 0 {
		return nil
	}

	fmt.Printf("\033[32m[+]\033[0m CBR USD->RUB rate: %.4f\n", rate)
	return &config.CurrencyInfo{
		USDToRUB:  rate,
		Source:    "cbr.ru",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
