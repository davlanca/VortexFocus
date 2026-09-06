package config

type Skin struct {
	Name       string        `json:"name"`
	Weapon     string        `json:"weapon,omitempty"`
	Type       string        `json:"type,omitempty"`
	Rarity     string        `json:"rarity"`
	Collection string        `json:"collection"`
	Price      Price         `json:"price"`
	Prices     []MarketPrice `json:"prices,omitempty"`
	URL        string        `json:"url"`
}

type Agent struct {
	Name        string        `json:"name"`
	Affiliation string        `json:"affiliation"`
	Side        string        `json:"side"`
	Collection  string        `json:"collection"`
	Rarity      string        `json:"rarity"`
	Price       PriceSimple   `json:"price"`
	Prices      []MarketPrice `json:"prices,omitempty"`
	URL         string        `json:"url"`
}

type DataOutput struct {
	Skins  []Skin  `json:"skins"`
	Agents []Agent `json:"agents"`
	Misc   []Skin  `json:"misc"`
}

type PriceValue struct {
	Value         float64 `json:"value"`
	StattrakValue float64 `json:"stattrak_value"`
	Unit          string  `json:"unit"`
}

type Price struct {
	PriceString         string     `json:"price_string"`
	PriceStattrakString string     `json:"price_stattrak_string"`
	Currency            string     `json:"currency"`
	Min                 PriceValue `json:"min"`
	Max                 PriceValue `json:"max"`
	UpdatedAt           string     `json:"updated_at"`
}

type PriceSimple struct {
	PriceString string     `json:"price_string"`
	Currency    string     `json:"currency"`
	From        PriceValue `json:"starts_from"`
	UpdatedAt   string     `json:"updated_at"`
}

type Item struct {
	Name       string        `json:"name"`
	Slug       string        `json:"slug"`
	URL        string        `json:"url"`
	ImageURL   string        `json:"image_url,omitempty"`
	Category   string        `json:"category"`
	Weapon     string        `json:"weapon,omitempty"`
	Rarity     string        `json:"rarity,omitempty"`
	Collection string        `json:"collection,omitempty"`
	Type       string        `json:"type,omitempty"`
	HasWear    bool          `json:"has_wear"`
	Prices     []MarketPrice `json:"prices"`
	ScrapedAt  string        `json:"scraped_at"`
}

type MarketPrice struct {
	Market   string  `json:"market"`
	Wear     string  `json:"wear,omitempty"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	HasPrice bool    `json:"has_price"`
	URL      string  `json:"url"`
	Lowest   bool    `json:"lowest,omitempty"`
}

type WearCondition string
const (
	WearFactoryNew    WearCondition = "FN"
	WearMinimalWear   WearCondition = "MW"
	WearFieldTested   WearCondition = "FT"
	WearWellWorn      WearCondition = "WW"
	WearBattleScarred WearCondition = "BS"
)
var AllWearConditions = []WearCondition{WearFactoryNew, WearMinimalWear, WearFieldTested, WearWellWorn, WearBattleScarred}
