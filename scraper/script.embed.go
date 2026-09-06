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
