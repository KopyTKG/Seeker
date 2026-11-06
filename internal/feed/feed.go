package feed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode"

	"github.com/kopytkg/golog/v2"
)

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

type Data struct {
	ID          string   `json:"cve_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Score       float32  `json:"cvss_score"`
	Severity    string   `json:"cvss_severity"`
	Vector      string   `json:"cvss_vector"`
	Weaknesses  []string `json:"weaknesses"`
	Vendors     []string `json:"vendors"`
	Link        string   `json:"link"`
}

func capitalize(str string) string {
	if len(str) == 0 {
		return ""
	}

	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func (d *Data) ToString() string {
	m, err := json.Marshal(d)
	if err != nil {
		golog.Errorf("Error during object to string parsing: %s", err)
		return ""
	}
	return string(m)
}

func (d *Data) VendorsToString() string {
	s := ""
	m := len(d.Vendors) - 1
	for i, v := range d.Vendors {
		if i < m {
			s = s + capitalize(v) + ", "
		} else {
			s = s + capitalize(v)
		}
	}
	return s
}

func DateToCET(date string) string {
	prague, err := time.LoadLocation("Europe/Prague")
	if err != nil {
		golog.Errorf("Error during timezone setup: %s", err)
	}

	layout := "2006-01-02T15:04:05Z07:00"
	time, err := time.Parse(layout, date)
	if err != nil {
		golog.Errorf("Error during datetime parsing: %s", err)
		return ""
	}

	return time.In(prague).Local().String()
}

type Query struct {
	Meta    Meta    `json:"meta"`
	Matches []Match `json:"matches"`
	Data    []Data  `json:"data"`
}

func PullFeed(uri string) (*Query, error) {
	if uri == "" {
		return nil, fmt.Errorf("URL is empty %s", "")
	}

	res, err := http.Get(uri)

	if err != nil {
		return nil, fmt.Errorf("Error during api fetch: %s", err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error during body fetching: %s", err)
	}

	var j Query

	if err = json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("Error during body parsing: %s", err)
	}

	return &j, nil
}
