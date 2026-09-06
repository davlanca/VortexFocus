package scraper

import (
	_ "embed"
)

//go:embed config/scripts/script.js
var ScriptJS string

//go:embed config/scripts/config.js
var ConfigJS string

//go:embed config/scripts/prices.js
var PricesJS string

//go:embed config/scripts/discovery.js
var DiscoveryJS string

//go:embed config/scripts/cases.js
var CasesJS string

//go:embed config/scripts/case_prices.js
var CasePricesJS string

//go:embed config/scripts/gloves.js
var GlovesJS string

//go:embed config/scripts/glove_prices.js
var GlovePricesJS string

//go:embed config/scripts/agents.js
var AgentsJS string
