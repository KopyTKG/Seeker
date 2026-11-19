package feed

import (
	"encoding/json"

	"github.com/kopytkg/golog/v2"
)

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
