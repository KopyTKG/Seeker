package feed

type Meta struct {
	CanonicalQuery string   `json:"canonical_query"`
	CanonicalTerm  []string `json:"canonical_term"`
	CacheTTL       int      `json:"cache_ttl"`
	Count          int      `json:"count"`
	Limit          int      `json:"limit"`
	UnmatchedTerms []string `json:"unmatched_terms"`
	AmbiquousTerms []string `json:"ambiquous_terms"`
}

type Match struct {
	Input      string   `json:"input"`
	Normalized string   `json:"normalized"`
	Matched    []Search `json:"matched"`
	Ambiquos   bool     `json:"ambiquos"`
	Unmatched  bool     `json:"unmatched"`
}

type Search struct {
	Type       string  `json:"type"`
	Vendor     string  `json:"vendor"`
	Product    string  `json:"product"`
	Source     string  `json:"source"`
	Confidence float32 `json:"confidence"`
}

type Query struct {
	Meta    Meta    `json:"meta"`
	Matches []Match `json:"matches"`
	Data    []Data  `json:"data"`
}

type VendorMatch struct {
	Vendor     string
	Input      string
	Type       string
	Confidence float32
	Product    string
}
